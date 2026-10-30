package authn

import (
	"net/http"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

func (j *JWT) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := Bearer(r.Header.Get("Authorization"))
		if raw == "" {
			httpx.WriteError(w, errx.ErrUnauthenticated)
			return
		}
		principal, err := j.ParseAccess(raw)
		if err != nil {
			httpx.WriteError(w, errx.ErrUnauthenticated)
			return
		}
		ctx := ctxkey.WithClinicID(r.Context(), principal.WorkspaceID.String())
		ctx = ctxkey.WithUserID(ctx, principal.UserID.String())
		ctx = ctxkey.WithRole(ctx, string(principal.Role))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (j *JWT) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := Bearer(r.Header.Get("Authorization"))
		if raw == "" {
			next.ServeHTTP(w, r)
			return
		}
		principal, err := j.ParseAccess(raw)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := ctxkey.WithClinicID(r.Context(), principal.WorkspaceID.String())
		ctx = ctxkey.WithUserID(ctx, principal.UserID.String())
		ctx = ctxkey.WithRole(ctx, string(principal.Role))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
