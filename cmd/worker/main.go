package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	w, err := bootstrap.NewWorker(ctx, cfg, log)
	cancel()
	if err != nil {
		log.Error("bootstrap worker failed", "error", err)
		os.Exit(1)
	}
	defer w.Close()

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
		<-stop
		log.Info("worker shutting down")
		w.Shutdown()
	}()

	log.Info("worker process started", "env", cfg.App.Env)
	if err := w.Run(); err != nil {
		log.Error("worker stopped with error", "error", err)
		os.Exit(1)
	}
}
