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
	d.pagePhoto("lab-photo", firstBytes(d.images.Lab, d.images.About, d.images.Cover), firstNonEmpty(d.snap.Clinic.Lab.Title, d.copy.LabTitle))
	d.pageLab()
	d.pageAwards()
	d.pagePhoto("award-photo", d.images.Award, d.copy.AwardsTitle)
	d.pagePhoto("hotel-photo", firstBytes(d.images.Hotel, d.images.Cover, d.images.About), firstNonEmpty(d.snap.Clinic.Hotel.Name, d.copy.HotelTitle))
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
	text := strings.TrimSpace(d.snap.Clinic.AboutText)
	welcome := strings.TrimSpace(d.snap.Clinic.WelcomeBody)
	intro := strings.TrimSpace(d.snap.Clinic.Intro)
	// If about text is empty or mistakenly identical to welcome letter, use dedicated clinic about text
	if text == "" || text == welcome || text == intro || text == "—" || text == "-" {
		text = d.copy.AboutDefault
	}
	d.multi(16, 160, 178, 6.5, "", 12, d.theme.Ink, text)
	d.footer()
}

func (d *doc) pageHighlights() {
	d.add()
	d.headerBar(d.copy.HighlightsTitle)
	var validStats []domain.Stat
	for _, st := range d.snap.Clinic.Highlights {
		v := strings.TrimSpace(st.Value)
		if v != "" && v != "—" && v != "-" {
			validStats = append(validStats, st)
		}
	}
	stats := validStats
	if len(stats) == 0 {
		if strings.ToLower(strings.TrimSpace(d.snap.Locale)) == "en" {
			stats = []domain.Stat{
				{Value: "15+", Label: "Years of Experience"},
				{Value: "99.4%", Label: "Patient Satisfaction"},
				{Value: "12,000+", Label: "Successful Treatments"},
				{Value: "3D CAD/CAM", Label: "Digital Smile Design"},
				{Value: "Class A+", Label: "Sterilization Protocol"},
				{Value: "24/7", Label: "Post-Care Support"},
			}
		} else {
			stats = []domain.Stat{
				{Value: "15+", Label: "Yıllık Klinik Deneyim"},
				{Value: "%99.4", Label: "Hasta Memnuniyeti"},
				{Value: "12.000+", Label: "Başarılı Tedavi"},
				{Value: "3D CAD/CAM", Label: "Dijital Tasarım Altyapısı"},
				{Value: "A+ Sınıfı", Label: "Sterilizasyon Protokolü"},
				{Value: "24/7", Label: "Tedavi Sonrası Destek"},
			}
		}
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
	bio := strings.TrimSpace(doc.Bio)
	if bio == "" || bio == "—" || bio == "-" {
		if strings.ToLower(strings.TrimSpace(d.snap.Locale)) == "en" {
			bio = "Our lead clinician specializes in aesthetic dentistry, implantology, and advanced smile design, dedicated to providing gentle and personalized treatment."
		} else {
			bio = "Kliniğimizin sorumlu hekimi estetik diş hekimliği, implantoloji ve dijital gülüş tasarımı alanlarında uzmanlaşmış olup, hasta konforunu ve klinik mükemmeliyeti esas almaktadır."
		}
	}
	d.multi(16, 70, 178, 6.5, "", 12, d.theme.Ink, bio)
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
	d.findingRow(16, 54, upper[:8])
	d.findingRow(16, 90, upper[8:])
	d.findingRow(16, 126, lower[:8])
	d.findingRow(16, 162, lower[8:])
	diags := d.snap.Situation.GeneralDiagnoses
	if len(diags) == 0 {
		diags = d.snap.Diagnoses
	}
	if len(diags) > 0 {
		d.font("B", 11)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(16, 206)
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
		{d.theme.Accent, d.copy.Planned},
		{domain.RGB{R: 220, G: 80, B: 80}, d.copy.Missing},
		{domain.RGB{R: 90, G: 160, B: 220}, d.copy.Filled},
		{d.theme.Primary, d.copy.Crown},
	}
	for i, it := range items {
		d.rect(x+float64(i)*35, y, 5, 5, it.c)
		d.font("", 8)
		d.setText(d.theme.Ink)
		d.pdf.SetXY(x+float64(i)*35+7, y-0.5)
		d.pdf.CellFormat(27, 6, it.l, "", 0, "L", false, 0, "")
	}
}

func (d *doc) toothFindingAndStatus(tooth string) (label string, note string, statusColor domain.RGB) {
	f := d.snap.Finding(tooth)
	rawStatus := strings.ToLower(strings.TrimSpace(f.Status))
	note = strings.TrimSpace(f.Note)

	// Look up if this tooth is included in any planned item
	var plannedName string
	for _, it := range d.snap.Items {
		match := false
		if it.ToothNumber != nil && *it.ToothNumber == tooth {
			match = true
		} else {
			for _, tStr := range it.Teeth {
				for _, t := range strings.FieldsFunc(tStr, func(r rune) bool {
					return r == ',' || r == ';' || r == ' ' || r == '\t'
				}) {
					if strings.TrimSpace(t) == tooth {
						match = true
						break
					}
				}
				if match {
					break
				}
			}
		}
		if match {
			plannedName = it.TreatmentName
			break
		}
	}

	switch rawStatus {
	case "missing":
		return d.copy.Missing, note, domain.RGB{R: 220, G: 80, B: 80}
	case "filled":
		return d.copy.Filled, note, domain.RGB{R: 90, G: 160, B: 220}
	case "crown":
		return d.copy.Crown, note, d.theme.Primary
	case "intact":
		if plannedName != "" {
			if note == "" {
				note = plannedName
			}
			return d.copy.Planned, note, d.theme.Accent
		}
		return d.copy.Intact, note, domain.RGB{R: 180, G: 210, B: 170}
	default:
		// If status is empty, infer from items
		if plannedName != "" {
			if note == "" {
				note = plannedName
			}
			return d.copy.Planned, note, d.theme.Accent
		}
		return d.copy.Intact, note, domain.RGB{R: 180, G: 210, B: 170}
	}
}

func (d *doc) findingRow(x, y float64, teeth []string) {
	w := 178.0 / float64(len(teeth))
	for i, t := range teeth {
		label, note, statusColor := d.toothFindingAndStatus(t)
		cx := x + float64(i)*w
		d.rect(cx, y, w-1.5, 30, d.theme.Soft)
		d.rect(cx, y, w-1.5, 3, statusColor)
		d.font("B", 9)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(cx, y+4)
		d.pdf.CellFormat(w-1.5, 5, t, "", 0, "C", false, 0, "")
		d.font("", 7.5)
		d.setText(d.theme.Ink)
		d.pdf.SetXY(cx, y+10.5)
		d.pdf.CellFormat(w-1.5, 5, label, "", 0, "C", false, 0, "")
		d.font("", 6.5)
		d.setText(d.theme.Muted)
		d.pdf.SetXY(cx, y+16.5)
		d.pdf.MultiCell(w-1.5, 3.5, note, "", "C", false)
	}
}

func (d *doc) pageVisit(index int) {
	d.add()
	title := d.copy.VisitTitle
	var visit domain.Visit
	isFinal := false

	if index == 0 {
		if len(d.snap.Visits) > 0 {
			visit = d.snap.Visits[0]
			if visit.Title != "" {
				title = visit.Title
			}
		} else {
			title = d.copy.VisitTitle + " 1"
		}
		if len(d.snap.Visits) <= 1 {
			isFinal = true
		}
	} else {
		// Index 1 (Final visit booklet page)
		isFinal = true
		if len(d.snap.Visits) > 1 {
			visit = d.snap.Visits[1]
			if visit.Title != "" {
				title = visit.Title
			}
			if len(d.snap.Visits) > 2 {
				title += " & Diğer Aşamalar"
			}
		} else {
			title = d.copy.VisitTitle + " 2"
		}
	}

	if visit.PhaseLabel != "" {
		title = visit.PhaseLabel + " · " + title
	}
	d.headerBar(title)
	d.odontogram(16, 38, 178, visit.ChartTeeth, nil)

	var items []domain.Item
	if index == 0 {
		items = d.snap.ItemsForVisit(visit.Title)
		if len(items) == 0 {
			items = d.snap.Items
		}
	} else {
		// On the second visit page, include Visit 1 items AND any subsequent/unassigned items
		items = d.itemsForSecondVisit()
	}

	d.itemTable(16, 82, items, index, isFinal)

	if strings.TrimSpace(visit.HealingNote) != "" {
		d.font("B", 10)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(16, 252)
		d.pdf.CellFormat(178, 5, d.copy.Healing, "", 0, "L", false, 0, "")
		d.multi(16, 258, 178, 4.5, "", 9, d.theme.Ink, visit.HealingNote)
	}
	d.footer()
}

func (d *doc) itemsForSecondVisit() []domain.Item {
	if len(d.snap.Visits) == 0 {
		return d.snap.Items
	}
	v0Title := d.snap.Visits[0].Title
	var out []domain.Item
	for _, it := range d.snap.Items {
		// Exclude items that were already rendered on Visit 0
		if it.VisitTitle != v0Title {
			out = append(out, it)
		}
	}
	if len(out) == 0 && len(d.snap.Visits) > 1 {
		out = d.snap.ItemsForVisit(d.snap.Visits[1].Title)
	}
	return out
}

func (d *doc) itemTable(x, y float64, items []domain.Item, pageIndex int, isFinal bool) {
	d.rect(x, y, 178, 7.5, d.theme.Primary)
	d.font("B", 8)
	d.setText(domain.RGB{R: 250, G: 248, B: 242})
	d.pdf.SetXY(x+2, y+1.5)
	d.pdf.CellFormat(70, 4.5, d.copy.EducationTitle, "", 0, "L", false, 0, "")
	d.pdf.SetXY(x+72, y+1.5)
	d.pdf.CellFormat(28, 4.5, d.copy.Teeth, "", 0, "L", false, 0, "")
	d.pdf.SetXY(x+100, y+1.5)
	d.pdf.CellFormat(16, 4.5, d.copy.Qty, "", 0, "R", false, 0, "")
	d.pdf.SetXY(x+116, y+1.5)
	d.pdf.CellFormat(30, 4.5, d.copy.Unit, "", 0, "R", false, 0, "")
	d.pdf.SetXY(x+146, y+1.5)
	d.pdf.CellFormat(30, 4.5, d.copy.Line, "", 0, "R", false, 0, "")

	rowY := y + 8.5
	var sub int64
	for i, it := range items {
		if i >= 14 {
			break
		}
		if i%2 == 0 {
			d.rect(x, rowY-0.5, 178, 7, d.theme.Soft)
		}
		d.font("", 8)
		d.setText(d.theme.Ink)
		name := it.TreatmentName
		if it.IsPackage {
			name += " (" + d.copy.Package + ")"
		}
		d.pdf.SetXY(x+2, rowY)
		d.pdf.CellFormat(70, 6, name, "", 0, "L", false, 0, "")

		// Deduplicate teeth numbers cleanly
		var teethList []string
		seenT := map[string]struct{}{}
		for _, tStr := range it.Teeth {
			for _, t := range strings.FieldsFunc(tStr, func(r rune) bool {
				return r == ',' || r == ';' || r == ' ' || r == '\t'
			}) {
				t = strings.TrimSpace(t)
				if t == "" {
					continue
				}
				if _, ok := seenT[t]; ok {
					continue
				}
				seenT[t] = struct{}{}
				teethList = append(teethList, t)
			}
		}
		if len(teethList) == 0 && it.ToothNumber != nil {
			for _, t := range strings.FieldsFunc(*it.ToothNumber, func(r rune) bool {
				return r == ',' || r == ';' || r == ' ' || r == '\t'
			}) {
				t = strings.TrimSpace(t)
				if t == "" {
					continue
				}
				if _, ok := seenT[t]; ok {
					continue
				}
				seenT[t] = struct{}{}
				teethList = append(teethList, t)
			}
		}
		teethText := strings.Join(teethList, ", ")
		d.pdf.SetXY(x+72, rowY)
		d.pdf.CellFormat(28, 6, teethText, "", 0, "L", false, 0, "")

		d.pdf.SetXY(x+100, rowY)
		d.pdf.CellFormat(16, 6, fmt.Sprintf("%d", it.Quantity), "", 0, "R", false, 0, "")
		d.pdf.SetXY(x+116, rowY)
		d.pdf.CellFormat(30, 6, formatMoney(it.UnitPrice, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
		d.pdf.SetXY(x+146, rowY)
		d.pdf.CellFormat(30, 6, formatMoney(it.LineTotal, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
		sub += it.LineTotal
		rowY += 7
	}

	totalsY := 216.0
	if !isFinal {
		// On non-final visit (e.g. Page 10), ONLY print this session's subtotal
		d.font("B", 9.5)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(x+80, totalsY+12)
		label := fmt.Sprintf("%d. %s %s", pageIndex+1, d.copy.VisitTitle, d.copy.Subtotal)
		d.pdf.CellFormat(66, 6, label, "", 0, "L", false, 0, "")
		d.pdf.CellFormat(32, 6, formatMoney(sub, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
	} else {
		// On final visit page (Page 12), show complete transparent arithmetic breakdown
		d.font("", 8.5)
		d.setText(d.theme.Ink)

		prevSub := d.snap.Totals.Subtotal - sub
		if prevSub > 0 {
			d.pdf.SetXY(x+80, totalsY)
			d.pdf.CellFormat(66, 5, "1. "+d.copy.VisitTitle+" "+d.copy.Subtotal, "", 0, "L", false, 0, "")
			d.pdf.CellFormat(32, 5, formatMoney(prevSub, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
			totalsY += 5.5
		}

		thisLabel := d.copy.Subtotal
		if prevSub > 0 {
			thisLabel = "2. " + d.copy.VisitTitle + " (ve Diğer) " + d.copy.Subtotal
		}
		d.pdf.SetXY(x+80, totalsY)
		d.pdf.CellFormat(66, 5, thisLabel, "", 0, "L", false, 0, "")
		d.pdf.CellFormat(32, 5, formatMoney(sub, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
		totalsY += 5.5

		if d.snap.Totals.Discount > 0 {
			d.pdf.SetXY(x+80, totalsY)
			d.pdf.CellFormat(66, 5, d.copy.Discount, "", 0, "L", false, 0, "")
			d.pdf.CellFormat(32, 5, "- "+formatMoney(d.snap.Totals.Discount, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
			totalsY += 5.5
		}

		d.font("B", 10.5)
		d.setText(d.theme.Primary)
		d.rect(x+80, totalsY, 98, 7.5, d.theme.Soft)
		d.pdf.SetXY(x+82, totalsY+1)
		d.pdf.CellFormat(64, 5.5, d.copy.Grand, "", 0, "L", false, 0, "")
		d.pdf.CellFormat(30, 5.5, formatMoney(d.snap.Totals.Grand, d.snap.Currency, d.snap.Locale), "", 0, "R", false, 0, "")
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
	if len(d.images.Smile) > 0 {
		d.photoOrPlate("result", d.images.Smile, 16, 40, 178, 200, d.copy.ResultTitle)
	} else {
		// Custom aesthetic smile design plate (never repeat the clinic reception photo!)
		d.rect(16, 40, 178, 230, d.theme.Soft)
		d.rect(16, 40, 4, 230, d.theme.Accent)

		d.font("B", 16)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(26, 52)
		if strings.ToLower(strings.TrimSpace(d.snap.Locale)) == "en" {
			d.pdf.CellFormat(160, 8, "Personalized Digital Smile Design", "", 0, "L", false, 0, "")
			d.font("", 11)
			d.setText(d.theme.Muted)
			d.pdf.SetXY(26, 62)
			d.pdf.CellFormat(160, 6, "Aesthetic harmony, facial balance, and bio-compatible function", "", 0, "L", false, 0, "")
			d.multi(26, 76, 158, 6, "", 11, d.theme.Ink,
				"Your smile transformation is planned in accordance with the golden ratio, your lip dynamics, and facial symmetry. The targeted morphology, natural translucency, and precise alignment are designed to achieve a healthy, confident, and timeless smile.")
		} else {
			d.pdf.CellFormat(160, 8, "Kişiye Özel Dijital Gülüş Tasarımı", "", 0, "L", false, 0, "")
			d.font("", 11)
			d.setText(d.theme.Muted)
			d.pdf.SetXY(26, 62)
			d.pdf.CellFormat(160, 6, "Yüz hatları ile uyumlu, estetik ve biyouyumlu hedef sonuç", "", 0, "L", false, 0, "")
			d.multi(26, 76, 158, 6, "", 11, d.theme.Ink,
				"Gülüş tasarımınız; yüz oranlarınız, dudak hareketleriniz ve altın oran prensipleri temel alınarak planlanmıştır. Tedavi sonunda hedeflenen diş formu, doğal ışık geçirgenliği ve çene kapanış dengesi bu kişiselleştirilmiş simülasyon doğrultusunda hayata geçirilecektir.")
		}

		cards := []struct {
			t string
			d string
		}{
			{"Altın Oran & Simetri", "Gülüş hattı, dudak kavisi ve yüz morfolojisi ile kusursuz estetik oran."},
			{"Doğal Doku & Renk", "Doğal diş minesinin ışık yansımasını taklit eden estetik seramik restorasyonlar."},
			{"Fonksiyonel Uyum", "Konforlu çiğneme, dengeli çene kapanışı ve uzun ömürlü stabilite."},
		}
		if strings.ToLower(strings.TrimSpace(d.snap.Locale)) == "en" {
			cards = []struct {
				t string
				d string
			}{
				{"Golden Proportion", "Aesthetic alignment tuned to your lip curve and facial proportions."},
				{"Vital Texture & Shade", "High-translucency ceramics mimicking natural enamel reflection."},
				{"Functional Occlusion", "Balanced bite dynamics for maximum comfort and long-term durability."},
			}
		}

		for i, c := range cards {
			y := 125.0 + float64(i)*38.0
			d.rect(26, y, 158, 30, domain.RGB{R: 255, G: 255, B: 255})
			d.rect(26, y, 2.5, 30, d.theme.Accent)
			d.font("B", 11)
			d.setText(d.theme.Primary)
			d.pdf.SetXY(34, y+4)
			d.pdf.CellFormat(146, 6, c.t, "", 0, "L", false, 0, "")
			d.font("", 9.5)
			d.setText(d.theme.Ink)
			d.pdf.SetXY(34, y+12)
			d.pdf.MultiCell(146, 5, c.d, "", "", false)
		}

		d.font("B", 10)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(26, 248)
		footerMsg := "Kliniğimiz, planlanan tedavi aşamalarında bu hedeflenen gülüş mimarisini eksiksiz uygulamayı taahhüt eder."
		if strings.ToLower(strings.TrimSpace(d.snap.Locale)) == "en" {
			footerMsg = "Our clinical team is committed to realizing this targeted smile design across each sequenced visit."
		}
		d.pdf.CellFormat(158, 6, footerMsg, "", 0, "C", false, 0, "")
	}
	d.footer()
}

func (d *doc) pageLab() {
	d.add()
	d.headerBar(firstNonEmpty(d.snap.Clinic.Lab.Title, d.copy.LabTitle))
	text := strings.TrimSpace(d.snap.Clinic.Lab.Text)
	if text == "" || text == "—" || text == "-" {
		text = d.copy.LabDefault
	}
	d.multi(16, 42, 178, 6.5, "", 12, d.theme.Ink, text)
	var validLabStats []domain.Stat
	for _, st := range d.snap.Clinic.Lab.Stats {
		v := strings.TrimSpace(st.Value)
		if v != "" && v != "—" && v != "-" {
			validLabStats = append(validLabStats, st)
		}
	}
	stats := validLabStats
	if len(stats) == 0 {
		if strings.ToLower(strings.TrimSpace(d.snap.Locale)) == "en" {
			stats = []domain.Stat{
				{Value: "Micron", Label: "CAD/CAM Precision"},
				{Value: "Class A+", Label: "Biocompatible Zirconia"},
				{Value: "100%", Label: "Digital Workflow"},
			}
		} else {
			stats = []domain.Stat{
				{Value: "Mikron", Label: "CAD/CAM Hassasiyeti"},
				{Value: "A+ Kalite", Label: "Biyouyumlu Zirkonyum"},
				{Value: "%100", Label: "Dijital İş Akışı"},
			}
		}
	}
	for i, st := range stats {
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
	body := strings.TrimSpace(d.snap.Clinic.Awards.Text)
	if body == "" || body == "—" || body == "-" || strings.Contains(body, "Treatment Plus standart markasıdır") || strings.Contains(body, "standard mark") {
		body = d.copy.AwardsDefault
	}
	d.multi(16, 42, 178, 6.5, "", 12, d.theme.Ink, body)
	d.photoOrPlate("award-inline", d.images.Award, 40, 120, 130, 130, d.copy.AwardsTitle)
	d.footer()
}

func (d *doc) pageHotel() {
	d.add()
	d.headerBar(firstNonEmpty(d.snap.Clinic.Hotel.Name, d.copy.HotelTitle))
	text := strings.TrimSpace(d.snap.Clinic.Hotel.Text)
	if text == "" || text == "—" || text == "-" {
		text = d.copy.HotelDefault
	}
	d.multi(16, 42, 178, 6.5, "", 12, d.theme.Ink, text)

	var stats []domain.Stat
	if strings.ToLower(strings.TrimSpace(d.snap.Locale)) == "en" {
		stats = []domain.Stat{
			{Value: "VIP", Label: "Airport & Clinic Transfers"},
			{Value: "4 & 5 Star", Label: "Partner Luxury Hotels"},
			{Value: "Concierge", Label: "Personal Patient Guide"},
		}
	} else {
		stats = []domain.Stat{
			{Value: "VIP", Label: "Havalimanı & Klinik Transferi"},
			{Value: "4 & 5 Yıldız", Label: "Seçkin Anlaşmalı Oteller"},
			{Value: "Rehber", Label: "Kişisel Tedavi Rehberliği"},
		}
	}
	for i, st := range stats {
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

func (d *doc) pageContact() {
	d.add()
	d.headerBar(d.copy.ContactTitle)
	d.putImage("logo-contact", d.images.Logo, 16, 42, 28, 28)
	d.font("B", 18)
	d.setText(d.theme.Primary)
	d.pdf.SetXY(50, 44)
	d.pdf.CellFormat(144, 8, firstNonEmpty(d.snap.Clinic.Name, "Clinic"), "", 0, "L", false, 0, "")
	d.font("", 11)
	d.setText(d.theme.Muted)
	d.pdf.SetXY(50, 54)
	d.pdf.CellFormat(144, 6, d.snap.Clinic.Slogan, "", 0, "L", false, 0, "")

	rows := [][2]string{
		{d.copy.Address, d.snap.Clinic.Address},
		{d.copy.Phone, d.snap.Clinic.Phone},
		{d.copy.Email, d.snap.Clinic.Email},
		{d.copy.Website, d.snap.Clinic.Website},
		{d.copy.Hours, d.snap.Clinic.Hours},
	}
	y := 76.0
	for _, row := range rows {
		if strings.TrimSpace(row[1]) == "" {
			continue
		}
		d.font("B", 10)
		d.setText(d.theme.Primary)
		d.pdf.SetXY(16, y)
		d.pdf.CellFormat(36, 6, row[0], "", 0, "L", false, 0, "")
		d.font("", 10)
		d.setText(d.theme.Ink)
		d.pdf.SetXY(54, y)
		d.pdf.MultiCell(140, 5, row[1], "", "", false)
		y += 11
	}

	legal := strings.TrimSpace(d.snap.Clinic.Legal)
	if legal != "" {
		clinicName := firstNonEmpty(d.snap.Clinic.Name, d.snap.Clinic.SignerName, d.snap.Clinic.Doctor.Name, "Kliniğimiz")
		legal = strings.ReplaceAll(legal, "[Klinik/Hekim Adı]", clinicName)
		legal = strings.ReplaceAll(legal, "[Klinik/Hekim Adı", clinicName)
		legal = strings.ReplaceAll(legal, "[Klinik Adı]", clinicName)
		legal = strings.ReplaceAll(legal, "[Hekim Adı]", clinicName)

		d.rect(16, y+4, 178, 0.5, d.theme.Soft)
		d.multi(16, y+8, 178, 3.8, "", 7.5, d.theme.Muted, legal)
	}
	d.footer()
}

func firstBytes(a, b, c []byte) []byte {
	if len(a) > 0 {
		return a
	}
	if len(b) > 0 {
		return b
	}
	return c
}
