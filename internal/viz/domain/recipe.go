package domain

// Primitive is a closed FX set. The player is an interpreter; new treatments add recipes, not engine forks (OCP).
type Primitive string

const (
	Cam       Primitive = "camera"
	Glow      Primitive = "glow"
	Fluid     Primitive = "fluid"
	Tool      Primitive = "tool"
	Overlay   Primitive = "overlay"
	Material  Primitive = "material"
	Hide      Primitive = "hide"
	XRay      Primitive = "xray"
	Gum       Primitive = "gum"
	Particles Primitive = "particles"
	Shade     Primitive = "shade"
	Seat      Primitive = "seat"
	Extract   Primitive = "extract"
	Hotspot   Primitive = "hotspot"
	Barrier   Primitive = "barrier"
	Light     Primitive = "light"
)

type Effect struct {
	Op     Primitive `json:"op"`
	Target string    `json:"target,omitempty"`
	Color  string    `json:"color,omitempty"`
	Mesh   string    `json:"mesh,omitempty"`
	From   string    `json:"from,omitempty"`
	To     string    `json:"to,omitempty"`
}

type Move struct {
	N          int      `json:"n"`
	Title      string   `json:"title"`
	DurationMS int      `json:"duration_ms"`
	Effects    []Effect `json:"effects"`
}

type Recipe struct {
	Code    Code   `json:"code"`
	Group   string `json:"group"`
	TitleTR string `json:"title_tr"`
	TitleEN string `json:"title_en"`
	Moves   []Move `json:"moves"`
}

type Clip struct {
	Code  Code     `json:"code"`
	Teeth []string `json:"teeth"`
	Title string   `json:"title"`
	Moves []Move   `json:"moves"`
}

type Playlist struct {
	PlanID string `json:"plan_id"`
	Locale string `json:"locale"`
	Clips  []Clip `json:"clips"`
}

func (r Recipe) Title(locale string) string {
	if locale == "en" {
		return r.TitleEN
	}
	return r.TitleTR
}

func AllPrimitives() []Primitive {
	return []Primitive{
		Cam, Glow, Fluid, Tool, Overlay, Material, Hide, XRay,
		Gum, Particles, Shade, Seat, Extract, Hotspot, Barrier, Light,
	}
}
