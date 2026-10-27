// Package mem is the in-memory identity adapter used by unit tests (LSP).
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
	clinics  map[id.ID]domain.Clinic
	byEmail  map[string]id.ID
	sessions map[id.ID]domain.Session
}

func New() *Store {
	return &Store{
		clinics:  make(map[id.ID]domain.Clinic),
		byEmail:  make(map[string]id.ID),
		sessions: make(map[id.ID]domain.Session),
	}
}

func (s *Store) GetVisible(ctx context.Context, clinicID id.ID) (domain.Clinic, error) {
	return s.FindByID(ctx, clinicID)
}

func (s *Store) FindByID(_ context.Context, clinicID id.ID) (domain.Clinic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clinics[clinicID]
	if !ok {
		return domain.Clinic{}, errx.ErrNotFound
	}
	return c, nil
}

func (s *Store) FindByEmail(_ context.Context, email string) (domain.Clinic, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.byEmail[email]
	if !ok {
		return domain.Clinic{}, errx.ErrNotFound
	}
	return s.clinics[id], nil
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

	clinics := cloneClinics(s.clinics)
	byEmail := cloneIDs(s.byEmail)
	sessions := cloneSessions(s.sessions)

	err := fn(ctx, &writeTx{store: s})
	if err != nil {
		s.clinics = clinics
		s.byEmail = byEmail
		s.sessions = sessions
	}
	return err
}

type writeTx struct {
	store *Store
}

func (t *writeTx) CreateClinic(_ context.Context, clinic domain.Clinic) error {
	if _, ok := t.store.byEmail[clinic.Email]; ok {
		return errx.New("email_taken", "email already registered", errx.ErrConflict)
	}
	t.store.clinics[clinic.ID] = clinic
	t.store.byEmail[clinic.Email] = clinic.ID
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

func cloneClinics(in map[id.ID]domain.Clinic) map[id.ID]domain.Clinic {
	out := make(map[id.ID]domain.Clinic, len(in))
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
