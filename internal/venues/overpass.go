package venues

import (
	"context"
	"fmt"
	"time"

	"github.com/ravio1i/whatdowedo/internal/httpx"
)

// Public Overpass instances, tried in order until one answers.
var overpassEndpoints = []string{
	"https://overpass.kumi.systems/api/interpreter",
	"https://overpass-api.de/api/interpreter",
	"https://maps.mail.ru/osm/tools/overpass/api/interpreter",
}

type overpassResp struct {
	Elements []struct {
		Type   string                      `json:"type"`
		Lat    float64                     `json:"lat"`
		Lon    float64                     `json:"lon"`
		Center *struct{ Lat, Lon float64 } `json:"center"`
		Tags   map[string]string           `json:"tags"`
	} `json:"elements"`
}

// Overpass queries OpenStreetMap directly (keyless). Used as fallback when
// Geoapify is not configured. Public Overpass instances are rate limited.
func Overpass(ctx context.Context, lat, lon float64, radiusM int, date time.Time, holiday bool) ([]Venue, error) {
	ql := fmt.Sprintf(`[out:json][timeout:10];
(
  nwr(around:%[1]d,%[2]f,%[3]f)["tourism"~"attraction|museum|gallery|zoo|theme_park|viewpoint|aquarium"]["name"];
  nwr(around:%[1]d,%[2]f,%[3]f)["leisure"~"park|garden|sports_centre|swimming_pool|bowling_alley|escape_game|ice_rink|water_park|miniature_golf|fitness_centre|nature_reserve"]["name"];
  nwr(around:%[1]d,%[2]f,%[3]f)["amenity"~"cinema|theatre|arts_centre|cafe|restaurant|biergarten|nightclub|planetarium|library|community_centre"]["name"];
);
out center tags 80;`, radiusM, lat, lon)
	var r overpassResp
	var err error
	for _, ep := range overpassEndpoints {
		if err = httpx.PostRaw(ctx, ep, "text/plain", []byte(ql), &r); err == nil {
			break
		}
		if ctx.Err() != nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("overpass: %w", err)
	}
	out := make([]Venue, 0, len(r.Elements))
	for _, e := range r.Elements {
		name := e.Tags["name"]
		if name == "" {
			continue
		}
		cat := ""
		for _, k := range []string{"tourism", "leisure", "amenity"} {
			if v := e.Tags[k]; v != "" {
				cat = k + " / " + v
				break
			}
		}
		v := Venue{
			Name: name, Category: cat, OpeningHours: e.Tags["opening_hours"],
			Website: firstNonEmpty(e.Tags["website"], e.Tags["contact:website"]),
			Address: joinAddr(e.Tags), Source: "openstreetmap", Lat: e.Lat, Lon: e.Lon,
		}
		if e.Center != nil {
			v.Lat, v.Lon = e.Center.Lat, e.Center.Lon
		}
		v.Status, v.HoursOnDate = EvaluateOSMHours(v.OpeningHours, date, holiday)
		out = append(out, v)
	}
	return out, nil
}

func joinAddr(t map[string]string) string {
	s := t["addr:street"]
	if n := t["addr:housenumber"]; n != "" {
		s += " " + n
	}
	if c := t["addr:city"]; c != "" {
		if s != "" {
			s += ", "
		}
		s += c
	}
	return s
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}
