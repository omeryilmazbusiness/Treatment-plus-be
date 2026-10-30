package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/output/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Service struct {
	plans  PlanReader
	render PDFRenderer
	assets AssetResolver
	docs   DocumentStore
	award  []byte
	now    func() time.Time
}

func NewService(plans PlanReader, render PDFRenderer, assets AssetResolver, docs DocumentStore, award []byte) *Service {
	return &Service{
		plans:  plans,
		render: render,
		assets: assets,
		docs:   docs,
		award:  award,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Generate(ctx context.Context, workspaceID, planID id.ID) (domain.Document, error) {
	if workspaceID.IsZero() || planID.IsZero() {
		return domain.Document{}, errx.New("invalid_id", "workspace and plan id required", errx.ErrInvalid)
	}
	if s.plans == nil || s.render == nil || s.docs == nil {
		return domain.Document{}, errx.New("pdf_unavailable", "pdf generator is not configured", errx.ErrUnavailable)
	}

	view, err := s.plans.GetView(ctx, workspaceID, planID)
	if err != nil {
		return domain.Document{}, err
	}
	var snap domain.Snapshot
	if err := json.Unmarshal(view.Payload, &snap); err != nil {
		return domain.Document{}, errx.New("invalid_snapshot", "ready snapshot is unreadable", errx.ErrInvalid)
	}
	if strings.TrimSpace(snap.PlanID) == "" {
		return domain.Document{}, errx.New("plan_not_ready", "plan must be ready before PDF generate", errx.ErrInvalid)
	}

	images := Images{Award: s.award}
	images.Logo = s.resolve(ctx, workspaceID, snap.Clinic.LogoKey)
	images.Cover = s.resolve(ctx, workspaceID, snap.Clinic.CoverKey)
	images.About = s.resolve(ctx, workspaceID, snap.Clinic.AboutKey)
	images.Doctor = s.resolve(ctx, workspaceID, snap.Clinic.Doctor.PhotoKey)
	images.Xray = s.resolve(ctx, workspaceID, snap.Situation.XrayKey)
	images.Lab = s.resolve(ctx, workspaceID, snap.Clinic.Lab.ImageKey)
	images.Hotel = s.resolve(ctx, workspaceID, snap.Clinic.Hotel.ImageKey)
	images.Smile = s.resolve(ctx, workspaceID, snap.Clinic.SmileKey)

	raw, pages, err := s.render.Render(ctx, snap, images)
	if err != nil {
		return domain.Document{}, err
	}
	if pages != 24 {
		return domain.Document{}, errx.New("pdf_page_count", "classic booklet must be 24 pages", errx.ErrInvalid)
	}
	if len(raw) < 5 || string(raw[:4]) != "%PDF" {
		return domain.Document{}, errx.New("pdf_invalid", "renderer did not produce a PDF", errx.ErrInvalid)
	}

	doc := domain.Document{
		PlanID:      planID,
		WorkspaceID: workspaceID,
		ContentType: domain.ContentTypePDF,
		PageCount:   pages,
		Bytes:       raw,
		CreatedAt:   s.now(),
	}
	if err := s.docs.SaveDocument(ctx, doc); err != nil {
		return domain.Document{}, err
	}
	return doc, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, planID id.ID) (domain.Document, error) {
	if workspaceID.IsZero() || planID.IsZero() {
		return domain.Document{}, errx.New("invalid_id", "workspace and plan id required", errx.ErrInvalid)
	}
	if s.docs == nil {
		return domain.Document{}, errx.New("pdf_unavailable", "pdf store is not configured", errx.ErrUnavailable)
	}
	return s.docs.GetDocument(ctx, workspaceID, planID)
}

func (s *Service) resolve(ctx context.Context, workspaceID id.ID, key string) []byte {
	key = strings.TrimSpace(key)
	if key == "" || s.assets == nil {
		return nil
	}
	b, err := s.assets.Resolve(ctx, workspaceID, key)
	if err != nil {
		return nil
	}
	return b
}
