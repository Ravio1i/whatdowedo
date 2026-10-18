// Package weather fetches a daily forecast from Open-Meteo (keyless).
package weather

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/ravio1i/whatdowedo/internal/httpx"
)

// Forecast is the daily summary for one date.
type Forecast struct {
	Date             string  `json:"date"`
	Summary          string  `json:"summary"`
	TempMaxC         float64 `json:"temp_max_c"`
	TempMinC         float64 `json:"temp_min_c"`
	PrecipitationMM  float64 `json:"precipitation_mm"`
	PrecipProbMaxPct int     `json:"precip_prob_max_pct"`
	WindMaxKMH       float64 `json:"wind_max_kmh"`
	SunshineHours    float64 `json:"sunshine_hours"`
	Sunrise          string  `json:"sunrise"`
	Sunset           string  `json:"sunset"`
	UVIndexMax       float64 `json:"uv_index_max"`
	WeatherCode      int     `json:"weather_code"`
	Available        bool    `json:"available"`
	Note             string  `json:"note,omitempty"`
}

type omResp struct {
	Daily struct {
		Time             []string  `json:"time"`
		WeatherCode      []int     `json:"weather_code"`
		TempMax          []float64 `json:"temperature_2m_max"`
		TempMin          []float64 `json:"temperature_2m_min"`
		Precip           []float64 `json:"precipitation_sum"`
		PrecipProb       []*int    `json:"precipitation_probability_max"`
		WindMax          []float64 `json:"wind_speed_10m_max"`
		SunshineDuration []float64 `json:"sunshine_duration"`
		Sunrise          []string  `json:"sunrise"`
		Sunset           []string  `json:"sunset"`
		UV               []float64 `json:"uv_index_max"`
	} `json:"daily"`
}

// MaxForecastDays is Open-Meteo's forecast horizon.
const MaxForecastDays = 16

// Get returns the forecast for date (YYYY-MM-DD) at the given coordinates.
// Dates beyond the forecast horizon return Available=false with a note.
func Get(ctx context.Context, lat, lon float64, date time.Time, tz string) (Forecast, error) {
	f := Forecast{Date: date.Format("2006-01-02")}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	diff := int(date.Sub(today).Hours() / 24)
	if diff < -7 {
		f.Note = "date is in the past; no forecast available"
		return f, nil
	}
	if diff > MaxForecastDays-1 {
		f.Note = fmt.Sprintf("date is more than %d days ahead; no forecast available, plan for seasonal conditions", MaxForecastDays)
		return f, nil
	}
	q := url.Values{
		"latitude":   {strconv.FormatFloat(lat, 'f', 4, 64)},
		"longitude":  {strconv.FormatFloat(lon, 'f', 4, 64)},
		"daily":      {"weather_code,temperature_2m_max,temperature_2m_min,precipitation_sum,precipitation_probability_max,wind_speed_10m_max,sunshine_duration,sunrise,sunset,uv_index_max"},
		"timezone":   {tz},
		"start_date": {f.Date},
		"end_date":   {f.Date},
	}
	if diff < 0 {
		q.Set("past_days", "7")
	}
	var r omResp
	if err := httpx.GetJSON(ctx, "https://api.open-meteo.com/v1/forecast?"+q.Encode(), nil, &r); err != nil {
		return f, fmt.Errorf("open-meteo: %w", err)
	}
	for i, d := range r.Daily.Time {
		if d != f.Date {
			continue
		}
		f.Available = true
		f.WeatherCode = at(r.Daily.WeatherCode, i)
		f.TempMaxC = at(r.Daily.TempMax, i)
		f.TempMinC = at(r.Daily.TempMin, i)
		f.PrecipitationMM = at(r.Daily.Precip, i)
		if i < len(r.Daily.PrecipProb) && r.Daily.PrecipProb[i] != nil {
			f.PrecipProbMaxPct = *r.Daily.PrecipProb[i]
		}
		f.WindMaxKMH = at(r.Daily.WindMax, i)
		f.SunshineHours = at(r.Daily.SunshineDuration, i) / 3600
		f.Sunrise = clock(at(r.Daily.Sunrise, i))
		f.Sunset = clock(at(r.Daily.Sunset, i))
		f.UVIndexMax = at(r.Daily.UV, i)
		f.Summary = Describe(f.WeatherCode)
	}
	if !f.Available {
		f.Note = "no forecast data returned for this date"
	}
	return f, nil
}

func at[T any](s []T, i int) T {
	var zero T
	if i < len(s) {
		return s[i]
	}
	return zero
}

func clock(iso string) string {
	if t, err := time.Parse("2006-01-02T15:04", iso); err == nil {
		return t.Format("15:04")
	}
	return iso
}

// Describe maps WMO weather codes to text.
func Describe(code int) string {
	switch {
	case code == 0:
		return "clear sky"
	case code <= 2:
		return "mainly clear / partly cloudy"
	case code == 3:
		return "overcast"
	case code == 45 || code == 48:
		return "fog"
	case code >= 51 && code <= 57:
		return "drizzle"
	case code >= 61 && code <= 67:
		return "rain"
	case code >= 71 && code <= 77:
		return "snow"
	case code >= 80 && code <= 82:
		return "rain showers"
	case code == 85 || code == 86:
		return "snow showers"
	case code >= 95:
		return "thunderstorm"
	}
	return "unknown"
}
