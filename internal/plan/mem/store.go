package mem

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type patientRow struct {
	patient   domain.Patient
	encrypted string
}

type planRow struct {
	plan      domain.Plan
	visits    []domain.Visit
	items     []domain.Item
	discounts []domain.Discount
	snapshot  []byte
}

type Store struct {
	mu       sync.RWMutex
	patients map[id.ID]patientRow
	plans    map[id.ID]planRow
}

func New() *Store {
	return &Store{
		patients: make(map[id.ID]patientRow),
		plans:    make(map[id.ID]planRow),
	}
}

func denyRead(ctx context.Context, wsID id.ID) bool {
	raw := ctxkey.ClinicID(ctx)
	return raw != "" && raw != wsID.String()
}

func denyWrite(ctx context.Context, wsID id.ID) error {
	raw := ctxkey.ClinicID(ctx)
	if raw != "" && raw != wsID.String() {
		return errx.ErrForbidden
	}
	return nil
}

func (s *Store) ListPatients(ctx context.Context, wsID id.ID) ([]domain.Patient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return nil, nil
	}
	var out []domain.Patient
	for _, row := range s.patients {
		if row.patient.WorkspaceID != wsID {
			continue
		}
		p := row.patient
		p.Contact = row.encrypted
		out = append(out, p)
	}
	return out, nil
}

func (s *Store) GetPatient(ctx context.Context, wsID id.ID, patientID id.ID) (domain.Patient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return domain.Patient{}, errx.ErrNotFound
	}
	row, ok := s.patients[patientID]
	if !ok || row.patient.WorkspaceID != wsID {
		return domain.Patient{}, errx.ErrNotFound
	}
	p := row.patient
	p.Contact = row.encrypted
	return p, nil
}

func (s *Store) SavePatient(ctx context.Context, p domain.Patient, encryptedContact string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, p.WorkspaceID); err != nil {
		return err
	}
	s.patients[p.ID] = patientRow{patient: p, encrypted: encryptedContact}
	return nil
}

func (s *Store) ListPlans(ctx context.Context, wsID id.ID) ([]domain.Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return nil, nil
	}
	var out []domain.Plan
	for _, row := range s.plans {
		if row.plan.WorkspaceID != wsID {
			continue
		}
		out = append(out, s.hydrate(row))
	}
	return out, nil
}

func (s *Store) GetPlan(ctx context.Context, wsID id.ID, planID id.ID) (domain.Plan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return domain.Plan{}, errx.ErrNotFound
	}
	row, ok := s.plans[planID]
	if !ok || row.plan.WorkspaceID != wsID {
		return domain.Plan{}, errx.ErrNotFound
	}
	return s.hydrate(row), nil
}

func (s *Store) CreatePlan(ctx context.Context, p domain.Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, p.WorkspaceID); err != nil {
		return err
	}
	if _, exists := s.plans[p.ID]; exists {
		return errx.ErrConflict
	}
	s.plans[p.ID] = planRow{plan: p}
	return nil
}

func (s *Store) UpdatePlanHeader(ctx context.Context, p domain.Plan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, p.WorkspaceID); err != nil {
		return err
	}
	row, ok := s.plans[p.ID]
	if !ok || row.plan.WorkspaceID != p.WorkspaceID || row.plan.Status != domain.StatusDraft {
		return errx.ErrNotFound
	}
	row.plan.Title = p.Title
	row.plan.Diagnoses = append([]string(nil), p.Diagnoses...)
	row.plan.Palette = p.Palette
	row.plan.Locale = p.Locale
	row.plan.Currency = p.Currency
	row.plan.Situation = p.Situation
	row.plan.TotalsSubtotal = p.TotalsSubtotal
	row.plan.TotalsDiscount = p.TotalsDiscount
	row.plan.TotalsGrand = p.TotalsGrand
	row.plan.WelcomeLetter = p.WelcomeLetter
	row.plan.VideoURL = p.VideoURL
	row.plan.BeforeAfterCaseIDs = append([]string(nil), p.BeforeAfterCaseIDs...)
	row.plan.UpdatedAt = time.Now().UTC()
	s.plans[p.ID] = row
	return nil
}

func (s *Store) ReplaceVisits(ctx context.Context, wsID id.ID, planID id.ID, visits []domain.Visit) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, wsID); err != nil {
		return err
	}
	row, ok := s.plans[planID]
	if !ok || row.plan.WorkspaceID != wsID || row.plan.Status != domain.StatusDraft {
		return errx.ErrNotFound
	}
	row.visits = append([]domain.Visit(nil), visits...)
	row.plan.UpdatedAt = time.Now().UTC()
	s.plans[planID] = row
	return nil
}

func (s *Store) ReplaceItems(ctx context.Context, wsID id.ID, planID id.ID, items []domain.Item, totals domain.Totals) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, wsID); err != nil {
		return err
	}
	row, ok := s.plans[planID]
	if !ok || row.plan.WorkspaceID != wsID || row.plan.Status != domain.StatusDraft {
		return errx.ErrNotFound
	}
	row.items = append([]domain.Item(nil), items...)
	row.plan.TotalsSubtotal = totals.Subtotal
	row.plan.TotalsDiscount = totals.Discount
	row.plan.TotalsGrand = totals.Grand
	row.plan.UpdatedAt = time.Now().UTC()
	s.plans[planID] = row
	return nil
}

func (s *Store) ReplaceDiscounts(ctx context.Context, wsID id.ID, planID id.ID, discounts []domain.Discount, totals domain.Totals) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, wsID); err != nil {
		return err
	}
	row, ok := s.plans[planID]
	if !ok || row.plan.WorkspaceID != wsID || row.plan.Status != domain.StatusDraft {
		return errx.ErrNotFound
	}
	row.discounts = append([]domain.Discount(nil), discounts...)
	row.plan.TotalsSubtotal = totals.Subtotal
	row.plan.TotalsDiscount = totals.Discount
	row.plan.TotalsGrand = totals.Grand
	row.plan.UpdatedAt = time.Now().UTC()
	s.plans[planID] = row
	return nil
}

func (s *Store) SetReady(ctx context.Context, wsID id.ID, planID id.ID, snapshotJSON []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, wsID); err != nil {
		return err
	}
	row, ok := s.plans[planID]
	if !ok || row.plan.WorkspaceID != wsID {
		return errx.New("cannot_set_ready", "plan not found", errx.ErrNotFound)
	}
	if row.plan.Status != domain.StatusDraft {
		return nil // already ready or past draft (idempotent)
	}
	row.plan.Status = domain.StatusReady
	row.snapshot = append([]byte(nil), snapshotJSON...)
	row.plan.UpdatedAt = time.Now().UTC()
	s.plans[planID] = row
	return nil
}

func (s *Store) ReopenDraft(ctx context.Context, wsID id.ID, planID id.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, wsID); err != nil {
		return err
	}
	row, ok := s.plans[planID]
	if !ok || row.plan.WorkspaceID != wsID {
		return errx.ErrNotFound
	}
	row.plan.Status = domain.StatusDraft
	row.plan.UpdatedAt = time.Now().UTC()
	s.plans[planID] = row
	return nil
}

func (s *Store) hydrate(row planRow) domain.Plan {
	p := row.plan
	p.Visits = append([]domain.Visit(nil), row.visits...)
	p.Items = append([]domain.Item(nil), row.items...)
	p.Discounts = append([]domain.Discount(nil), row.discounts...)
	if len(row.snapshot) > 0 {
		p.SnapshotJSON = json.RawMessage(append([]byte(nil), row.snapshot...))
	}
	if pat, ok := s.patients[p.PatientID]; ok {
		copied := pat.patient
		copied.Contact = pat.encrypted
		p.Patient = &copied
	}
	return p
}
