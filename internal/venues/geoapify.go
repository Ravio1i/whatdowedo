package venues

import (
	"context"
	"fmt"
	"net/url"

	"strings"
	"time"
)
import "github.com/ravio1i/whatdowedo/internal/httpx"

// Geoapify categories chosen for "things to do". Works well in Europe (OSM based).
const geoapifyCategories = "entertainment,leisure,tourism.attraction,tourism.sights,catering.cafe,catering.restaurant,sport.swimming_pool,sport.fitness,natural,commercial.shopping_mall"

type geoapifyResp struct {
	Features []struct {
		Properties struct {
			Name         string   `json:"name"`
			Formatted    string   `json:"formatted"`
			Categories   []string `json:"categories"`
			OpeningHours string   `json:"opening_hours"`
			Website      string   `json:"website"`
			Lat          float64  `json:"lat"`
			Lon          float64  `json:"lon"`
		} `json:"properties"`
	} `json:"features"`
}

// Geoapify fetches venues around lat/lon within radiusM.
func Geoapify(ctx context.Context, apiKey string, lat, lon float64, radiusM int, date time.Time, holiday bool) ([]Venue, error) {
	q := url.Values{
		"categories": {geoapifyCategories},
		"filter":     {fmt.Sprintf("circle:%.5f,%.5f,%d", lon, lat, radiusM)},
		"bias":       {fmt.Sprintf("proximity:%.5f,%.5f", lon, lat)},
		"limit":      {"60"},
		"apiKey":     {apiKey},
	}
	var r geoapifyResp
	if err := httpx.GetJSON(ctx, "https://api.geoapify.com/v2/places?"+q.Encode(), nil, &r); err != nil {
		return nil, fmt.Errorf("geoapify: %w", err)
	}
	out := make([]Venue, 0, len(r.Features))
	for _, f := range r.Features {
		p := f.Properties
		if p.Name == "" {
			continue
		}
		v := Venue{
			Name: p.Name, Address: p.Formatted, OpeningHours: p.OpeningHours,
			Website: p.Website, Source: "geoapify", Lat: p.Lat, Lon: p.Lon,
			Category: mostSpecificCategory(p.Categories),
		}
		v.Status, v.HoursOnDate = EvaluateOSMHours(p.OpeningHours, date, holiday)
		out = append(out, v)
	}
	return out, nil
}

func mostSpecificCategory(cats []string) string {
	best := ""
	for _, c := range cats {
		if strings.HasPrefix(c, "building") || strings.HasPrefix(c, "wheelchair") || strings.HasPrefix(c, "internet_access") || strings.HasPrefix(c, "access") || strings.HasPrefix(c, "fee") || strings.HasPrefix(c, "no_fee") {
			continue
		}
		if strings.Count(c, ".") >= strings.Count(best, ".") {
			best = c
		}
	}
	return strings.ReplaceAll(best, ".", " / ")
}
