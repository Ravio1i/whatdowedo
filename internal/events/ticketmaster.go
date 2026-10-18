// Package events fetches local events from the Ticketmaster Discovery API.
package events

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ravio1i/whatdowedo/internal/httpx"
)

// Event is a normalised event.
type Event struct {
	Name      string  `json:"name"`
	Category  string  `json:"category,omitempty"`
	Genre     string  `json:"genre,omitempty"`
	Venue     string  `json:"venue,omitempty"`
	City      string  `json:"city,omitempty"`
	Start     string  `json:"start"` // local time, e.g. 2026-10-18 20:00
	URL       string  `json:"url,omitempty"`
	PriceFrom float64 `json:"price_from,omitempty"`
	Currency  string  `json:"currency,omitempty"`
	Source    string  `json:"source"`
}

type tmResp struct {
	Embedded struct {
		Events []struct {
			Name  string `json:"name"`
			URL   string `json:"url"`
			Dates struct {
				Start struct {
					LocalDate string `json:"localDate"`
					LocalTime string `json:"localTime"`
				} `json:"start"`
			} `json:"dates"`
			Classifications []struct {
				Segment struct{ Name string } `json:"segment"`
				Genre   struct{ Name string } `json:"genre"`
			} `json:"classifications"`
			PriceRanges []struct {
				Min      float64 `json:"min"`
				Currency string  `json:"currency"`
			} `json:"priceRanges"`
			Embedded struct {
				Venues []struct {
					Name string                `json:"name"`
					City struct{ Name string } `json:"city"`
				} `json:"venues"`
			} `json:"_embedded"`
		} `json:"events"`
	} `json:"_embedded"`
}

// Ticketmaster queries events on the given date within radiusKM of lat/lon.
func Ticketmaster(ctx context.Context, apiKey string, lat, lon float64, date time.Time, radiusKM int, loc *time.Location) ([]Event, error) {
	start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc).UTC()
	end := start.Add(24*time.Hour - time.Second)
	q := url.Values{
		"apikey":        {apiKey},
		"latlong":       {fmt.Sprintf("%.4f,%.4f", lat, lon)},
		"radius":        {strconv.Itoa(radiusKM)},
		"unit":          {"km"},
		"startDateTime": {start.Format("2006-01-02T15:04:05Z")},
		"endDateTime":   {end.Format("2006-01-02T15:04:05Z")},
		"size":          {"25"},
		"sort":          {"date,asc"},
		"locale":        {"*"},
	}
	var r tmResp
	if err := httpx.GetJSON(ctx, "https://app.ticketmaster.com/discovery/v2/events.json?"+q.Encode(), nil, &r); err != nil {
		return nil, fmt.Errorf("ticketmaster: %w", err)
	}
	out := make([]Event, 0, len(r.Embedded.Events))
	for _, e := range r.Embedded.Events {
		ev := Event{Name: e.Name, URL: e.URL, Source: "ticketmaster"}
		ev.Start = strings.TrimSpace(e.Dates.Start.LocalDate + " " + strings.TrimSuffix(e.Dates.Start.LocalTime, ":00"))
		if len(e.Classifications) > 0 {
			ev.Category = e.Classifications[0].Segment.Name
			ev.Genre = e.Classifications[0].Genre.Name
		}
		if len(e.PriceRanges) > 0 {
			ev.PriceFrom = e.PriceRanges[0].Min
			ev.Currency = e.PriceRanges[0].Currency
		}
		if len(e.Embedded.Venues) > 0 {
			ev.Venue = e.Embedded.Venues[0].Name
			ev.City = e.Embedded.Venues[0].City.Name
		}
		out = append(out, ev)
	}
	return out, nil
}
