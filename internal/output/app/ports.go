package app

import (
	"context"

	"github.com/treatment-plus/treatment-plus-be/internal/output/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type PlanReader interface {
	GetView(ctx context.Context, clinicID, planID id.ID) (domain.PlanView, error)
}

type PDFRenderer interface {
	Render(ctx context.Context, snap domain.Snapshot, images Images) (pdf []byte, pages int, err error)
}

type Images struct {
	Logo   []byte
	Cover  []byte
	About  []byte
	Doctor []byte
	Xray   []byte
	Lab    []byte
	Hotel  []byte
	Award  []byte
	Smile  []byte
}

type AssetResolver interface {
	Resolve(ctx context.Context, workspaceID id.ID, key string) ([]byte, error)
}

type DocumentStore interface {
	SaveDocument(ctx context.Context, doc domain.Document) error
	GetDocument(ctx context.Context, workspaceID, planID id.ID) (domain.Document, error)
}
