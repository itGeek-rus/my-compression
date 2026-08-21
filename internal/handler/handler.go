package handler

import (
	"context"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"my-compression/internal/archive"
	"my-compression/internal/config"
	"my-compression/internal/job"
	"my-compression/internal/service"
	"my-compression/web"
)

type Handler struct {
	cfg       config.Config
	jobs      *job.Store
	processor *service.Processor
	tmpl      *template.Template
	log       *slog.Logger
	static    http.Handler
}

func New(cfg config.Config, jobs *job.Store, log *slog.Logger) (*Handler, error) {
	tmpl, err := template.ParseFS(web.Assets, "templates/*.html")
	if err != nil {
		return nil, err
	}

	staticFS, err := fs.Sub(web.Assets, "static")
	if err != nil {
		return nil, err
	}

	return &Handler{
		cfg:       cfg,
		jobs:      jobs,
		processor: service.NewProcessor(jobs, cfg.TempDir),
		tmpl:      tmpl,
		log:       log,
		static:    http.FileServer(http.FS(staticFS)),
	}, nil
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		h.log.Error("render index", "err", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *Handler) Process(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxUploadBytes)
	if err := r.ParseMultipartForm(h.cfg.MaxUploadBytes); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large or invalid form"})
		return
	}

	action := r.FormValue("action")
	format := r.FormValue("format")
	if action != "archive" && action != "extract" {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "action: archive|extract"})
		return
	}
	if _, err := archive.ParseFormat(format); err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "format: zip|tar.gz|zstd|7z|tar.xz"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	j := h.jobs.Create(action, format)

	workDir := filepath.Join(h.cfg.TempDir, "my-compression", j.ID)
	srcName := service.SanitizeFilename(header.Filename)
	srcPath := filepath.Join(workDir, "in", srcName)

	size, err := service.SaveUpload(srcPath, io.LimitReader(file, h.cfg.MaxUploadBytes+1))
	if err != nil {
		_ = os.RemoveAll(workDir)
		_, _ = h.jobs.Update(j.ID, func(jobState *job.Job) {
			jobState.Status = job.StatusFailed
			jobState.Message = "Failed"
			jobState.Error = err.Error()
			jobState.Progress = 100
		})
		h.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save upload"})
		return
	}
	if size > h.cfg.MaxUploadBytes {
		_ = os.RemoveAll(workDir)
		_, _ = h.jobs.Update(j.ID, func(jobState *job.Job) {
			jobState.Status = job.StatusFailed
			jobState.Message = "Failed"
			jobState.Error = "file too large"
			jobState.Progress = 100
		})
		h.writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "file too large"})
		return
	}

	_, _ = h.jobs.Update(j.ID, func(jobState *job.Job) {
		jobState.OriginalSize = size
		jobState.Status = job.StatusQueued
		jobState.Message = "Queued"
		jobState.Progress = 0
	})

	go h.processor.Run(context.Background(), service.Input{
		JobID:    j.ID,
		Action:   action,
		Format:   format,
		Filename: srcName,
		SrcPath:  srcPath,
		WorkDir:  workDir,
	})

	h.log.Info("job accepted",
		"id", j.ID,
		"action", action,
		"format", format,
		"filename", srcName,
		"size", size,
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
	id := r.PathValue("id")
	j, err := h.jobs.Get(id)
	if err != nil {
		h.writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	if j.Status != job.StatusDone {
		h.writeJSON(w, http.StatusConflict, map[string]string{"error": "job is not ready"})
		return
	}
	if j.ResultPath == "" {
		h.writeJSON(w, http.StatusNotFound, map[string]string{"error": "result not found"})
		return
	}
	if _, err := os.Stat(j.ResultPath); err != nil {
		h.writeJSON(w, http.StatusNotFound, map[string]string{"error": "result file missing"})
		return
	}

	name := j.DownloadName
	if name == "" {
		name = filepath.Base(j.ResultPath)
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(name))
	http.ServeFile(w, r, j.ResultPath)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) Favicon(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(web.Assets, "static/favicon.svg")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write(data)
}

func (h *Handler) writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
