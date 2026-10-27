package mem

import (
	"context"
	"sync"

	"github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	mu   sync.RWMutex
	data map[id.ID]domain.Settings
}

func New() *Store {
	return &Store{data: make(map[id.ID]domain.Settings)}
}

func (s *Store) GetSettings(ctx context.Context, workspaceID id.ID) (domain.Settings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Tenant isolation check (RLS in memory)
	tenantClinicID := ctxkey.ClinicID(ctx)
	if tenantClinicID != "" && tenantClinicID != workspaceID.String() {
		// Mismatched tenant: returns 0 rows / not found
		return domain.Settings{}, errx.ErrNotFound
	}

	st, ok := s.data[workspaceID]
	if !ok {
		return domain.Settings{}, errx.ErrNotFound
	}
	return st, nil
}

func (s *Store) SaveSettings(ctx context.Context, settings domain.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Tenant isolation check (RLS in memory)
	tenantClinicID := ctxkey.ClinicID(ctx)
	if tenantClinicID != "" && tenantClinicID != settings.WorkspaceID.String() {
		return errx.ErrForbidden
	}

	s.data[settings.WorkspaceID] = settings
	return nil
}
