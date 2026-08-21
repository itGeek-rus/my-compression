package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"my-compression/internal/config"
	"my-compression/internal/handler"
	"my-compression/internal/job"
)

var version = "dev"

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg := config.Load()
	jobs := job.NewStore()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	h, err := handler.New(cfg, jobs, log, stop)
	if err != nil {
		log.Error("init handler", "err", err)
		os.Exit(1)
	}

	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		log.Error("listen", "err", err)
		os.Exit(1)
	}

	url := "http://" + displayAddr(ln.Addr().String())
	srv := &http.Server{
		Handler:           h.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info("starting app", "url", url, "version", version)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("app failed", "err", err)
			os.Exit(1)
		}
	}()

	go openBrowser(url)

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("app shutdown", "err", err)
		os.Exit(1)
	}
	log.Info("app stopped")
}

func displayAddr(addr string) string {
	host, port, ok := strings.Cut(addr, ":")
	if ok {
		return "127.0.0.1:" + port
	}
	if strings.HasPrefix(addr, ":") {
		return "127.0.0.1" + addr
	}
	if strings.HasPrefix(addr, "0.0.0.0:") {
		return "127.0.0.1" + strings.TrimPrefix(addr, "0.0.0.0:")
	}
	_ = host
	return addr
}
