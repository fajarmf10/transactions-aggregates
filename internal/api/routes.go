package api

import "net/http"

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	h.registerHealthRoutes(mux)
	h.registerTransactionRoutes(mux)
	return mux
}
