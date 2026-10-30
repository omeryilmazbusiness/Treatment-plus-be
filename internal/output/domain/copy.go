package domain

import "strings"

type Copy struct {
	CoverKicker     string
	WelcomeTitle    string
	WelcomeDefault  string
	AboutTitle      string
	AboutDefault    string
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
	LabDefault      string
	AwardsTitle     string
	AwardsDefault   string
	HotelTitle      string
	HotelDefault    string
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
	Planned         string
}

func CopyFor(locale string) Copy {
	if strings.ToLower(strings.TrimSpace(locale)) == "en" {
		return Copy{
			CoverKicker:     "Treatment plan",
			WelcomeTitle:    "Welcome",
			WelcomeDefault:  "Dear %s,\n\nThank you for trusting us with your smile. This plan describes your current situation and the visits we recommend, with transparent fees.\n\nWe will walk every step with you.",
			AboutTitle:      "About the clinic",
			AboutDefault:    "Our clinic is a modern dental care centre combining cutting-edge technology with a dedicated team of specialist clinicians and patient-centred care.\n\nEquipped with 3D imaging, digital impression technology, and CAD/CAM milling systems, we perform all diagnostic and restorative procedures with micron-level precision. In full compliance with international sterilization standards, our priority is to deliver healthy, confident, and long-lasting smiles in a relaxed, comfortable environment.",
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
			UltimateTitle:   "3D Simulation",
			UltimateBody:    "Every step of your personalized treatment plan is visualised in interactive 3D on a digital jaw and tooth model. You can review the treatment stages, materials used, and expected clinical outcomes step by step in full detail.",
			VisualTitle:     "Planned result",
			VisualBody:      "The next pages show the clinical and laboratory path to your new smile.",
			MaterialTitle:   "Materials",
			MaterialBody:    "We prefer high-strength ceramics such as zirconia for implant restorations: biocompatible, colour-stable, and kind to the opposing teeth.",
			ResultTitle:     "The smile we aim for",
			LabTitle:        "Laboratory",
			LabDefault:      "Our clinic utilizes in-house and partner high-tech dental laboratories equipped with state-of-the-art CAD/CAM digital milling and 3D scanning systems. Restorations are crafted with micron-level precision from high-strength biocompatible zirconia and lithium disilicate blocks, ensuring superior aesthetics and long-lasting durability.",
			AwardsTitle:     "Awards & standards",
			AwardsDefault:   "Our clinic adheres to the highest international standards of patient safety, clinical excellence, and advanced sterilization protocols. We are dedicated to providing every patient with the highest quality care and comfort.",
			HotelTitle:      "Stay",
			HotelDefault:    "To ensure a seamless and relaxing experience for our out-of-town and international patients, we arrange accommodation at partner 4 and 5-star hotels alongside private VIP airport and clinic transfers.\n\nOur patient coordinators manage your entire schedule, from greeting you at the airport to transportation between your hotel and the clinic, providing comfort throughout your dental journey.",
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
			Planned:         "Planned",
		}
	}
	return Copy{
		CoverKicker:     "Tedavi planı",
		WelcomeTitle:    "Hoş geldiniz",
		WelcomeDefault:  "Sayın %s,\n\nGülüşünüzü bize emanet ettiğiniz için teşekkür ederiz. Bu plan mevcut durumunuzu ve önerdiğimiz seansları, şeffaf ücretlerle anlatır.\n\nHer adımda yanınızdayız.",
		AboutTitle:      "Kliniğimiz",
		AboutDefault:    "Kliniğimiz, ileri dental teknolojileri uzman hekim kadrosu ve hasta odaklı yaklaşımıyla bir araya getiren modern bir ağız ve diş sağlığı merkezidir.\n\nDijital diş hekimliği altyapımız, 3D tomografi ve CAD/CAM sistemlerimiz ile tüm tanı ve tedavi süreçlerini en yüksek hassasiyetle yürütüyoruz. Uluslararası sterilizasyon protokollerine tam uyum içinde, hastalarımızın güvenliğini ve konforunu her aşamada ön planda tutarak kalıcı ve estetik gülüşler tasarlıyoruz.",
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
		UltimateTitle:   "3D Tedavi Simülasyonu",
		UltimateBody:    "Planlanan tüm tedavi adımlarınız, kliniğimizin dijital altyapısında 3 boyutlu çene ve diş modeli üzerinde aşama aşama canlandırılır. Tedavi sürecinizi, kullanılacak materyalleri ve her seansın sonucunu interaktif 3D simülasyon üzerinden ayrıntılı olarak inceleyebilirsiniz.",
		VisualTitle:     "Hedeflenen sonuç",
		VisualBody:      "Sonraki sayfalar yeni gülüşünüze giden klinik ve laboratuvar yolunu gösterir.",
		MaterialTitle:   "Malzemeler",
		MaterialBody:    "İmplant üstü restorasyonlarda yüksek dayanımlı seramikleri, özellikle zirkonyayı tercih ederiz: biyouyumlu, renk sabit, karşıt dişe nazik.",
		ResultTitle:     "Hedeflediğimiz gülüş",
		LabTitle:        "Laboratuvar",
		LabDefault:      "Kliniğimiz bünyesinde ve anlaşmalı yüksek teknolojili dental laboratuvarımızda, en ileri CAD/CAM dijital kazıma ve 3D tarama sistemleri kullanılmaktadır. Hastalarımızın restorasyonları mikron düzeyinde hassasiyetle tasarlanmakta, yüksek dayanımlı biyouyumlu zirkonyum ve e-max porselen bloklardan kişiye özel olarak üretilmektedir. Dijital iş akışımız sayesinde prova süreleri kısalmakta ve estetik uyum mükemmelleştirilmektedir.",
		AwardsTitle:     "Ödüller ve standartlar",
		AwardsDefault:   "Kliniğimiz; uluslararası hasta güvenliği, sterilizasyon protokolleri ve klinik mükemmeliyet standartlarına tam uyumla hizmet vermektedir. Hastalarımıza en yüksek kalitede ve konforlu tedavi deneyimi sunmayı taahhüt ediyoruz.",
		HotelTitle:      "Konaklama",
		HotelDefault:    "Şehir dışından ve yurt dışından gelen hastalarımızın tedavi süreçlerini konforlu ve huzurlu geçirmeleri için anlaşmalı seçkin otellerimizde konaklama ve VIP transfer hizmeti organize edilmektedir.\n\nHavalimanı karşılama, otel-klinik arası ulaşım ve tedavi randevuları klinik asistanlarımız tarafından titizlikle planlanır. Tedavi seanslarınız arasında dinlenebileceğiniz, kliniğimize yakın merkezi ve nezih lokasyonlarda konforlu konaklama imkanı sunulmaktadır.",
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
		Planned:         "Planlandı",
	}
}
