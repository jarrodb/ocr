package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func (s *Server) handleStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	now := time.Now().UTC()
	reset := monthStart(now).AddDate(0, 1, 0)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(StatisticsResponse{
		Usage: Usage{
			Month:    int(s.monthStart.Month()),
			Year:     s.monthStart.Year(),
			Calls:    int(s.monthCalls.Load()),
			ResetsOn: reset.Format(time.RFC3339),
		},
		TotalCalls: int(s.totalCalls.Load()),
	}); err != nil {
		log.Printf("encode statistics: %v", err)
	}
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	resp := InfoResponse{
		Version:    "mock-1.0.0",
		LicenseKey: "MOCK",
		TotalCalls: int(s.totalCalls.Load()),
		Webhooks:   []any{},
	}
	resp.Usage.Calls = int(s.monthCalls.Load())
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("encode info: %v", err)
	}
}

// handleStatus is a debug endpoint (no auth) for tests and humans poking the
// mock. Not part of the real PR API.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"mode":        s.cfg.Mode,
		"total_calls": s.totalCalls.Load(),
		"month_calls": s.monthCalls.Load(),
		"tesseract":   s.cfg.Tesseract.Enabled,
	})
}

func writeError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{Detail: detail, StatusCode: status})
}

func (s *Server) simulateDelay() {
	if s.cfg.Delay == "" || s.cfg.Delay == "0s" {
		return
	}
	if d, err := time.ParseDuration(s.cfg.Delay); err == nil {
		time.Sleep(d)
	}
}
