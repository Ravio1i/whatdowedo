// Command server runs the whatdowedo HTTP service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ravio1i/whatdowedo/internal/config"
	"github.com/ravio1i/whatdowedo/internal/suggest"
	"github.com/ravio1i/whatdowedo/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	var lvl slog.Level
	_ = lvl.UnmarshalText([]byte(cfg.LogLevel))
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))

	svc := suggest.New(cfg, log)
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           web.New(svc, log),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Info("listening", "addr", srv.Addr, "llm_model", cfg.LLMModel, "llm_base_url", cfg.LLMBaseURL,
			"ticketmaster", cfg.TicketmasterAPIKey != "", "geoapify", cfg.GeoapifyAPIKey != "", "google_places", cfg.GooglePlacesAPIKey != "")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server", "err", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
