package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type Service struct {
	planStore    PlanStore
	patientStore PatientStore
	catalog      CatalogReader
	settings     SettingsReader
	encrypter    Encrypter
}

func NewService(
	planStore PlanStore,
	patientStore PatientStore,
	catalog CatalogReader,
	settings SettingsReader,
	encrypter Encrypter,
) *Service {
	return &Service{
		planStore:    planStore,
		patientStore: patientStore,
		catalog:      catalog,
		settings:     settings,
		encrypter:    encrypter,
	}
}

// Patients
func (s *Service) ListPatients(ctx context.Context, wsID id.ID) ([]domain.Patient, error) {
	if wsID.IsZero() {
		return nil, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	patients, err := s.patientStore.ListPatients(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for i := range patients {
		if s.encrypter != nil && patients[i].Contact != "" {
			dec, err := s.encrypter.Decrypt(patients[i].Contact)
			if err == nil {
				patients[i].Contact = dec
			}
		}
	}
	return patients, nil
}

func (s *Service) GetPatient(ctx context.Context, wsID id.ID, patID id.ID) (domain.Patient, error) {
	if wsID.IsZero() || patID.IsZero() {
		return domain.Patient{}, errx.New("invalid_id", "workspace and patient id required", errx.ErrInvalid)
	}
	p, err := s.patientStore.GetPatient(ctx, wsID, patID)
	if err != nil {
		return domain.Patient{}, err
	}
	if s.encrypter != nil && p.Contact != "" {
		dec, err := s.encrypter.Decrypt(p.Contact)
		if err == nil {
			p.Contact = dec
		}
	}
	return p, nil
}

func (s *Service) CreatePatient(ctx context.Context, wsID id.ID, name, contact string) (domain.Patient, error) {
	if wsID.IsZero() {
		return domain.Patient{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	p := domain.Patient{
		ID:          id.New(),
		WorkspaceID: wsID,
		Name:        strings.TrimSpace(name),
		Contact:     strings.TrimSpace(contact),
	}
	if err := p.Validate(); err != nil {
		return domain.Patient{}, err
	}
	encrypted := p.Contact
	if s.encrypter != nil && p.Contact != "" {
		enc, err := s.encrypter.Encrypt(p.Contact)
		if err != nil {
			return domain.Patient{}, err
		}
		encrypted = enc
	}
	if err := s.patientStore.SavePatient(ctx, p, encrypted); err != nil {
		return domain.Patient{}, err
	}
	return p, nil
}

func (s *Service) UpdatePatient(ctx context.Context, wsID id.ID, patID id.ID, name, contact *string) (domain.Patient, error) {
	current, err := s.GetPatient(ctx, wsID, patID)
	if err != nil {
		return domain.Patient{}, err
	}
	if name != nil {
		current.Name = strings.TrimSpace(*name)
	}
	if contact != nil {
		current.Contact = strings.TrimSpace(*contact)
	}
	if err := current.Validate(); err != nil {
		return domain.Patient{}, err
	}
	encrypted := current.Contact
	if s.encrypter != nil && current.Contact != "" {
		enc, err := s.encrypter.Encrypt(current.Contact)
		if err != nil {
			return domain.Patient{}, err
		}
		encrypted = enc
	}
	if err := s.patientStore.SavePatient(ctx, current, encrypted); err != nil {
		return domain.Patient{}, err
	}
	return current, nil
}

// Plans
func (s *Service) ListPlans(ctx context.Context, wsID id.ID) ([]domain.Plan, error) {
	if wsID.IsZero() {
		return nil, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	plans, err := s.planStore.ListPlans(ctx, wsID)
	if err != nil {
		return nil, err
	}
	for i := range plans {
		if s.encrypter != nil && plans[i].Patient != nil && plans[i].Patient.Contact != "" {
			dec, err := s.encrypter.Decrypt(plans[i].Patient.Contact)
			if err == nil {
				plans[i].Patient.Contact = dec
			}
		}
	}
	return plans, nil
}

func (s *Service) GetPlan(ctx context.Context, wsID id.ID, planID id.ID) (domain.Plan, error) {
	if wsID.IsZero() || planID.IsZero() {
		return domain.Plan{}, errx.New("invalid_id", "workspace and plan id required", errx.ErrInvalid)
	}
	p, err := s.planStore.GetPlan(ctx, wsID, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	if !p.PatientID.IsZero() {
		pat, err := s.GetPatient(ctx, wsID, p.PatientID)
		if err == nil {
			p.Patient = &pat
		}
	}
	return p, nil
}

type CreatePlanInput struct {
	PatientID id.ID    `json:"patient_id"`
	Title     string   `json:"title"`
	Currency  string   `json:"currency"`
	Diagnoses []string `json:"diagnoses"`
}

func (s *Service) CreatePlan(ctx context.Context, wsID id.ID, input CreatePlanInput) (domain.Plan, error) {
	if wsID.IsZero() || input.PatientID.IsZero() {
		return domain.Plan{}, errx.New("invalid_id", "workspace and patient id required", errx.ErrInvalid)
	}
	// Verify patient exists in this workspace
	_, err := s.patientStore.GetPatient(ctx, wsID, input.PatientID)
	if err != nil {
		return domain.Plan{}, err
	}

	curr := money.Currency(strings.ToUpper(strings.TrimSpace(input.Currency)))
	if !money.ValidCurrency(curr) {
		curr = money.TRY
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "Tedavi Planı"
	}

	p := domain.Plan{
		ID:             id.New(),
		WorkspaceID:    wsID,
		PatientID:      input.PatientID,
		Title:          title,
		Kind:           domain.KindClassic, // Only classic!
		Status:         domain.StatusDraft,
		Currency:       curr,
		Diagnoses:      input.Diagnoses,
		TotalsSubtotal: 0,
		TotalsDiscount: 0,
		TotalsGrand:    0,
	}

	if err := s.planStore.CreatePlan(ctx, p); err != nil {
		return domain.Plan{}, err
	}

	return s.GetPlan(ctx, wsID, p.ID)
}

func (s *Service) UpdatePlanHeader(ctx context.Context, wsID id.ID, planID id.ID, title *string, diagnoses *[]string) (domain.Plan, error) {
	current, err := s.GetPlan(ctx, wsID, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	if current.Status == domain.StatusReady {
		return domain.Plan{}, errx.New("plan_locked", "cannot modify ready plan", errx.ErrInvalid)
	}
	if title != nil {
		current.Title = strings.TrimSpace(*title)
	}
	if diagnoses != nil {
		current.Diagnoses = *diagnoses
	}
	if err := s.planStore.UpdatePlanHeader(ctx, current); err != nil {
		return domain.Plan{}, err
	}
	return s.GetPlan(ctx, wsID, planID)
}

type PutVisitInput struct {
	ID       *string `json:"id,omitempty"`
	OrderNum int     `json:"order_num"`
	Title    string  `json:"title"`
}

func (s *Service) PutVisits(ctx context.Context, wsID id.ID, planID id.ID, visits []PutVisitInput) (domain.Plan, error) {
	current, err := s.GetPlan(ctx, wsID, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	if current.Status == domain.StatusReady {
		return domain.Plan{}, errx.New("plan_locked", "cannot modify ready plan", errx.ErrInvalid)
	}

	var domainVisits []domain.Visit
	for i, v := range visits {
		var vID id.ID
		if v.ID != nil && *v.ID != "" {
			parsed, err := id.Parse(*v.ID)
			if err == nil {
				vID = parsed
			}
		}
		if vID.IsZero() {
			vID = id.New()
		}
		orderNum := v.OrderNum
		if orderNum == 0 {
			orderNum = i + 1
		}
		domainVisits = append(domainVisits, domain.Visit{
			ID:          vID,
			PlanID:      planID,
			WorkspaceID: wsID,
			OrderNum:    orderNum,
			Title:       strings.TrimSpace(v.Title),
		})
	}

	if err := s.planStore.ReplaceVisits(ctx, wsID, planID, domainVisits); err != nil {
		return domain.Plan{}, err
	}
	return s.GetPlan(ctx, wsID, planID)
}

type PutItemInput struct {
	ID            *string `json:"id,omitempty"`
	VisitID       *string `json:"visit_id,omitempty"`
	TreatmentID   *string `json:"treatment_id,omitempty"`
	TreatmentName string  `json:"treatment_name"`
	ToothNumber   *string `json:"tooth_number,omitempty"`
	Quantity      int     `json:"quantity"`
	UnitPrice     *int64  `json:"unit_price,omitempty"` // minor unit
	IsPackage     bool    `json:"is_package"`
	OrderNum      int     `json:"order_num"`
}

func (s *Service) PutItems(ctx context.Context, wsID id.ID, planID id.ID, items []PutItemInput) (domain.Plan, error) {
	current, err := s.GetPlan(ctx, wsID, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	if current.Status == domain.StatusReady {
		return domain.Plan{}, errx.New("plan_locked", "cannot modify ready plan", errx.ErrInvalid)
	}

	var domainItems []domain.Item
	for i, it := range items {
		var itID id.ID
		if it.ID != nil && *it.ID != "" {
			if parsed, err := id.Parse(*it.ID); err == nil {
				itID = parsed
			}
		}
		if itID.IsZero() {
			itID = id.New()
		}

		var visitID *id.ID
		if it.VisitID != nil && *it.VisitID != "" {
			if parsed, err := id.Parse(*it.VisitID); err == nil {
				visitID = &parsed
			}
		}

		var treatID *id.ID
		treatName := strings.TrimSpace(it.TreatmentName)
		var unitPrice int64
		if it.UnitPrice != nil {
			unitPrice = *it.UnitPrice
		}

		if it.TreatmentID != nil && *it.TreatmentID != "" {
			if parsed, err := id.Parse(*it.TreatmentID); err == nil {
				treatID = &parsed
				// If name or price is missing, query CatalogReader
				if s.catalog != nil {
					_, cName, cPrice, cErr := s.catalog.GetTreatment(ctx, wsID, parsed)
					if cErr == nil {
						if treatName == "" {
							treatName = cName
						}
						if it.UnitPrice == nil && cPrice > 0 {
							unitPrice = cPrice
						}
					}
				}
			}
		}

		qty := it.Quantity
		if qty <= 0 {
			qty = 1
		}
		orderNum := it.OrderNum
		if orderNum == 0 {
			orderNum = i + 1
		}

		var toothNum *string
		if it.ToothNumber != nil && strings.TrimSpace(*it.ToothNumber) != "" {
			tn := strings.TrimSpace(*it.ToothNumber)
			toothNum = &tn
		}

		domainItems = append(domainItems, domain.Item{
			ID:            itID,
			PlanID:        planID,
			VisitID:       visitID,
			WorkspaceID:   wsID,
			TreatmentID:   treatID,
			TreatmentName: treatName,
			ToothNumber:   toothNum,
			Quantity:      qty,
			UnitPrice:     unitPrice,
			IsPackage:     it.IsPackage,
			OrderNum:      orderNum,
		})
	}

	totals, err := domain.CalculateTotals(domainItems, current.Discounts, current.Currency)
	if err != nil {
		return domain.Plan{}, err
	}

	if err := s.planStore.ReplaceItems(ctx, wsID, planID, domainItems, totals); err != nil {
		return domain.Plan{}, err
	}
	return s.GetPlan(ctx, wsID, planID)
}

type PutDiscountInput struct {
	ID       *string `json:"id,omitempty"`
	Label    string  `json:"label"`
	Amount   int64   `json:"amount"` // minor unit
	OrderNum int     `json:"order_num"`
}

func (s *Service) PutDiscounts(ctx context.Context, wsID id.ID, planID id.ID, discounts []PutDiscountInput) (domain.Plan, error) {
	current, err := s.GetPlan(ctx, wsID, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	if current.Status == domain.StatusReady {
		return domain.Plan{}, errx.New("plan_locked", "cannot modify ready plan", errx.ErrInvalid)
	}

	var domainDiscounts []domain.Discount
	for i, d := range discounts {
		var dID id.ID
		if d.ID != nil && *d.ID != "" {
			if parsed, err := id.Parse(*d.ID); err == nil {
				dID = parsed
			}
		}
		if dID.IsZero() {
			dID = id.New()
		}
		orderNum := d.OrderNum
		if orderNum == 0 {
			orderNum = i + 1
		}
		domainDiscounts = append(domainDiscounts, domain.Discount{
			ID:          dID,
			PlanID:      planID,
			WorkspaceID: wsID,
			Label:       strings.TrimSpace(d.Label),
			Amount:      d.Amount,
			OrderNum:    orderNum,
		})
	}

	totals, err := domain.CalculateTotals(current.Items, domainDiscounts, current.Currency)
	if err != nil {
		return domain.Plan{}, err
	}

	if err := s.planStore.ReplaceDiscounts(ctx, wsID, planID, domainDiscounts, totals); err != nil {
		return domain.Plan{}, err
	}
	return s.GetPlan(ctx, wsID, planID)
}

func (s *Service) SetReady(ctx context.Context, wsID id.ID, planID id.ID) (domain.Plan, error) {
	current, err := s.GetPlan(ctx, wsID, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	if current.Status == domain.StatusReady {
		return current, nil // Already locked
	}

	// Read clinic settings
	var clinicSnap domain.ClinicSnapshot
	if s.settings != nil {
		settings, err := s.settings.GetClinicSettings(ctx, wsID)
		if err == nil {
			clinicSnap = domain.ClinicSnapshot{
				Name:    settings.Name,
				Phone:   settings.Phone,
				Palette: settings.Palette,
				LogoKey: settings.LogoKey,
				Intro:   settings.Intro,
				Legal:   settings.Legal,
			}
		}
	}

	// Read patient details
	var patientSnap domain.PatientSnapshot
	if current.Patient != nil {
		patientSnap = domain.PatientSnapshot{
			ID:      current.Patient.ID.String(),
			Name:    current.Patient.Name,
			Contact: current.Patient.Contact,
		}
	}

	// Map visit titles for items
	visitMap := make(map[string]string)
	for _, v := range current.Visits {
		visitMap[v.ID.String()] = v.Title
	}

	var itemSnaps []domain.ItemSnapshot
	for _, it := range current.Items {
		vTitle := ""
		if it.VisitID != nil {
			vTitle = visitMap[it.VisitID.String()]
		}
		itemSnaps = append(itemSnaps, domain.ItemSnapshot{
			VisitTitle:    vTitle,
			TreatmentName: it.TreatmentName,
			ToothNumber:   it.ToothNumber,
			Quantity:      it.Quantity,
			UnitPrice:     it.UnitPrice,
			IsPackage:     it.IsPackage,
			LineTotal:     it.LineTotal(),
		})
	}

	totals := domain.Totals{
		Subtotal: current.TotalsSubtotal,
		Discount: current.TotalsDiscount,
		Grand:    current.TotalsGrand,
	}

	snapshot := domain.Snapshot{
		PlanID:    current.ID.String(),
		Title:     current.Title,
		Kind:      string(current.Kind),
		Currency:  string(current.Currency),
		Clinic:    clinicSnap,
		Patient:   patientSnap,
		Diagnoses: current.Diagnoses,
		Visits:    current.Visits,
		Items:     itemSnaps,
		Discounts: current.Discounts,
		Totals:    totals,
		ReadyAt:   time.Now().UTC(),
	}

	snapBytes, err := json.Marshal(snapshot)
	if err != nil {
		return domain.Plan{}, err
	}

	if err := s.planStore.SetReady(ctx, wsID, planID, snapBytes); err != nil {
		return domain.Plan{}, err
	}

	return s.GetPlan(ctx, wsID, planID)
}
