package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/redisx"
)

// Worker is the async process. Phase 0 is idle: it holds connections and waits.
// Phase 7+ drains outbox (PDF, AI, mail, expiry).
type Worker struct {
	Pool   *pgxpool.Pool
	Redis  *redis.Client
	Log    *slog.Logger
	tick   time.Duration
	stop   chan struct{}
	closed atomic.Bool
}

func NewWorker(ctx context.Context, cfg config.Config, log *slog.Logger) (*Worker, error) {
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	rdb, err := redisx.New(cfg.Redis)
	if err != nil {
		pool.Close()
		return nil, err
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		pool.Close()
		_ = rdb.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	return &Worker{
		Pool:  pool,
		Redis: rdb,
		Log:   log,
		tick:  15 * time.Second,
		stop:  make(chan struct{}),
	}, nil
}

func (w *Worker) Run() error {
	w.Log.Info("worker idle; outbox empty until phase 7")
	t := time.NewTicker(w.tick)
	defer t.Stop()
	for {
		select {
		case <-w.stop:
			return nil
		case <-t.C:
			if err := w.Pool.Ping(context.Background()); err != nil {
				w.Log.Warn("worker postgres ping", "error", err)
			}
		}
	}
}

func (w *Worker) Shutdown() {
	if w.closed.CompareAndSwap(false, true) {
		close(w.stop)
	}
}

func (w *Worker) Close() {
	w.Shutdown()
	if w.Redis != nil {
		_ = w.Redis.Close()
	}
	if w.Pool != nil {
		w.Pool.Close()
	}
}
