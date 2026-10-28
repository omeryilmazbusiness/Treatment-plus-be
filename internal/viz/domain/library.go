package domain

func fx(op Primitive, target, color, mesh string) Effect {
	return Effect{Op: op, Target: target, Color: color, Mesh: mesh}
}

func move(n int, title, cam string, ms int, extra ...Effect) Move {
	fxs := []Effect{{Op: Cam, Target: cam}}
	fxs = append(fxs, extra...)
	return Move{N: n, Title: title, DurationMS: ms, Effects: fxs}
}

func rec(code Code, group, tr, en string, moves ...Move) Recipe {
	return Recipe{Code: code, Group: group, TitleTR: tr, TitleEN: en, Moves: moves}
}

func Library() []Recipe {
	return []Recipe{
		rec(FissureSealant, "preventive", "Fissür Örtücü", "Fissure sealant",
			move(1, "Odak ve temizlik", "occlusal", 2200, fx(Glow, "occlusal", "#3b82f6", "")),
			move(2, "Uygulama", "occlusal", 2400, fx(Fluid, "fissure", "#93c5fd", "sealant")),
			move(3, "Polimerizasyon", "occlusal", 2200, fx(Tool, "led", "#60a5fa", ""), fx(Material, "enamel", "#f8fafc", "")),
		),
		rec(TopicalFluoride, "preventive", "Topikal Flor", "Topical fluoride",
			move(1, "İzolasyon", "arch", 1800, fx(Gum, "retract", "", "")),
			move(2, "Jel", "arch", 2600, fx(Tool, "brush", "#67e8f9", ""), fx(Fluid, "enamel", "#a5f3fc", "fluoride"), fx(Particles, "foam", "#e0f2fe", "")),
		),
		rec(Desensitizer, "preventive", "Hassasiyet giderici", "Desensitizer",
			move(1, "Hassas boyun", "cervical", 2000, fx(Hotspot, "cervical", "#ef4444", "")),
			move(2, "Ajan", "cervical", 2400, fx(Fluid, "cervical", "#22d3ee", "desensitizer"), fx(Hotspot, "cervical", "#4ade80", "")),
		),
		rec(NightGuard, "preventive", "Gece plağı / splint", "Night guard",
			move(1, "Kapanış", "arch", 1800, fx(Gum, "close", "", "")),
			move(2, "Plak oturur", "arch", 2800, fx(Seat, "arch", "#e0f2fe", "splint")),
		),
		rec(CompositeFilling, "restorative", "Kompozit dolgu", "Composite filling",
			move(1, "Çürük temizliği", "occlusal", 2600, fx(Hotspot, "caries", "#44403c", ""), fx(Tool, "bur", "#94a3b8", ""), fx(Hide, "caries", "", "")),
			move(2, "Kompozit", "occlusal", 2200, fx(Fluid, "cavity", "#f5d0c5", "composite")),
			move(3, "Işık ve cila", "occlusal", 2200, fx(Tool, "led", "#60a5fa", ""), fx(Material, "enamel", "#fde8d8", "")),
		),
		rec(AmalgamFilling, "restorative", "Amalgam dolgu", "Amalgam filling",
			move(1, "Kaviteye amalgam", "occlusal", 2400, fx(Overlay, "cavity", "#9ca3af", "amalgam")),
			move(2, "Metal parlaklık", "occlusal", 1800, fx(Material, "metal", "#d1d5db", "")),
		),
		rec(InlayOnlay, "restorative", "İnley / onley / overlay", "Inlay / onlay",
			move(1, "Hazır boşluk", "occlusal", 2000, fx(Hotspot, "cavity", "#78716c", "")),
			move(2, "Blok oturur", "occlusal", 2800, fx(Seat, "occlusal", "#f5e6d3", "inlay")),
		),
		rec(OfficeWhitening, "restorative", "Ofis tipi beyazlatma", "In-office whitening",
			move(1, "Bariyer", "smile", 2000, fx(Shade, "enamel", "#e8d5a3", ""), fx(Barrier, "gums", "#3b82f6", "")),
			move(2, "Jel ve ışık", "smile", 2800, fx(Fluid, "enamel", "#f8fafc", "bleach"), fx(Light, "lamp", "#e9d5ff", "")),
			move(3, "Ton açılır", "smile", 2200, fx(Shade, "enamel", "#fffaf3", "")),
		),
		rec(CompositeBonding, "restorative", "Kompozit bonding", "Composite bonding",
			move(1, "Kırık uç", "facial", 1800, fx(Hotspot, "incisal", "#fca5a5", "")),
			move(2, "Form tamamlanır", "facial", 2600, fx(Fluid, "incisal", "#f5d0c5", "bonding"), fx(Material, "enamel", "#fde8d8", "")),
		),
		rec(RootCanal, "endo", "Kanal tedavisi", "Root canal",
			move(1, "Kesit ve pulpa", "xray", 2400, fx(XRay, "on", "", ""), fx(Hotspot, "pulp", "#ef4444", "")),
			move(2, "Eğe", "xray", 2600, fx(Tool, "file", "#94a3b8", ""), fx(Hide, "pulp", "", "")),
			move(3, "Guta-perka", "xray", 2400, fx(Fluid, "canal", "#f9a8d4", "gutta")),
		),
		rec(Retreatment, "endo", "Kanal yenileme", "Retreatment",
			move(1, "Eski dolgu", "xray", 2000, fx(XRay, "on", "", ""), fx(Hotspot, "old_fill", "#a8a29e", "")),
			move(2, "Söküm", "xray", 2600, fx(Tool, "file", "#64748b", ""), fx(Hide, "old_fill", "", "")),
			move(3, "Yeniden doldur", "xray", 2400, fx(Fluid, "canal", "#f9a8d4", "gutta")),
		),
		rec(PulpCap, "endo", "Kuafaj / pulpotomi", "Pulp cap",
			move(1, "Derin çürük", "xray", 2000, fx(XRay, "on", "", ""), fx(Hotspot, "near_pulp", "#f97316", "")),
			move(2, "Örtücü", "xray", 2400, fx(Overlay, "pulp_roof", "#f8fafc", "liner")),
		),
		rec(PostCore, "endo", "Post-core", "Post and core",
			move(1, "Madde kaybı", "xray", 1800, fx(XRay, "on", "", ""), fx(Hotspot, "loss", "#78716c", "")),
			move(2, "Post", "xray", 2400, fx(Seat, "canal", "#64748b", "post")),
			move(3, "Kor", "occlusal", 2200, fx(Overlay, "core", "#e7e5e4", "core")),
		),
		rec(Apexification, "endo", "Apeksifikasyon", "Apexification",
			move(1, "Açık apeks", "xray", 2000, fx(XRay, "on", "", ""), fx(Hotspot, "apex", "#f87171", "")),
			move(2, "Tıkaç", "xray", 2600, fx(Overlay, "apex", "#e7e5e4", "plug")),
		),
		rec(ZirconiaCrown, "prosthetic", "Zirkonyum kron", "Zirconia crown",
			move(1, "Güdük", "facial", 2000, fx(Material, "stump", "#e7e5e4", "")),
			move(2, "Altyapı", "facial", 2200, fx(Seat, "crown", "#e5e7eb", "coping")),
			move(3, "Kaplama", "facial", 2600, fx(Seat, "crown", "#f8fafc", "zirconia")),
		),
		rec(PFMCrown, "prosthetic", "Metal destekli porselen", "PFM crown",
			move(1, "Metal başlık", "facial", 2200, fx(Seat, "crown", "#9ca3af", "metal_cap")),
			move(2, "Porselen", "facial", 2600, fx(Seat, "crown", "#f5e6d3", "porcelain")),
		),
		rec(EmaxVeneer, "prosthetic", "E-max / lamine", "E-max veneer",
			move(1, "Ön yüzey prep", "facial", 2000, fx(Material, "facial", "#e7e5e4", "")),
			move(2, "Yaprak oturur", "facial", 2800, fx(Seat, "facial", "#f8fafc", "veneer")),
		),
		rec(TempCrown, "prosthetic", "Geçici kron", "Temporary crown",
			move(1, "Geçici oturur", "facial", 2400, fx(Seat, "crown", "#fde68a", "temp_crown")),
		),
		rec(Scaling, "perio", "Diş taşı temizliği", "Scaling",
			move(1, "Tartar", "cervical", 1800, fx(Hotspot, "tartar", "#a16207", "")),
			move(2, "Scaler", "cervical", 2800, fx(Tool, "scaler", "#94a3b8", ""), fx(Particles, "chip", "#d6d3d1", ""), fx(Glow, "enamel", "#f8fafc", "")),
		),
		rec(RootPlaning, "perio", "Küretaj", "Root planing",
			move(1, "Cep", "intra", 2000, fx(Gum, "pocket", "", ""), fx(Hotspot, "root", "#fca5a5", "")),
			move(2, "Küret", "intra", 2800, fx(Tool, "curette", "#94a3b8", ""), fx(Gum, "pink", "#f9a8d4", "")),
		),
		rec(Gingivectomy, "perio", "Gingivektomi", "Gingivectomy",
			move(1, "Fazla et", "smile", 1800, fx(Hotspot, "gummy", "#ef4444", "")),
			move(2, "Tıraş", "smile", 2600, fx(Tool, "laser", "#22d3ee", ""), fx(Gum, "trim", "", "")),
		),
		rec(Frenectomy, "perio", "Frenektomi", "Frenectomy",
			move(1, "Frenulum", "smile", 1800, fx(Hotspot, "frenum", "#f97316", "")),
			move(2, "Serbestleşme", "smile", 2400, fx(Tool, "laser", "#22d3ee", ""), fx(Hide, "frenum", "", "")),
		),
		rec(GumGraft, "perio", "Diş eti grefti", "Gum graft",
			move(1, "Çekilme", "cervical", 1800, fx(Hotspot, "recession", "#fca5a5", "")),
			move(2, "Yama", "cervical", 2800, fx(Seat, "root", "#fda4af", "graft")),
		),
		rec(Extraction, "surgery", "Normal çekim", "Extraction",
			move(1, "Diş yuvada", "socket", 1600, fx(Hotspot, "tooth", "#78716c", "")),
			move(2, "Pens ile çekim", "socket", 3000, fx(Tool, "forceps", "#64748b", ""), fx(Extract, "tooth", "", "")),
		),
		rec(SurgicalExtraction, "surgery", "Cerrahi / gömülü çekim", "Surgical extraction",
			move(1, "Flep ve kemik", "socket", 2400, fx(Gum, "flap", "", ""), fx(XRay, "bone", "", ""), fx(Hotspot, "impacted", "#a8a29e", "")),
			move(2, "Bölerek çıkarma", "socket", 3000, fx(Extract, "split", "", ""), fx(Gum, "suture", "", "")),
		),
		rec(Apicoectomy, "surgery", "Apikal rezeksiyon", "Apicoectomy",
			move(1, "Kist", "xray", 2000, fx(XRay, "on", "", ""), fx(Hotspot, "cyst", "#ef4444", "")),
			move(2, "Kök ucu kesilir", "xray", 2600, fx(Tool, "bur", "#94a3b8", ""), fx(Hide, "cyst", "", "")),
		),
		rec(Implant, "surgery", "Diş implantı", "Dental implant",
			move(1, "Boş sırt", "socket", 1800, fx(Hide, "tooth", "", "")),
			move(2, "Vida", "socket", 3200, fx(Seat, "bone", "#94a3b8", "implant")),
			move(3, "Abutment ve kron", "socket", 2800, fx(Seat, "implant", "#d1d5db", "abutment"), fx(Seat, "crown", "#f8fafc", "zirconia")),
		),
		rec(BoneGraftSinus, "surgery", "Kemik grefti ve sinüs lifting", "Bone graft / sinus lift",
			move(1, "İnce kemik", "sinus", 2000, fx(XRay, "on", "", ""), fx(Hotspot, "sinus", "#7dd3fc", "")),
			move(2, "Kaldır ve doldur", "sinus", 3200, fx(Gum, "lift_sinus", "", ""), fx(Particles, "granule", "#e7e5e4", "")),
		),
		rec(Braces, "ortho", "Braket", "Braces",
			move(1, "Çapraşıklık", "smile", 1800, fx(Hotspot, "crowding", "#fbbf24", "")),
			move(2, "Braketler", "smile", 2600, fx(Overlay, "facial", "#e5e7eb", "bracket")),
			move(3, "Ark teli", "smile", 2400, fx(Overlay, "arch", "#94a3b8", "archwire")),
		),
		rec(AlignerAttachment, "ortho", "Şeffaf plak attachment", "Aligner attachments",
			move(1, "Attachment", "smile", 2000, fx(Overlay, "facial", "#fde68a", "attachment")),
			move(2, "Plak oturur", "smile", 2800, fx(Seat, "arch", "#e0f2fe", "aligner")),
		),
		rec(ImpactedExposure, "ortho", "Gömülü diş sürdürme", "Impacted exposure",
			move(1, "Gömülü kanin", "intra", 2000, fx(XRay, "on", "", ""), fx(Hotspot, "impacted", "#a8a29e", "")),
			move(2, "Çekme", "intra", 3000, fx(Overlay, "button", "#e5e7eb", "button"), fx(Overlay, "vector", "#22c55e", "chain")),
		),
	}
}

func RecipeByCode(code Code) (Recipe, bool) {
	for _, r := range Library() {
		if r.Code == code {
			return r, true
		}
	}
	return Recipe{}, false
}

func DefaultTeeth(code Code) []string {
	switch code {
	case EmaxVeneer, CompositeBonding, Frenectomy, Gingivectomy, Braces, AlignerAttachment:
		return []string{"11", "21"}
	case NightGuard:
		return []string{"11", "21", "31", "41"}
	case SurgicalExtraction:
		return []string{"18"}
	case ImpactedExposure:
		return []string{"13"}
	case BoneGraftSinus:
		return []string{"16"}
	default:
		return []string{"16"}
	}
}

func init() {
	for _, r := range Library() {
		registerAlias(string(r.Code), r.Code)
		registerAlias(r.TitleTR, r.Code)
		registerAlias(r.TitleEN, r.Code)
	}
}
