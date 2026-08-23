// Package api exposes HTTP endpoints for the stacklift operator console.
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/lacsar712/stacklift/internal/alarm"
	"github.com/lacsar712/stacklift/internal/clock"
	"github.com/lacsar712/stacklift/internal/config"
	"github.com/lacsar712/stacklift/internal/crane"
	"github.com/lacsar712/stacklift/internal/interlock"
	"github.com/lacsar712/stacklift/internal/journal"
	"github.com/lacsar712/stacklift/internal/model"
	"github.com/lacsar712/stacklift/internal/store"
	"github.com/lacsar712/stacklift/internal/telemetry"
)

type SlewSink interface {
	RequestSlew(ctx context.Context, rigID string, targetAz, targetRadius float64, reason string) error
}

type TransitionSink interface {
	Transition(ctx context.Context, rigID string, to model.DutyMode) (model.TransitionResult, error)
	EmergencyStop(rigID string) error
}

type IngestSink interface {
	IngestLoad(sample model.LoadSample)
	IngestWind(sample model.WindSample) error
}

type ServiceSink interface {
	SlewSink
	TransitionSink
	IngestSink
}

type Server struct {
	Cfg       config.Config
	Slew      SlewSink
	Transit   TransitionSink
	Ingest    IngestSink
	Group     *crane.Group
	Store     *store.RigStore
	Guard     *interlock.Guard
	Alarms    *alarm.Bus
	Telemetry *telemetry.Bus
	Journal   *journal.Writer
	Clock     *clock.ProcessClock
	Counts    *telemetry.Counters
	Mux       *http.ServeMux
}

func New(cfg config.Config, sink ServiceSink, group *crane.Group, st *store.RigStore, guard *interlock.Guard, alarms *alarm.Bus, telem *telemetry.Bus, journal *journal.Writer, clk *clock.ProcessClock) *Server {
	s := &Server{
		Cfg: cfg, Slew: sink, Transit: sink, Ingest: sink,
		Group: group, Store: st, Guard: guard, Alarms: alarms, Telemetry: telem,
		Journal: journal, Clock: clk, Counts: telemetry.NewCounters(), Mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.Mux.HandleFunc("/api/health", s.handleHealth)
	s.Mux.HandleFunc("/api/status", s.handleStatus)
	s.Mux.HandleFunc("/api/poses", s.handlePoses)
	s.Mux.HandleFunc("/api/alarms", s.handleAlarms)
	s.Mux.HandleFunc("/api/telemetry", s.handleTelemetry)
	s.Mux.HandleFunc("/api/interlock", s.handleInterlock)
	s.Mux.HandleFunc("/api/journal", s.handleJournal)
	s.Mux.HandleFunc("/api/slew", s.handleSlew)
	s.Mux.HandleFunc("/api/transition", s.handleTransition)
	s.Mux.HandleFunc("/api/emergency", s.handleEmergency)
	s.Mux.HandleFunc("/api/ingest/load", s.handleIngestLoad)
	s.Mux.HandleFunc("/api/ingest/wind", s.handleIngestWind)
	s.Mux.HandleFunc("/api/metrics", s.handleMetrics)
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Counts.Inc("http_requests", 1)
		s.Mux.ServeHTTP(w, r)
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"ok": true, "tick": s.Clock.Tick(), "rigs": len(s.Cfg.RigIDs)})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{
		"cranes": s.Group.StatusAll(), "summary": crane.Summarize(s.Group),
		"alarms": s.Alarms.CountActive(), "critical": s.Alarms.HasCritical(), "tick": s.Clock.Tick(),
	})
}

func (s *Server) handlePoses(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"poses": s.Store.Snapshot(), "statuses": s.Store.StatusSnapshot(), "version": s.Store.Revision()})
}

func (s *Server) handleAlarms(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"active": s.Alarms.ActiveSnapshot(), "history": s.Alarms.HistorySnapshot()})
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"events": s.Telemetry.Snapshot()})
}

func (s *Server) handleInterlock(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"items": s.Guard.Snapshot()})
}

func (s *Server) handleJournal(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{"lines": s.Journal.Snapshot()})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, s.Counts.Snapshot())
}

func (s *Server) handleSlew(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST"})
		return
	}
	var body struct {
		RigID  string  `json:"rig_id"`
		Target float64 `json:"target_az"`
		Radius float64 `json:"target_radius"`
		Reason string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad json"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.Slew.RequestSlew(ctx, body.RigID, body.Target, body.Radius, body.Reason); err != nil {
		s.writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleTransition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST"})
		return
	}
	var body struct {
		RigID string         `json:"rig_id"`
		To    model.DutyMode `json:"to"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	res, err := s.Transit.Transition(r.Context(), body.RigID, body.To)
	if err != nil {
		s.writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "result": res})
		return
	}
	s.writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleEmergency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST"})
		return
	}
	var body struct{ RigID string `json:"rig_id"` }
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := s.Transit.EmergencyStop(body.RigID); err != nil {
		s.writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "emergency"})
}

func (s *Server) handleIngestLoad(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST"})
		return
	}
	var body model.LoadSample
	_ = json.NewDecoder(r.Body).Decode(&body)
	body.ProcessTick = s.Clock.Tick()
	s.Ingest.IngestLoad(body)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleIngestWind(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST"})
		return
	}
	var body model.WindSample
	_ = json.NewDecoder(r.Body).Decode(&body)
	body.ProcessTick = s.Clock.Tick()
	err := s.Ingest.IngestWind(body)
	if err != nil {
		s.writeJSON(w, http.StatusOK, map[string]any{"status": "fault", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
