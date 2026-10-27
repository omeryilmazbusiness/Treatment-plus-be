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
