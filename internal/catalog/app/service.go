package app

import (
	"context"
	"strings"

	"github.com/treatment-plus/treatment-plus-be/internal/catalog/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type Service struct {
	store CatalogStore
}

func NewService(store CatalogStore) *Service {
	return &Service{store: store}
}

// Diagnoses
func (s *Service) ListDiagnoses(ctx context.Context, wsID id.ID, includeInactive bool) ([]domain.Diagnosis, error) {
	if wsID.IsZero() {
		return nil, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	return s.store.ListDiagnoses(ctx, wsID, includeInactive)
}

func (s *Service) CreateDiagnosis(ctx context.Context, wsID id.ID, code, name string) (domain.Diagnosis, error) {
	if wsID.IsZero() {
		return domain.Diagnosis{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	d := domain.Diagnosis{
		ID:          id.New(),
		WorkspaceID: wsID,
		Code:        strings.TrimSpace(code),
		Name:        strings.TrimSpace(name),
		IsActive:    true,
	}
	if err := d.Validate(); err != nil {
		return domain.Diagnosis{}, err
	}
	if err := s.store.SaveDiagnosis(ctx, d); err != nil {
		return domain.Diagnosis{}, err
	}
	return d, nil
}

func (s *Service) UpdateDiagnosis(ctx context.Context, wsID id.ID, diagID id.ID, name *string, isActive *bool) (domain.Diagnosis, error) {
	if wsID.IsZero() || diagID.IsZero() {
		return domain.Diagnosis{}, errx.New("invalid_id", "workspace and diagnosis id required", errx.ErrInvalid)
	}
	current, err := s.store.GetDiagnosis(ctx, wsID, diagID)
	if err != nil {
		return domain.Diagnosis{}, err
	}
	if name != nil {
		current.Name = strings.TrimSpace(*name)
	}
	if isActive != nil {
		current.IsActive = *isActive
	}
	if err := current.Validate(); err != nil {
		return domain.Diagnosis{}, err
	}
	if err := s.store.SaveDiagnosis(ctx, current); err != nil {
		return domain.Diagnosis{}, err
	}
	return current, nil
}

// Treatments
func (s *Service) ListTreatments(ctx context.Context, wsID id.ID, includeInactive bool) ([]domain.Treatment, error) {
	if wsID.IsZero() {
		return nil, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	return s.store.ListTreatments(ctx, wsID, includeInactive)
}

func (s *Service) CreateTreatment(ctx context.Context, wsID id.ID, code, name string) (domain.Treatment, error) {
	if wsID.IsZero() {
		return domain.Treatment{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	t := domain.Treatment{
		ID:          id.New(),
		WorkspaceID: wsID,
		Code:        strings.TrimSpace(code),
		Name:        strings.TrimSpace(name),
		IsActive:    true,
	}
	if err := t.Validate(); err != nil {
		return domain.Treatment{}, err
	}
	if err := s.store.SaveTreatment(ctx, t); err != nil {
		return domain.Treatment{}, err
	}
	return t, nil
}

func (s *Service) UpdateTreatment(ctx context.Context, wsID id.ID, treatID id.ID, name *string, isActive *bool) (domain.Treatment, error) {
	if wsID.IsZero() || treatID.IsZero() {
		return domain.Treatment{}, errx.New("invalid_id", "workspace and treatment id required", errx.ErrInvalid)
	}
	current, err := s.store.GetTreatment(ctx, wsID, treatID)
	if err != nil {
		return domain.Treatment{}, err
	}
	if name != nil {
		current.Name = strings.TrimSpace(*name)
	}
	if isActive != nil {
		current.IsActive = *isActive
	}
	if err := current.Validate(); err != nil {
		return domain.Treatment{}, err
	}
	if err := s.store.SaveTreatment(ctx, current); err != nil {
		return domain.Treatment{}, err
	}
	return current, nil
}

func (s *Service) DeleteTreatment(ctx context.Context, wsID id.ID, treatID id.ID) error {
	if wsID.IsZero() || treatID.IsZero() {
		return errx.New("invalid_id", "workspace and treatment id required", errx.ErrInvalid)
	}
	return s.store.DeleteTreatment(ctx, wsID, treatID)
}

// Price Lists
func (s *Service) ListPriceLists(ctx context.Context, wsID id.ID) ([]domain.PriceList, error) {
	if wsID.IsZero() {
		return nil, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	return s.store.ListPriceLists(ctx, wsID)
}

func (s *Service) GetPriceList(ctx context.Context, wsID id.ID, listID id.ID) (domain.PriceList, []domain.PriceItem, error) {
	if wsID.IsZero() || listID.IsZero() {
		return domain.PriceList{}, nil, errx.New("invalid_id", "workspace and price list id required", errx.ErrInvalid)
	}
	return s.store.GetPriceList(ctx, wsID, listID)
}

func (s *Service) CreatePriceList(ctx context.Context, wsID id.ID, name string, currency money.Currency, isDefault bool) (domain.PriceList, error) {
	if wsID.IsZero() {
		return domain.PriceList{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	pl := domain.PriceList{
		ID:          id.New(),
		WorkspaceID: wsID,
		Name:        strings.TrimSpace(name),
		Currency:    currency,
		IsDefault:   isDefault,
		IsActive:    true,
	}
	if err := pl.Validate(); err != nil {
		return domain.PriceList{}, err
	}
	if err := s.store.SavePriceList(ctx, pl); err != nil {
		return domain.PriceList{}, err
	}
	return pl, nil
}

type SetPriceItemInput struct {
	TreatmentID id.ID `json:"treatment_id"`
	Amount      int64 `json:"amount"` // Minor unit
}

func (s *Service) SetPriceItems(ctx context.Context, wsID id.ID, listID id.ID, items []SetPriceItemInput) ([]domain.PriceItem, error) {
	if wsID.IsZero() || listID.IsZero() {
		return nil, errx.New("invalid_id", "workspace and price list id required", errx.ErrInvalid)
	}
	var domainItems []domain.PriceItem
	for _, it := range items {
		pi := domain.PriceItem{
			ID:          id.New(),
			WorkspaceID: wsID,
			PriceListID: listID,
			TreatmentID: it.TreatmentID,
			Amount:      it.Amount,
		}
		if err := pi.Validate(); err != nil {
			return nil, err
		}
		domainItems = append(domainItems, pi)
	}
	if err := s.store.SavePriceItems(ctx, wsID, listID, domainItems); err != nil {
		return nil, err
	}
	return domainItems, nil
}

// PlanReader helper
func (s *Service) GetTreatmentPrice(ctx context.Context, wsID id.ID, treatmentID id.ID) (int64, domain.Treatment, error) {
	t, err := s.store.GetTreatment(ctx, wsID, treatmentID)
	if err != nil {
		return 0, domain.Treatment{}, err
	}
	_, items, err := s.store.GetDefaultPriceList(ctx, wsID)
	if err != nil {
		return 0, t, nil
	}
	for _, item := range items {
		if item.TreatmentID == treatmentID {
			return item.Amount, t, nil
		}
	}
	return 0, t, nil
}
