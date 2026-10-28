package mem

import (
	"context"
	"sync"

	"github.com/treatment-plus/treatment-plus-be/internal/output/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	mu   sync.RWMutex
	docs map[id.ID]domain.Document
}

func New() *Store {
	return &Store{docs: make(map[id.ID]domain.Document)}
}

func deny(ctx context.Context, wsID id.ID) bool {
	raw := ctxkey.ClinicID(ctx)
	return raw != "" && raw != wsID.String()
}

func (s *Store) SaveDocument(ctx context.Context, doc domain.Document) error {
	if deny(ctx, doc.WorkspaceID) {
		return errx.ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs[doc.PlanID] = doc
	return nil
}

func (s *Store) GetDocument(ctx context.Context, workspaceID, planID id.ID) (domain.Document, error) {
	if deny(ctx, workspaceID) {
		return domain.Document{}, errx.ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.docs[planID]
	if !ok || doc.WorkspaceID != workspaceID {
		return domain.Document{}, errx.ErrNotFound
	}
	return doc, nil
}
