// Package config loads runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime settings. Empty provider keys mean "skip provider".
type Config struct {
	Port     string
	LogLevel string

	LLMBaseURL string
	LLMAPIKey  string
	LLMModel   string

	TicketmasterAPIKey string
	GeoapifyAPIKey     string
	GooglePlacesAPIKey string

	ProviderTimeout time.Duration
	CacheTTL        time.Duration
	VenueRadiusM    int
	EventRadiusKM   int
}

// Load reads the environment and applies defaults.
func Load() (Config, error) {
	c := Config{
		Port:               env("PORT", "8080"),
		LogLevel:           env("LOG_LEVEL", "info"),
		LLMBaseURL:         env("LLM_BASE_URL", "https://api.groq.com/openai/v1"),
		LLMAPIKey:          os.Getenv("LLM_API_KEY"),
		LLMModel:           env("LLM_MODEL", "openai/gpt-oss-120b"),
		TicketmasterAPIKey: os.Getenv("TICKETMASTER_API_KEY"),
		GeoapifyAPIKey:     os.Getenv("GEOAPIFY_API_KEY"),
		GooglePlacesAPIKey: os.Getenv("GOOGLE_PLACES_API_KEY"),
	}
	var err error
	if c.ProviderTimeout, err = time.ParseDuration(env("PROVIDER_TIMEOUT", "8s")); err != nil {
		return c, fmt.Errorf("PROVIDER_TIMEOUT: %w", err)
	}
	if c.CacheTTL, err = time.ParseDuration(env("CACHE_TTL", "10m")); err != nil {
		return c, fmt.Errorf("CACHE_TTL: %w", err)
	}
	if c.VenueRadiusM, err = strconv.Atoi(env("VENUE_RADIUS_M", "5000")); err != nil {
		return c, fmt.Errorf("VENUE_RADIUS_M: %w", err)
	}
	if c.EventRadiusKM, err = strconv.Atoi(env("EVENT_RADIUS_KM", "30")); err != nil {
		return c, fmt.Errorf("EVENT_RADIUS_KM: %w", err)
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
