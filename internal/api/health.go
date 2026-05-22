package api

import "net/http"

func (h *Handler) registerHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /healthz", h.health)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
