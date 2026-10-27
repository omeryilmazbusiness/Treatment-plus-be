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
	catalogapp "github.com/treatment-plus/treatment-plus-be/internal/catalog/app"
	cataloghttp "github.com/treatment-plus/treatment-plus-be/internal/catalog/http"
	catalogpg "github.com/treatment-plus/treatment-plus-be/internal/catalog/pg"
	clinicapp "github.com/treatment-plus/treatment-plus-be/internal/clinic/app"
	clinichttp "github.com/treatment-plus/treatment-plus-be/internal/clinic/http"
	clinicpg "github.com/treatment-plus/treatment-plus-be/internal/clinic/pg"
	identityapp "github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	identityhttp "github.com/treatment-plus/treatment-plus-be/internal/identity/http"
	identitypg "github.com/treatment-plus/treatment-plus-be/internal/identity/pg"
	mediahttp "github.com/treatment-plus/treatment-plus-be/internal/media/http"
	outputhttp "github.com/treatment-plus/treatment-plus-be/internal/output/http"
	planapp "github.com/treatment-plus/treatment-plus-be/internal/plan/app"
	planhttp "github.com/treatment-plus/treatment-plus-be/internal/plan/http"
	planpg "github.com/treatment-plus/treatment-plus-be/internal/plan/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/redisx"
	sharehttp "github.com/treatment-plus/treatment-plus-be/internal/share/http"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/crypto"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type API struct {
	Server *httpx.Server
	Pool   *pgxpool.Pool
	Redis  *redis.Client
	Log    *slog.Logger
}

type catalogAdapter struct {
	svc *catalogapp.Service
}

func (c *catalogAdapter) GetTreatment(ctx context.Context, wsID id.ID, treatID id.ID) (string, string, int64, error) {
	price, t, err := c.svc.GetTreatmentPrice(ctx, wsID, treatID)
	if err != nil {
		return "", "", 0, err
	}
	return t.Code, t.Name, price, nil
}

type clinicSettingsAdapter struct {
	svc *clinicapp.Service
}

func (c *clinicSettingsAdapter) GetClinicSettings(ctx context.Context, wsID id.ID) (planapp.ClinicSettingsDTO, error) {
	st, err := c.svc.GetSettings(ctx, wsID)
	if err != nil {
		return planapp.ClinicSettingsDTO{}, err
	}
	return planapp.ClinicSettingsDTO{
		Name:    st.Name,
		Phone:   st.Phone,
		Palette: string(st.Palette),
		LogoKey: st.LogoKey,
		Intro:   st.Intro,
		Legal:   st.Legal,
	}, nil
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

	clk := clock.System{}
	tokens, err := authn.NewJWT(cfg.Auth, clk)
	if err != nil {
		pool.Close()
		_ = rdb.Close()
		return nil, err
	}

	identityStore := identitypg.New(pool)
	identitySvc, err := identityapp.NewService(
		identityStore,
		identityStore,
		identityStore,
		identityStore,
		authn.DefaultArgon2id(),
		tokens,
		clk,
		cfg.Auth.RefreshTTL,
	)
	if err != nil {
		pool.Close()
		_ = rdb.Close()
		return nil, err
	}

	clinicStore := clinicpg.New(pool)
	clinicSvc := clinicapp.NewService(clinicStore)

	catalogStore := catalogpg.New(pool)
	catalogSvc := catalogapp.NewService(catalogStore)

	encrypter := crypto.NewAESGCM(cfg.Auth.KMSKey)
	planStore := planpg.New(pool)
	planSvc := planapp.NewService(
		planStore,
		planStore,
		&catalogAdapter{svc: catalogSvc},
		&clinicSettingsAdapter{svc: clinicSvc},
		encrypter,
	)

	srv := httpx.NewServer(httpx.Options{
		Config: cfg,
		Log:    log,
		Ready: []httpx.Readiness{
			postgres.Checker{Pool: pool},
			redisx.Checker{Client: rdb},
		},
		Registrars: []httpx.RouteRegistrar{
			identityhttp.NewHandler(identitySvc, tokens),
			clinichttp.NewHandler(clinicSvc, tokens),
			cataloghttp.NewHandler(catalogSvc, tokens),
			planhttp.NewHandler(planSvc, tokens),
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
