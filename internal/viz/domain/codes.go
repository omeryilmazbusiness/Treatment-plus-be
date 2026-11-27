package domain

import (
	"strings"
	"unicode"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

// Code is a closed set of patient-facing treatment clips. Clinic form stores this, not free text.
type Code string

const (
	FissureSealant     Code = "fissure_sealant"
	TopicalFluoride    Code = "topical_fluoride"
	Desensitizer       Code = "desensitizer"
	NightGuard         Code = "night_guard"
	CompositeFilling   Code = "composite_filling"
	AmalgamFilling     Code = "amalgam_filling"
	InlayOnlay         Code = "inlay_onlay"
	OfficeWhitening    Code = "office_whitening"
	CompositeBonding   Code = "composite_bonding"
	RootCanal          Code = "root_canal"
	Retreatment        Code = "retreatment"
	PulpCap            Code = "pulp_cap"
	PostCore           Code = "post_core"
	Apexification      Code = "apexification"
	ZirconiaCrown      Code = "zirconia_crown"
	PFMCrown           Code = "pfm_crown"
	EmaxVeneer         Code = "emax_veneer"
	TempCrown          Code = "temp_crown"
	Bridge             Code = "bridge"
	Scaling            Code = "scaling"
	RootPlaning        Code = "root_planing"
	Gingivectomy       Code = "gingivectomy"
	Frenectomy         Code = "frenectomy"
	GumGraft           Code = "gum_graft"
	Extraction         Code = "extraction"
	SurgicalExtraction Code = "surgical_extraction"
	Apicoectomy        Code = "apicoectomy"
	Implant            Code = "implant"
	BoneGraftSinus     Code = "bone_graft_sinus"
	Braces             Code = "braces"
	AlignerAttachment  Code = "aligner_attachment"
	ImpactedExposure   Code = "impacted_exposure"
)

func AllCodes() []Code {
	return []Code{
		FissureSealant, TopicalFluoride, Desensitizer, NightGuard,
		CompositeFilling, AmalgamFilling, InlayOnlay, OfficeWhitening, CompositeBonding,
		RootCanal, Retreatment, PulpCap, PostCore, Apexification,
		ZirconiaCrown, PFMCrown, EmaxVeneer, TempCrown, Bridge,
		Scaling, RootPlaning, Gingivectomy, Frenectomy, GumGraft,
		Extraction, SurgicalExtraction, Apicoectomy, Implant, BoneGraftSinus,
		Braces, AlignerAttachment, ImpactedExposure,
	}
}

var turkishReplacer = strings.NewReplacer(
	"İ", "i",
	"I", "ı",
	"\u0307", "",
)

var asciiFoldReplacer = strings.NewReplacer(
	"ı", "i",
	"ş", "s",
	"ğ", "g",
	"ü", "u",
	"ö", "o",
	"ç", "c",
)

func asciiFold(s string) string {
	return asciiFoldReplacer.Replace(s)
}

func Normalize(s string) string {
	s = turkishReplacer.Replace(strings.TrimSpace(s))
	s = strings.ToLower(s)
	var b strings.Builder
	prev := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prev = false
			continue
		}
		if !prev {
			b.WriteByte('_')
			prev = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func ParseCode(s string) (Code, error) {
	raw := Normalize(s)
	if raw == "" {
		return "", errx.New("invalid_animation_code", "unknown treatment animation", errx.ErrInvalid)
	}
	if c, ok := matchCodeExact(raw); ok {
		return c, nil
	}
	folded := asciiFold(raw)
	if c, ok := matchCodeExact(folded); ok {
		return c, nil
	}
	// Token n-gram fallback (longest multi-word phrase first) for descriptive clinical names
	// e.g. "Üst Çene Zirkonyum Kuron" or "Kompozit Dolgu (2 Yüzlü)".
	tokens := strings.Split(raw, "_")
	for width := len(tokens) - 1; width >= 1; width-- {
		for start := 0; start+width <= len(tokens); start++ {
			sub := strings.Join(tokens[start:start+width], "_")
			if c, ok := matchCodeExact(sub); ok {
				return c, nil
			}
			if c, ok := matchCodeExact(asciiFold(sub)); ok {
				return c, nil
			}
		}
	}
	return "", errx.New("invalid_animation_code", "unknown treatment animation", errx.ErrInvalid)
}

func matchCodeExact(candidate string) (Code, bool) {
	if alias, ok := aliases[candidate]; ok {
		return alias, true
	}
	for _, c := range AllCodes() {
		if string(c) == candidate {
			return c, true
		}
	}
	return "", false
}

func registerAlias(s string, c Code) {
	if k := Normalize(s); k != "" {
		if _, exists := aliases[k]; !exists {
			aliases[k] = c
		}
		f := asciiFold(k)
		if f != "" {
			if _, exists := aliases[f]; !exists {
				aliases[f] = c
			}
		}
	}
}

var aliases = map[string]Code{
	"fissür_örtücü": FissureSealant, "fissur_ortucu": FissureSealant, "sealant": FissureSealant, "fissür": FissureSealant, "fissur": FissureSealant,
	"topikal_flor": TopicalFluoride, "flor": TopicalFluoride, "fluoride": TopicalFluoride, "florür": TopicalFluoride, "florur": TopicalFluoride,
	"hassasiyet": Desensitizer, "desensitizer": Desensitizer, "hassasiyet_giderici": Desensitizer,
	"gece_plağı": NightGuard, "gece_plagi": NightGuard, "splint": NightGuard, "night_guard": NightGuard, "bruksizm_plağı": NightGuard, "bruksizm_plagi": NightGuard,
	"kompozit_dolgu": CompositeFilling, "composite": CompositeFilling, "dolgu": CompositeFilling, "filling": CompositeFilling, "estetik_dolgu": CompositeFilling,
	"amalgam": AmalgamFilling, "amalgam_dolgu": AmalgamFilling,
	"inley": InlayOnlay, "onley": InlayOnlay, "overlay": InlayOnlay, "inlay": InlayOnlay, "onlay": InlayOnlay,
	"beyazlatma": OfficeWhitening, "whitening": OfficeWhitening, "bleaching": OfficeWhitening, "diş_beyazlatma": OfficeWhitening, "dis_beyazlatma": OfficeWhitening,
	"bonding": CompositeBonding, "kompozit_bonding": CompositeBonding,
	"kanal": RootCanal, "kanal_tedavisi": RootCanal, "root_canal": RootCanal, "endo": RootCanal, "endodonti": RootCanal, "hedstrom_cleaning": RootCanal, "hedstrom": RootCanal,
	"retreatment": Retreatment, "kanal_yenileme": Retreatment, "kanal_tekrarı": Retreatment, "kanal_tekrari": Retreatment,
	"kuafaj": PulpCap, "pulpotomi": PulpCap, "pulp_cap": PulpCap, "direkt_kuafaj": PulpCap, "indirekt_kuafaj": PulpCap,
	"post": PostCore, "post_core": PostCore, "post_kor": PostCore, "fiber_post": PostCore,
	"apeksifikasyon": Apexification, "rejeneratif": Apexification,
	"zirkonyum": ZirconiaCrown, "zirkonya": ZirconiaCrown, "zirconia": ZirconiaCrown, "zirkonyum_kron": ZirconiaCrown, "zirkonyum_kuron": ZirconiaCrown, "zirkonya_kron": ZirconiaCrown, "zirkonya_kuron": ZirconiaCrown, "zirkonyum_kaplama": ZirconiaCrown, "kron": ZirconiaCrown, "kuron": ZirconiaCrown, "crown": ZirconiaCrown,
	"metal_destekli": PFMCrown, "porselen_kron": PFMCrown, "porselen_kuron": PFMCrown, "pfm": PFMCrown, "metal_destekli_porselen": PFMCrown, "metal_porselen": PFMCrown,
	"e_max": EmaxVeneer, "emax": EmaxVeneer, "lamine": EmaxVeneer, "lamina": EmaxVeneer, "veneer": EmaxVeneer, "vener": EmaxVeneer, "yaprak_porselen": EmaxVeneer, "emax_kron": EmaxVeneer, "emax_kuron": EmaxVeneer, "emax_lamine": EmaxVeneer,
	"geçici_kron": TempCrown, "gecici_kron": TempCrown, "geçici_kuron": TempCrown, "gecici_kuron": TempCrown, "temporary": TempCrown, "geçici_kaplama": TempCrown, "gecici_kaplama": TempCrown,
	"köprü": Bridge, "kopru": Bridge, "bridge": Bridge, "dental_bridge": Bridge, "zirkonyum_köprü": Bridge, "zirkonyum_kopru": Bridge, "porselen_köprü": Bridge, "porselen_kopru": Bridge,
	"detertraj": Scaling, "detartraj": Scaling, "diş_taşı": Scaling, "dis_tasi": Scaling, "diş_taşı_temizliği": Scaling, "dis_tasi_temizligi": Scaling, "scaling": Scaling, "tartar": Scaling,
	"küretaj": RootPlaning, "kuretaj": RootPlaning, "root_planing": RootPlaning, "subgingival_küretaj": RootPlaning, "subgingival_kuretaj": RootPlaning,
	"gingivektomi": Gingivectomy, "gingivoplasti": Gingivectomy, "pembe_estetik": Gingivectomy,
	"frenektomi": Frenectomy,
	"greft":      GumGraft, "diş_eti_grefti": GumGraft, "dis_eti_grefti": GumGraft, "gum_graft": GumGraft, "bağ_dokusu_grefti": GumGraft,
	"çekim": Extraction, "cekim": Extraction, "diş_çekimi": Extraction, "dis_cekimi": Extraction, "normal_çekim": Extraction, "normal_cekim": Extraction, "extraction": Extraction,
	"gömülü": SurgicalExtraction, "gomulu": SurgicalExtraction, "cerrahi_çekim": SurgicalExtraction, "cerrahi_cekim": SurgicalExtraction, "gömülü_çekim": SurgicalExtraction, "gomulu_cekim": SurgicalExtraction, "wisdom": SurgicalExtraction, "yirmi_yaş": SurgicalExtraction, "yirmi_yas": SurgicalExtraction, "20_yaş": SurgicalExtraction, "20_yas": SurgicalExtraction,
	"apikal": Apicoectomy, "rezeksiyon": Apicoectomy, "apikal_rezeksiyon": Apicoectomy, "apicoectomy": Apicoectomy,
	"implant": Implant, "diş_implantı": Implant, "diş_implanti": Implant, "dis_implanti": Implant, "dental_implant": Implant,
	"sinüs": BoneGraftSinus, "sinus": BoneGraftSinus, "kemik_grefti": BoneGraftSinus, "kemik_tozu": BoneGraftSinus, "sinus_lifting": BoneGraftSinus, "sinüs_lifting": BoneGraftSinus, "sinus_lift": BoneGraftSinus, "bone_graft": BoneGraftSinus,
	"braket": Braces, "ortodonti": Braces, "braces": Braces, "tel": Braces, "diş_teli": Braces, "dis_teli": Braces,
	"aligner": AlignerAttachment, "şeffaf_plak": AlignerAttachment, "seffaf_plak": AlignerAttachment, "invisalign": AlignerAttachment,
	"sürdürme": ImpactedExposure, "surdurme": ImpactedExposure, "gömülü_sürdürme": ImpactedExposure, "gomulu_surdurme": ImpactedExposure,
}
