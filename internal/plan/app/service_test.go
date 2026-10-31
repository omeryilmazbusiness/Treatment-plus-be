package app_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/plan/app"
	"github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

// ---- Mocks & Stubs for Unit Testing (Self-Test Requirement) ----

type stubEncrypter struct {
	failEncrypt bool
	failDecrypt bool
}

func (e *stubEncrypter) Encrypt(plaintext string) (string, error) {
	if e.failEncrypt {
		return "", errors.New("kms error: encryption key expired")
	}
	return "enc:" + plaintext, nil
}

func (e *stubEncrypter) Decrypt(ciphertext string) (string, error) {
	if e.failDecrypt {
		return "", errors.New("kms error: decryption failed")
	}
	return strings.TrimPrefix(ciphertext, "enc:"), nil
}

type stubCatalog struct {
	treatments map[id.ID]struct {
		code  string
		name  string
		price int64
	}
	fail bool
}

func newStubCatalog() *stubCatalog {
	return &stubCatalog{
		treatments: make(map[id.ID]struct {
			code  string
			name  string
			price int64
		}),
	}
}

func (c *stubCatalog) GetTreatment(_ context.Context, _ id.ID, treatID id.ID) (string, string, int64, error) {
	if c.fail {
		return "", "", 0, errors.New("catalog service unavailable")
	}
	t, ok := c.treatments[treatID]
	if !ok {
		return "", "", 0, errx.ErrNotFound
	}
	return t.code, t.name, t.price, nil
}

type stubSettings struct {
	settings map[id.ID]app.ClinicSettingsDTO
	fail     bool
}

func newStubSettings() *stubSettings {
	return &stubSettings{settings: make(map[id.ID]app.ClinicSettingsDTO)}
}

func (s *stubSettings) GetClinicSettings(_ context.Context, wsID id.ID) (app.ClinicSettingsDTO, error) {
	if s.fail {
		return app.ClinicSettingsDTO{}, errors.New("settings storage failure")
	}
	st, ok := s.settings[wsID]
	if !ok {
		return app.ClinicSettingsDTO{
			Name:     "Default Clinic",
			Palette:  "navy",
			Locale:   "tr",
			Currency: "TRY",
		}, nil
	}
	return st, nil
}

type mockPatientStore struct {
	mu       sync.Mutex
	patients map[id.ID]domain.Patient
	contacts map[id.ID]string
	saveErr  error
	getErr   error
}

func newMockPatientStore() *mockPatientStore {
	return &mockPatientStore{
		patients: make(map[id.ID]domain.Patient),
		contacts: make(map[id.ID]string),
	}
}

func (m *mockPatientStore) ListPatients(_ context.Context, wsID id.ID) ([]domain.Patient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Patient
	for _, p := range m.patients {
		if p.WorkspaceID == wsID {
			p.Contact = m.contacts[p.ID]
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *mockPatientStore) GetPatient(_ context.Context, wsID id.ID, patID id.ID) (domain.Patient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.getErr != nil {
		return domain.Patient{}, m.getErr
	}
	p, ok := m.patients[patID]
	if !ok || p.WorkspaceID != wsID {
		return domain.Patient{}, errx.ErrNotFound
	}
	p.Contact = m.contacts[patID]
	return p, nil
}

func (m *mockPatientStore) SavePatient(_ context.Context, p domain.Patient, encryptedContact string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.saveErr != nil {
		return m.saveErr
	}
	m.patients[p.ID] = p
	m.contacts[p.ID] = encryptedContact
	return nil
}

type mockPlanStore struct {
	mu        sync.Mutex
	plans     map[id.ID]domain.Plan
	visits    map[id.ID][]domain.Visit
	items     map[id.ID][]domain.Item
	discounts map[id.ID][]domain.Discount
	snapshots map[id.ID][]byte
	createErr error
	updateErr error
}

func newMockPlanStore() *mockPlanStore {
	return &mockPlanStore{
		plans:     make(map[id.ID]domain.Plan),
		visits:    make(map[id.ID][]domain.Visit),
		items:     make(map[id.ID][]domain.Item),
		discounts: make(map[id.ID][]domain.Discount),
		snapshots: make(map[id.ID][]byte),
	}
}

func (m *mockPlanStore) ListPlans(_ context.Context, wsID id.ID) ([]domain.Plan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Plan
	for _, p := range m.plans {
		if p.WorkspaceID == wsID {
			p.Visits = m.visits[p.ID]
			p.Items = m.items[p.ID]
			p.Discounts = m.discounts[p.ID]
			p.SnapshotJSON = m.snapshots[p.ID]
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *mockPlanStore) GetPlan(_ context.Context, wsID id.ID, planID id.ID) (domain.Plan, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plans[planID]
	if !ok || p.WorkspaceID != wsID {
		return domain.Plan{}, errx.ErrNotFound
	}
	p.Visits = m.visits[planID]
	p.Items = m.items[planID]
	p.Discounts = m.discounts[planID]
	if snap := m.snapshots[planID]; len(snap) > 0 {
		p.SnapshotJSON = snap
	}
	return p, nil
}

func (m *mockPlanStore) CreatePlan(_ context.Context, p domain.Plan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	m.plans[p.ID] = p
	return nil
}

func (m *mockPlanStore) UpdatePlanHeader(_ context.Context, p domain.Plan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.updateErr != nil {
		return m.updateErr
	}
	current, ok := m.plans[p.ID]
	if !ok || current.WorkspaceID != p.WorkspaceID || !current.CanModify() {
		return errx.ErrNotFound
	}
	current.Title = p.Title
	current.Palette = p.Palette
	current.Locale = p.Locale
	current.Currency = p.Currency
	current.Diagnoses = p.Diagnoses
	current.Situation = p.Situation
	current.TotalsSubtotal = p.TotalsSubtotal
	current.TotalsDiscount = p.TotalsDiscount
	current.TotalsGrand = p.TotalsGrand
	current.WelcomeLetter = p.WelcomeLetter
	current.VideoURL = p.VideoURL
	current.BeforeAfterCaseIDs = p.BeforeAfterCaseIDs
	current.UpdatedAt = time.Now().UTC()
	m.plans[p.ID] = current
	return nil
}

func (m *mockPlanStore) ReplaceVisits(_ context.Context, wsID id.ID, planID id.ID, visits []domain.Visit) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plans[planID]
	if !ok || p.WorkspaceID != wsID || !p.CanModify() {
		return errx.ErrNotFound
	}
	m.visits[planID] = append([]domain.Visit(nil), visits...)
	p.UpdatedAt = time.Now().UTC()
	m.plans[planID] = p
	return nil
}

func (m *mockPlanStore) ReplaceItems(_ context.Context, wsID id.ID, planID id.ID, items []domain.Item, totals domain.Totals) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plans[planID]
	if !ok || p.WorkspaceID != wsID || !p.CanModify() {
		return errx.ErrNotFound
	}
	m.items[planID] = append([]domain.Item(nil), items...)
	p.TotalsSubtotal = totals.Subtotal
	p.TotalsDiscount = totals.Discount
	p.TotalsGrand = totals.Grand
	p.UpdatedAt = time.Now().UTC()
	m.plans[planID] = p
	return nil
}

func (m *mockPlanStore) ReplaceDiscounts(_ context.Context, wsID id.ID, planID id.ID, discounts []domain.Discount, totals domain.Totals) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plans[planID]
	if !ok || p.WorkspaceID != wsID || !p.CanModify() {
		return errx.ErrNotFound
	}
	m.discounts[planID] = append([]domain.Discount(nil), discounts...)
	p.TotalsSubtotal = totals.Subtotal
	p.TotalsDiscount = totals.Discount
	p.TotalsGrand = totals.Grand
	p.UpdatedAt = time.Now().UTC()
	m.plans[planID] = p
	return nil
}

func (m *mockPlanStore) SetReady(_ context.Context, wsID id.ID, planID id.ID, snapshotJSON []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plans[planID]
	if !ok || p.WorkspaceID != wsID {
		return errx.New("cannot_set_ready", "plan not found", errx.ErrNotFound)
	}
	if !p.CanModify() {
		return nil // idempotent
	}
	p.Status = domain.StatusReady
	p.UpdatedAt = time.Now().UTC()
	m.plans[planID] = p
	m.snapshots[planID] = append([]byte(nil), snapshotJSON...)
	return nil
}

func (m *mockPlanStore) ReopenDraft(_ context.Context, wsID id.ID, planID id.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.plans[planID]
	if !ok || p.WorkspaceID != wsID {
		return errx.New("cannot_reopen", "plan not found", errx.ErrNotFound)
	}
	p.Status = domain.StatusDraft
	p.UpdatedAt = time.Now().UTC()
	m.plans[planID] = p
	return nil
}

func ptr[T any](v T) *T { return &v }

func setupTestService() (*app.Service, *mockPlanStore, *mockPatientStore, *stubCatalog, *stubSettings, *stubEncrypter) {
	planStore := newMockPlanStore()
	patStore := newMockPatientStore()
	cat := newStubCatalog()
	sett := newStubSettings()
	enc := &stubEncrypter{}

	svc := app.NewService(planStore, patStore, cat, sett, enc)
	return svc, planStore, patStore, cat, sett, enc
}

// ---- Unit Test Cases ----

func TestService_PatientManagement(t *testing.T) {
	svc, _, patStore, _, _, enc := setupTestService()
	ctx := context.Background()
	wsID := id.New()

	// 1. Validation error on empty name
	_, err := svc.CreatePatient(ctx, wsID, "   ", "555-1234")
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for empty patient name, got %v", err)
	}

	// 2. Successful creation with contact encryption
	p, err := svc.CreatePatient(ctx, wsID, "  Fatma Hanım  ", "0555 123 4567")
	if err != nil {
		t.Fatalf("unexpected error creating patient: %v", err)
	}
	if p.Name != "Fatma Hanım" {
		t.Errorf("expected trimmed name 'Fatma Hanım', got %q", p.Name)
	}
	if p.Contact != "0555 123 4567" {
		t.Errorf("expected plaintext in domain output, got %q", p.Contact)
	}
	// Verify stored contact is encrypted
	if patStore.contacts[p.ID] != "enc:0555 123 4567" {
		t.Errorf("expected encrypted contact in storage, got %q", patStore.contacts[p.ID])
	}

	// 3. GetPatient with decryption
	fetched, err := svc.GetPatient(ctx, wsID, p.ID)
	if err != nil {
		t.Fatalf("unexpected error getting patient: %v", err)
	}
	if fetched.Contact != "0555 123 4567" {
		t.Errorf("expected decrypted contact, got %q", fetched.Contact)
	}

	// 4. UpdatePatient
	updated, err := svc.UpdatePatient(ctx, wsID, p.ID, ptr("Fatma Kaya"), ptr("0555 999 8888"))
	if err != nil {
		t.Fatalf("unexpected error updating patient: %v", err)
	}
	if updated.Name != "Fatma Kaya" || updated.Contact != "0555 999 8888" {
		t.Errorf("updated patient mismatch: %+v", updated)
	}

	// 5. Decryption error is safely handled without crash
	enc.failDecrypt = true
	list, err := svc.ListPatients(ctx, wsID)
	if err != nil {
		t.Fatalf("ListPatients should not fail even if decryption fails: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 patient in list, got %d", len(list))
	}
}

func TestService_CreatePlan_ValidationAndDefaults(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	wsID := id.New()

	// Zero workspace or patient ID
	_, err := svc.CreatePlan(ctx, id.ID{}, app.CreatePlanInput{PatientID: id.New()})
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected ErrInvalid for zero workspace, got %v", err)
	}

	// Non-existent patient
	_, err = svc.CreatePlan(ctx, wsID, app.CreatePlanInput{PatientID: id.New()})
	if err == nil || !errors.Is(err, errx.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for non-existent patient, got %v", err)
	}

	// Create valid patient first
	pat, err := svc.CreatePatient(ctx, wsID, "Ali Veli", "555-1111")
	if err != nil {
		t.Fatal(err)
	}

	// Create plan with default fallback values
	plan, err := svc.CreatePlan(ctx, wsID, app.CreatePlanInput{
		PatientID: pat.ID,
		Title:     "", // Should default to "Tedavi Planı"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Title != "Tedavi Planı" {
		t.Errorf("expected default title, got %q", plan.Title)
	}
	if plan.Status != domain.StatusDraft {
		t.Errorf("expected draft status, got %q", plan.Status)
	}
	if plan.Currency != money.TRY {
		t.Errorf("expected default currency TRY, got %q", plan.Currency)
	}
	if plan.Palette != domain.PaletteNavy {
		t.Errorf("expected default palette navy, got %q", plan.Palette)
	}
}

func TestService_PlanModification_DraftVersusLocked(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	wsID := id.New()

	pat, _ := svc.CreatePatient(ctx, wsID, "Test Patient", "555")
	plan, err := svc.CreatePlan(ctx, wsID, app.CreatePlanInput{
		PatientID: pat.ID,
		Title:     "Draft Plan",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 1. Add visits on draft plan -> should succeed
	plan, err = svc.PutVisits(ctx, wsID, plan.ID, []app.PutVisitInput{
		{Title: "Visit 1", PhaseLabel: "Phase A"},
	})
	if err != nil {
		t.Fatalf("PutVisits failed on draft plan: %v", err)
	}
	if len(plan.Visits) != 1 || plan.Visits[0].OrderNum != 1 {
		t.Fatalf("visits mismatch: %+v", plan.Visits)
	}

	// 2. Add items on draft plan -> should succeed
	plan, err = svc.PutItems(ctx, wsID, plan.ID, []app.PutItemInput{
		{TreatmentName: "Dolgu", Quantity: 2, UnitPrice: ptr(int64(50000))},
	})
	if err != nil {
		t.Fatalf("PutItems failed on draft plan: %v", err)
	}
	if plan.TotalsSubtotal != 100000 || plan.TotalsGrand != 100000 {
		t.Fatalf("expected subtotal 100000, got %d", plan.TotalsSubtotal)
	}

	// 3. Add discount on draft plan -> should succeed
	plan, err = svc.PutDiscounts(ctx, wsID, plan.ID, []app.PutDiscountInput{
		{Label: "İndirim", Amount: 20000},
	})
	if err != nil {
		t.Fatalf("PutDiscounts failed on draft plan: %v", err)
	}
	if plan.TotalsDiscount != 20000 || plan.TotalsGrand != 80000 {
		t.Fatalf("expected grand total 80000, got %d", plan.TotalsGrand)
	}

	// 4. Lock plan with SetReady
	readyPlan, err := svc.SetReady(ctx, wsID, plan.ID)
	if err != nil {
		t.Fatalf("SetReady failed: %v", err)
	}
	if readyPlan.Status != domain.StatusReady {
		t.Fatalf("expected ready status, got %s", readyPlan.Status)
	}
	if readyPlan.CanModify() {
		t.Fatalf("locked plan must report CanModify() = false")
	}

	// 5. Attempt modifying locked plan -> MUST FAIL with ErrPlanLocked
	_, err = svc.UpdatePlanHeader(ctx, wsID, plan.ID, app.UpdatePlanHeaderInput{
		Title: ptr("New Title"),
	})
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected ErrPlanLocked on UpdatePlanHeader, got %v", err)
	}

	_, err = svc.PutVisits(ctx, wsID, plan.ID, []app.PutVisitInput{{Title: "Visit 2"}})
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected ErrPlanLocked on PutVisits, got %v", err)
	}

	_, err = svc.PutItems(ctx, wsID, plan.ID, []app.PutItemInput{{TreatmentName: "Yeni Dolgu", Quantity: 1}})
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected ErrPlanLocked on PutItems, got %v", err)
	}

	_, err = svc.PutDiscounts(ctx, wsID, plan.ID, []app.PutDiscountInput{{Label: "Yeni İndirim", Amount: 1000}})
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("expected ErrPlanLocked on PutDiscounts, got %v", err)
	}

	// 6. Calling SetReady again on already ready plan is IDEMPOTENT (returns plan, no error)
	againPlan, err := svc.SetReady(ctx, wsID, plan.ID)
	if err != nil {
		t.Fatalf("idempotent SetReady failed: %v", err)
	}
	if againPlan.Status != domain.StatusReady {
		t.Fatalf("expected status ready, got %s", againPlan.Status)
	}
}

func TestService_PutItems_CatalogLookupAndTotals(t *testing.T) {
	svc, _, _, catalog, _, _ := setupTestService()
	ctx := context.Background()
	wsID := id.New()
	treatID := id.New()

	catalog.treatments[treatID] = struct {
		code  string
		name  string
		price int64
	}{
		code:  "IMP-01",
		name:  "İsviçre İmplant",
		price: 2500000, // 25,000.00 TL
	}

	pat, _ := svc.CreatePatient(ctx, wsID, "İmplant Hastası", "555")
	plan, _ := svc.CreatePlan(ctx, wsID, app.CreatePlanInput{PatientID: pat.ID})

	// Put item referring to catalog treatment without providing name or price
	plan, err := svc.PutItems(ctx, wsID, plan.ID, []app.PutItemInput{
		{
			TreatmentID: ptr(treatID.String()),
			Quantity:    2,
			ToothNumber: ptr("36"),
			Teeth:       []string{"36", "46"},
		},
	})
	if err != nil {
		t.Fatalf("PutItems failed: %v", err)
	}

	if len(plan.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(plan.Items))
	}
	it := plan.Items[0]
	if it.TreatmentName != "İsviçre İmplant" {
		t.Errorf("expected catalog name 'İsviçre İmplant', got %q", it.TreatmentName)
	}
	if it.UnitPrice != 2500000 {
		t.Errorf("expected catalog unit price 2500000, got %d", it.UnitPrice)
	}
	if plan.TotalsSubtotal != 5000000 { // 2 * 25000.00 = 50000.00
		t.Errorf("expected subtotal 5000000, got %d", plan.TotalsSubtotal)
	}
}

func TestService_CreateClassic_EndToEnd(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	wsID := id.New()

	input := app.ClassicPlanInput{
		Palette: "forest",
		Locale:  "en",
		Patient: app.ClassicPatientInput{
			Name:    "John Doe",
			Contact: "+1 555 123 4567",
		},
		Title:     "Comprehensive Restoration",
		Diagnoses: []string{"Caries"},
		Situation: domain.Situation{
			GeneralDiagnoses: []string{"Bruxism"},
		},
		Visits: []app.PutVisitInput{
			{Title: "Session 1 - Preparation"},
			{Title: "Session 2 - Cementation"},
		},
		Items: []app.PutItemInput{
			{TreatmentName: "Zirconia Crown", Quantity: 2, UnitPrice: ptr(int64(300000))},
		},
		Discounts: []app.PutDiscountInput{
			{Label: "First Visit Discount", Amount: 50000},
		},
	}

	plan, err := svc.CreateClassic(ctx, wsID, input)
	if err != nil {
		t.Fatalf("CreateClassic failed: %v", err)
	}

	if plan.Title != "Comprehensive Restoration" {
		t.Errorf("expected title, got %q", plan.Title)
	}
	if plan.Palette != domain.PaletteForest || plan.Locale != domain.LocaleEN {
		t.Errorf("presentation mismatch: palette=%s locale=%s", plan.Palette, plan.Locale)
	}
	if len(plan.Visits) != 2 {
		t.Errorf("expected 2 visits, got %d", len(plan.Visits))
	}
	if len(plan.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(plan.Items))
	}
	if plan.TotalsGrand != 550000 { // 600000 - 50000
		t.Errorf("expected grand total 550000, got %d", plan.TotalsGrand)
	}
}

func TestService_CreateUltimate_EndToEnd(t *testing.T) {
	svc, _, _, _, _, _ := setupTestService()
	ctx := context.Background()
	wsID := id.New()

	input := app.UltimatePlanInput{
		Palette: "rose",
		Locale:  "tr",
		Patient: app.ClassicPatientInput{
			Name:    "Zeynep Demir",
			Contact: "0532 000 0000",
		},
		Title:              "Gülüş Tasarımı",
		WelcomeLetter:      "Kliniğimize hoş geldiniz.",
		VideoURL:           "https://example.com/video.mp4",
		BeforeAfterCaseIDs: []string{"case-1", "case-2"},
		Visits: []app.PutVisitInput{
			{Title: "1. Seans", PhaseLabel: "Hazırlık"},
		},
		Items: []app.PutItemInput{
			{TreatmentName: "E-Max Lamine", Quantity: 6, UnitPrice: ptr(int64(400000))},
		},
	}

	plan, err := svc.CreateUltimate(ctx, wsID, input)
	if err != nil {
		t.Fatalf("CreateUltimate failed: %v", err)
	}

	if plan.Kind != domain.KindUltimate {
		t.Errorf("expected KindUltimate, got %q", plan.Kind)
	}
	if plan.WelcomeLetter != "Kliniğimize hoş geldiniz." {
		t.Errorf("welcome letter mismatch: %q", plan.WelcomeLetter)
	}
	if len(plan.BeforeAfterCaseIDs) != 2 {
		t.Errorf("before after case IDs mismatch: %v", plan.BeforeAfterCaseIDs)
	}
	if plan.TotalsGrand != 2400000 {
		t.Errorf("expected grand total 2400000, got %d", plan.TotalsGrand)
	}
}

