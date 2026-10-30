package domain

import "strings"

const (
	ToothIntact  = "intact"
	ToothMissing = "missing"
	ToothFilled  = "filled"
	ToothCrown   = "crown"
)

// ToothFinding is one FDI cell on the current-situation odontogram (Wise p.9).
type ToothFinding struct {
	Tooth  string `json:"tooth"`
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}

func (f ToothFinding) Normalize() ToothFinding {
	return ToothFinding{
		Tooth:  strings.TrimSpace(f.Tooth),
		Status: strings.ToLower(strings.TrimSpace(f.Status)),
		Note:   strings.TrimSpace(f.Note),
	}
}

func (f ToothFinding) Valid() bool {
	if f.Tooth == "" {
		return false
	}
	switch f.Status {
	case "", ToothIntact, ToothMissing, ToothFilled, ToothCrown:
		return true
	default:
		return false
	}
}

// Situation is the per-PDF current state: panoramic x-ray + odontogram + general notes (Wise p.8–9).
type Situation struct {
	XrayKey          string         `json:"xray_key,omitempty"`
	ChartTeeth       []string       `json:"chart_teeth,omitempty"`
	ToothFindings    []ToothFinding `json:"tooth_findings,omitempty"`
	GeneralDiagnoses []string       `json:"general_diagnoses,omitempty"`
}

func (s Situation) Normalize() Situation {
	out := Situation{XrayKey: strings.TrimSpace(s.XrayKey)}
	seenTooth := map[string]struct{}{}
	for _, t := range s.ChartTeeth {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seenTooth[t]; ok {
			continue
		}
		seenTooth[t] = struct{}{}
		out.ChartTeeth = append(out.ChartTeeth, t)
	}
	for _, f := range s.ToothFindings {
		n := f.Normalize()
		if n.Valid() {
			out.ToothFindings = append(out.ToothFindings, n)
		}
	}
	seenDiag := map[string]struct{}{}
	for _, d := range s.GeneralDiagnoses {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if _, ok := seenDiag[d]; ok {
			continue
		}
		seenDiag[d] = struct{}{}
		out.GeneralDiagnoses = append(out.GeneralDiagnoses, d)
	}
	return out
}

func (s Situation) Diagnoses() []string {
	return append([]string(nil), s.GeneralDiagnoses...)
}

func MergeSituation(diagnoses []string, sit Situation) Situation {
	sit = sit.Normalize()
	seen := map[string]struct{}{}
	for _, d := range sit.GeneralDiagnoses {
		seen[d] = struct{}{}
	}
	for _, d := range diagnoses {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if _, ok := seen[d]; ok {
			continue
		}
		seen[d] = struct{}{}
		sit.GeneralDiagnoses = append(sit.GeneralDiagnoses, d)
	}
	return sit
}

// ItemTeeth keeps tooth_number as the first FDI code and teeth as the full set.
func ItemTeeth(tooth *string, teeth []string) (*string, []string) {
	seen := map[string]struct{}{}
	var all []string
	add := func(raw string) {
		for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '\t'
		}) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			all = append(all, part)
		}
	}
	if tooth != nil {
		add(*tooth)
	}
	for _, t := range teeth {
		add(t)
	}
	if len(all) == 0 {
		return nil, nil
	}
	first := all[0]
	return &first, all
}

func CleanStrings(in []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
