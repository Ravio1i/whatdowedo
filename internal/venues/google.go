package venues

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ravio1i/whatdowedo/internal/httpx"
)

var googleTypes = []string{
	"tourist_attraction", "museum", "art_gallery", "park", "amusement_park", "zoo", "aquarium",
	"bowling_alley", "movie_theater", "performing_arts_theater", "spa", "cafe", "restaurant",
	"night_club", "hiking_area", "swimming_pool", "shopping_mall", "botanical_garden",
}

type googleResp struct {
	Places []struct {
		DisplayName         struct{ Text string }                 `json:"displayName"`
		FormattedAddress    string                                `json:"formattedAddress"`
		PrimaryType         string                                `json:"primaryType"`
		Rating              float64                               `json:"rating"`
		WebsiteURI          string                                `json:"websiteUri"`
		Location            struct{ Latitude, Longitude float64 } `json:"location"`
		RegularOpeningHours *struct {
			WeekdayDescriptions []string `json:"weekdayDescriptions"`
		} `json:"regularOpeningHours"`
	} `json:"places"`
}

// Google uses the Places API (New) Nearby Search.
func Google(ctx context.Context, apiKey string, lat, lon float64, radiusM int, date time.Time) ([]Venue, error) {
	if radiusM > 50000 {
		radiusM = 50000
	}
	body := map[string]any{
		"includedTypes":  googleTypes,
		"maxResultCount": 20,
		"rankPreference": "POPULARITY",
		"locationRestriction": map[string]any{
			"circle": map[string]any{
				"center": map[string]float64{"latitude": lat, "longitude": lon},
				"radius": float64(radiusM),
			},
		},
	}
	headers := map[string]string{
		"X-Goog-Api-Key":   apiKey,
		"X-Goog-FieldMask": "places.displayName,places.formattedAddress,places.primaryType,places.rating,places.websiteUri,places.location,places.regularOpeningHours.weekdayDescriptions",
	}
	var r googleResp
	if err := httpx.PostJSON(ctx, "https://places.googleapis.com/v1/places:searchNearby", headers, body, &r); err != nil {
		return nil, fmt.Errorf("google places: %w", err)
	}
	weekday := date.Weekday().String()
	out := make([]Venue, 0, len(r.Places))
	for _, p := range r.Places {
		v := Venue{
			Name: p.DisplayName.Text, Address: p.FormattedAddress, Rating: p.Rating,
			Website: p.WebsiteURI, Source: "google", Status: StatusUnknown,
			Category: strings.ReplaceAll(p.PrimaryType, "_", " "),
			Lat:      p.Location.Latitude, Lon: p.Location.Longitude,
		}
		if p.RegularOpeningHours != nil {
			v.OpeningHours = strings.Join(p.RegularOpeningHours.WeekdayDescriptions, "; ")
			for _, d := range p.RegularOpeningHours.WeekdayDescriptions {
				if strings.HasPrefix(d, weekday+":") {
					h := strings.TrimSpace(strings.TrimPrefix(d, weekday+":"))
					v.HoursOnDate = h
					if strings.EqualFold(h, "Closed") {
						v.Status = StatusClosed
					} else {
						v.Status = StatusOpen
					}
				}
			}
		}
		out = append(out, v)
	}
	return out, nil
}
