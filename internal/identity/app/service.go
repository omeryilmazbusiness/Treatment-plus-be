package app

import (
	"context"
	"errors"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

const dummyPassword = "not-a-real-password-used-only-for-timing"

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type ClinicView struct {
	ID    id.ID
	Name  string
	Email string
}

type Tokens struct {
	Access    string
	Refresh   string
	ExpiresIn int
}

type AuthResult struct {
	Clinic ClinicView
	Tokens Tokens
}

type Service struct {
	visible    ClinicReader
	lookup     ClinicLookup
	sessions   SessionReader
	tx         TxRunner
	hasher     PasswordHasher
	access     AccessIssuer
	clock      clock.Clock
	refreshTTL time.Duration
	dummyHash  string
}

func NewService(
	visible ClinicReader,
	lookup ClinicLookup,
	sessions SessionReader,
	tx TxRunner,
	hasher PasswordHasher,
	access AccessIssuer,
	clk clock.Clock,
	refreshTTL time.Duration,
) (*Service, error) {
	if clk == nil {
		clk = clock.System{}
	}
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	dummy, err := hasher.Hash(dummyPassword)
	if err != nil {
		return nil, err
	}
	return &Service{
		visible:    visible,
		lookup:     lookup,
		sessions:   sessions,
		tx:         tx,
		hasher:     hasher,
		access:     access,
		clock:      clk,
		refreshTTL: refreshTTL,
		dummyHash:  dummy,
	}, nil
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (AuthResult, error) {
	name, err := domain.NormalizeName(in.Name)
	if err != nil {
		return AuthResult{}, err
	}
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return AuthResult{}, err
	}
	if err := domain.ValidatePassword(in.Password); err != nil {
		return AuthResult{}, err
	}

	if _, err := s.lookup.FindByEmail(ctx, email); err == nil {
		return AuthResult{}, errx.New("email_taken", "email already registered", errx.ErrConflict)
	} else if !errors.Is(err, errx.ErrNotFound) {
		return AuthResult{}, err
	}

	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return AuthResult{}, err
	}

	now := s.clock.Now()
	clinic := domain.Clinic{
		ID:           id.New(),
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    now,
	}
	session, rawRefresh, err := s.newSession(clinic.ID, id.New(), now)
	if err != nil {
		return AuthResult{}, err
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx WriteTx) error {
		if err := tx.CreateClinic(ctx, clinic); err != nil {
			return err
		}
		return tx.SaveSession(ctx, session)
	})
	if err != nil {
		return AuthResult{}, err
	}
	return s.issue(clinic, rawRefresh, now)
}

func (s *Service) Login(ctx context.Context, in LoginInput) (AuthResult, error) {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return AuthResult{}, err
	}

	clinic, err := s.lookup.FindByEmail(ctx, email)
	if err != nil {
		_ = s.hasher.Verify(s.dummyHash, in.Password)
		if errors.Is(err, errx.ErrNotFound) {
			return AuthResult{}, errx.ErrUnauthenticated
		}
		return AuthResult{}, err
	}
	if err := s.hasher.Verify(clinic.PasswordHash, in.Password); err != nil {
		return AuthResult{}, errx.ErrUnauthenticated
	}

	now := s.clock.Now()
	session, rawRefresh, err := s.newSession(clinic.ID, id.New(), now)
	if err != nil {
		return AuthResult{}, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx WriteTx) error {
		return tx.SaveSession(ctx, session)
	})
	if err != nil {
		return AuthResult{}, err
	}
	return s.issue(clinic, rawRefresh, now)
}

func (s *Service) Refresh(ctx context.Context, rawRefresh string) (AuthResult, error) {
	if rawRefresh == "" {
		return AuthResult{}, errx.ErrUnauthenticated
	}
	now := s.clock.Now()
	sess, err := s.sessions.GetByRefreshHash(ctx, domain.HashRefresh(rawRefresh))
	if err != nil {
		if errors.Is(err, errx.ErrNotFound) {
			return AuthResult{}, errx.ErrUnauthenticated
		}
		return AuthResult{}, err
	}

	if err := sess.Usable(now); err != nil {
		if errors.Is(err, domain.ErrRefreshReused) {
			_ = s.tx.WithinTx(ctx, func(ctx context.Context, tx WriteTx) error {
				return tx.RevokeFamily(ctx, sess.FamilyID, now)
			})
		}
		return AuthResult{}, errx.ErrUnauthenticated
	}

	next, raw, err := s.newSession(sess.ClinicID, sess.FamilyID, now)
	if err != nil {
		return AuthResult{}, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx WriteTx) error {
		if err := tx.MarkReused(ctx, sess.ID, now); err != nil {
			return err
		}
		return tx.SaveSession(ctx, next)
	})
	if err != nil {
		return AuthResult{}, err
	}

	clinic, err := s.lookup.FindByID(ctx, sess.ClinicID)
	if err != nil {
		return AuthResult{}, err
	}
	return s.issue(clinic, raw, now)
}

func (s *Service) Logout(ctx context.Context, rawRefresh string) error {
	if rawRefresh == "" {
		return nil
	}
	sess, err := s.sessions.GetByRefreshHash(ctx, domain.HashRefresh(rawRefresh))
	if err != nil {
		if errors.Is(err, errx.ErrNotFound) {
			return nil
		}
		return err
	}
	return s.tx.WithinTx(ctx, func(ctx context.Context, tx WriteTx) error {
		return tx.RevokeFamily(ctx, sess.FamilyID, s.clock.Now())
	})
}

func (s *Service) Me(ctx context.Context) (ClinicView, error) {
	clinicID, err := clinicFromCtx(ctx)
	if err != nil {
		return ClinicView{}, err
	}
	return s.GetClinic(ctx, clinicID)
}

func (s *Service) GetClinic(ctx context.Context, requested id.ID) (ClinicView, error) {
	actor, err := clinicFromCtx(ctx)
	if err != nil {
		return ClinicView{}, err
	}
	if actor != requested {
		return ClinicView{}, errx.ErrForbidden
	}
	clinic, err := s.visible.GetVisible(ctx, requested)
	if err != nil {
		if errors.Is(err, errx.ErrNotFound) {
			return ClinicView{}, errx.ErrForbidden
		}
		return ClinicView{}, err
	}
	return viewOf(clinic), nil
}

func (s *Service) newSession(clinicID, familyID id.ID, now time.Time) (domain.Session, string, error) {
	raw, hash, err := domain.NewRefreshSecret()
	if err != nil {
		return domain.Session{}, "", err
	}
	if familyID.IsZero() {
		familyID = id.New()
	}
	return domain.Session{
		ID:          id.New(),
		ClinicID:    clinicID,
		RefreshHash: hash,
		FamilyID:    familyID,
		ExpiresAt:   now.Add(s.refreshTTL),
		CreatedAt:   now,
	}, raw, nil
}

func (s *Service) issue(clinic domain.Clinic, rawRefresh string, now time.Time) (AuthResult, error) {
	access, ttl, err := s.access.Issue(clinic.ID, now)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{
		Clinic: viewOf(clinic),
		Tokens: Tokens{
			Access:    access,
			Refresh:   rawRefresh,
			ExpiresIn: int(ttl / time.Second),
		},
	}, nil
}

func viewOf(c domain.Clinic) ClinicView {
	return ClinicView{ID: c.ID, Name: c.Name, Email: c.Email}
}

func clinicFromCtx(ctx context.Context) (id.ID, error) {
	raw := ctxkey.ClinicID(ctx)
	if raw == "" {
		return id.ID{}, errx.ErrUnauthenticated
	}
	parsed, err := id.Parse(raw)
	if err != nil {
		return id.ID{}, errx.ErrUnauthenticated
	}
	return parsed, nil
}
