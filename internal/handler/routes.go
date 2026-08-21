package handler

import "net/http"

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", h.static))
	mux.HandleFunc("GET /favicon.ico", h.Favicon)
	mux.HandleFunc("GET /{$}", h.Index)
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("POST /api/process", h.Process)
	mux.HandleFunc("GET /api/jobs/{id}", h.Status)
	mux.HandleFunc("GET /api/jobs/{id}/download", h.Download)

	return mux
}
