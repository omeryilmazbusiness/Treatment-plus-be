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
	Role     string
}

type LoginInput struct {
	Email    string
	Password string
}

type UserView struct {
	ID    id.ID
	Email string
	Role  domain.Role
}

type WorkspaceView struct {
	ID   id.ID
	Name string
	Kind domain.Role
}

type AccountView struct {
	User      UserView
	Workspace WorkspaceView
}

type Tokens struct {
	Access    string
	Refresh   string
	ExpiresIn int
}

type AuthResult struct {
	Account AccountView
	Tokens  Tokens
}

type Service struct {
	visible    WorkspaceReader
	lookup     AccountLookup
	sessions   SessionReader
	tx         TxRunner
	hasher     PasswordHasher
	access     AccessIssuer
	clock      clock.Clock
	refreshTTL time.Duration
	dummyHash  string
}

func NewService(
	visible WorkspaceReader,
	lookup AccountLookup,
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
	role, err := domain.ParseRole(in.Role)
	if err != nil {
		return AuthResult{}, err
	}
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
	userID, workspaceID := id.New(), id.New()
	account := domain.Account{
		User: domain.User{
			ID:           userID,
			Email:        email,
			PasswordHash: hash,
			CreatedAt:    now,
		},
		Workspace: domain.Workspace{
			ID:          workspaceID,
			OwnerUserID: userID,
			Name:        name,
			Kind:        role,
			CreatedAt:   now,
		},
		Role: role,
	}
	session, rawRefresh, err := s.newSession(userID, workspaceID, role, id.New(), now)
	if err != nil {
		return AuthResult{}, err
	}

	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx WriteTx) error {
		if err := tx.CreateAccount(ctx, account); err != nil {
			return err
		}
		return tx.SaveSession(ctx, session)
	})
	if err != nil {
		return AuthResult{}, err
	}
	return s.issue(account, rawRefresh, now)
}

func (s *Service) Login(ctx context.Context, in LoginInput) (AuthResult, error) {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return AuthResult{}, err
	}

	account, err := s.lookup.FindByEmail(ctx, email)
	if err != nil {
		_ = s.hasher.Verify(s.dummyHash, in.Password)
		if errors.Is(err, errx.ErrNotFound) {
			return AuthResult{}, errx.ErrUnauthenticated
		}
		return AuthResult{}, err
	}
	if err := s.hasher.Verify(account.User.PasswordHash, in.Password); err != nil {
		return AuthResult{}, errx.ErrUnauthenticated
	}

	now := s.clock.Now()
	session, rawRefresh, err := s.newSession(account.User.ID, account.Workspace.ID, account.Role, id.New(), now)
	if err != nil {
		return AuthResult{}, err
	}
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx WriteTx) error {
		return tx.SaveSession(ctx, session)
	})
	if err != nil {
		return AuthResult{}, err
	}
	return s.issue(account, rawRefresh, now)
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

	next, raw, err := s.newSession(sess.UserID, sess.ClinicID, sess.Role, sess.FamilyID, now)
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

	account, err := s.lookup.FindByWorkspaceID(ctx, sess.ClinicID)
	if err != nil {
		return AuthResult{}, err
	}
	return s.issue(account, raw, now)
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

func (s *Service) Me(ctx context.Context) (AccountView, error) {
	workspaceID, err := workspaceFromCtx(ctx)
	if err != nil {
		return AccountView{}, err
	}
	return s.GetWorkspace(ctx, workspaceID)
}

func (s *Service) GetWorkspace(ctx context.Context, requested id.ID) (AccountView, error) {
	actor, err := workspaceFromCtx(ctx)
	if err != nil {
		return AccountView{}, err
	}
	if actor != requested {
		return AccountView{}, errx.ErrForbidden
	}
	if _, err := s.visible.GetVisible(ctx, requested); err != nil {
		if errors.Is(err, errx.ErrNotFound) {
			return AccountView{}, errx.ErrForbidden
		}
		return AccountView{}, err
	}
	account, err := s.lookup.FindByWorkspaceID(ctx, requested)
	if err != nil {
		if errors.Is(err, errx.ErrNotFound) {
			return AccountView{}, errx.ErrForbidden
		}
		return AccountView{}, err
	}
	return viewOf(account), nil
}

func (s *Service) newSession(userID, workspaceID id.ID, role domain.Role, familyID id.ID, now time.Time) (domain.Session, string, error) {
	raw, hash, err := domain.NewRefreshSecret()
	if err != nil {
		return domain.Session{}, "", err
	}
	if familyID.IsZero() {
		familyID = id.New()
	}
	return domain.Session{
		ID:          id.New(),
		UserID:      userID,
		ClinicID:    workspaceID,
		Role:        role,
		RefreshHash: hash,
		FamilyID:    familyID,
		ExpiresAt:   now.Add(s.refreshTTL),
		CreatedAt:   now,
	}, raw, nil
}

func (s *Service) issue(account domain.Account, rawRefresh string, now time.Time) (AuthResult, error) {
	access, ttl, err := s.access.Issue(account.User.ID, account.Workspace.ID, account.Role, now)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{
		Account: viewOf(account),
		Tokens: Tokens{
			Access:    access,
			Refresh:   rawRefresh,
			ExpiresIn: int(ttl / time.Second),
		},
	}, nil
}

func viewOf(a domain.Account) AccountView {
	return AccountView{
		User: UserView{
			ID:    a.User.ID,
			Email: a.User.Email,
			Role:  a.Role,
		},
		Workspace: WorkspaceView{
			ID:   a.Workspace.ID,
			Name: a.Workspace.Name,
			Kind: a.Workspace.Kind,
		},
	}
}

func workspaceFromCtx(ctx context.Context) (id.ID, error) {
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
