package pdf

import (
	"bytes"
	"fmt"
	"strings"

	gofpdf "github.com/go-pdf/fpdf"

	"github.com/treatment-plus/treatment-plus-be/internal/output/domain"
)

func (d *doc) setFill(c domain.RGB) { d.pdf.SetFillColor(c.R, c.G, c.B) }
func (d *doc) setDraw(c domain.RGB) { d.pdf.SetDrawColor(c.R, c.G, c.B) }
func (d *doc) setText(c domain.RGB) { d.pdf.SetTextColor(c.R, c.G, c.B) }

func (d *doc) font(style string, size float64) {
	d.pdf.SetFont("sans", style, size)
}

func (d *doc) rect(x, y, w, h float64, c domain.RGB) {
	d.setFill(c)
	d.pdf.Rect(x, y, w, h, "F")
}

func (d *doc) text(x, y, w float64, align, style string, size float64, c domain.RGB, s string) {
	d.font(style, size)
	d.setText(c)
	d.pdf.SetXY(x, y)
	d.pdf.CellFormat(w, size*0.45, s, "", 0, align, false, 0, "")
}

func (d *doc) multi(x, y, w, h float64, style string, size float64, c domain.RGB, s string) {
	d.font(style, size)
	d.setText(c)
	d.pdf.SetXY(x, y)
	d.pdf.MultiCell(w, h, s, "", "", false)
}

func (d *doc) footer() {
	d.rect(0, 287, 210, 10, d.theme.Primary)
	d.setText(domain.RGB{R: 250, G: 248, B: 242})
	d.font("", 8)
	d.pdf.SetXY(16, 289.5)
	name := strings.TrimSpace(d.snap.Clinic.Name)
	if name == "" {
		name = "Treatment Plus"
	}
	d.pdf.CellFormat(120, 5, name, "", 0, "L", false, 0, "")
	d.pdf.SetXY(140, 289.5)
	d.pdf.CellFormat(54, 5, fmt.Sprintf("%s %d / 24", d.copy.Page, d.pdf.PageNo()), "", 0, "R", false, 0, "")
}

func (d *doc) headerBar(title string) {
	d.rect(0, 0, 210, 28, d.theme.Primary)
	d.rect(0, 28, 210, 1.4, d.theme.Accent)
	d.setText(domain.RGB{R: 250, G: 248, B: 242})
	d.font("B", 16)
	d.pdf.SetXY(16, 10)
	d.pdf.CellFormat(178, 10, title, "", 0, "L", false, 0, "")
}

func (d *doc) putImage(name string, raw []byte, x, y, w, h float64) bool {
	if len(raw) == 0 {
		return false
	}
	kind := detectImage(raw)
	if kind == "" {
		return false
	}
	opt := gofpdf.ImageOptions{ImageType: kind, ReadDpi: false}
	info := d.pdf.RegisterImageOptionsReader(name, opt, bytes.NewReader(raw))
	if info == nil {
		return false
	}
	d.pdf.ImageOptions(name, x, y, w, h, false, opt, 0, "")
	return true
}

func (d *doc) photoOrPlate(name string, raw []byte, x, y, w, h float64, label string) {
	d.rect(x, y, w, h, d.theme.Soft)
	if !d.putImage(name, raw, x, y, w, h) {
		d.font("", 10)
		d.setText(d.theme.Muted)
		d.pdf.SetXY(x, y+h/2-3)
		d.pdf.CellFormat(w, 6, label, "", 0, "C", false, 0, "")
	}
}

func detectImage(raw []byte) string {
	if len(raw) >= 3 && raw[0] == 0xFF && raw[1] == 0xD8 {
		return "JPEG"
	}
	if len(raw) >= 8 && raw[0] == 0x89 && raw[1] == 0x50 && raw[2] == 0x4E && raw[3] == 0x47 {
		return "PNG"
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
