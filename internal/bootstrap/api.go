// Package bootstrap is the composition root helper. Only cmd/* imports this package.
// Concrete adapters are wired here; use-cases never see pgx, Redis, or HTTP types.
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	aihttp "github.com/treatment-plus/treatment-plus-be/internal/ai/http"
	cataloghttp "github.com/treatment-plus/treatment-plus-be/internal/catalog/http"
	clinichttp "github.com/treatment-plus/treatment-plus-be/internal/clinic/http"
	identityhttp "github.com/treatment-plus/treatment-plus-be/internal/identity/http"
	mediahttp "github.com/treatment-plus/treatment-plus-be/internal/media/http"
	outputhttp "github.com/treatment-plus/treatment-plus-be/internal/output/http"
	planhttp "github.com/treatment-plus/treatment-plus-be/internal/plan/http"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/redisx"
	sharehttp "github.com/treatment-plus/treatment-plus-be/internal/share/http"
)

type API struct {
	Server *httpx.Server
	Pool   *pgxpool.Pool
	Redis  *redis.Client
	Log    *slog.Logger
}

func NewAPI(ctx context.Context, cfg config.Config, log *slog.Logger) (*API, error) {
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

	srv := httpx.NewServer(httpx.Options{
		Config: cfg,
		Log:    log,
		Ready: []httpx.Readiness{
			postgres.Checker{Pool: pool},
			redisx.Checker{Client: rdb},
		},
		Registrars: []httpx.RouteRegistrar{
			identityhttp.NewHandler(),
			clinichttp.NewHandler(),
			cataloghttp.NewHandler(),
			planhttp.NewHandler(),
			mediahttp.NewHandler(),
			outputhttp.NewHandler(),
			sharehttp.NewHandler(),
			aihttp.NewHandler(),
		},
	})

	return &API{Server: srv, Pool: pool, Redis: rdb, Log: log}, nil
}

func (a *API) Close() {
	if a.Redis != nil {
		_ = a.Redis.Close()
	}
	if a.Pool != nil {
		a.Pool.Close()
	}
}
