package mem

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/share/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	mu     sync.RWMutex
	links  map[id.ID]domain.Link
	events []domain.Event
}

func New() *Store {
	return &Store{
		links: make(map[id.ID]domain.Link),
	}
}

func (s *Store) CreateLink(ctx context.Context, link domain.Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw := ctxkey.ClinicID(ctx)
	if raw != "" && raw != link.ClinicID.String() {
		return errx.ErrForbidden
	}
	s.links[link.ID] = link
	return nil
}

func (s *Store) GetLink(ctx context.Context, clinicID, linkID id.ID) (domain.Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw := ctxkey.ClinicID(ctx)
	if raw != "" && raw != clinicID.String() {
		return domain.Link{}, errx.ErrNotFound
	}
	l, ok := s.links[linkID]
	if !ok || l.ClinicID != clinicID {
		return domain.Link{}, errx.ErrNotFound
	}
	return l, nil
}

func (s *Store) ListLinks(ctx context.Context, clinicID, planID id.ID) ([]domain.Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw := ctxkey.ClinicID(ctx)
	if raw != "" && raw != clinicID.String() {
		return nil, nil
	}
	var out []domain.Link
	for _, l := range s.links {
		if l.ClinicID == clinicID && l.PlanID == planID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (s *Store) RevokeLink(ctx context.Context, clinicID, linkID id.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw := ctxkey.ClinicID(ctx)
	if raw != "" && raw != clinicID.String() {
		return errx.ErrForbidden
	}
	l, ok := s.links[linkID]
	if !ok || l.ClinicID != clinicID {
		return errx.ErrNotFound
	}
	now := time.Now().UTC()
	l.RevokedAt = &now
	l.UpdatedAt = now
	s.links[linkID] = l
	return nil
}

func (s *Store) ResolveToken(ctx context.Context, tokenHash []byte) (domain.Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, l := range s.links {
		if bytes.Equal(l.TokenHash, tokenHash) {
			now := time.Now().UTC()
			if l.IsRevoked() {
				return l, errx.New("link_revoked", "this share link has been revoked", errx.ErrForbidden)
			}
			if l.IsExpired(now) {
				return l, errx.New("link_expired", "this share link has expired", errx.ErrForbidden)
			}
			l.ViewsCount++
			l.LastViewedAt = &now
			l.UpdatedAt = now
			s.links[k] = l
			return l, nil
		}
	}
	return domain.Link{}, errx.New("invalid_token", "share link not found", errx.ErrNotFound)
}

func (s *Store) RecordEvent(ctx context.Context, tokenHash []byte, eventID id.ID, kind domain.EventKind, metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, l := range s.links {
		if bytes.Equal(l.TokenHash, tokenHash) {
			now := time.Now().UTC()
			if l.IsRevoked() || l.IsExpired(now) {
				return errx.New("invalid_token", "share link not found or inactive", errx.ErrNotFound)
			}
			s.events = append(s.events, domain.Event{
				ID:        eventID,
				ClinicID:  l.ClinicID,
				LinkID:    l.ID,
				Kind:      kind,
				Metadata:  metadata,
				CreatedAt: now,
			})
			return nil
		}
	}
	return errx.New("invalid_token", "share link not found or inactive", errx.ErrNotFound)
}

func (s *Store) AcceptPlan(ctx context.Context, tokenHash []byte, signature, ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, l := range s.links {
		if bytes.Equal(l.TokenHash, tokenHash) {
			now := time.Now().UTC()
			if l.IsRevoked() || l.IsExpired(now) {
				return errx.New("invalid_token", "share link not found or inactive", errx.ErrNotFound)
			}
			l.AcceptedAt = &now
			l.AcceptedSignature = signature
			l.AcceptedIP = ip
			l.UpdatedAt = now
			s.links[k] = l

			s.events = append(s.events, domain.Event{
				ID:        id.New(),
				ClinicID:  l.ClinicID,
				LinkID:    l.ID,
				Kind:      domain.EventAccepted,
				Metadata:  map[string]any{"signature": signature, "ip": ip},
				CreatedAt: now,
			})
			return nil
		}
	}
	return errx.New("invalid_token", "share link not found or inactive", errx.ErrNotFound)
}

func (s *Store) ListEvents(ctx context.Context, clinicID, planID id.ID) ([]domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	raw := ctxkey.ClinicID(ctx)
	if raw != "" && raw != clinicID.String() {
		return nil, nil
	}
	linkIDs := make(map[id.ID]struct{})
	for _, l := range s.links {
		if l.ClinicID == clinicID && l.PlanID == planID {
			linkIDs[l.ID] = struct{}{}
		}
	}

	var out []domain.Event
	for i := len(s.events) - 1; i >= 0; i-- {
		ev := s.events[i]
		if _, ok := linkIDs[ev.LinkID]; ok {
			out = append(out, ev)
		}
	}
	return out, nil
}
