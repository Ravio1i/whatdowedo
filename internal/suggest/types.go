// Package suggest orchestrates data providers and the LLM into activity suggestions.
package suggest

import (
	"github.com/ravio1i/whatdowedo/internal/events"
	"github.com/ravio1i/whatdowedo/internal/geo"
	"github.com/ravio1i/whatdowedo/internal/venues"
	"github.com/ravio1i/whatdowedo/internal/weather"
)

// Request is what the user asks for.
type Request struct {
	Date        string   `json:"date"`     // YYYY-MM-DD
	Location    string   `json:"location"` // free text, e.g. "Berlin" or "Erlangen, Germany"
	Preferences string   `json:"preferences,omitempty"`
	Tags        []string `json:"tags,omitempty"` // e.g. indoor, outdoor, kids, budget, food, culture, active
}

// Suggestion is one proposed activity.
type Suggestion struct {
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Category      string   `json:"category"`
	TimeOfDay     string   `json:"time_of_day"` // morning, afternoon, evening, all day
	Indoor        bool     `json:"indoor"`
	Why           string   `json:"why"`
	EstimatedCost string   `json:"estimated_cost"`
	Sources       []string `json:"sources,omitempty"` // venue or event names used
	URL           string   `json:"url,omitempty"`
}

// ProviderStatus records what each provider contributed.
type ProviderStatus struct {
	Name   string `json:"name"`
	Used   bool   `json:"used"`
	Count  int    `json:"count,omitempty"`
	Reason string `json:"reason,omitempty"` // why skipped or failed
}

// Context is the gathered world state handed to the LLM and echoed to the client.
type Context struct {
	Place     geo.Place        `json:"place"`
	Date      string           `json:"date"`
	Weekday   string           `json:"weekday"`
	Weather   weather.Forecast `json:"weather"`
	Events    []events.Event   `json:"events"`
	Venues    []venues.Venue   `json:"venues"`
	Providers []ProviderStatus `json:"providers"`
}

// Response is the full result.
type Response struct {
	Summary     string       `json:"summary"`
	Suggestions []Suggestion `json:"suggestions"`
	Caveats     []string     `json:"caveats,omitempty"`
	Context     Context      `json:"context"`
	Cached      bool         `json:"cached"`
	LLMModel    string       `json:"llm_model"`
}
