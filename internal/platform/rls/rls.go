// Package rls sets PostgreSQL session locals used by row-level security.
// clinic_id is taken from the authenticated session, never from the request body.
package rls

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

const settingClinicID = "app.clinic_id"

// Apply sets SET LOCAL app.clinic_id on the current transaction.
// The value is the UUID text form of the clinic ULID (Postgres uuid, not ULID charset).
func Apply(ctx context.Context, tx pgx.Tx) error {
	raw := ctxkey.ClinicID(ctx)
	if raw == "" {
		return fmt.Errorf("rls: clinic id missing from context")
	}
	clinicID, err := id.Parse(raw)
	if err != nil {
		return fmt.Errorf("rls: clinic id: %w", err)
	}
	_, err = tx.Exec(ctx, "SELECT set_config($1, $2, true)", settingClinicID, clinicID.UUIDString())
	if err != nil {
		return fmt.Errorf("rls: set clinic id: %w", err)
	}
	return nil
}
