// Package config loads process configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App      App
	HTTP     HTTP
	Database Database
	Redis    Redis
	Auth     Auth
	Object   Object
	AI       AI
	OTel     OTel
}

type App struct {
	Name          string
	Env           string
	Version       string
	URL           string
	InternalToken string
	LogLevel      string
	LogFormat     string
}

type HTTP struct {
	Addr            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxBodyBytes    int64
	CORSOrigins     []string
}

type Database struct {
	URL             string
	AppURL          string
	ReadURL         string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// RuntimeURL is the API role. Prefer a non-superuser so FORCE RLS is not bypassed.
func (d Database) RuntimeURL() string {
	if d.AppURL != "" {
		return d.AppURL
	}
	return d.URL
}

type Redis struct {
	URL string
}

type Auth struct {
	JWTAccessSecret  string
	JWTRefreshSecret string
	KMSKey           string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration
}

type Object struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
}

type AI struct {
	OpenAIKey string
}

type OTel struct {
	Enabled     bool
	ServiceName string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		App: App{
			Name:          env("APP_NAME", "treatment-plus-be"),
			Env:           env("APP_ENV", "local"),
			Version:       env("APP_VERSION", "0.0.0"),
			URL:           env("APP_URL", "http://localhost:8080"),
			InternalToken: env("INTERNAL_TOKEN", ""),
			LogLevel:      env("LOG_LEVEL", "info"),
			LogFormat:     env("LOG_FORMAT", "json"),
		},
		HTTP: HTTP{
			Addr:            env("HTTP_ADDR", ":8080"),
			ReadTimeout:     envDuration("HTTP_READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    envDuration("HTTP_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:     envDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: envDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
			MaxBodyBytes:    envInt64("HTTP_MAX_BODY_BYTES", 8<<20),
			CORSOrigins:     splitCSV(env("CORS_ORIGINS", "http://localhost:3000")),
		},
		Database: Database{
			URL:             env("DATABASE_URL", "postgres://treatment:treatment@localhost:5434/treatment_plus?sslmode=disable"),
			AppURL:          env("DATABASE_APP_URL", ""),
			ReadURL:         env("DATABASE_RO_URL", ""),
			MaxConns:        int32(envInt("DB_MAX_CONNS", 20)),
			MinConns:        int32(envInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime: envDuration("DB_MAX_CONN_LIFETIME", 45*time.Minute),
			MaxConnIdleTime: envDuration("DB_MAX_CONN_IDLE_TIME", 10*time.Minute),
		},
		Redis: Redis{
			URL: env("REDIS_URL", "redis://localhost:6380/0"),
		},
		Auth: Auth{
			JWTAccessSecret:  env("JWT_ACCESS_SECRET", ""),
			JWTRefreshSecret: env("JWT_REFRESH_SECRET", ""),
			KMSKey:           env("KMS_KEY", ""),
			AccessTTL:        envDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:       envDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Object: Object{
			Endpoint:  env("S3_ENDPOINT", ""),
			Region:    env("S3_REGION", "auto"),
			Bucket:    env("S3_BUCKET", ""),
			AccessKey: env("S3_ACCESS_KEY", ""),
			SecretKey: env("S3_SECRET_KEY", ""),
		},
		AI: AI{
			OpenAIKey: env("OPENAI_API_KEY", ""),
		},
		OTel: OTel{
			Enabled:     envBool("OTEL_ENABLED", false),
			ServiceName: env("OTEL_SERVICE_NAME", "treatment-plus-api"),
		},
	}

	if cfg.Database.URL == "" {
		return Config{}, fmt.Errorf("config: DATABASE_URL is required")
	}
	if cfg.Redis.URL == "" {
		return Config{}, fmt.Errorf("config: REDIS_URL is required")
	}
	if cfg.IsLocal() && cfg.Database.AppURL == "" {
		cfg.Database.AppURL = "postgres://treatment_app:treatment@localhost:5434/treatment_plus?sslmode=disable"
	}
	if err := cfg.Auth.apply(cfg.IsLocal()); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

const LocalJWTAccessSecret = "local-dev-jwt-access-secret-min-32b"
const LocalKMSKey = "local-dev-kms-key-must-be-at-least-32b"

func (a *Auth) apply(local bool) error {
	if a.JWTAccessSecret == "" && local {
		a.JWTAccessSecret = LocalJWTAccessSecret
	}
	if len(a.JWTAccessSecret) < 32 {
		return fmt.Errorf("config: JWT_ACCESS_SECRET must be at least 32 bytes")
	}
	if a.KMSKey == "" && local {
		a.KMSKey = LocalKMSKey
	}
	if len(a.KMSKey) < 32 {
		return fmt.Errorf("config: KMS_KEY must be at least 32 bytes")
	}
	return nil
}

func (c Config) IsLocal() bool { return c.App.Env == "local" || c.App.Env == "test" }

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envInt64(key string, fallback int64) int64 {
	return int64(envInt(key, int(fallback)))
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
