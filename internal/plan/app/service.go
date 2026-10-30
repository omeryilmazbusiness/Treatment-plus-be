package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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
	sub          SubscriptionEnforcer
	encrypter    Encrypter
	log          *slog.Logger
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
		log:          slog.Default(),
	}
}

// WithSubscription configures the subscription & quota enforcer.
func (s *Service) WithSubscription(sub SubscriptionEnforcer) *Service {
	s.sub = sub
	return s
}

// WithLogger configures structured logger on the service.
func (s *Service) WithLogger(log *slog.Logger) *Service {
	if log != nil {
		s.log = log
	}
	return s
}

func (s *Service) logger() *slog.Logger {
	if s.log != nil {
		return s.log
	}
	return slog.Default()
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
			} else {
				s.logger().WarnContext(ctx, "failed to decrypt patient contact", "patient_id", patients[i].ID, "error", err)
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
		} else {
			s.logger().WarnContext(ctx, "failed to decrypt patient contact", "patient_id", p.ID, "error", err)
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
			s.logger().ErrorContext(ctx, "failed to encrypt patient contact", "patient_id", p.ID, "error", err)
			return domain.Patient{}, err
		}
		encrypted = enc
	}
	if err := s.patientStore.SavePatient(ctx, p, encrypted); err != nil {
		return domain.Patient{}, err
	}
	s.logger().InfoContext(ctx, "patient created", "workspace_id", wsID, "patient_id", p.ID)
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
			s.logger().ErrorContext(ctx, "failed to encrypt patient contact", "patient_id", current.ID, "error", err)
			return domain.Patient{}, err
		}
		encrypted = enc
	}
	if err := s.patientStore.SavePatient(ctx, current, encrypted); err != nil {
		return domain.Patient{}, err
	}
	s.logger().InfoContext(ctx, "patient updated", "workspace_id", wsID, "patient_id", current.ID)
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
			} else {
				s.logger().WarnContext(ctx, "failed to decrypt patient contact in plan", "plan_id", plans[i].ID, "error", err)
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
		} else if !errors.Is(err, errx.ErrNotFound) {
			s.logger().WarnContext(ctx, "failed to load patient for plan", "workspace_id", wsID, "patient_id", p.PatientID, "error", err)
		}
	}
	return p, nil
}

type CreatePlanInput struct {
	PatientID          id.ID            `json:"patient_id"`
	Title              string           `json:"title"`
	Kind               string           `json:"kind"`
	Currency           string           `json:"currency"`
	Palette            string           `json:"palette"`
	Locale             string           `json:"locale"`
	WelcomeLetter      string           `json:"welcome_letter"`
	VideoURL           string           `json:"video_url"`
	BeforeAfterCaseIDs []string         `json:"before_after_case_ids"`
	Diagnoses          []string         `json:"diagnoses"`
	Situation          domain.Situation `json:"situation"`
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

	curr := s.workspaceCurrency(ctx, wsID, input.Currency)
	palette, locale, err := s.resolvePresentation(ctx, wsID, input.Palette, input.Locale, false)
	if err != nil {
		return domain.Plan{}, err
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "Tedavi Planı"
	}
	situation := domain.MergeSituation(input.Diagnoses, input.Situation)

	planKind := domain.KindClassic
	if input.Kind == string(domain.KindUltimate) {
		planKind = domain.KindUltimate
	} else if input.Kind == string(domain.KindBoth) {
		planKind = domain.KindBoth
	}

	p := domain.Plan{
		ID:                 id.New(),
		WorkspaceID:        wsID,
		PatientID:          input.PatientID,
		Title:              title,
		Kind:               planKind,
		Status:             domain.StatusDraft,
		Currency:           curr,
		Palette:            palette,
		Locale:             locale,
		WelcomeLetter:      strings.TrimSpace(input.WelcomeLetter),
		VideoURL:           strings.TrimSpace(input.VideoURL),
		BeforeAfterCaseIDs: domain.CleanStrings(input.BeforeAfterCaseIDs),
		Diagnoses:          situation.Diagnoses(),
		Situation:          situation,
		TotalsSubtotal:     0,
		TotalsDiscount:     0,
		TotalsGrand:        0,
	}

	if err := p.Validate(); err != nil {
		return domain.Plan{}, err
	}

	if s.sub != nil {
		if err := s.sub.CanCreatePlan(ctx, wsID, string(planKind)); err != nil {
			return domain.Plan{}, err
		}
	}

	if err := s.planStore.CreatePlan(ctx, p); err != nil {
		return domain.Plan{}, err
	}

	if s.sub != nil {
		_ = s.sub.RecordPlanUsage(ctx, wsID)
	}

	s.logger().InfoContext(ctx, "plan created", "workspace_id", wsID, "plan_id", p.ID, "kind", p.Kind)
	return s.GetPlan(ctx, wsID, p.ID)
}

type UpdatePlanHeaderInput struct {
	Title              *string
	Diagnoses          *[]string
	Palette            *string
	Locale             *string
	Currency           *string
	WelcomeLetter      *string
	VideoURL           *string
	BeforeAfterCaseIDs *[]string
	Situation          *domain.Situation
}

func (s *Service) UpdatePlanHeader(ctx context.Context, wsID id.ID, planID id.ID, in UpdatePlanHeaderInput) (domain.Plan, error) {
	current, err := s.GetPlan(ctx, wsID, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	if !current.CanModify() {
		return domain.Plan{}, domain.ErrPlanLocked
	}
	if in.Title != nil {
		current.Title = strings.TrimSpace(*in.Title)
	}
	if in.Palette != nil {
		pal, err := domain.ParsePalette(*in.Palette)
		if err != nil {
			return domain.Plan{}, err
		}
		current.Palette = pal
	}
	if in.Locale != nil {
		loc, err := domain.ParseLocale(*in.Locale)
		if err != nil {
			return domain.Plan{}, err
		}
		current.Locale = loc
	}
	if in.Currency != nil {
		curr, err := moneyCurrency(*in.Currency)
		if err != nil {
			return domain.Plan{}, err
		}
		current.Currency = curr
		if len(current.Items) > 0 || len(current.Discounts) > 0 {
			totals, err := domain.CalculateTotals(current.Items, current.Discounts, current.Currency)
			if err != nil {
				s.logger().ErrorContext(ctx, "failed to recalculate totals after currency update", "plan_id", planID, "error", err)
				return domain.Plan{}, err
			}
			current.TotalsSubtotal = totals.Subtotal
			current.TotalsDiscount = totals.Discount
			current.TotalsGrand = totals.Grand
		}
	}
	if in.Situation != nil {
		current.Situation = domain.MergeSituation(nil, *in.Situation)
		current.Diagnoses = current.Situation.Diagnoses()
	}
	if in.Diagnoses != nil {
		current.Situation = domain.MergeSituation(*in.Diagnoses, current.Situation)
		current.Diagnoses = current.Situation.Diagnoses()
	}
	if in.WelcomeLetter != nil {
		current.WelcomeLetter = strings.TrimSpace(*in.WelcomeLetter)
	}
	if in.VideoURL != nil {
		current.VideoURL = strings.TrimSpace(*in.VideoURL)
	}
	if in.BeforeAfterCaseIDs != nil {
		current.BeforeAfterCaseIDs = domain.CleanStrings(*in.BeforeAfterCaseIDs)
	}
	if err := s.planStore.UpdatePlanHeader(ctx, current); err != nil {
		return domain.Plan{}, err
	}
	s.logger().InfoContext(ctx, "plan header updated", "workspace_id", wsID, "plan_id", planID)
	return s.GetPlan(ctx, wsID, planID)
}

type PutVisitInput struct {
	ID          *string  `json:"id,omitempty"`
	OrderNum    int      `json:"order_num"`
	Title       string   `json:"title"`
	PhaseLabel  string   `json:"phase_label"`
	HealingNote string   `json:"healing_note"`
	ChartTeeth  []string `json:"chart_teeth"`
}

func (s *Service) PutVisits(ctx context.Context, wsID id.ID, planID id.ID, visits []PutVisitInput) (domain.Plan, error) {
	current, err := s.GetPlan(ctx, wsID, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	if !current.CanModify() {
		return domain.Plan{}, domain.ErrPlanLocked
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
			PhaseLabel:  strings.TrimSpace(v.PhaseLabel),
			HealingNote: strings.TrimSpace(v.HealingNote),
			ChartTeeth:  domain.CleanStrings(v.ChartTeeth),
		})
	}

	if err := s.planStore.ReplaceVisits(ctx, wsID, planID, domainVisits); err != nil {
		return domain.Plan{}, err
	}
	s.logger().InfoContext(ctx, "plan visits replaced", "workspace_id", wsID, "plan_id", planID, "count", len(domainVisits))
	return s.GetPlan(ctx, wsID, planID)
}

type PutItemInput struct {
	ID            *string  `json:"id,omitempty"`
	VisitID       *string  `json:"visit_id,omitempty"`
	TreatmentID   *string  `json:"treatment_id,omitempty"`
	TreatmentName string   `json:"treatment_name"`
	AnimationCode string   `json:"animation_code,omitempty"`
	ToothNumber   *string  `json:"tooth_number,omitempty"`
	Teeth         []string `json:"teeth,omitempty"`
	Description   string   `json:"description"`
	Quantity      int      `json:"quantity"`
	UnitPrice     *int64   `json:"unit_price,omitempty"` // minor unit
	IsPackage     bool     `json:"is_package"`
	OrderNum      int      `json:"order_num"`
}

func (s *Service) PutItems(ctx context.Context, wsID id.ID, planID id.ID, items []PutItemInput) (domain.Plan, error) {
	current, err := s.GetPlan(ctx, wsID, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	if !current.CanModify() {
		return domain.Plan{}, domain.ErrPlanLocked
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
					} else {
						s.logger().WarnContext(ctx, "catalog lookup failed for treatment", "treatment_id", parsed, "error", cErr)
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

		toothNum, teeth := domain.ItemTeeth(it.ToothNumber, it.Teeth)

		anim := strings.TrimSpace(it.AnimationCode)
		if anim == "" {
			anim = treatName
		}

		domainItems = append(domainItems, domain.Item{
			ID:            itID,
			PlanID:        planID,
			VisitID:       visitID,
			WorkspaceID:   wsID,
			TreatmentID:   treatID,
			TreatmentName: treatName,
			AnimationCode: anim,
			ToothNumber:   toothNum,
			Teeth:         teeth,
			Description:   strings.TrimSpace(it.Description),
			Quantity:      qty,
			UnitPrice:     unitPrice,
			IsPackage:     it.IsPackage,
			OrderNum:      orderNum,
		})
	}

	totals, err := domain.CalculateTotals(domainItems, current.Discounts, current.Currency)
	if err != nil {
		s.logger().ErrorContext(ctx, "failed to calculate totals for items", "plan_id", planID, "error", err)
		return domain.Plan{}, err
	}

	if err := s.planStore.ReplaceItems(ctx, wsID, planID, domainItems, totals); err != nil {
		return domain.Plan{}, err
	}
	s.logger().InfoContext(ctx, "plan items replaced", "workspace_id", wsID, "plan_id", planID, "count", len(domainItems))
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
	if !current.CanModify() {
		return domain.Plan{}, domain.ErrPlanLocked
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
		s.logger().ErrorContext(ctx, "failed to calculate totals for discounts", "plan_id", planID, "error", err)
		return domain.Plan{}, err
	}

	if err := s.planStore.ReplaceDiscounts(ctx, wsID, planID, domainDiscounts, totals); err != nil {
		return domain.Plan{}, err
	}
	s.logger().InfoContext(ctx, "plan discounts replaced", "workspace_id", wsID, "plan_id", planID, "count", len(domainDiscounts))
	return s.GetPlan(ctx, wsID, planID)
}

func (s *Service) SetReady(ctx context.Context, wsID id.ID, planID id.ID) (domain.Plan, error) {
	current, err := s.GetPlan(ctx, wsID, planID)
	if err != nil {
		return domain.Plan{}, err
	}
	if !current.CanModify() {
		return current, nil // Already locked or past draft (ready, sent, viewed, accepted) - idempotent success
	}

	var clinicSnap domain.ClinicSnapshot
	if s.settings != nil {
		settings, err := s.settings.GetClinicSettings(ctx, wsID)
		if err == nil {
			clinicSnap = snapshotClinic(settings)
			if current.Palette == "" {
				if pal, err := domain.ParsePalette(settings.Palette); err == nil {
					current.Palette = pal
				}
			}
			if current.Locale == "" {
				if loc, err := domain.ParseLocale(settings.Locale); err == nil {
					current.Locale = loc
				}
			}
		} else {
			s.logger().WarnContext(ctx, "failed to fetch clinic settings for snapshot, fallback to defaults", "workspace_id", wsID, "error", err)
		}
	}
	clinicSnap.Palette = string(current.Palette)
	clinicSnap.Locale = string(current.Locale)

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
			AnimationCode: it.AnimationCode,
			ToothNumber:   it.ToothNumber,
			Teeth:         it.Teeth,
			Description:   it.Description,
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
		Palette:   string(current.Palette),
		Locale:    string(current.Locale),
		Clinic:    clinicSnap,
		Patient:   patientSnap,
		Diagnoses: current.Diagnoses,
		Situation: current.Situation,
		Visits:    current.Visits,
		Items:     itemSnaps,
		Discounts: current.Discounts,
		Totals:    totals,
		ReadyAt:   time.Now().UTC(),
	}

	snapBytes, err := json.Marshal(snapshot)
	if err != nil {
		s.logger().ErrorContext(ctx, "failed to marshal plan snapshot", "plan_id", planID, "error", err)
		return domain.Plan{}, err
	}

	if err := s.planStore.SetReady(ctx, wsID, planID, snapBytes); err != nil {
		return domain.Plan{}, err
	}

	s.logger().InfoContext(ctx, "plan marked ready and locked", "workspace_id", wsID, "plan_id", planID)
	return s.GetPlan(ctx, wsID, planID)
}

type ClassicPatientInput struct {
	Name    string
	Contact string
}

type ClassicPlanInput struct {
	Palette   string
	Locale    string
	Patient   ClassicPatientInput
	Title     string
	Diagnoses []string
	Situation domain.Situation
	Visits    []PutVisitInput
	Items     []PutItemInput
	Discounts []PutDiscountInput
}

type ClassicForm struct {
	Product  string               `json:"product"`
	Clinic   ClinicSettingsDTO    `json:"clinic"`
	Sections []domain.FormSection `json:"sections"`
}

func (s *Service) ClassicForm(ctx context.Context, wsID id.ID) (ClassicForm, error) {
	if wsID.IsZero() {
		return ClassicForm{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	clinic := ClinicSettingsDTO{}
	if s.settings != nil {
		if st, err := s.settings.GetClinicSettings(ctx, wsID); err == nil {
			clinic = st
		}
	}
	return ClassicForm{
		Product:  string(domain.KindClassic),
		Clinic:   clinic,
		Sections: domain.ClassicFormSections(),
	}, nil
}

func (s *Service) CreateClassic(ctx context.Context, wsID id.ID, input ClassicPlanInput) (domain.Plan, error) {
	if wsID.IsZero() {
		return domain.Plan{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	patient, err := s.CreatePatient(ctx, wsID, input.Patient.Name, input.Patient.Contact)
	if err != nil {
		return domain.Plan{}, err
	}
	palette, locale, err := s.resolvePresentation(ctx, wsID, input.Palette, input.Locale, true)
	if err != nil {
		return domain.Plan{}, err
	}
	plan, err := s.CreatePlan(ctx, wsID, CreatePlanInput{
		PatientID: patient.ID,
		Title:     input.Title,
		Palette:   string(palette),
		Locale:    string(locale),
		Diagnoses: input.Diagnoses,
		Situation: input.Situation,
	})
	if err != nil {
		return domain.Plan{}, err
	}
	if len(input.Visits) > 0 {
		plan, err = s.PutVisits(ctx, wsID, plan.ID, input.Visits)
		if err != nil {
			return domain.Plan{}, err
		}
	}
	if len(input.Items) > 0 {
		plan, err = s.PutItems(ctx, wsID, plan.ID, input.Items)
		if err != nil {
			return domain.Plan{}, err
		}
	}
	if len(input.Discounts) > 0 {
		plan, err = s.PutDiscounts(ctx, wsID, plan.ID, input.Discounts)
		if err != nil {
			return domain.Plan{}, err
		}
	}
	s.logger().InfoContext(ctx, "classic plan created successfully", "workspace_id", wsID, "plan_id", plan.ID)
	return plan, nil
}

type UltimatePlanInput struct {
	Palette            string
	Locale             string
	Patient            ClassicPatientInput
	Title              string
	WelcomeLetter      string
	VideoURL           string
	BeforeAfterCaseIDs []string
	Diagnoses          []string
	Situation          domain.Situation
	Visits             []PutVisitInput
	Items              []PutItemInput
	Discounts          []PutDiscountInput
}

type UltimateForm struct {
	Product  string               `json:"product"`
	Clinic   ClinicSettingsDTO    `json:"clinic"`
	Sections []domain.FormSection `json:"sections"`
}

func (s *Service) UltimateForm(ctx context.Context, wsID id.ID) (UltimateForm, error) {
	if wsID.IsZero() {
		return UltimateForm{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	clinic := ClinicSettingsDTO{}
	if s.settings != nil {
		if st, err := s.settings.GetClinicSettings(ctx, wsID); err == nil {
			clinic = st
		}
	}
	return UltimateForm{
		Product:  string(domain.KindUltimate),
		Clinic:   clinic,
		Sections: domain.UltimateFormSections(),
	}, nil
}

func (s *Service) CreateUltimate(ctx context.Context, wsID id.ID, input UltimatePlanInput) (domain.Plan, error) {
	if wsID.IsZero() {
		return domain.Plan{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	patient, err := s.CreatePatient(ctx, wsID, input.Patient.Name, input.Patient.Contact)
	if err != nil {
		return domain.Plan{}, err
	}
	palette, locale, err := s.resolvePresentation(ctx, wsID, input.Palette, input.Locale, true)
	if err != nil {
		return domain.Plan{}, err
	}
	plan, err := s.CreatePlan(ctx, wsID, CreatePlanInput{
		PatientID:          patient.ID,
		Title:              input.Title,
		Kind:               string(domain.KindUltimate),
		Palette:            string(palette),
		Locale:             string(locale),
		WelcomeLetter:      input.WelcomeLetter,
		VideoURL:           input.VideoURL,
		BeforeAfterCaseIDs: input.BeforeAfterCaseIDs,
		Diagnoses:          input.Diagnoses,
		Situation:          input.Situation,
	})
	if err != nil {
		return domain.Plan{}, err
	}
	if len(input.Visits) > 0 {
		plan, err = s.PutVisits(ctx, wsID, plan.ID, input.Visits)
		if err != nil {
			return domain.Plan{}, err
		}
	}
	if len(input.Items) > 0 {
		plan, err = s.PutItems(ctx, wsID, plan.ID, input.Items)
		if err != nil {
			return domain.Plan{}, err
		}
	}
	if len(input.Discounts) > 0 {
		plan, err = s.PutDiscounts(ctx, wsID, plan.ID, input.Discounts)
		if err != nil {
			return domain.Plan{}, err
		}
	}
	s.logger().InfoContext(ctx, "ultimate plan created successfully", "workspace_id", wsID, "plan_id", plan.ID)
	return plan, nil
}

func snapshotClinic(st ClinicSettingsDTO) domain.ClinicSnapshot {
	var highlights []domain.ClinicStatSnapshot
	for _, h := range st.Highlights {
		highlights = append(highlights, domain.ClinicStatSnapshot{Value: h.Value, Label: h.Label})
	}
	var doctorStats []domain.ClinicStatSnapshot
	for _, h := range st.Doctor.Stats {
		doctorStats = append(doctorStats, domain.ClinicStatSnapshot{Value: h.Value, Label: h.Label})
	}
	var labStats []domain.ClinicStatSnapshot
	for _, h := range st.Lab.Stats {
		labStats = append(labStats, domain.ClinicStatSnapshot{Value: h.Value, Label: h.Label})
	}
	return domain.ClinicSnapshot{
		Name: st.Name, Slogan: st.Slogan, Phone: st.Phone, Email: st.Email, Website: st.Website,
		Address: st.Address, Hours: st.Hours, Palette: st.Palette, Locale: st.Locale, Currency: st.Currency,
		LogoKey: st.LogoKey, CoverKey: st.CoverKey, MapURL: st.MapURL, Intro: st.Intro, Legal: st.Legal,
		AboutText: st.AboutText, AboutKey: st.AboutKey, WelcomeBody: st.WelcomeBody,
		SignerName: st.SignerName, SignerTitle: st.SignerTitle, Highlights: highlights,
		Doctor: domain.ClinicDoctorSnapshot{
			Name: st.Doctor.Name, Title: st.Doctor.Title, Subtitle: st.Doctor.Subtitle,
			Bio: st.Doctor.Bio, PhotoKey: st.Doctor.PhotoKey, Stats: doctorStats,
		},
		Lab: domain.ClinicLabSnapshot{
			Title: st.Lab.Title, Text: st.Lab.Text, ImageKey: st.Lab.ImageKey, Stats: labStats,
		},
		Awards: domain.ClinicAwardsSnapshot{Text: st.Awards.Text, BadgeKeys: st.Awards.BadgeKeys},
		Hotel:  domain.ClinicHotelSnapshot{Name: st.Hotel.Name, Text: st.Hotel.Text, ImageKey: st.Hotel.ImageKey},
	}
}

func (s *Service) resolvePresentation(ctx context.Context, wsID id.ID, palette, locale string, required bool) (domain.Palette, domain.Locale, error) {
	if strings.TrimSpace(palette) != "" {
		pal, err := domain.ParsePalette(palette)
		if err != nil {
			return "", "", err
		}
		palette = string(pal)
	}
	if strings.TrimSpace(locale) != "" {
		loc, err := domain.ParseLocale(locale)
		if err != nil {
			return "", "", err
		}
		locale = string(loc)
	}
	if required {
		if strings.TrimSpace(palette) == "" {
			return "", "", errx.New("plan_palette_required", "color palette must be selected for this PDF", errx.ErrInvalid)
		}
		if strings.TrimSpace(locale) == "" {
			return "", "", errx.New("plan_locale_required", "language must be selected for this PDF", errx.ErrInvalid)
		}
	} else if s.settings != nil && (palette == "" || locale == "") {
		if st, err := s.settings.GetClinicSettings(ctx, wsID); err == nil {
			if palette == "" {
				palette = st.Palette
			}
			if locale == "" {
				locale = st.Locale
			}
		}
	}
	if palette == "" {
		palette = string(domain.PaletteNavy)
	}
	if locale == "" {
		locale = string(domain.LocaleTR)
	}
	pal, err := domain.ParsePalette(palette)
	if err != nil {
		return "", "", err
	}
	loc, err := domain.ParseLocale(locale)
	if err != nil {
		return "", "", err
	}
	return pal, loc, nil
}

func (s *Service) workspaceCurrency(ctx context.Context, wsID id.ID, fallback string) money.Currency {
	if fallback != "" {
		if curr, err := moneyCurrency(fallback); err == nil {
			return curr
		}
	}
	if s.settings != nil {
		if st, err := s.settings.GetClinicSettings(ctx, wsID); err == nil {
			if curr, err := moneyCurrency(st.Currency); err == nil {
				return curr
			}
		}
	}
	return money.TRY
}

func moneyCurrency(raw string) (money.Currency, error) {
	curr := money.Currency(strings.ToUpper(strings.TrimSpace(raw)))
	if !money.ValidCurrency(curr) {
		return "", errx.New("invalid_currency", "unsupported currency", errx.ErrInvalid)
	}
	return curr, nil
}
