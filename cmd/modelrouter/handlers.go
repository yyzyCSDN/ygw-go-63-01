package main

import (
	"encoding/json"
	"io"
	"net/http"

	"modelrouter/internal/gateway"
	"modelrouter/internal/model"
)

type registerRequest struct {
	Model     string          `json:"model"`
	Version   string          `json:"version"`
	Instances []seedInstance  `json:"instances"`
}

type publishRequest struct {
	Model   string `json:"model"`
	Version string `json:"version"`
}

type aliasRequest struct {
	Alias   string `json:"alias"`
	Model   string `json:"model"`
	Version string `json:"version"`
}

type retireRequest struct {
	Model   string `json:"model"`
	Version string `json:"version"`
}

func (s *Server) handleInfer(w http.ResponseWriter, r *http.Request) {
	modelName := r.PathValue("model")
	if modelName == "" {
		writeError(w, http.StatusBadRequest, "missing model")
		return
	}
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "read payload: "+err.Error())
		return
	}
	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		requestID = newRequestID()
	}
	resp, err := s.gw.ServeWithTimeout(r.Context(), modelName, requestID, payload, 0)
	if err != nil {
		writeError(w, gateway.MapError(err), err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(resp.Status)
	_, _ = w.Write(resp.Body)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if _, err := s.reg.Register(req.Model, req.Version); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	for _, inst := range req.Instances {
		if err := s.reg.AddInstance(req.Model, req.Version, inst.ID, inst.Addr); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := s.health.Observe(model.NewInstance(req.Model, req.Version, inst.ID, inst.Addr)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "registered"})
}

func (s *Server) handlePublish(w http.ResponseWriter, r *http.Request) {
	var req publishRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.reg.Publish(req.Model, req.Version); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
}

func (s *Server) handleAlias(w http.ResponseWriter, r *http.Request) {
	var req aliasRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.reg.SetAlias(req.Alias, req.Model, req.Version); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "aliased"})
}

func (s *Server) handleRetire(w http.ResponseWriter, r *http.Request) {
	var req retireRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.reg.Retire(req.Model, req.Version); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "retired"})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if err := s.reg.Sync(); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "synced"})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	states := make(map[string]any)
	for _, name := range s.reg.Models() {
		states[name] = s.reg.VersionStates(name)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"models":     s.reg.Models(),
		"active":     s.reg.ActiveVersions(),
		"table":      s.gw.Table().Snapshot(),
		"route":      s.gw.Table().Report(),
		"registry":   s.reg.Report(),
		"states":     states,
		"generation": s.reg.Generation(),
	})
}

func (s *Server) handleInstances(w http.ResponseWriter, r *http.Request) {
	snapshot := s.health.Snapshot()
	writeJSON(w, http.StatusOK, map[string]any{
		"healthy": snapshot.Instances(),
		"ids":     snapshot.IDs(),
		"count":   snapshot.Count(),
	})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"counters": s.metrics.Snapshot(),
		"names":    s.metrics.Names(),
		"quota": map[string]any{
			"inFlight": s.gw.Limiter().InFlight(),
			"counter":  s.gw.Limiter().CounterState(),
			"capacity": s.gw.Limiter().Capacity(),
		},
		"timeout":       s.gw.Timeout().String(),
		"fallbackTries": s.gw.FallbackPolicy().MaxAttempts,
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"uptime":  timeSince(s.started),
		"quota":   s.gw.Limiter().InFlight(),
		"syncErr": errorString(s.reg.LastError()),
		"lastSync": s.reg.LastSync().At.Format("2006-01-02T15:04:05Z07:00"),
		"route":    s.gw.Table().Describe(),
	})
}

func (s *Server) handleConsole(w http.ResponseWriter, r *http.Request) {
	content, err := ServeConsole()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "decode json: "+err.Error())
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
