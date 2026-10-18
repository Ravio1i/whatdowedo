// Package web serves the HTML UI and the JSON API.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ravio1i/whatdowedo/internal/suggest"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

// Server holds handlers.
type Server struct {
	svc  *suggest.Service
	tpl  *template.Template
	log  *slog.Logger
	tags []string
}

// New builds the HTTP handler.
func New(svc *suggest.Service, log *slog.Logger) http.Handler {
	s := &Server{
		svc:  svc,
		log:  log,
		tpl:  template.Must(template.ParseFS(templateFS, "templates/*.html")),
		tags: []string{"indoor", "outdoor", "kids", "budget", "food", "culture", "active", "relax", "nightlife", "date night"},
	}
	mux := http.NewServeMux()
	static, _ := fs.Sub(staticFS, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("POST /suggest", s.suggestHTML)
	mux.HandleFunc("POST /api/suggest", s.suggestJSON)
	mux.HandleFunc("GET /api/suggest", s.suggestJSON)
	return logging(log, mux)
}

type pageData struct {
	Today string
	Tags  []string
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	s.render(w, "index.html", pageData{Today: time.Now().Format("2006-01-02"), Tags: s.tags})
}

func (s *Server) suggestHTML(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	req := suggest.Request{
		Date:        r.FormValue("date"),
		Location:    r.FormValue("location"),
		Preferences: r.FormValue("preferences"),
		Tags:        r.Form["tags"],
	}
	resp, err := s.svc.Suggest(r.Context(), req)
	if err != nil {
		w.WriteHeader(statusFor(err))
		s.render(w, "error.html", map[string]string{"Error": err.Error()})
		return
	}
	s.render(w, "results.html", resp)
}

func (s *Server) suggestJSON(w http.ResponseWriter, r *http.Request) {
	var req suggest.Request
	if r.Method == http.MethodGet {
		q := r.URL.Query()
		req = suggest.Request{Date: q.Get("date"), Location: q.Get("location"), Preferences: q.Get("preferences"), Tags: q["tags"]}
		if t := q.Get("tags"); t != "" && strings.Contains(t, ",") {
			req.Tags = strings.Split(t, ",")
		}
	} else if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	resp, err := s.svc.Suggest(r.Context(), req)
	if err != nil {
		writeJSON(w, statusFor(err), map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func statusFor(err error) int {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "required"), strings.Contains(msg, "must be"), strings.Contains(msg, "too long"), strings.Contains(msg, "could not find location"):
		return http.StatusBadRequest
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout
	}
	return http.StatusBadGateway
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tpl.ExecuteTemplate(w, name, data); err != nil {
		s.log.Error("template", "name", name, "err", err)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, code: 200}
		next.ServeHTTP(rw, r)
		if !strings.HasPrefix(r.URL.Path, "/static/") && r.URL.Path != "/healthz" {
			log.Info("http", "method", r.Method, "path", r.URL.Path, "status", rw.code, "dur", time.Since(start).Round(time.Millisecond))
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (s *statusWriter) WriteHeader(c int) { s.code = c; s.ResponseWriter.WriteHeader(c) }
