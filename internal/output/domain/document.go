package domain

import (
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

const ContentTypePDF = "application/pdf"

type Document struct {
	PlanID      id.ID
	WorkspaceID id.ID
	ContentType string
	PageCount   int
	Bytes       []byte
	CreatedAt   time.Time
}

func (d Document) Size() int {
	return len(d.Bytes)
}
