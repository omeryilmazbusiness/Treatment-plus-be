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
		clinicID, err := j.ParseAccess(raw)
		if err != nil {
			httpx.WriteError(w, errx.ErrUnauthenticated)
			return
		}
		ctx := ctxkey.WithClinicID(r.Context(), clinicID.String())
		ctx = ctxkey.WithUserID(ctx, clinicID.String())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
