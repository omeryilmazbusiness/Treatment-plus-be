package pdf

import "github.com/treatment-plus/treatment-plus-be/internal/output/domain"

var upper = []string{"18", "17", "16", "15", "14", "13", "12", "11", "21", "22", "23", "24", "25", "26", "27", "28"}
var lower = []string{"48", "47", "46", "45", "44", "43", "42", "41", "31", "32", "33", "34", "35", "36", "37", "38"}

func (d *doc) odontogram(x, y, w float64, highlight []string, findings []domain.ToothFinding) {
	cell := w / 16
	find := map[string]string{}
	for _, f := range findings {
		find[f.Tooth] = f.Status
	}
	mark := map[string]struct{}{}
	for _, t := range highlight {
		mark[t] = struct{}{}
	}
	d.drawArch(x, y, cell, upper, find, mark)
	d.drawArch(x, y+18, cell, lower, find, mark)
}

func (d *doc) drawArch(x, y, cell float64, teeth []string, find map[string]string, mark map[string]struct{}) {
	for i, t := range teeth {
		cx := x + float64(i)*cell
		fill := domain.RGB{R: 255, G: 255, B: 255}
		switch find[t] {
		case "missing":
			fill = domain.RGB{R: 220, G: 80, B: 80}
		case "filled":
			fill = domain.RGB{R: 90, G: 160, B: 220}
		case "crown":
			fill = d.theme.Accent
		case "intact":
			fill = domain.RGB{R: 180, G: 210, B: 170}
		}
		if _, ok := mark[t]; ok && find[t] == "" {
			fill = d.theme.Soft
		}
		d.setFill(fill)
		d.setDraw(d.theme.Primary)
		d.pdf.Rect(cx, y, cell-1.2, 12, "FD")
		d.font("", 6)
		d.setText(d.theme.Ink)
		d.pdf.SetXY(cx, y+3.5)
		d.pdf.CellFormat(cell-1.2, 5, t, "", 0, "C", false, 0, "")
	}
}
