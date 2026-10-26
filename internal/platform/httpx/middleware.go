package httpx

import (
	"net/http"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

const headerRequestID = "X-Request-Id"

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get(headerRequestID)
		if rid == "" {
			rid = id.New().String()
		}
		w.Header().Set(headerRequestID, rid)
		ctx := ctxkey.WithTraceID(r.Context(), rid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func maxBytes(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if n > 0 && r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(sw, r)
		if l := loggerFrom(r.Context()); l != nil {
			l.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.code,
				"dur_ms", time.Since(start).Milliseconds(),
				"trace_id", ctxkey.TraceID(r.Context()),
			)
		}
	})
}
