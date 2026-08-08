package handler

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"log/slog"
	"my-compression/internal/config"
	"my-compression/internal/job"
	"my-compression/web"
	"net/http"
)

type Handler struct {
	cfg    config.Config
	jobs   *job.Store
	tmpl   *template.Template
	log    *slog.Logger
	static http.Handler
}

func New(cfg config.Config, jobs *job.Store, log *slog.Logger) (*Handler, error) {
	tmpl, err := template.ParseFS(web.Assets, "templates/*.tmpl")
	if err != nil {
		return nil, err
	}

	staticFS, err := fs.Sub(web.Assets, "static")
	if err != nil {
		return nil, err
	}

	return &Handler{
		cfg:    cfg,
		jobs:   jobs,
		tmpl:   tmpl,
		log:    log,
		static: http.FileServer(http.FS(staticFS)),
	}, nil
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "index.tmpl", nil); err != nil {
		h.log.Error("render index", "err", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *Handler) Process(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.cfg.MaxUploadBytes); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "incorrect form"})
		return
	}

	action := r.FormValue("action")
	format := r.FormValue("format")
	if action != "archive" && action != "extract" {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action: archive|extract"})
		return
	}
	if format != "zip" && format != "tar.gz" {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "format: zip|tar.gz"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "The file is required"})
		return
	}
	defer file.Close()

	j := h.jobs.Create(action, format)
	_, _ = h.jobs.Update(j.ID, func(jobState *job.Job) {
		jobState.OriginalSize = header.Size
		jobState.Message = "The file has been received (processing is not yet complete)"
		jobState.Status = job.StatusQueued
		jobState.Progress = 0
	})

	h.log.Info("job created",
		"id", j.ID,
		"action", action,
		"format", format,
		"filename", header.Filename,
		"size", header.Size,
	)

	updated, _ := h.jobs.Get(j.ID)
	h.writeJSON(w, http.StatusAccepted, updated)
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	j, err := h.jobs.Get(id)
	if err != nil {
		h.writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	h.writeJSON(w, http.StatusOK, j)
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusAccepted, map[string]string{
		"error": "download is unavailable yet",
	})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
