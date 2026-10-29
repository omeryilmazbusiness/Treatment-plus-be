package pdf

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"

	"github.com/treatment-plus/treatment-plus-be/internal/output/app"
	"github.com/treatment-plus/treatment-plus-be/internal/output/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
)

type Renderer struct{}

func NewRenderer() *Renderer { return &Renderer{} }

type doc struct {
	pdf    *fpdf.Fpdf
	snap   domain.Snapshot
	theme  domain.Theme
	copy   domain.Copy
	images app.Images
}

func (r *Renderer) Render(_ context.Context, snap domain.Snapshot, images app.Images) ([]byte, int, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddUTF8FontFromBytes("sans", "", fontRegular)
	pdf.AddUTF8FontFromBytes("sans", "B", fontBold)

	d := &doc{
		pdf:    pdf,
		snap:   snap,
		theme:  domain.ThemeFor(firstNonEmpty(snap.Palette, snap.Clinic.Palette)),
		copy:   domain.CopyFor(firstNonEmpty(snap.Locale, snap.Clinic.Locale)),
		images: images,
	}

	d.pageCover()
	d.pageWelcome()
	d.pagePhoto("welcome-photo", d.images.Cover, d.snap.Clinic.Name)
	d.pageAbout()
	d.pageHighlights()
	d.pageDoctor()
	d.pagePhoto("doctor-photo", d.images.Doctor, d.snap.Clinic.Doctor.Name)
	d.pageSituation()
	d.pageFindings()
	d.pageVisit(0)
	d.pageEducation()
	d.pageVisit(1)
	d.pageAftercare()
	d.pageUltimate()
	d.pageVisual()
	d.pageMaterial()
	d.pageResult()
	d.pagePhoto("lab-photo", d.images.Lab, firstNonEmpty(d.snap.Clinic.Lab.Title, d.copy.LabTitle))
	d.pageLab()
	d.pageAwards()
	d.pagePhoto("award-photo", d.images.Award, d.copy.AwardsTitle)
	d.pagePhoto("hotel-photo", d.images.Hotel, firstNonEmpty(d.snap.Clinic.Hotel.Name, d.copy.HotelTitle))
	d.pageHotel()
	d.pageContact()

	if pdf.Err() {
		return nil, 0, errx.New("pdf_render", pdf.Error().Error(), errx.ErrInvalid)
	}
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, 0, errx.New("pdf_render", err.Error(), errx.ErrInvalid)
	}
	return buf.Bytes(), pdf.PageCount(), nil
}

func (d *doc) add() {
	d.pdf.AddPage()
	d.pdf.SetFillColor(d.theme.Paper.R, d.theme.Paper.G, d.theme.Paper.B)
	d.pdf.Rect(0, 0, 210, 297, "F")
}

func (d *doc) pageCover() {
	d.add()
	d.rect(0, 0, 210, 297, d.theme.Primary)
	d.rect(0, 0, 8, 297, d.theme.Accent)
	d.putImage("cover", d.images.Cover, 0, 0, 210, 160)
	d.rect(0, 150, 210, 147, d.theme.Primary)
	d.rect(16, 168, 36, 2, d.theme.Accent)
	d.putImage("logo-cover", d.images.Logo, 16, 178, 28, 28)
	d.setText(domain.RGB{R: 250, G: 248, B: 242})
	d.font("", 11)
	d.pdf.SetXY(16, 214)
	d.pdf.CellFormat(178, 6, d.copy.CoverKicker, "", 0, "L", false, 0, "")
	d.font("B", 28)
	d.pdf.SetXY(16, 224)
	d.pdf.MultiCell(178, 12, firstNonEmpty(d.snap.Clinic.Name, "Clinic"), "", "", false)
	d.font("", 14)
	d.pdf.SetXY(16, 250)
	d.pdf.MultiCell(178, 7, firstNonEmpty(d.snap.Clinic.Slogan, d.snap.Title), "", "", false)
	d.footer()
}

func (d *doc) pageWelcome() {
	d.add()
	d.headerBar(d.copy.WelcomeTitle)
	name := firstNonEmpty(d.snap.Patient.Name, " ")
	body := d.snap.Clinic.WelcomeBody
	if strings.TrimSpace(body) == "" {
		body = fmt.Sprintf(d.copy.WelcomeDefault, name)
	} else if !strings.Contains(body, name) && strings.TrimSpace(name) != "" {
		body = name + ",\n\n" + body
	}
	d.multi(16, 42, 178, 7, "", 12, d.theme.Ink, body)
	d.font("B", 13)
	d.setText(d.theme.Primary)
	d.pdf.SetXY(16, 230)
	d.pdf.CellFormat(178, 7, firstNonEmpty(d.snap.Clinic.SignerName, d.snap.Clinic.Name), "", 0, "L", false, 0, "")
	d.font("", 11)
	d.setText(d.theme.Muted)
	d.pdf.SetXY(16, 237)
	d.pdf.CellFormat(178, 6, d.snap.Clinic.SignerTitle, "", 0, "L", false, 0, "")
	d.footer()
}

func (d *doc) pagePhoto(name string, raw []byte, caption string) {
	d.add()
	d.rect(0, 0, 210, 297, d.theme.Primary)
	d.photoOrPlate(name, raw, 12, 18, 186, 240, caption)
	d.footer()
}

func (d *doc) pageAbout() {
	d.add()
	d.headerBar(d.copy.AboutTitle)
	d.photoOrPlate("about", d.images.About, 16, 40, 178, 110, d.copy.AboutTitle)
	text := firstNonEmpty(d.snap.Clinic.AboutText, d.snap.Clinic.Intro)
	d.multi(16, 160, 178, 6.5, "", 12, d.theme.Ink, text)
	d.footer()
}

func (d *doc) pageHighlights() {
	d.add()
	d.headerBar(d.copy.HighlightsTitle)
	stats := d.snap.Clinic.Highlights
	if len(stats) == 0 {
		stats = []domain.Stat{{Value: "—", Label: d.copy.HighlightsTitle}}
	}
	cols := 3
	cardW, cardH := 56.0, 42.0
	for i, st := range stats {
		if i >= 6 {
			break
		}
		col := i % cols
		row := i / cols
		x := 16 + float64(col)*(cardW+8)
		y := 48 + float64(row)*(cardH+12)
		d.rect(x, y, cardW, cardH, d.theme.Soft)
		d.rect(x, y, 3, cardH, d.theme.Accent)
		d.font("B", 18)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(x+8, y+10)
		d.pdf.CellFormat(cardW-12, 10, st.Value, "", 0, "L", false, 0, "")
		d.font("", 10)
		d.setText(d.theme.Muted)
		d.pdf.SetXY(x+8, y+24)
		d.pdf.MultiCell(cardW-12, 5, st.Label, "", "", false)
	}
	d.footer()
}

func (d *doc) pageDoctor() {
	d.add()
	d.headerBar(d.copy.DoctorTitle)
	doc := d.snap.Clinic.Doctor
	d.font("B", 20)
	d.setText(d.theme.Primary)
	d.pdf.SetXY(16, 42)
	d.pdf.CellFormat(178, 10, firstNonEmpty(doc.Name, d.snap.Clinic.SignerName), "", 0, "L", false, 0, "")
	d.font("", 12)
	d.setText(d.theme.Muted)
	d.pdf.SetXY(16, 54)
	d.pdf.CellFormat(178, 7, strings.TrimSpace(doc.Title+"  "+doc.Subtitle), "", 0, "L", false, 0, "")
	d.multi(16, 70, 178, 6.5, "", 12, d.theme.Ink, doc.Bio)
	for i, st := range doc.Stats {
		if i >= 3 {
			break
		}
		x := 16 + float64(i)*60
		d.rect(x, 210, 54, 36, d.theme.Soft)
		d.font("B", 16)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(x+6, 218)
		d.pdf.CellFormat(42, 8, st.Value, "", 0, "L", false, 0, "")
		d.font("", 9)
		d.setText(d.theme.Muted)
		d.pdf.SetXY(x+6, 228)
		d.pdf.MultiCell(42, 4.5, st.Label, "", "", false)
	}
	d.footer()
}

func (d *doc) pageSituation() {
	d.add()
	d.headerBar(d.copy.SituationTitle)
	d.font("", 11)
	d.setText(d.theme.Muted)
	d.pdf.SetXY(16, 38)
	d.pdf.CellFormat(178, 6, d.copy.SituationLead, "", 0, "L", false, 0, "")
	d.odontogram(16, 50, 178, d.snap.Situation.ChartTeeth, d.snap.Situation.ToothFindings)
	d.photoOrPlate("xray", d.images.Xray, 16, 100, 178, 150, d.copy.SituationTitle)
	d.footer()
}

func (d *doc) pageFindings() {
	d.add()
	d.headerBar(d.copy.FindingsTitle)
	d.legend(16, 38)
	d.findingRow(16, 56, upper[:8])
	d.findingRow(16, 92, upper[8:])
	d.findingRow(16, 128, lower[:8])
	d.findingRow(16, 164, lower[8:])
	diags := d.snap.Situation.GeneralDiagnoses
	if len(diags) == 0 {
		diags = d.snap.Diagnoses
	}
	if len(diags) > 0 {
		d.font("B", 12)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(16, 210)
		d.pdf.CellFormat(178, 7, strings.Join(diags, " · "), "", 0, "L", false, 0, "")
	}
	d.footer()
}

func (d *doc) legend(x, y float64) {
	items := []struct {
		c domain.RGB
		l string
	}{
		{domain.RGB{R: 180, G: 210, B: 170}, d.copy.Intact},
		{domain.RGB{R: 220, G: 80, B: 80}, d.copy.Missing},
		{domain.RGB{R: 90, G: 160, B: 220}, d.copy.Filled},
		{d.theme.Accent, d.copy.Crown},
	}
	for i, it := range items {
		d.rect(x+float64(i)*44, y, 6, 6, it.c)
		d.font("", 8)
		d.setText(d.theme.Ink)
		d.pdf.SetXY(x+float64(i)*44+8, y)
		d.pdf.CellFormat(34, 6, it.l, "", 0, "L", false, 0, "")
	}
}

func (d *doc) findingRow(x, y float64, teeth []string) {
	w := 178.0 / float64(len(teeth))
	for i, t := range teeth {
		f := d.snap.Finding(t)
		cx := x + float64(i)*w
		d.rect(cx, y, w-1.5, 28, d.theme.Soft)
		d.font("B", 9)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(cx, y+2)
		d.pdf.CellFormat(w-1.5, 6, t, "", 0, "C", false, 0, "")
		d.font("", 7)
		d.setText(d.theme.Ink)
		d.pdf.SetXY(cx, y+10)
		label := f.Status
		if label == "" {
			label = "—"
		}
		d.pdf.CellFormat(w-1.5, 5, label, "", 0, "C", false, 0, "")
		d.pdf.SetXY(cx, y+16)
		d.pdf.CellFormat(w-1.5, 8, f.Note, "", 0, "C", false, 0, "")
	}
}

func (d *doc) pageVisit(index int) {
	d.add()
	title := d.copy.VisitTitle
	var visit domain.Visit
	if index < len(d.snap.Visits) {
		visit = d.snap.Visits[index]
		if visit.Title != "" {
			title = visit.Title
		}
	} else if index == 0 && len(d.snap.Visits) == 0 {
		title = d.copy.VisitTitle + " 1"
	} else if index > 0 && len(d.snap.Visits) <= 1 {
		title = d.copy.Grand
	}
	if visit.PhaseLabel != "" {
		title = visit.PhaseLabel + " · " + title
	}
	d.headerBar(title)
	d.odontogram(16, 38, 178, visit.ChartTeeth, nil)

	items := d.snap.ItemsForVisit(visit.Title)
	if len(items) == 0 && index == 0 {
		items = d.snap.Items
	}
	if index > 0 && len(d.snap.Visits) <= 1 {
		items = d.snap.Items
	}
	d.itemTable(16, 82, items)

	if strings.TrimSpace(visit.HealingNote) != "" {
		d.font("B", 11)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(16, 248)
		d.pdf.CellFormat(178, 6, d.copy.Healing, "", 0, "L", false, 0, "")
		d.multi(16, 255, 178, 5, "", 10, d.theme.Ink, visit.HealingNote)
	}
	d.footer()
}

func (d *doc) itemTable(x, y float64, items []domain.Item) {
	d.rect(x, y, 178, 8, d.theme.Primary)
	d.font("B", 8)
	d.setText(domain.RGB{R: 250, G: 248, B: 242})
	d.pdf.SetXY(x+2, y+1.5)
	d.pdf.CellFormat(70, 5, d.copy.EducationTitle, "", 0, "L", false, 0, "")
	d.pdf.SetXY(x+72, y+1.5)
	d.pdf.CellFormat(28, 5, d.copy.Teeth, "", 0, "L", false, 0, "")
	d.pdf.SetXY(x+100, y+1.5)
	d.pdf.CellFormat(16, 5, d.copy.Qty, "", 0, "R", false, 0, "")
	d.pdf.SetXY(x+116, y+1.5)
	d.pdf.CellFormat(30, 5, d.copy.Unit, "", 0, "R", false, 0, "")
	d.pdf.SetXY(x+146, y+1.5)
	d.pdf.CellFormat(30, 5, d.copy.Line, "", 0, "R", false, 0, "")

	rowY := y + 10
	var sub int64
	for i, it := range items {
		if i >= 12 {
			break
		}
		if i%2 == 0 {
			d.rect(x, rowY-1, 178, 8, d.theme.Soft)
		}
		d.font("", 8)
		d.setText(d.theme.Ink)
		name := it.TreatmentName
		if it.IsPackage {
			name += " (" + d.copy.Package + ")"
		}
		d.pdf.SetXY(x+2, rowY)
		d.pdf.CellFormat(70, 6, name, "", 0, "L", false, 0, "")
		d.pdf.SetXY(x+72, rowY)
		d.pdf.CellFormat(28, 6, strings.Join(it.Teeth, ","), "", 0, "L", false, 0, "")
		d.pdf.SetXY(x+100, rowY)
		d.pdf.CellFormat(16, 6, fmt.Sprintf("%d", it.Quantity), "", 0, "R", false, 0, "")
		d.pdf.SetXY(x+116, rowY)
		d.pdf.CellFormat(30, 6, formatMoney(it.UnitPrice, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
		d.pdf.SetXY(x+146, rowY)
		d.pdf.CellFormat(30, 6, formatMoney(it.LineTotal, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
		sub += it.LineTotal
		rowY += 8
	}

	d.font("B", 10)
	d.setText(d.theme.Primary)
	d.pdf.SetXY(x+100, 230)
	d.pdf.CellFormat(46, 6, d.copy.Subtotal, "", 0, "L", false, 0, "")
	d.pdf.CellFormat(32, 6, formatMoney(sub, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
	if indexTotals := d.snap.Totals; indexTotals.Grand > 0 && len(items) > 0 {
		d.pdf.SetXY(x+100, 237)
		d.pdf.CellFormat(46, 6, d.copy.Grand, "", 0, "L", false, 0, "")
		d.pdf.CellFormat(32, 6, formatMoney(d.snap.Totals.Grand, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
	}
}

func (d *doc) pageEducation() {
	d.add()
	d.headerBar(d.copy.EducationTitle)
	d.multi(16, 48, 178, 7, "", 13, d.theme.Ink, d.copy.EducationBody)
	d.footer()
}

func (d *doc) pageAftercare() {
	d.add()
	d.headerBar(d.copy.AftercareTitle)
	d.multi(16, 48, 178, 7, "", 13, d.theme.Ink, d.copy.AftercareBody)
	d.footer()
}

func (d *doc) pageUltimate() {
	d.add()
	d.headerBar(d.copy.UltimateTitle)
	d.multi(16, 48, 178, 7, "", 13, d.theme.Ink, d.copy.UltimateBody)
	d.footer()
}

func (d *doc) pageVisual() {
	d.add()
	d.headerBar(d.copy.VisualTitle)
	d.multi(16, 48, 178, 7, "", 13, d.theme.Ink, d.copy.VisualBody)
	d.photoOrPlate("visual", d.images.Cover, 16, 100, 178, 150, d.copy.VisualTitle)
	d.footer()
}

func (d *doc) pageMaterial() {
	d.add()
	d.headerBar(d.copy.MaterialTitle)
	d.multi(16, 48, 178, 7, "", 13, d.theme.Ink, d.copy.MaterialBody)
	d.footer()
}

func (d *doc) pageResult() {
	d.add()
	d.headerBar(d.copy.ResultTitle)
	d.photoOrPlate("result", firstBytes(d.images.About, d.images.Cover), 16, 40, 178, 200, d.copy.ResultTitle)
	d.footer()
}

func (d *doc) pageLab() {
	d.add()
	d.headerBar(firstNonEmpty(d.snap.Clinic.Lab.Title, d.copy.LabTitle))
	d.multi(16, 42, 178, 6.5, "", 12, d.theme.Ink, d.snap.Clinic.Lab.Text)
	for i, st := range d.snap.Clinic.Lab.Stats {
		if i >= 3 {
			break
		}
		x := 16 + float64(i)*60
		d.rect(x, 200, 54, 40, d.theme.Soft)
		d.font("B", 16)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(x+6, 210)
		d.pdf.CellFormat(42, 8, st.Value, "", 0, "L", false, 0, "")
		d.font("", 9)
		d.setText(d.theme.Muted)
		d.pdf.SetXY(x+6, 220)
		d.pdf.MultiCell(42, 4.5, st.Label, "", "", false)
	}
	d.footer()
}

func (d *doc) pageAwards() {
	d.add()
	d.headerBar(d.copy.AwardsTitle)
	body := firstNonEmpty(d.snap.Clinic.Awards.Text, d.copy.AwardsDefault)
	d.multi(16, 42, 178, 6.5, "", 12, d.theme.Ink, body)
	d.photoOrPlate("award-inline", d.images.Award, 40, 120, 130, 130, d.copy.AwardsTitle)
	d.footer()
}

func (d *doc) pageHotel() {
	d.add()
	d.headerBar(firstNonEmpty(d.snap.Clinic.Hotel.Name, d.copy.HotelTitle))
	d.multi(16, 42, 178, 6.5, "", 12, d.theme.Ink, d.snap.Clinic.Hotel.Text)
	d.footer()
}

func (d *doc) pageContact() {
	d.add()
	d.headerBar(d.copy.ContactTitle)
	d.putImage("logo-contact", d.images.Logo, 16, 42, 28, 28)
	d.font("B", 18)
	d.setText(d.theme.Primary)
	d.pdf.SetXY(50, 46)
	d.pdf.CellFormat(144, 10, firstNonEmpty(d.snap.Clinic.Name, "Clinic"), "", 0, "L", false, 0, "")
	d.font("", 12)
	d.setText(d.theme.Muted)
	d.pdf.SetXY(50, 58)
	d.pdf.CellFormat(144, 7, d.snap.Clinic.Slogan, "", 0, "L", false, 0, "")

	rows := [][2]string{
		{d.copy.Address, d.snap.Clinic.Address},
		{d.copy.Phone, d.snap.Clinic.Phone},
		{d.copy.Email, d.snap.Clinic.Email},
		{d.copy.Website, d.snap.Clinic.Website},
		{d.copy.Hours, d.snap.Clinic.Hours},
	}
	y := 90.0
	for _, row := range rows {
		if strings.TrimSpace(row[1]) == "" {
			continue
		}
		d.font("B", 11)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(16, y)
		d.pdf.CellFormat(40, 7, row[0], "", 0, "L", false, 0, "")
		d.font("", 11)
		d.setText(d.theme.Ink)
		d.pdf.SetXY(56, y)
		d.pdf.MultiCell(138, 6, row[1], "", "", false)
		y += 16
	}
	if d.snap.Clinic.Legal != "" {
		d.multi(16, 250, 178, 5, "", 8, d.theme.Muted, d.snap.Clinic.Legal)
	}
	d.footer()
}

func firstBytes(a, b []byte) []byte {
	if len(a) > 0 {
		return a
	}
	return b
}
