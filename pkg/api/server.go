package api

import (
	"encoding/json"
	"net/http"

	"github.com/theshedman/shedman/pkg/core"
)

// Server exposes a minimal HTTP API for shedman.
type Server struct {
	engine *core.Engine
	mux    *http.ServeMux
}

// NewServer creates a new API server.
func NewServer(engine *core.Engine) *Server {
	s := &Server{
		engine: engine,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// Serve starts the API server on the given address.
func (s *Server) Serve(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", s.handleHealth)
	s.mux.HandleFunc("/search", s.handleSearch)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query", http.StatusBadRequest)
		return
	}

	if s.engine == nil {
		http.Error(w, "engine not configured", http.StatusInternalServerError)
		return
	}

	results, err := s.engine.Search(query)
	if err != nil {
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}
