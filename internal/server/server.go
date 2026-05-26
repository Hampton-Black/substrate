package server

import (
	"context"
	"encoding/json"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/Hampton-Black/substrate/internal/core"
	"github.com/Hampton-Black/substrate/web"
	"github.com/go-chi/chi/v5"
)

// Server serves the Substrate HTTP dashboard and JSON API.
type Server struct {
	svc  *core.Service
	log  *slog.Logger
	tmpl *template.Template
}

// New constructs an HTTP server backed by core.Service.
func New(svc *core.Service, log *slog.Logger) (*Server, error) {
	if log == nil {
		log = slog.Default()
	}
	tmpl, err := template.ParseFS(web.Templates, "templates/index.html")
	if err != nil {
		return nil, err
	}
	return &Server{svc: svc, log: log, tmpl: tmpl}, nil
}

// Handler returns the chi router for the server.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/", s.handleIndex)
	r.Get("/api/workstreams", s.handleAPIWorkstreams)
	r.Get("/api/gaps", s.handleAPIGaps)
	return r
}

// ListenAndServe starts the HTTP server until ctx is canceled.
func (s *Server) ListenAndServe(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

type indexData struct {
	GeneratedAt string
	Workstreams []workstreamView
	Gaps        []gapView
}

type workstreamView struct {
	PodID            string
	Title            string
	Status           string
	StatusClass      string
	Branch           string
	Intent           string
	LastActivityText string
}

type gapView struct {
	Priority      int
	PriorityClass string
	Category      string
	Frequency     int
	PodID         string
	Description   string
	OccurredText  string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	workstreams, err := s.svc.QueryActiveWork(r.Context(), core.WorkFilters{})
	if err != nil {
		s.log.Error("dashboard workstreams failed", "error", err)
		http.Error(w, "failed to load workstreams", http.StatusInternalServerError)
		return
	}

	gaps, err := s.svc.ListCapabilityGaps(r.Context(), core.GapFilters{
		Statuses: []core.GapStatus{core.GapOpen, core.GapAcknowledged},
	})
	if err != nil {
		s.log.Error("dashboard gaps failed", "error", err)
		http.Error(w, "failed to load gaps", http.StatusInternalServerError)
		return
	}

	data := indexData{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Workstreams: make([]workstreamView, 0, len(workstreams)),
		Gaps:        make([]gapView, 0, len(gaps)),
	}
	for _, ws := range workstreams {
		data.Workstreams = append(data.Workstreams, workstreamView{
			PodID:            ws.PodID,
			Title:            ws.Title,
			Status:           string(ws.Status),
			StatusClass:      statusBadgeClass(string(ws.Status)),
			Branch:           ws.Branch,
			Intent:           ws.Intent,
			LastActivityText: relativeTime(ws.LastActivity),
		})
	}
	for _, g := range gaps {
		data.Gaps = append(data.Gaps, gapView{
			Priority:      g.Priority,
			PriorityClass: priorityClass(g.Priority),
			Category:      string(g.Category),
			Frequency:     g.Frequency,
			PodID:         g.PodID,
			Description:   g.Description,
			OccurredText:  relativeTime(g.OccurredAt),
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, data); err != nil {
		s.log.Error("render dashboard failed", "error", err)
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleAPIWorkstreams(w http.ResponseWriter, r *http.Request) {
	f := core.WorkFilters{}
	if pod := r.URL.Query().Get("pod"); pod != "" {
		f.PodID = &pod
	}
	if status := r.URL.Query().Get("status"); status != "" {
		st := core.WorkstreamStatus(status)
		f.Status = &st
	}

	workstreams, err := s.svc.QueryActiveWork(r.Context(), f)
	if err != nil {
		s.log.Error("api workstreams failed", "error", err)
		http.Error(w, "failed to list workstreams", http.StatusInternalServerError)
		return
	}
	if workstreams == nil {
		workstreams = []core.Workstream{}
	}
	writeJSON(w, workstreams)
}

func (s *Server) handleAPIGaps(w http.ResponseWriter, r *http.Request) {
	f := core.GapFilters{}
	if pod := r.URL.Query().Get("pod"); pod != "" {
		f.PodID = &pod
	}
	if status := r.URL.Query().Get("status"); status != "" {
		for _, part := range splitComma(status) {
			st := core.GapStatus(part)
			f.Statuses = append(f.Statuses, st)
		}
	} else {
		f.Statuses = []core.GapStatus{core.GapOpen, core.GapAcknowledged}
	}

	gaps, err := s.svc.ListCapabilityGaps(r.Context(), f)
	if err != nil {
		s.log.Error("api gaps failed", "error", err)
		http.Error(w, "failed to list gaps", http.StatusInternalServerError)
		return
	}
	if gaps == nil {
		gaps = []core.CapabilityGap{}
	}
	writeJSON(w, gaps)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func statusBadgeClass(status string) string {
	switch status {
	case "active":
		return "badge-green"
	case "blocked":
		return "badge-red"
	case "review":
		return "badge-amber"
	default:
		return "badge-grey"
	}
}

func priorityClass(priority int) string {
	switch priority {
	case 1:
		return "priority-high"
	case 2:
		return "priority-medium"
	default:
		return "priority-low"
	}
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			if part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}
