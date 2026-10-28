package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/share/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Service struct {
	store   ShareStore
	plans   PlanReader
	clinics ClinicReader
	viz     VizReader
	pdf     PDFReader
	baseURL string
}

func NewService(
	store ShareStore,
	plans PlanReader,
	clinics ClinicReader,
	viz VizReader,
	pdf PDFReader,
	baseURL string,
) *Service {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Service{
		store:   store,
		plans:   plans,
		clinics: clinics,
		viz:     viz,
		pdf:     pdf,
		baseURL: baseURL,
	}
}

type ShareResult struct {
	Link      domain.Link `json:"link"`
	Token     string      `json:"token"`
	PublicURL string      `json:"public_url"`
}

func (s *Service) CreateShareLink(ctx context.Context, clinicID, planID id.ID, ttl time.Duration) (ShareResult, error) {
	if clinicID.IsZero() || planID.IsZero() {
		return ShareResult{}, errx.New("invalid_id", "clinic and plan id are required", errx.ErrInvalid)
	}
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour // Default 30 days
	}

	rawBytes := make([]byte, 24)
	if _, err := rand.Read(rawBytes); err != nil {
		return ShareResult{}, fmt.Errorf("share: generate token: %w", err)
	}
	rawToken := base64.RawURLEncoding.EncodeToString(rawBytes)
	tokenHash := domain.HashToken(rawToken)
	prefix := rawToken
	if len(prefix) > 8 {
		prefix = prefix[:8]
	}

	now := time.Now().UTC()
	link := domain.Link{
		ID:          id.New(),
		ClinicID:    clinicID,
		PlanID:      planID,
		TokenHash:   tokenHash,
		TokenPrefix: prefix,
		ExpiresAt:   now.Add(ttl),
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.CreateLink(ctx, link); err != nil {
		return ShareResult{}, err
	}

	publicURL := fmt.Sprintf("%s/p/%s", s.baseURL, rawToken)
	return ShareResult{
		Link:      link,
		Token:     rawToken,
		PublicURL: publicURL,
	}, nil
}

func (s *Service) ListShareLinks(ctx context.Context, clinicID, planID id.ID) ([]domain.Link, error) {
	if clinicID.IsZero() || planID.IsZero() {
		return nil, errx.New("invalid_id", "clinic and plan id are required", errx.ErrInvalid)
	}
	return s.store.ListLinks(ctx, clinicID, planID)
}

func (s *Service) RevokeShareLink(ctx context.Context, clinicID, linkID id.ID) error {
	if clinicID.IsZero() || linkID.IsZero() {
		return errx.New("invalid_id", "clinic and link id are required", errx.ErrInvalid)
	}
	return s.store.RevokeLink(ctx, clinicID, linkID)
}

type UltimatePlanDTO struct {
	Token       string                  `json:"token"`
	PublicURL   string                  `json:"public_url"`
	Plan        SharedPlanData          `json:"plan"`
	Clinic      SharedClinicData        `json:"clinic"`
	Viz         *SharedVizPlaylist      `json:"viz,omitempty"`
	BeforeAfter []SharedBeforeAfterCase `json:"before_after"`
	Accepted    bool                    `json:"accepted"`
	AcceptedAt  *time.Time              `json:"accepted_at,omitempty"`
	Signature   string                  `json:"signature,omitempty"`
	ExpiresAt   time.Time               `json:"expires_at"`
}

func (s *Service) GetUltimatePlan(ctx context.Context, rawToken string) (UltimatePlanDTO, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return UltimatePlanDTO{}, errx.New("invalid_token", "token is required", errx.ErrInvalid)
	}
	hash := domain.HashToken(rawToken)

	link, err := s.store.ResolveToken(ctx, hash)
	if err != nil {
		return UltimatePlanDTO{}, err
	}

	// Downstream readers (plans, clinics, viz, pdf) query tenant-isolated tables
	// governed by RLS. Populate the resolved tenant ID onto context.
	ctx = ctxkey.WithClinicID(ctx, link.ClinicID.String())

	planData, err := s.plans.GetPlanForShare(ctx, link.ClinicID, link.PlanID)
	if err != nil {
		return UltimatePlanDTO{}, err
	}

	clinicData, err := s.clinics.GetClinicBrand(ctx, link.ClinicID)
	if err != nil {
		return UltimatePlanDTO{}, err
	}

	// Filter before/after cases: if plan specified case IDs, filter; else use clinic's cases
	selectedCases := clinicData.BeforeAfter
	if len(planData.BeforeAfterCaseIDs) > 0 {
		lookup := make(map[string]struct{}, len(planData.BeforeAfterCaseIDs))
		for _, id := range planData.BeforeAfterCaseIDs {
			lookup[id] = struct{}{}
		}
		var filtered []SharedBeforeAfterCase
		for _, c := range clinicData.BeforeAfter {
			if _, ok := lookup[c.ID]; ok {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) > 0 {
			selectedCases = filtered
		}
	}

	var vizPlaylist *SharedVizPlaylist
	if s.viz != nil {
		if vp, err := s.viz.GetVizPlaylist(ctx, link.ClinicID, link.PlanID); err == nil && len(vp.Clips) > 0 {
			vizPlaylist = &vp
		}
	}

	publicURL := fmt.Sprintf("%s/p/%s", s.baseURL, rawToken)
	return UltimatePlanDTO{
		Token:       rawToken,
		PublicURL:   publicURL,
		Plan:        planData,
		Clinic:      clinicData,
		Viz:         vizPlaylist,
		BeforeAfter: selectedCases,
		Accepted:    link.IsAccepted(),
		AcceptedAt:  link.AcceptedAt,
		Signature:   link.AcceptedSignature,
		ExpiresAt:   link.ExpiresAt,
	}, nil
}

func (s *Service) RecordEvent(ctx context.Context, rawToken string, kind domain.EventKind, metadata map[string]any) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return errx.New("invalid_token", "token is required", errx.ErrInvalid)
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	hash := domain.HashToken(rawToken)
	return s.store.RecordEvent(ctx, hash, id.New(), kind, metadata)
}

func (s *Service) AcceptPlan(ctx context.Context, rawToken, signature, ip string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return errx.New("invalid_token", "token is required", errx.ErrInvalid)
	}
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return errx.New("signature_required", "patient signature or name is required", errx.ErrInvalid)
	}
	hash := domain.HashToken(rawToken)
	return s.store.AcceptPlan(ctx, hash, signature, ip)
}

func (s *Service) GetVizForToken(ctx context.Context, rawToken string) (SharedVizPlaylist, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return SharedVizPlaylist{}, errx.New("invalid_token", "token is required", errx.ErrInvalid)
	}
	hash := domain.HashToken(rawToken)
	link, err := s.store.ResolveToken(ctx, hash)
	if err != nil {
		return SharedVizPlaylist{}, err
	}
	ctx = ctxkey.WithClinicID(ctx, link.ClinicID.String())
	if s.viz == nil {
		return SharedVizPlaylist{}, errx.New("viz_unavailable", "visualization service unavailable", errx.ErrNotFound)
	}
	return s.viz.GetVizPlaylist(ctx, link.ClinicID, link.PlanID)
}

func (s *Service) GetPDFForToken(ctx context.Context, rawToken string) ([]byte, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, errx.New("invalid_token", "token is required", errx.ErrInvalid)
	}
	hash := domain.HashToken(rawToken)
	link, err := s.store.ResolveToken(ctx, hash)
	if err != nil {
		return nil, err
	}
	ctx = ctxkey.WithClinicID(ctx, link.ClinicID.String())
	if s.pdf == nil {
		return nil, errx.New("pdf_unavailable", "pdf service unavailable", errx.ErrNotFound)
	}
	return s.pdf.GetPlanPDF(ctx, link.ClinicID, link.PlanID)
}

type AnalyticsSummaryDTO struct {
	TotalViews   int            `json:"total_views"`
	UniqueLinks  int            `json:"unique_links"`
	IsAccepted   bool           `json:"is_accepted"`
	AcceptedAt   *time.Time     `json:"accepted_at,omitempty"`
	Signature    string         `json:"signature,omitempty"`
	LastViewedAt *time.Time     `json:"last_viewed_at,omitempty"`
	EventsCount  int            `json:"events_count"`
	EventsByKind map[string]int `json:"events_by_kind"`
	RecentEvents []domain.Event `json:"recent_events"`
	Links        []domain.Link  `json:"links"`
}

func (s *Service) GetAnalytics(ctx context.Context, clinicID, planID id.ID) (AnalyticsSummaryDTO, error) {
	if clinicID.IsZero() || planID.IsZero() {
		return AnalyticsSummaryDTO{}, errx.New("invalid_id", "clinic and plan id are required", errx.ErrInvalid)
	}
	links, err := s.store.ListLinks(ctx, clinicID, planID)
	if err != nil {
		return AnalyticsSummaryDTO{}, err
	}
	events, err := s.store.ListEvents(ctx, clinicID, planID)
	if err != nil {
		return AnalyticsSummaryDTO{}, err
	}

	summary := AnalyticsSummaryDTO{
		UniqueLinks:  len(links),
		EventsCount:  len(events),
		EventsByKind: make(map[string]int),
		RecentEvents: events,
		Links:        links,
	}

	for _, l := range links {
		summary.TotalViews += l.ViewsCount
		if l.AcceptedAt != nil {
			summary.IsAccepted = true
			summary.AcceptedAt = l.AcceptedAt
			summary.Signature = l.AcceptedSignature
		}
		if l.LastViewedAt != nil {
			if summary.LastViewedAt == nil || l.LastViewedAt.After(*summary.LastViewedAt) {
				summary.LastViewedAt = l.LastViewedAt
			}
		}
	}

	for _, e := range events {
		summary.EventsByKind[string(e.Kind)]++
	}

	return summary, nil
}
