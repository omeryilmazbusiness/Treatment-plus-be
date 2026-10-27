package pg

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	"github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres/dbgen"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/rls"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: dbgen.New(pool)}
}

func (s *Store) GetVisible(ctx context.Context, clinicID id.ID) (domain.Clinic, error) {
	var out domain.Clinic
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		row, err := s.q.WithTx(tx).IdentityGetClinicByID(ctx, toUUID(clinicID))
		if err != nil {
			return mapNotFound(err)
		}
		out = clinicFrom(row.ID, row.Name, row.Email, row.PasswordHash, row.CreatedAt)
		return nil
	})
	return out, err
}

func (s *Store) FindByID(ctx context.Context, clinicID id.ID) (domain.Clinic, error) {
	row, err := s.q.IdentityGetClinicByIDAuth(ctx, toUUID(clinicID))
	if err != nil {
		return domain.Clinic{}, mapNotFound(err)
	}
	return clinicFrom(row.ID, row.Name, row.Email, row.PasswordHash, row.CreatedAt), nil
}

func (s *Store) FindByEmail(ctx context.Context, email string) (domain.Clinic, error) {
	row, err := s.q.IdentityFindClinicByEmail(ctx, email)
	if err != nil {
		return domain.Clinic{}, mapNotFound(err)
	}
	return clinicFrom(row.ID, row.Name, row.Email, row.PasswordHash, row.CreatedAt), nil
}

func (s *Store) GetByRefreshHash(ctx context.Context, hash []byte) (domain.Session, error) {
	row, err := s.q.IdentityGetSessionByRefreshHash(ctx, hash)
	if err != nil {
		return domain.Session{}, mapNotFound(err)
	}
	return sessionFrom(row), nil
}

func (s *Store) WithinTx(ctx context.Context, fn func(context.Context, app.WriteTx) error) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return fn(ctx, &writeTx{q: s.q.WithTx(tx)})
	})
}

type writeTx struct {
	q *dbgen.Queries
}

func (t *writeTx) CreateClinic(ctx context.Context, clinic domain.Clinic) error {
	_, err := t.q.IdentityRegisterClinic(ctx, dbgen.IdentityRegisterClinicParams{
		ID:           toUUID(clinic.ID),
		Name:         clinic.Name,
		Email:        clinic.Email,
		PasswordHash: clinic.PasswordHash,
	})
	return mapWrite(err)
}

func (t *writeTx) SaveSession(ctx context.Context, session domain.Session) error {
	_, err := t.q.IdentitySaveSession(ctx, dbgen.IdentitySaveSessionParams{
		ID:          toUUID(session.ID),
		ClinicID:    toUUID(session.ClinicID),
		RefreshHash: session.RefreshHash,
		FamilyID:    toUUID(session.FamilyID),
		ExpiresAt:   toTimestamptz(session.ExpiresAt),
	})
	return mapWrite(err)
}

func (t *writeTx) MarkReused(ctx context.Context, sessionID id.ID, at time.Time) error {
	return mapWrite(t.q.IdentityMarkSessionReused(ctx, dbgen.IdentityMarkSessionReusedParams{
		ID:       toUUID(sessionID),
		ReusedAt: toTimestamptz(at),
	}))
}

func (t *writeTx) RevokeFamily(ctx context.Context, familyID id.ID, at time.Time) error {
	return mapWrite(t.q.IdentityRevokeFamily(ctx, dbgen.IdentityRevokeFamilyParams{
		FamilyID:  toUUID(familyID),
		RevokedAt: toTimestamptz(at),
	}))
}

func clinicFrom(rawID pgtype.UUID, name, email, hash string, created pgtype.Timestamptz) domain.Clinic {
	return domain.Clinic{
		ID:           mustID(rawID),
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    created.Time,
	}
}

func sessionFrom(row dbgen.Session) domain.Session {
	var reused, revoked *time.Time
	if row.ReusedAt.Valid {
		t := row.ReusedAt.Time
		reused = &t
	}
	if row.RevokedAt.Valid {
		t := row.RevokedAt.Time
		revoked = &t
	}
	return domain.Session{
		ID:          mustID(row.ID),
		ClinicID:    mustID(row.ClinicID),
		RefreshHash: row.RefreshHash,
		FamilyID:    mustID(row.FamilyID),
		ReusedAt:    reused,
		RevokedAt:   revoked,
		ExpiresAt:   row.ExpiresAt.Time,
		CreatedAt:   row.CreatedAt.Time,
	}
}

func toUUID(v id.ID) pgtype.UUID {
	var u pgtype.UUID
	copy(u.Bytes[:], v.Bytes())
	u.Valid = true
	return u
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t.UTC(), Valid: true}
}

func mustID(u pgtype.UUID) id.ID {
	if !u.Valid {
		return id.ID{}
	}
	out, err := id.FromBytes(u.Bytes[:])
	if err != nil {
		return id.ID{}
	}
	return out
}

func mapNotFound(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return errx.ErrNotFound
	}
	return err
}

func mapWrite(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return errx.New("email_taken", "email already registered", errx.ErrConflict)
	}
	return err
}
