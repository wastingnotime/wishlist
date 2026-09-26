package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/wastingnotime/wishlist/apps/api/internal/application"
	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
)

type Server struct{ board *application.Board }

func New(board *application.Board) *Server { return &Server{board: board} }

func (server *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.health)
	mux.HandleFunc("GET /v1/apps", server.apps)
	mux.HandleFunc("GET /v1/features", server.features)
	return mux
}

func (server *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (server *Server) apps(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"apps": server.board.Apps()})
}

func (server *Server) features(w http.ResponseWriter, r *http.Request) {
	view := r.URL.Query().Get("view")
	if view == "" {
		view = string(domain.Voting)
	}
	rows, err := server.board.Features(domain.Status(view), r.URL.Query().Get("app"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"code": "invalid_request"}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"features": rows})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
