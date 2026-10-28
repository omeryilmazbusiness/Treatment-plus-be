package domain

import "strings"

type Copy struct {
	CoverKicker     string
	WelcomeTitle    string
	WelcomeDefault  string
	AboutTitle      string
	HighlightsTitle string
	DoctorTitle     string
	SituationTitle  string
	SituationLead   string
	FindingsTitle   string
	VisitTitle      string
	Healing         string
	EducationTitle  string
	EducationBody   string
	AftercareTitle  string
	AftercareBody   string
	UltimateTitle   string
	UltimateBody    string
	VisualTitle     string
	VisualBody      string
	MaterialTitle   string
	MaterialBody    string
	ResultTitle     string
	LabTitle        string
	AwardsTitle     string
	AwardsDefault   string
	HotelTitle      string
	ContactTitle    string
	Phone           string
	Email           string
	Hours           string
	Website         string
	Address         string
	Qty             string
	Unit            string
	Line            string
	Teeth           string
	Subtotal        string
	Discount        string
	Grand           string
	Package         string
	Page            string
	Intact          string
	Missing         string
	Filled          string
	Crown           string
}

func CopyFor(locale string) Copy {
	if strings.ToLower(strings.TrimSpace(locale)) == "en" {
		return Copy{
			CoverKicker:     "Treatment plan",
			WelcomeTitle:    "Welcome",
			WelcomeDefault:  "Dear %s,\n\nThank you for trusting us with your smile. This plan describes your current situation and the visits we recommend, with transparent fees.\n\nWe will walk every step with you.",
			AboutTitle:      "About the clinic",
			HighlightsTitle: "In numbers",
			DoctorTitle:     "Your clinician",
			SituationTitle:  "Current situation",
			SituationLead:   "Panoramic overview and FDI chart.",
			FindingsTitle:   "Tooth-by-tooth",
			VisitTitle:      "Visit",
			Healing:         "Healing",
			EducationTitle:  "Your treatment",
			EducationBody:   "Each visit is sequenced so hard and soft tissues can heal before the next phase. Materials and techniques are selected for longevity and a natural result.",
			AftercareTitle:  "Aftercare",
			AftercareBody:   "Follow the written aftercare sheet: hygiene, diet in the first days, and the review appointment. Call us if swelling, bleeding or pain increases.",
			UltimateTitle:   "3D preview",
			UltimateBody:    "Photorealistic 3D animation is part of Treatment Plus Ultimate. Classic includes this page as a placeholder so the booklet structure stays complete.",
			VisualTitle:     "Planned result",
			VisualBody:      "The next pages show the clinical and laboratory path to your new smile.",
			MaterialTitle:   "Materials",
			MaterialBody:    "We prefer high-strength ceramics such as zirconia for implant restorations: biocompatible, colour-stable, and kind to the opposing teeth.",
			ResultTitle:     "The smile we aim for",
			LabTitle:        "Laboratory",
			AwardsTitle:     "Awards & standards",
			AwardsDefault:   "Our clinic is recognised for clinical excellence and patient safety. The award imagery in this booklet is the Treatment Plus standard mark.",
			HotelTitle:      "Stay",
			ContactTitle:    "Contact",
			Phone:           "Phone",
			Email:           "Email",
			Hours:           "Hours",
			Website:         "Web",
			Address:         "Address",
			Qty:             "Qty",
			Unit:            "Unit",
			Line:            "Line",
			Teeth:           "Teeth",
			Subtotal:        "Subtotal",
			Discount:        "Discount",
			Grand:           "Total",
			Package:         "Package",
			Page:            "Page",
			Intact:          "Intact",
			Missing:         "Missing",
			Filled:          "Filled",
			Crown:           "Crown",
		}
	}
	return Copy{
		CoverKicker:     "Tedavi planı",
		WelcomeTitle:    "Hoş geldiniz",
		WelcomeDefault:  "Sayın %s,\n\nGülüşünüzü bize emanet ettiğiniz için teşekkür ederiz. Bu plan mevcut durumunuzu ve önerdiğimiz seansları, şeffaf ücretlerle anlatır.\n\nHer adımda yanınızdayız.",
		AboutTitle:      "Kliniğimiz",
		HighlightsTitle: "Rakamlarla",
		DoctorTitle:     "Hekiminiz",
		SituationTitle:  "Mevcut durum",
		SituationLead:   "Panoramik görünüm ve FDI şeması.",
		FindingsTitle:   "Diş diş değerlendirme",
		VisitTitle:      "Seans",
		Healing:         "İyileşme",
		EducationTitle:  "Tedaviniz",
		EducationBody:   "Seanslar, bir sonraki evreden önce dokuların iyileşmesi için sıralanır. Malzeme ve teknikler uzun ömür ve doğal görünüm için seçilir.",
		AftercareTitle:  "Bakım",
		AftercareBody:   "Yazılı bakım formuna uyun: hijyen, ilk günlerin diyeti ve kontrol randevusu. Şişlik, kanama veya ağrı artarsa bizi arayın.",
		UltimateTitle:   "3D önizleme",
		UltimateBody:    "Fotogerçekçi 3D animasyon Treatment Plus Ultimate kapsamındadır. Classic kitapçıkta bu sayfa yapıyı tamamlamak için yer tutar.",
		VisualTitle:     "Hedeflenen sonuç",
		VisualBody:      "Sonraki sayfalar yeni gülüşünüze giden klinik ve laboratuvar yolunu gösterir.",
		MaterialTitle:   "Malzemeler",
		MaterialBody:    "İmplant üstü restorasyonlarda yüksek dayanımlı seramikleri, özellikle zirkonyayı tercih ederiz: biyouyumlu, renk sabit, karşıt dişe nazik.",
		ResultTitle:     "Hedeflediğimiz gülüş",
		LabTitle:        "Laboratuvar",
		AwardsTitle:     "Ödüller ve standartlar",
		AwardsDefault:   "Kliniğimiz klinik mükemmeliyet ve hasta güvenliği ile tanınır. Kitapçıktaki ödül görseli Treatment Plus standart markasıdır.",
		HotelTitle:      "Konaklama",
		ContactTitle:    "İletişim",
		Phone:           "Telefon",
		Email:           "E-posta",
		Hours:           "Saatler",
		Website:         "Web",
		Address:         "Adres",
		Qty:             "Adet",
		Unit:            "Birim",
		Line:            "Tutar",
		Teeth:           "Dişler",
		Subtotal:        "Ara toplam",
		Discount:        "İskonto",
		Grand:           "Genel toplam",
		Package:         "Paket",
		Page:            "Sayfa",
		Intact:          "Sağlam",
		Missing:         "Eksik",
		Filled:          "Dolgulu",
		Crown:           "Kuron",
	}
}
