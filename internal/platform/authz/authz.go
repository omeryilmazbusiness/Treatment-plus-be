package authz

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

func ClinicID(ctx context.Context) (string, error) {
	id := ctxkey.ClinicID(ctx)
	if id == "" {
		return "", errx.ErrUnauthenticated
	}
	return id, nil
}

func Role(ctx context.Context) (string, error) {
	role := ctxkey.Role(ctx)
	if role == "" {
		return "", errx.ErrUnauthenticated
	}
	return role, nil
}

func RequireRole(ctx context.Context, allowed ...string) error {
	got, err := Role(ctx)
	if err != nil {
		return err
	}
	for _, role := range allowed {
		if got == role {
			return nil
		}
	}
	return errx.ErrForbidden
}

func AllowClinic(ctx context.Context, clinicID string) error {
	got, err := ClinicID(ctx)
	if err != nil {
		return err
	}
	if got != clinicID {
		return errx.ErrForbidden
	}
	return nil
}
