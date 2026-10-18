// Package venues aggregates points of interest from several providers and
// estimates whether they are open on the requested date.
package venues

// OpenStatus is the best-effort availability on the requested date.
type OpenStatus string

const (
	StatusOpen    OpenStatus = "open"
	StatusClosed  OpenStatus = "closed"
	StatusUnknown OpenStatus = "unknown"
)

// Venue is a normalised point of interest.
type Venue struct {
	Name         string     `json:"name"`
	Category     string     `json:"category"`
	Address      string     `json:"address,omitempty"`
	OpeningHours string     `json:"opening_hours,omitempty"` // raw provider string
	Status       OpenStatus `json:"status_on_date"`
	HoursOnDate  string     `json:"hours_on_date,omitempty"`
	Rating       float64    `json:"rating,omitempty"`
	Website      string     `json:"website,omitempty"`
	Source       string     `json:"source"`
	Lat          float64    `json:"-"`
	Lon          float64    `json:"-"`
}
