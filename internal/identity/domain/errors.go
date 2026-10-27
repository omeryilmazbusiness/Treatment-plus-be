package domain

import "github.com/treatment-plus/treatment-plus-be/internal/shared/errx"

// ErrRefreshReused is a presented refresh token that was already rotated.
// Callers must revoke the whole session family.
var ErrRefreshReused = errx.New("refresh_reused", "refresh token reused", errx.ErrUnauthenticated)
