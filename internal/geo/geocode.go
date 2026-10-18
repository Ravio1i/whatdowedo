// Package geo resolves free-text locations to coordinates via Open-Meteo geocoding.
package geo

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ravio1i/whatdowedo/internal/httpx"
)

// Place is a resolved location.
type Place struct {
	Name        string  `json:"name"`
	Admin       string  `json:"admin,omitempty"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Timezone    string  `json:"timezone"`
}

// Display returns "Name, Admin, Country".
func (p Place) Display() string {
	s := p.Name
	if p.Admin != "" && p.Admin != p.Name {
		s += ", " + p.Admin
	}
	if p.Country != "" {
		s += ", " + p.Country
	}
	return s
}

type geoResp struct {
	Results []struct {
		Name        string  `json:"name"`
		Admin1      string  `json:"admin1"`
		Country     string  `json:"country"`
		CountryCode string  `json:"country_code"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		Timezone    string  `json:"timezone"`
	} `json:"results"`
}

// Resolve geocodes a location name. Open-Meteo geocoding is keyless.
func Resolve(ctx context.Context, query string) (Place, error) {
	u := "https://geocoding-api.open-meteo.com/v1/search?" + url.Values{
		"name":     {query},
		"count":    {"1"},
		"language": {"en"},
		"format":   {"json"},
	}.Encode()
	var r geoResp
	if err := httpx.GetJSON(ctx, u, nil, &r); err != nil {
		return Place{}, fmt.Errorf("geocode: %w", err)
	}
	if len(r.Results) == 0 {
		return Place{}, fmt.Errorf("geocode: no results for %q", query)
	}
	x := r.Results[0]
	return Place{
		Name: x.Name, Admin: x.Admin1, Country: x.Country, CountryCode: x.CountryCode,
		Latitude: x.Latitude, Longitude: x.Longitude, Timezone: x.Timezone,
	}, nil
}
