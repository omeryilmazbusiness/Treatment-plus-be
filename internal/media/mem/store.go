package mem

import (
	"context"
	"sync"

	"github.com/treatment-plus/treatment-plus-be/internal/media/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	mu   sync.RWMutex
	data map[string]domain.Asset
}

func New() *Store {
	return &Store{data: make(map[string]domain.Asset)}
}

func denyRead(ctx context.Context, wsID id.ID) bool {
	raw := ctxkey.ClinicID(ctx)
	return raw != "" && raw != wsID.String()
}

func (s *Store) SaveAsset(ctx context.Context, asset domain.Asset) error {
	if denyRead(ctx, asset.WorkspaceID) {
		return errx.ErrForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := asset
	cp.Bytes = append([]byte(nil), asset.Bytes...)
	s.data[asset.WorkspaceID.String()+"|"+asset.Key] = cp
	return nil
}

func (s *Store) GetAsset(ctx context.Context, workspaceID id.ID, key string) (domain.Asset, error) {
	if denyRead(ctx, workspaceID) {
		return domain.Asset{}, errx.ErrNotFound
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.data[workspaceID.String()+"|"+key]
	if !ok {
		return domain.Asset{}, errx.ErrNotFound
	}
	return a, nil
}
