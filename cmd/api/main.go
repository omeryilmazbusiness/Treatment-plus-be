package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/treatment-plus/treatment-plus-be/internal/bootstrap"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/logger"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/otelx"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := logger.New(cfg.App.LogLevel, cfg.App.LogFormat)

	_, shutdownTracer := otelx.Init(cfg.OTel)
	defer func() { _ = shutdownTracer(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	api, err := bootstrap.NewAPI(ctx, cfg, log)
	cancel()
	if err != nil {
		log.Error("bootstrap api failed", "error", err)
		os.Exit(1)
	}
	defer api.Close()

	go func() {
		log.Info("http server listening",
			"addr", cfg.HTTP.Addr,
			"env", cfg.App.Env,
			"version", cfg.App.Version,
		)
		if err := api.Server.HTTP.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()
	log.Info("shutting down")
	if err := api.Server.HTTP.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
}
