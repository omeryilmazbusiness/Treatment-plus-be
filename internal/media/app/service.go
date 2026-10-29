package app

import (
	"context"
	"strings"

	"github.com/treatment-plus/treatment-plus-be/internal/media/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Service struct {
	store AssetStore
}

func NewService(store AssetStore) *Service {
	return &Service{store: store}
}

func (s *Service) Upload(ctx context.Context, workspaceID id.ID, kind, contentType string, raw []byte) (domain.Asset, error) {
	if workspaceID.IsZero() {
		return domain.Asset{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	k, err := domain.ParseKind(kind)
	if err != nil {
		return domain.Asset{}, err
	}
	ct := sniff(contentType, raw)
	asset := domain.Asset{
		ID:          id.New(),
		WorkspaceID: workspaceID,
		Key:         workspaceID.String() + "/" + k.String() + "/" + id.New().String(),
		Kind:        k,
		ContentType: ct,
		Bytes:       raw,
	}
	if err := asset.Validate(); err != nil {
		return domain.Asset{}, err
	}
	if err := s.store.SaveAsset(ctx, asset); err != nil {
		return domain.Asset{}, err
	}
	return asset, nil
}

func (s *Service) Get(ctx context.Context, workspaceID id.ID, key string) (domain.Asset, error) {
	if workspaceID.IsZero() || strings.TrimSpace(key) == "" {
		return domain.Asset{}, errx.New("invalid_id", "workspace and key required", errx.ErrInvalid)
	}
	return s.store.GetAsset(ctx, workspaceID, key)
}

func (s *Service) Resolve(ctx context.Context, workspaceID id.ID, key string) ([]byte, error) {
	a, err := s.Get(ctx, workspaceID, key)
	if err != nil {
		return nil, err
	}
	return a.Bytes, nil
}

func sniff(declared string, raw []byte) string {
	if kind := domain.DetectImage(raw); kind != "" {
		return kind
	}
	switch strings.ToLower(strings.TrimSpace(declared)) {
	case "image/jpeg", "image/jpg":
		return "image/jpeg"
	case "image/png":
		return "image/png"
	case "image/webp":
		return "image/webp"
	default:
		return declared
	}
}
