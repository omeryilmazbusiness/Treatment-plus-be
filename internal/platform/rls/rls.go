// Package rls sets PostgreSQL session locals used by row-level security.
// clinic_id is taken from the authenticated session, never from the request body.
package rls

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
)

const settingClinicID = "app.clinic_id"

// Apply sets SET LOCAL app.clinic_id on the current transaction.
func Apply(ctx context.Context, tx pgx.Tx) error {
	clinicID := ctxkey.ClinicID(ctx)
	if clinicID == "" {
		return fmt.Errorf("rls: clinic id missing from context")
	}
	_, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", settingClinicID, clinicID)
	if err != nil {
		return fmt.Errorf("rls: set clinic id: %w", err)
	}
	return nil
}
