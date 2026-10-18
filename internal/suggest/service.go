package suggest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ravio1i/whatdowedo/internal/config"
	"github.com/ravio1i/whatdowedo/internal/events"
	"github.com/ravio1i/whatdowedo/internal/geo"
	"github.com/ravio1i/whatdowedo/internal/llm"
	"github.com/ravio1i/whatdowedo/internal/venues"
	"github.com/ravio1i/whatdowedo/internal/weather"
)

// Service wires providers together.
type Service struct {
	cfg   config.Config
	llm   llm.Client
	cache *cache
	log   *slog.Logger
}

// New builds a Service.
func New(cfg config.Config, log *slog.Logger) *Service {
	return &Service{
		cfg:   cfg,
		llm:   llm.Client{BaseURL: cfg.LLMBaseURL, APIKey: cfg.LLMAPIKey, Model: cfg.LLMModel},
		cache: newCache(cfg.CacheTTL),
		log:   log,
	}
}

// Validate checks and normalises the request.
func (r *Request) Validate() (time.Time, error) {
	r.Location = strings.TrimSpace(r.Location)
	r.Preferences = strings.TrimSpace(r.Preferences)
	if r.Location == "" {
		return time.Time{}, errors.New("location is required")
	}
	if len(r.Location) > 120 || len(r.Preferences) > 1000 {
		return time.Time{}, errors.New("input too long")
	}
	if r.Date == "" {
		r.Date = time.Now().Format("2006-01-02")
	}
	d, err := time.Parse("2006-01-02", r.Date)
	if err != nil {
		return time.Time{}, errors.New("date must be YYYY-MM-DD")
	}
	clean := r.Tags[:0]
	for _, t := range r.Tags {
		if t = strings.ToLower(strings.TrimSpace(t)); t != "" {
			clean = append(clean, t)
		}
	}
	r.Tags = clean
	return d, nil
}

// Suggest runs the full pipeline.
func (s *Service) Suggest(ctx context.Context, req Request) (Response, error) {
	date, err := req.Validate()
	if err != nil {
		return Response{}, err
	}
	key := fmt.Sprintf("%s|%s|%s|%s", req.Date, strings.ToLower(req.Location), strings.ToLower(req.Preferences), strings.Join(req.Tags, ","))
	if r, ok := s.cache.get(key); ok {
		r.Cached = true
		return r, nil
	}

	place, err := geo.Resolve(ctx, req.Location)
	if err != nil {
		return Response{}, fmt.Errorf("could not find location %q", req.Location)
	}
	loc, err := time.LoadLocation(place.Timezone)
	if err != nil {
		loc = time.UTC
	}
	localDate := time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, loc)

	wctx := Context{Place: place, Date: req.Date, Weekday: localDate.Weekday().String()}
	wctx.Providers, wctx.Weather, wctx.Events, wctx.Venues = s.gather(ctx, place, localDate)

	resp := Response{Context: wctx, LLMModel: s.llm.Model}
	if err := s.generate(ctx, req, &resp); err != nil {
		return Response{}, err
	}
	s.cache.set(key, resp)
	return resp, nil
}

// gather fans out to all providers concurrently, each with its own timeout.
func (s *Service) gather(ctx context.Context, place geo.Place, date time.Time) ([]ProviderStatus, weather.Forecast, []events.Event, []venues.Venue) {
	var (
		mu       sync.Mutex
		statuses []ProviderStatus
		wg       sync.WaitGroup
		wx       weather.Forecast
		evs      []events.Event
		vGeo     []venues.Venue
		vOSM     []venues.Venue
		vGoogle  []venues.Venue
		isHolday = false // public holiday lookup is a candidate follow-up (Nager.Date)
	)
	add := func(st ProviderStatus) {
		mu.Lock()
		statuses = append(statuses, st)
		mu.Unlock()
	}
	run := func(name string, enabled bool, skipReason string, fn func(context.Context) (int, error)) {
		if !enabled {
			add(ProviderStatus{Name: name, Reason: skipReason})
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			pctx, cancel := context.WithTimeout(ctx, s.cfg.ProviderTimeout)
			defer cancel()
			n, err := fn(pctx)
			if err != nil {
				s.log.Warn("provider failed", "provider", name, "err", err)
				add(ProviderStatus{Name: name, Reason: trimErr(err)})
				return
			}
			add(ProviderStatus{Name: name, Used: true, Count: n})
		}()
	}

	run("open-meteo", true, "", func(c context.Context) (int, error) {
		var err error
		wx, err = weather.Get(c, place.Latitude, place.Longitude, date, place.Timezone)
		if wx.Available {
			return 1, err
		}
		return 0, err
	})
	run("ticketmaster", s.cfg.TicketmasterAPIKey != "", "TICKETMASTER_API_KEY not set", func(c context.Context) (int, error) {
		var err error
		evs, err = events.Ticketmaster(c, s.cfg.TicketmasterAPIKey, place.Latitude, place.Longitude, date, s.cfg.EventRadiusKM, date.Location())
		return len(evs), err
	})
	useGeoapify := s.cfg.GeoapifyAPIKey != ""
	run("geoapify", useGeoapify, "GEOAPIFY_API_KEY not set", func(c context.Context) (int, error) {
		var err error
		vGeo, err = venues.Geoapify(c, s.cfg.GeoapifyAPIKey, place.Latitude, place.Longitude, s.cfg.VenueRadiusM, date, isHolday)
		return len(vGeo), err
	})
	// Overpass is the keyless fallback; only hit the public instance when Geoapify is absent.
	run("openstreetmap-overpass", !useGeoapify, "skipped because Geoapify is configured", func(c context.Context) (int, error) {
		var err error
		vOSM, err = venues.Overpass(c, place.Latitude, place.Longitude, s.cfg.VenueRadiusM, date, isHolday)
		return len(vOSM), err
	})
	run("google-places", s.cfg.GooglePlacesAPIKey != "", "GOOGLE_PLACES_API_KEY not set", func(c context.Context) (int, error) {
		var err error
		vGoogle, err = venues.Google(c, s.cfg.GooglePlacesAPIKey, place.Latitude, place.Longitude, s.cfg.VenueRadiusM, date)
		return len(vGoogle), err
	})
	wg.Wait()
	if evs == nil {
		evs = []events.Event{}
	}
	sort.Slice(statuses, func(i, j int) bool { return statuses[i].Name < statuses[j].Name })
	return statuses, wx, evs, venues.Merge(40, vGoogle, vGeo, vOSM)
}

const systemPrompt = `You are a local activity planner. You receive structured facts (location, date, weather forecast, events happening that day, nearby venues with their opening status) and the user's preferences.
Propose 5 to 8 concrete, varied activities for that specific day and place. Ground every suggestion in the provided facts where possible: prefer venues whose status_on_date is "open" and events listed; never recommend a venue marked "closed". If the weather is bad, favour indoor options and say so. Respect the user's preferences and tags strictly. If facts are thin, fall back to well-known typical activities for that kind of place and say the availability is unverified.
Answer ONLY with a JSON object of this exact shape:
{"summary": "one or two sentences about the day",
 "suggestions": [{"title": "", "description": "2-3 sentences", "category": "culture|outdoor|food|nightlife|sport|family|relax|shopping|event", "time_of_day": "morning|afternoon|evening|all day", "indoor": true, "why": "why it fits weather/preferences", "estimated_cost": "free|€|€€|€€€", "sources": ["venue or event names from the facts"], "url": "link from the facts if any, else empty"}],
 "caveats": ["short notes, e.g. unverified opening hours, forecast horizon"]}
Write in the same language the user wrote their preferences in; default to English.`

func (s *Service) generate(ctx context.Context, req Request, resp *Response) error {
	facts := map[string]any{
		"place":       resp.Context.Place,
		"date":        resp.Context.Date,
		"weekday":     resp.Context.Weekday,
		"weather":     resp.Context.Weather,
		"events":      resp.Context.Events,
		"venues":      resp.Context.Venues,
		"preferences": req.Preferences,
		"tags":        req.Tags,
	}
	factsJSON, _ := json.Marshal(facts)
	user := "FACTS:\n" + string(factsJSON) + "\n\nUser preferences: " + orNone(req.Preferences) + "\nTags: " + orNone(strings.Join(req.Tags, ", "))

	lctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	raw, err := s.llm.CompleteJSON(lctx, systemPrompt, user)
	if err != nil {
		if !s.llm.Configured() {
			// Degrade gracefully: return the facts with a rule-based summary.
			resp.Summary = "LLM not configured; showing raw facts only."
			resp.Caveats = []string{"Set LLM_API_KEY to get generated suggestions."}
			return nil
		}
		return fmt.Errorf("suggestion generation failed: %w", err)
	}
	var out struct {
		Summary     string       `json:"summary"`
		Suggestions []Suggestion `json:"suggestions"`
		Caveats     []string     `json:"caveats"`
	}
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		s.log.Warn("llm returned non-json", "err", err, "raw", truncate(raw, 400))
		resp.Summary = strings.TrimSpace(raw)
		resp.Caveats = []string{"Model response could not be parsed as JSON; shown verbatim."}
		return nil
	}
	resp.Summary, resp.Suggestions, resp.Caveats = out.Summary, out.Suggestions, out.Caveats
	return nil
}

// extractJSON tolerates models that wrap JSON in code fences or prose.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "{"); i >= 0 {
		if j := strings.LastIndex(s, "}"); j > i {
			return s[i : j+1]
		}
	}
	return s
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func trimErr(err error) string { return truncate(err.Error(), 160) }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
