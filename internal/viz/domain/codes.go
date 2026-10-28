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
		ZirconiaCrown, PFMCrown, EmaxVeneer, TempCrown,
		Scaling, RootPlaning, Gingivectomy, Frenectomy, GumGraft,
		Extraction, SurgicalExtraction, Apicoectomy, Implant, BoneGraftSinus,
		Braces, AlignerAttachment, ImpactedExposure,
	}
}

func Normalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
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
	if alias, ok := aliases[raw]; ok {
		raw = string(alias)
	}
	for _, c := range AllCodes() {
		if string(c) == raw {
			return c, nil
		}
	}
	return "", errx.New("invalid_animation_code", "unknown treatment animation", errx.ErrInvalid)
}

func registerAlias(s string, c Code) {
	if k := Normalize(s); k != "" {
		if _, exists := aliases[k]; !exists {
			aliases[k] = c
		}
	}
}

var aliases = map[string]Code{
	"fissür_örtücü": FissureSealant, "fissur_ortucu": FissureSealant, "sealant": FissureSealant,
	"topikal_flor": TopicalFluoride, "flor": TopicalFluoride, "fluoride": TopicalFluoride,
	"hassasiyet": Desensitizer, "desensitizer": Desensitizer,
	"gece_plağı": NightGuard, "gece_plagi": NightGuard, "splint": NightGuard, "night_guard": NightGuard,
	"kompozit_dolgu": CompositeFilling, "composite": CompositeFilling, "dolgu": CompositeFilling,
	"amalgam": AmalgamFilling, "amalgam_dolgu": AmalgamFilling,
	"inley": InlayOnlay, "onley": InlayOnlay, "overlay": InlayOnlay, "inlay": InlayOnlay,
	"beyazlatma": OfficeWhitening, "whitening": OfficeWhitening, "bleaching": OfficeWhitening,
	"bonding": CompositeBonding, "kompozit_bonding": CompositeBonding,
	"kanal": RootCanal, "kanal_tedavisi": RootCanal, "root_canal": RootCanal, "endo": RootCanal,
	"retreatment": Retreatment, "kanal_yenileme": Retreatment,
	"kuafaj": PulpCap, "pulpotomi": PulpCap, "pulp_cap": PulpCap,
	"post": PostCore, "post_core": PostCore, "post_kor": PostCore,
	"apeksifikasyon": Apexification, "rejeneratif": Apexification,
	"zirkonyum": ZirconiaCrown, "zirkonya": ZirconiaCrown, "zirconia": ZirconiaCrown, "zirkonyum_kron": ZirconiaCrown,
	"metal_destekli": PFMCrown, "porselen_kron": PFMCrown, "pfm": PFMCrown,
	"e_max": EmaxVeneer, "emax": EmaxVeneer, "lamine": EmaxVeneer, "veneer": EmaxVeneer, "vener": EmaxVeneer,
	"geçici_kron": TempCrown, "gecici_kron": TempCrown, "temporary": TempCrown,
	"detertraj": Scaling, "diş_taşı": Scaling, "dis_tasi": Scaling, "scaling": Scaling,
	"küretaj": RootPlaning, "kuretaj": RootPlaning, "root_planing": RootPlaning,
	"gingivektomi": Gingivectomy, "gingivoplasti": Gingivectomy,
	"frenektomi": Frenectomy,
	"greft":      GumGraft, "diş_eti_grefti": GumGraft, "gum_graft": GumGraft,
	"çekim": Extraction, "cekim": Extraction, "extraction": Extraction,
	"gömülü": SurgicalExtraction, "gomulu": SurgicalExtraction, "cerrahi_çekim": SurgicalExtraction, "wisdom": SurgicalExtraction,
	"apikal": Apicoectomy, "rezeksiyon": Apicoectomy, "apicoectomy": Apicoectomy,
	"implant": Implant, "diş_implanti": Implant,
	"sinüs": BoneGraftSinus, "sinus": BoneGraftSinus, "kemik_grefti": BoneGraftSinus, "sinus_lifting": BoneGraftSinus,
	"braket": Braces, "ortodonti": Braces, "braces": Braces, "tel": Braces,
	"aligner": AlignerAttachment, "şeffaf_plak": AlignerAttachment, "seffaf_plak": AlignerAttachment, "invisalign": AlignerAttachment,
	"sürdürme": ImpactedExposure, "surdurme": ImpactedExposure, "gömülü_sürdürme": ImpactedExposure,
}
