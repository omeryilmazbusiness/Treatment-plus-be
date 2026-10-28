package domain

type RGB struct {
	R, G, B int
}

type Theme struct {
	Name    string
	Primary RGB
	Accent  RGB
	Ink     RGB
	Muted   RGB
	Paper   RGB
	Soft    RGB
}

func ThemeFor(palette string) Theme {
	switch palette {
	case "forest":
		return Theme{Name: "forest", Primary: RGB{27, 67, 50}, Accent: RGB{196, 165, 82}, Ink: RGB{22, 28, 24}, Muted: RGB{90, 102, 96}, Paper: RGB{248, 250, 247}, Soft: RGB{232, 240, 234}}
	case "rose":
		return Theme{Name: "rose", Primary: RGB{122, 36, 50}, Accent: RGB{201, 154, 120}, Ink: RGB{36, 22, 24}, Muted: RGB{110, 92, 94}, Paper: RGB{252, 248, 247}, Soft: RGB{246, 232, 232}}
	case "slate":
		return Theme{Name: "slate", Primary: RGB{51, 65, 85}, Accent: RGB{148, 163, 184}, Ink: RGB{15, 23, 42}, Muted: RGB{100, 116, 139}, Paper: RGB{248, 250, 252}, Soft: RGB{226, 232, 240}}
	default:
		return Theme{Name: "navy", Primary: RGB{11, 31, 58}, Accent: RGB{201, 162, 39}, Ink: RGB{17, 24, 39}, Muted: RGB{90, 98, 112}, Paper: RGB{250, 248, 242}, Soft: RGB{232, 236, 244}}
	}
}
