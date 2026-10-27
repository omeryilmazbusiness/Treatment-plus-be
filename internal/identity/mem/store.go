package mem

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	"github.com/treatment-plus/treatment-plus-be/internal/identity/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	mu       sync.Mutex
	accounts map[id.ID]domain.Account // workspace id
	byEmail  map[string]id.ID
	sessions map[id.ID]domain.Session
}

func New() *Store {
	return &Store{
		accounts: make(map[id.ID]domain.Account),
		byEmail:  make(map[string]id.ID),
		sessions: make(map[id.ID]domain.Session),
	}
}

func (s *Store) GetVisible(ctx context.Context, workspaceID id.ID) (domain.Workspace, error) {
	acc, err := s.FindByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return domain.Workspace{}, err
	}
	return acc.Workspace, nil
}

func (s *Store) FindByWorkspaceID(_ context.Context, workspaceID id.ID) (domain.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acc, ok := s.accounts[workspaceID]
	if !ok {
		return domain.Account{}, errx.ErrNotFound
	}
	return acc, nil
}

func (s *Store) FindByEmail(_ context.Context, email string) (domain.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wid, ok := s.byEmail[email]
	if !ok {
		return domain.Account{}, errx.ErrNotFound
	}
	return s.accounts[wid], nil
}

func (s *Store) GetByRefreshHash(_ context.Context, hash []byte) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sess := range s.sessions {
		if bytes.Equal(sess.RefreshHash, hash) {
			return sess, nil
		}
	}
	return domain.Session{}, errx.ErrNotFound
}

func (s *Store) WithinTx(ctx context.Context, fn func(context.Context, app.WriteTx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	accounts := cloneAccounts(s.accounts)
	byEmail := cloneIDs(s.byEmail)
	sessions := cloneSessions(s.sessions)

	err := fn(ctx, &writeTx{store: s})
	if err != nil {
		s.accounts = accounts
		s.byEmail = byEmail
		s.sessions = sessions
	}
	return err
}

type writeTx struct {
	store *Store
}

func (t *writeTx) CreateAccount(_ context.Context, account domain.Account) error {
	if _, ok := t.store.byEmail[account.User.Email]; ok {
		return errx.New("email_taken", "email already registered", errx.ErrConflict)
	}
	t.store.accounts[account.Workspace.ID] = account
	t.store.byEmail[account.User.Email] = account.Workspace.ID
	return nil
}

func (t *writeTx) SaveSession(_ context.Context, session domain.Session) error {
	t.store.sessions[session.ID] = session
	return nil
}

func (t *writeTx) MarkReused(_ context.Context, sessionID id.ID, at time.Time) error {
	sess, ok := t.store.sessions[sessionID]
	if !ok {
		return errx.ErrNotFound
	}
	if sess.ReusedAt == nil {
		sess.ReusedAt = &at
		t.store.sessions[sessionID] = sess
	}
	return nil
}

func (t *writeTx) RevokeFamily(_ context.Context, familyID id.ID, at time.Time) error {
	for k, sess := range t.store.sessions {
		if sess.FamilyID == familyID && sess.RevokedAt == nil {
			sess.RevokedAt = &at
			t.store.sessions[k] = sess
		}
	}
	return nil
}

func cloneAccounts(in map[id.ID]domain.Account) map[id.ID]domain.Account {
	out := make(map[id.ID]domain.Account, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneIDs(in map[string]id.ID) map[string]id.ID {
	out := make(map[string]id.ID, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneSessions(in map[id.ID]domain.Session) map[id.ID]domain.Session {
	out := make(map[id.ID]domain.Session, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
