package server

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jarrodb/ocr/pkg/config"
	"github.com/jarrodb/ocr/pkg/ocr"
)

type Server struct {
	cfg    *config.Config
	engine *ocr.Engine

	httpClient *http.Client

	mu         sync.Mutex
	totalCalls atomic.Int64
	monthCalls atomic.Int64
	monthStart time.Time
}

func New(cfg *config.Config) (*Server, error) {
	return &Server{
		cfg:        cfg,
		engine:     ocr.NewEngine(cfg),
		httpClient: &http.Client{Timeout: 15 * time.Second},
		monthStart: monthStart(time.Now().UTC()),
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/plate-reader/", s.handlePlateReader)
	mux.HandleFunc("/v1/statistics/", s.handleStatistics)
	mux.HandleFunc("/info/", s.handleInfo)

	mux.HandleFunc("/status", s.handleStatus)

	return s.withCORS(s.withAuth(mux))
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.cfg.Port)
	log.Printf("Starting Platerecognizer mock on %s", addr)
	log.Printf("Mode: %s", s.cfg.Mode)
	log.Printf("POST /v1/plate-reader/")
	log.Printf("GET  /v1/statistics/")
	log.Printf("GET  /info/")
	return http.ListenAndServe(addr, s.Handler())
}

func (s *Server) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			for _, allowed := range s.cfg.CORSOrigins {
				if allowed == "*" || allowed == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					break
				}
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withAuth enforces the `Authorization: Token <value>` header on API routes.
// Mirrors the real PR API, which returns 403 for missing/invalid tokens (not
// 401 — surprising but correct per their docs).
func (s *Server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" || r.Method == http.MethodOptions || !s.cfg.AuthRequired {
			next.ServeHTTP(w, r)
			return
		}
		got := r.Header.Get("Authorization")
		want := "Token " + s.cfg.APIToken
		if got != want {
			writeError(w, http.StatusForbidden, "Please provide a valid API token.")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func (s *Server) recordCall() {
	now := time.Now().UTC()
	if now.Year() != s.monthStart.Year() || now.Month() != s.monthStart.Month() {
		s.mu.Lock()
		s.monthStart = monthStart(now)
		s.monthCalls.Store(0)
		s.mu.Unlock()
	}
	s.totalCalls.Add(1)
	s.monthCalls.Add(1)
}
