package domain

import "time"

// Snapshot is the locked Classic payload. Output never recomputes prices.
type Snapshot struct {
	PlanID    string     `json:"plan_id"`
	Title     string     `json:"title"`
	Kind      string     `json:"kind"`
	Currency  string     `json:"currency"`
	Palette   string     `json:"palette"`
	Locale    string     `json:"locale"`
	Clinic    Clinic     `json:"clinic"`
	Patient   Patient    `json:"patient"`
	Diagnoses []string   `json:"diagnoses"`
	Situation Situation  `json:"situation"`
	Visits    []Visit    `json:"visits"`
	Items     []Item     `json:"items"`
	Discounts []Discount `json:"discounts"`
	Totals    Totals     `json:"totals"`
	ReadyAt   time.Time  `json:"ready_at"`
}

type Clinic struct {
	Name        string `json:"name"`
	Slogan      string `json:"slogan"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	Website     string `json:"website"`
	Address     string `json:"address"`
	Hours       string `json:"hours"`
	Palette     string `json:"palette"`
	Locale      string `json:"locale"`
	Currency    string `json:"currency"`
	LogoKey     string `json:"logo_key"`
	CoverKey    string `json:"cover_key"`
	MapURL      string `json:"map_url"`
	Intro       string `json:"intro"`
	Legal       string `json:"legal"`
	AboutText   string `json:"about_text"`
	AboutKey    string `json:"about_key"`
	WelcomeBody string `json:"welcome_body"`
	SignerName  string `json:"signer_name"`
	SignerTitle string `json:"signer_title"`
	Highlights  []Stat `json:"highlights"`
	Doctor      Doctor `json:"doctor"`
	Lab         Lab    `json:"lab"`
	Awards      Awards `json:"awards"`
	Hotel       Hotel  `json:"hotel"`
	SmileKey    string `json:"smile_key,omitempty"`
}

type Stat struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type Doctor struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	Bio      string `json:"bio"`
	PhotoKey string `json:"photo_key"`
	Stats    []Stat `json:"stats"`
}

type Lab struct {
	Title    string `json:"title"`
	Text     string `json:"text"`
	ImageKey string `json:"image_key"`
	Stats    []Stat `json:"stats"`
}

type Awards struct {
	Text      string   `json:"text"`
	BadgeKeys []string `json:"badge_keys"`
}

type Hotel struct {
	Name     string `json:"name"`
	Text     string `json:"text"`
	ImageKey string `json:"image_key"`
}

type Patient struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Contact string `json:"contact"`
}

type Situation struct {
	XrayKey          string         `json:"xray_key"`
	ChartTeeth       []string       `json:"chart_teeth"`
	ToothFindings    []ToothFinding `json:"tooth_findings"`
	GeneralDiagnoses []string       `json:"general_diagnoses"`
}

type ToothFinding struct {
	Tooth  string `json:"tooth"`
	Status string `json:"status"`
	Note   string `json:"note"`
}

type Visit struct {
	ID          string   `json:"id"`
	OrderNum    int      `json:"order_num"`
	Title       string   `json:"title"`
	PhaseLabel  string   `json:"phase_label"`
	HealingNote string   `json:"healing_note"`
	ChartTeeth  []string `json:"chart_teeth"`
}

type Item struct {
	VisitTitle    string   `json:"visit_title"`
	TreatmentName string   `json:"treatment_name"`
	AnimationCode string   `json:"animation_code,omitempty"`
	ToothNumber   *string  `json:"tooth_number"`
	Teeth         []string `json:"teeth"`
	Description   string   `json:"description"`
	Quantity      int      `json:"quantity"`
	UnitPrice     int64    `json:"unit_price"`
	IsPackage     bool     `json:"is_package"`
	LineTotal     int64    `json:"line_total"`
}

type Discount struct {
	Label  string `json:"label"`
	Amount int64  `json:"amount"`
}

type Totals struct {
	Subtotal int64 `json:"subtotal"`
	Discount int64 `json:"discount"`
	Grand    int64 `json:"grand"`
}

func (s Snapshot) ItemsForVisit(title string) []Item {
	var out []Item
	for _, it := range s.Items {
		if it.VisitTitle == title {
			out = append(out, it)
		}
	}
	return out
}

func (s Snapshot) Finding(tooth string) ToothFinding {
	for _, f := range s.Situation.ToothFindings {
		if f.Tooth == tooth {
			return f
		}
	}
	return ToothFinding{}
}
