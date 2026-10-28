package http

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
	"github.com/treatment-plus/treatment-plus-be/internal/share/app"
	"github.com/treatment-plus/treatment-plus-be/internal/share/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Handler struct {
	svc  *app.Service
	auth *authn.JWT
}

func NewHandler(svc *app.Service, auth *authn.JWT) *Handler {
	return &Handler{svc: svc, auth: auth}
}

func (h *Handler) Register(r chi.Router) {
	// Public unauthenticated routes
	r.Get("/public/plans/{token}", h.getPublicPlan)
	r.Post("/public/plans/{token}/events", h.recordPublicEvent)
	r.Post("/public/plans/{token}/accept", h.acceptPublicPlan)
	r.Get("/public/plans/{token}/viz", h.getPublicViz)
	r.Get("/public/plans/{token}/pdf", h.downloadPublicPDF)
	r.Get("/p/{token}", h.renderPublicHTML)

	// Authenticated clinic routes
	r.Group(func(r chi.Router) {
		if h.auth != nil {
			r.Use(h.auth.Require)
		}
		r.Post("/plans/{id}/share", h.createShareLink)
		r.Get("/plans/{id}/share", h.listShareLinks)
		r.Delete("/plans/{id}/share/{linkID}", h.revokeShareLink)
		r.Get("/plans/{id}/analytics", h.getPlanAnalytics)
	})
}

type createShareRequest struct {
	TTLHours int `json:"ttl_hours,omitempty"`
}

func (h *Handler) createShareLink(w http.ResponseWriter, r *http.Request) {
	clinicID, err := h.getClinicID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed plan id", errx.ErrInvalid))
		return
	}

	var req createShareRequest
	_ = httpx.DecodeJSON(r, &req)

	var ttl time.Duration
	if req.TTLHours > 0 {
		ttl = time.Duration(req.TTLHours) * time.Hour
	}

	res, err := h.svc.CreateShareLink(r.Context(), clinicID, planID, ttl)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, res)
}

func (h *Handler) listShareLinks(w http.ResponseWriter, r *http.Request) {
	clinicID, err := h.getClinicID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed plan id", errx.ErrInvalid))
		return
	}

	links, err := h.svc.ListShareLinks(r.Context(), clinicID, planID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"links": links})
}

func (h *Handler) revokeShareLink(w http.ResponseWriter, r *http.Request) {
	clinicID, err := h.getClinicID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	linkID, err := id.Parse(chi.URLParam(r, "linkID"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed link id", errx.ErrInvalid))
		return
	}

	if err := h.svc.RevokeShareLink(r.Context(), clinicID, linkID); err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

func (h *Handler) getPlanAnalytics(w http.ResponseWriter, r *http.Request) {
	clinicID, err := h.getClinicID(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	planID, err := id.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, errx.New("invalid_id", "malformed plan id", errx.ErrInvalid))
		return
	}

	analytics, err := h.svc.GetAnalytics(r.Context(), clinicID, planID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, analytics)
}

func (h *Handler) getPublicPlan(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	planDTO, err := h.svc.GetUltimatePlan(r.Context(), token)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, planDTO)
}

type recordEventRequest struct {
	Kind     string         `json:"kind"`
	Metadata map[string]any `json:"metadata"`
}

func (h *Handler) recordPublicEvent(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	var req recordEventRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Metadata == nil {
		req.Metadata = make(map[string]any)
	}
	req.Metadata["ip"] = clientIP(r)
	req.Metadata["user_agent"] = r.UserAgent()

	err := h.svc.RecordEvent(r.Context(), token, domain.ParseEventKind(req.Kind), req.Metadata)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type acceptPlanRequest struct {
	Signature string `json:"signature"`
}

func (h *Handler) acceptPublicPlan(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	var req acceptPlanRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}

	ip := clientIP(r)
	if err := h.svc.AcceptPlan(r.Context(), token, req.Signature, ip); err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status":      "accepted",
		"accepted_at": time.Now().UTC(),
	})
}

func (h *Handler) getPublicViz(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	playlist, err := h.svc.GetVizForToken(r.Context(), token)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, playlist)
}

func (h *Handler) downloadPublicPDF(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	pdfBytes, err := h.svc.GetPDFForToken(r.Context(), token)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"treatment-plan.pdf\"")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdfBytes)
}

func (h *Handler) renderPublicHTML(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	planDTO, err := h.svc.GetUltimatePlan(r.Context(), token)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	// Render interactive mobile-first HTML view of the Ultimate Treatment Plan
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_ = renderUltimateHTML(w, planDTO)
}

func (h *Handler) getClinicID(r *http.Request) (id.ID, error) {
	raw := ctxkey.ClinicID(r.Context())
	if raw == "" {
		return id.ID{}, errx.ErrUnauthenticated
	}
	parsed, err := id.Parse(raw)
	if err != nil {
		return id.ID{}, errx.ErrUnauthenticated
	}
	return parsed, nil
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		return strings.TrimSpace(rip)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func renderUltimateHTML(w http.ResponseWriter, p app.UltimatePlanDTO) error {
	dataJSON, _ := json.Marshal(p)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="tr" class="scroll-smooth">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>%s | %s - Tedavi Planı</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700;800&family=Playfair+Display:wght@500;700&display=swap" rel="stylesheet">
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/lucide@latest"></script>
    <style>
        body { font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif; background-color: #0b0f19; color: #f3f4f6; }
        .serif { font-family: 'Playfair Display', serif; }
        .glass-card { background: rgba(255, 255, 255, 0.05); backdrop-filter: blur(16px); border: 1px solid rgba(255, 255, 255, 0.1); }
        .gradient-brand { background: linear-gradient(135deg, #06b6d4 0%%, #3b82f6 50%%, #8b5cf6 100%%); }
        .gradient-text { background: linear-gradient(135deg, #38bdf8, #818cf8); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
    </style>
</head>
<body class="min-h-screen pb-28 selection:bg-cyan-500 selection:text-white">
    <!-- Header / Branding -->
    <header class="sticky top-0 z-40 bg-[#0b0f19]/80 backdrop-blur-md border-b border-white/10 px-4 py-3.5 flex items-center justify-between">
        <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-xl gradient-brand flex items-center justify-center font-bold text-white shadow-lg shadow-cyan-500/20">
                %s
            </div>
            <div>
                <h1 class="text-sm font-bold text-white tracking-tight">%s</h1>
                <p class="text-[11px] text-gray-400 font-medium">%s</p>
            </div>
        </div>
        <div class="flex items-center gap-2">
            %s
            <a href="tel:%s" class="p-2 rounded-full glass-card hover:bg-white/10 text-gray-300">
                <i data-lucide="phone" class="w-4 h-4"></i>
            </a>
        </div>
    </header>

    <main class="max-w-xl mx-auto px-4 pt-6 space-y-6">
        <!-- Hero Card -->
        <section class="glass-card rounded-[2rem] p-6 relative overflow-hidden text-center shadow-2xl">
            <div class="absolute -top-24 -right-24 w-48 h-48 bg-cyan-500/20 rounded-full blur-3xl pointer-events-none"></div>
            <div class="absolute -bottom-24 -left-24 w-48 h-48 bg-purple-500/20 rounded-full blur-3xl pointer-events-none"></div>

            <span class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-semibold bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 mb-4">
                <i data-lucide="sparkles" class="w-3.5 h-3.5"></i> Kişiselleştirilmiş Tedavi Deneyimi
            </span>
            <h2 class="text-2xl sm:text-3xl font-extrabold text-white tracking-tight mb-2 serif">%s</h2>
            <p class="text-sm text-gray-300 font-light mb-4">Hazırlayan: <span class="font-medium text-white">%s</span></p>

            <div class="grid grid-cols-2 gap-3 pt-3 border-t border-white/10 text-left">
                <div class="p-3 rounded-xl bg-white/5">
                    <p class="text-[11px] uppercase tracking-wider text-gray-400 font-semibold">Tarih</p>
                    <p class="text-sm font-bold text-white mt-0.5">%s</p>
                </div>
                <div class="p-3 rounded-xl bg-white/5">
                    <p class="text-[11px] uppercase tracking-wider text-gray-400 font-semibold">Toplam Tutar</p>
                    <p class="text-sm font-bold text-cyan-400 mt-0.5">%s</p>
                </div>
            </div>
        </section>

        <!-- Welcome Message -->
        %s

        <!-- Treatment Plan Timeline -->
        <section class="space-y-4">
            <div class="flex items-center justify-between">
                <div>
                    <h3 class="text-lg font-bold text-white tracking-tight">Tedavi Aşamaları</h3>
                    <p class="text-xs text-gray-400">Adım adım planlanan seanslar ve işlemler</p>
                </div>
                <span class="text-xs px-2.5 py-1 rounded-lg bg-white/5 text-gray-300 border border-white/10">%d Seans</span>
            </div>

            <div class="space-y-3">
                %s
            </div>
        </section>

        <!-- Doctor & Team -->
        <section class="glass-card rounded-[2rem] p-6 space-y-4">
            <div class="flex items-center gap-4">
                <div class="w-14 h-14 rounded-2xl gradient-brand flex items-center justify-center text-white text-xl font-bold shadow-lg">
                    <i data-lucide="stethoscope" class="w-7 h-7"></i>
                </div>
                <div>
                    <p class="text-xs font-semibold text-cyan-400 uppercase tracking-wider">Hekiminiz</p>
                    <h4 class="text-base font-bold text-white">%s</h4>
                    <p class="text-xs text-gray-400">%s</p>
                </div>
            </div>
            %s
        </section>

        <!-- Action / Acceptance -->
        <section class="glass-card rounded-[2rem] p-6 text-center space-y-4 border border-cyan-500/30">
            <h3 class="text-xl font-bold text-white">Yeni Gülüşünüz İçin Hazır mısınız?</h3>
            <p class="text-xs text-gray-300 leading-relaxed">
                Tedavi planınızı hemen dijital olarak onaylayabilir veya hekimimizle WhatsApp üzerinden iletişime geçebilirsiniz.
            </p>
            <div class="space-y-2 pt-2">
                <button onclick="acceptPlan()" id="acceptBtn" class="w-full py-3.5 px-6 rounded-xl gradient-brand text-white font-bold text-sm shadow-lg shadow-cyan-500/25 hover:opacity-90 active:scale-[0.98] transition">
                    %s
                </button>
                <a href="/public/plans/%s/pdf" download class="inline-flex items-center justify-center gap-2 w-full py-2.5 px-4 rounded-xl bg-white/5 hover:bg-white/10 text-xs font-medium text-gray-300 transition">
                    <i data-lucide="file-down" class="w-4 h-4 text-cyan-400"></i> Klasik PDF Versiyonunu İndir
                </a>
            </div>
        </section>
    </main>

    <!-- Floating Sticky Bottom Bar -->
    <div class="fixed bottom-0 inset-x-0 z-40 bg-[#0b0f19]/90 backdrop-blur-xl border-t border-white/10 px-4 py-3 flex items-center justify-between max-w-xl mx-auto">
        <div>
            <p class="text-[10px] uppercase tracking-wider text-gray-400 font-bold">Toplam Yatırım</p>
            <p class="text-lg font-black text-white">%s</p>
        </div>
        <div class="flex items-center gap-2">
            %s
        </div>
    </div>

    <script>
        const PLAN_DATA = %s;
        lucide.createIcons();

        // Send telemetry: opened
        fetch('/public/plans/' + PLAN_DATA.token + '/events', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ kind: 'opened', metadata: { width: window.innerWidth } })
        }).catch(() => {});

        async function acceptPlan() {
            if (PLAN_DATA.accepted) {
                alert('Bu plan zaten daha önce onaylanmıştır.');
                return;
            }
            const name = prompt('Lütfen planı onaylamak için Adınızı ve Soyadınızı giriniz:', '%s');
            if (!name) return;
            try {
                const res = await fetch('/public/plans/' + PLAN_DATA.token + '/accept', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ signature: name })
                });
                if (res.ok) {
                    alert('Tedavi planınız başarıyla onaylandı! Kliniğimiz en kısa sürede sizinle iletişime geçecektir.');
                    location.reload();
                } else {
                    alert('Bir hata oluştu, lütfen tekrar deneyiniz.');
                }
            } catch (e) {
                alert('Bağlantı hatası.');
            }
        }
    </script>
</body>
</html>`,
		escape(p.Plan.PatientName), escape(p.Clinic.Name),
		initials(p.Clinic.Name), escape(p.Clinic.Name), escape(p.Clinic.Slogan),
		whatsAppHeader(p.Clinic.CTA.WhatsAppNumber), p.Clinic.Phone,
		escape(p.Plan.PatientName), escape(p.Clinic.Doctor.Name),
		p.Plan.CreatedAt.Format("02.01.2006"), formatMoney(p.Plan.TotalsGrand, p.Plan.Currency),
		renderWelcome(p.Plan.WelcomeLetter, p.Clinic.WelcomeBody),
		len(p.Plan.Visits),
		renderVisitsHTML(p.Plan.Visits, p.Plan.Currency),
		escape(p.Clinic.Doctor.Name), escape(p.Clinic.Doctor.Title),
		renderDoctorBio(p.Clinic.Doctor.Bio),
		acceptanceBtnText(p.Accepted), p.Token,
		formatMoney(p.Plan.TotalsGrand, p.Plan.Currency),
		whatsAppFooter(p.Clinic.CTA.WhatsAppNumber),
		string(dataJSON),
		escape(p.Plan.PatientName),
	)
	_, err := w.Write([]byte(html))
	return err
}

func escape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func initials(name string) string {
	parts := strings.Fields(name)
	if len(parts) >= 2 {
		return string([]rune(parts[0])[:1]) + string([]rune(parts[1])[:1])
	}
	if len(parts) == 1 && len(parts[0]) > 0 {
		return string([]rune(parts[0])[:1])
	}
	return "TP"
}

func formatMoney(amount int64, curr string) string {
	major := float64(amount) / 100.0
	symbol := "₺"
	if curr == "EUR" {
		symbol = "€"
	} else if curr == "USD" {
		symbol = "$"
	}
	return fmt.Sprintf("%.2f %s", major, symbol)
}

func renderWelcome(planWelcome, clinicWelcome string) string {
	text := planWelcome
	if text == "" {
		text = clinicWelcome
	}
	if text == "" {
		return ""
	}
	return fmt.Sprintf(`
        <section class="glass-card rounded-[2rem] p-6 space-y-2 border-l-4 border-cyan-400">
            <h3 class="text-sm font-bold text-white uppercase tracking-wider text-cyan-400">Hekiminizden Mesaj</h3>
            <p class="text-sm text-gray-300 font-light leading-relaxed">%s</p>
        </section>`, escape(text))
}

func renderDoctorBio(bio string) string {
	if bio == "" {
		return ""
	}
	return fmt.Sprintf(`<p class="text-xs text-gray-300 font-light leading-relaxed">%s</p>`, escape(bio))
}

func renderVisitsHTML(visits []app.SharedVisit, curr string) string {
	if len(visits) == 0 {
		return `<div class="p-6 rounded-2xl glass-card text-center text-sm text-gray-400">Henüz tedavi kalemi eklenmemiş.</div>`
	}
	var sb strings.Builder
	for i, v := range visits {
		phase := v.PhaseLabel
		if phase == "" {
			phase = fmt.Sprintf("%d. Seans", i+1)
		}
		sb.WriteString(fmt.Sprintf(`
        <div class="glass-card rounded-2xl p-4 space-y-3">
            <div class="flex items-center justify-between pb-2 border-b border-white/5">
                <div>
                    <span class="text-[11px] font-bold uppercase tracking-wider text-cyan-400">%s</span>
                    <h4 class="text-sm font-bold text-white">%s</h4>
                </div>
            </div>`, escape(phase), escape(v.Title)))

		if v.HealingNote != "" {
			sb.WriteString(fmt.Sprintf(`<p class="text-xs text-amber-300/80 bg-amber-500/10 px-3 py-1.5 rounded-lg border border-amber-500/20">ℹ️ %s</p>`, escape(v.HealingNote)))
		}

		if len(v.Items) > 0 {
			sb.WriteString(`<div class="space-y-2 pt-1">`)
			for _, item := range v.Items {
				teethBadge := ""
				if len(item.Teeth) > 0 {
					teethBadge = fmt.Sprintf(`<span class="text-[10px] px-2 py-0.5 rounded bg-white/10 text-gray-300 font-mono">Diş %s</span>`, strings.Join(item.Teeth, ", "))
				}
				sb.WriteString(fmt.Sprintf(`
                <div class="flex items-center justify-between p-2.5 rounded-xl bg-white/5 text-xs">
                    <div class="space-y-0.5">
                        <div class="flex items-center gap-2">
                            <span class="font-semibold text-white">%s</span>
                            %s
                        </div>
                        <p class="text-[11px] text-gray-400">%d Adet</p>
                    </div>
                    <span class="font-bold text-cyan-400">%s</span>
                </div>`, escape(item.TreatmentName), teethBadge, item.Quantity, formatMoney(item.UnitPrice*int64(item.Quantity), curr)))
			}
			sb.WriteString(`</div>`)
		}
		sb.WriteString(`</div>`)
	}
	return sb.String()
}

func whatsAppHeader(num string) string {
	if num == "" {
		return ""
	}
	clean := strings.ReplaceAll(num, "+", "")
	clean = strings.ReplaceAll(clean, " ", "")
	return fmt.Sprintf(`<a href="https://wa.me/%s" target="_blank" class="p-2 rounded-full glass-card hover:bg-emerald-500/20 text-emerald-400 transition"><i data-lucide="message-circle" class="w-4 h-4"></i></a>`, clean)
}

func whatsAppFooter(num string) string {
	if num == "" {
		return ""
	}
	clean := strings.ReplaceAll(num, "+", "")
	clean = strings.ReplaceAll(clean, " ", "")
	return fmt.Sprintf(`<a href="https://wa.me/%s" target="_blank" class="px-4 py-2.5 rounded-xl bg-emerald-500 hover:bg-emerald-600 text-white text-xs font-bold flex items-center gap-2 shadow-lg shadow-emerald-500/20 transition"><i data-lucide="message-circle" class="w-4 h-4"></i> WhatsApp</a>`, clean)
}

func acceptanceBtnText(accepted bool) string {
	if accepted {
		return "✓ Tedavi Planı Onaylandı"
	}
	return "Tedavi Planını Onayla"
}
