package httpx

import (
	"context"
	"log/slog"
	"net/http"
)

type logKey struct{}

func withLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), logKey{}, log)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func loggerFrom(ctx context.Context) *slog.Logger {
	l, _ := ctx.Value(logKey{}).(*slog.Logger)
	return l
}
