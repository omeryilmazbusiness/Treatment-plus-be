package domain

// FormField describes one Classic input. Clinic brand fields never appear here.
type FormField struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Repeatable  bool     `json:"repeatable,omitempty"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// FormSection is one Classic heading, sorted by Wise PDF page order.
type FormSection struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Order       int         `json:"order"`
	Required    bool        `json:"required"`
	Description string      `json:"description,omitempty"`
	Fields      []FormField `json:"fields"`
}

func ClassicFormSections() []FormSection {
	return []FormSection{
		{
			ID:          "presentation",
			Title:       "Sunum",
			Order:       1,
			Required:    true,
			Description: "Bu PDF'in paleti ve dili. Klinik markası hesapta kalır; her planda yeniden seçilir.",
			Fields: []FormField{
				{Name: "palette", Label: "Renk paleti", Type: "enum", Required: true, Enum: []string{"navy", "forest", "rose", "slate"}},
				{Name: "locale", Label: "Dil", Type: "enum", Required: true, Enum: []string{"tr", "en"}},
			},
		},
		{
			ID:          "patient",
			Title:       "Hasta",
			Order:       2,
			Required:    true,
			Description: "Karşılama mektubundaki hasta adı (Wise s.2).",
			Fields: []FormField{
				{Name: "patient.name", Label: "Hasta adı", Type: "string", Required: true},
				{Name: "patient.contact", Label: "İletişim", Type: "string", Required: false, Description: "Telefon veya e-posta. Şifreli saklanır."},
			},
		},
		{
			ID:          "situation",
			Title:       "Mevcut durum",
			Order:       3,
			Required:    false,
			Description: "Panoramik röntgen, FDI odontogram ve genel tanılar (Wise s.8–9).",
			Fields: []FormField{
				{Name: "title", Label: "Plan başlığı", Type: "string", Required: false},
				{Name: "situation.xray_key", Label: "Röntgen görseli", Type: "string", Required: false},
				{Name: "situation.chart_teeth", Label: "Mevcut odontogram dişleri", Type: "string[]", Required: false, Repeatable: true},
				{Name: "situation.tooth_findings.tooth", Label: "Diş no (FDI)", Type: "string", Required: false, Repeatable: true},
				{Name: "situation.tooth_findings.status", Label: "Diş durumu", Type: "enum", Required: false, Repeatable: true, Enum: []string{"intact", "missing", "filled", "crown"}},
				{Name: "situation.tooth_findings.note", Label: "Diş notu", Type: "string", Required: false, Repeatable: true},
				{Name: "situation.general_diagnoses", Label: "Genel tanılar", Type: "string[]", Required: false, Repeatable: true, Description: "Örn. gingivitis."},
			},
		},
		{
			ID:          "visits",
			Title:       "Seanslar / tedaviler",
			Order:       4,
			Required:    true,
			Description: "Planlı odontogram, satır kalemleri, seans toplamı ve iyileşme notu (Wise s.10, 12). Paket kalemi toplama girmez. Tutarlar kuruştur.",
			Fields: []FormField{
				{Name: "visits.title", Label: "Seans adı", Type: "string", Required: true, Repeatable: true},
				{Name: "visits.phase_label", Label: "Faz", Type: "string", Required: false, Repeatable: true},
				{Name: "visits.healing_note", Label: "İyileşme notu", Type: "string", Required: false, Repeatable: true},
				{Name: "visits.chart_teeth", Label: "Planlı odontogram dişleri", Type: "string[]", Required: false, Repeatable: true},
				{Name: "items.treatment_name", Label: "Tedavi adı", Type: "string", Required: true, Repeatable: true},
				{Name: "items.animation_code", Label: "3D sahne kodu", Type: "string", Required: false, Repeatable: true, Description: "Örn. implant, zirconia_crown. Boşsa tedavi adından çözülür. Katalog: GET /v1/viz/recipes"},
				{Name: "items.description", Label: "Açıklama", Type: "string", Required: false, Repeatable: true},
				{Name: "items.teeth", Label: "Dişler", Type: "string[]", Required: false, Repeatable: true},
				{Name: "items.quantity", Label: "Adet", Type: "integer", Required: true, Repeatable: true},
				{Name: "items.unit_price", Label: "Birim fiyat (kuruş)", Type: "integer", Required: true, Repeatable: true},
				{Name: "items.is_package", Label: "Paket", Type: "boolean", Required: false, Repeatable: true},
				{Name: "items.visit_id", Label: "Seans", Type: "id", Required: false, Repeatable: true},
				{Name: "items.treatment_id", Label: "Katalog tedavi", Type: "id", Required: false, Repeatable: true},
			},
		},
		{
			ID:          "discounts",
			Title:       "İskontolar",
			Order:       5,
			Required:    false,
			Description: "İsteğe bağlı. Clinic markası burada sorulmaz.",
			Fields: []FormField{
				{Name: "discounts.label", Label: "İskonto adı", Type: "string", Required: true, Repeatable: true},
				{Name: "discounts.amount", Label: "Tutar (kuruş)", Type: "integer", Required: true, Repeatable: true},
			},
		},
	}
}

func UltimateFormSections() []FormSection {
	sections := ClassicFormSections()
	// Enrich presentation section with welcome letter and video url
	for i := range sections {
		if sections[i].ID == "presentation" {
			sections[i].Fields = append(sections[i].Fields,
				FormField{Name: "welcome_letter", Label: "Kişisel hekim mektubu", Type: "text", Required: false, Description: "Hastaya hitaben samimi karşılama mesajı"},
				FormField{Name: "video_url", Label: "Tanıtım / karşılama videosu URL", Type: "string", Required: false, Description: "YouTube, Vimeo veya doğrudan MP4 linki"},
			)
		}
	}
	// Add Before/After cases section
	sections = append(sections, FormSection{
		ID:          "before_after",
		Title:       "Öncesi / Sonrası Vaka Seçimi",
		Order:       6,
		Required:    false,
		Description: "Bu hastaya özel planda gösterilecek klinik öncesi/sonrası vaka referansları.",
		Fields: []FormField{
			{Name: "before_after_case_ids", Label: "Seçili vaka ID'leri", Type: "string[]", Required: false, Repeatable: true},
		},
	})
	return sections
}

func FormSectionsByKind(kind string) []FormSection {
	if kind == "ultimate" || kind == "both" {
		return UltimateFormSections()
	}
	return ClassicFormSections()
}
