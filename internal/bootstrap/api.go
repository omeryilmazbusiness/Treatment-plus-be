// Package bootstrap is the composition root helper. Only cmd/* imports this package.
// Concrete adapters are wired here; use-cases never see pgx, Redis, or HTTP types.
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	aihttp "github.com/treatment-plus/treatment-plus-be/internal/ai/http"
	catalogapp "github.com/treatment-plus/treatment-plus-be/internal/catalog/app"
	cataloghttp "github.com/treatment-plus/treatment-plus-be/internal/catalog/http"
	catalogpg "github.com/treatment-plus/treatment-plus-be/internal/catalog/pg"
	clinicapp "github.com/treatment-plus/treatment-plus-be/internal/clinic/app"
	clinicdomain "github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	clinichttp "github.com/treatment-plus/treatment-plus-be/internal/clinic/http"
	clinicpg "github.com/treatment-plus/treatment-plus-be/internal/clinic/pg"
	identityapp "github.com/treatment-plus/treatment-plus-be/internal/identity/app"
	identityhttp "github.com/treatment-plus/treatment-plus-be/internal/identity/http"
	identitypg "github.com/treatment-plus/treatment-plus-be/internal/identity/pg"
	mediaapp "github.com/treatment-plus/treatment-plus-be/internal/media/app"
	mediahttp "github.com/treatment-plus/treatment-plus-be/internal/media/http"
	mediapg "github.com/treatment-plus/treatment-plus-be/internal/media/pg"
	outputapp "github.com/treatment-plus/treatment-plus-be/internal/output/app"
	outputhttp "github.com/treatment-plus/treatment-plus-be/internal/output/http"
	outputpdf "github.com/treatment-plus/treatment-plus-be/internal/output/pdf"
	outputpg "github.com/treatment-plus/treatment-plus-be/internal/output/pg"
	planapp "github.com/treatment-plus/treatment-plus-be/internal/plan/app"
	planhttp "github.com/treatment-plus/treatment-plus-be/internal/plan/http"
	planpg "github.com/treatment-plus/treatment-plus-be/internal/plan/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/authn"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/config"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/httpx"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/postgres"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/redisx"
	shareapp "github.com/treatment-plus/treatment-plus-be/internal/share/app"
	sharehttp "github.com/treatment-plus/treatment-plus-be/internal/share/http"
	sharepg "github.com/treatment-plus/treatment-plus-be/internal/share/pg"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/clock"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/crypto"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	vizapp "github.com/treatment-plus/treatment-plus-be/internal/viz/app"
	vizhttp "github.com/treatment-plus/treatment-plus-be/internal/viz/http"
)

type API struct {
	Server *httpx.Server
	Pool   *pgxpool.Pool
	Redis  *redis.Client
	Log    *slog.Logger
}

type catalogAdapter struct {
	svc *catalogapp.Service
}

func (c *catalogAdapter) GetTreatment(ctx context.Context, wsID id.ID, treatID id.ID) (string, string, int64, error) {
	price, t, err := c.svc.GetTreatmentPrice(ctx, wsID, treatID)
	if err != nil {
		return "", "", 0, err
	}
	return t.Code, t.Name, price, nil
}

type clinicSettingsAdapter struct {
	svc *clinicapp.Service
}

func (c *clinicSettingsAdapter) GetClinicSettings(ctx context.Context, wsID id.ID) (planapp.ClinicSettingsDTO, error) {
	st, err := c.svc.GetSettings(ctx, wsID)
	if err != nil {
		return planapp.ClinicSettingsDTO{}, err
	}
	return planapp.ClinicSettingsDTO{
		Name:        st.Name,
		Slogan:      st.Slogan,
		Phone:       st.Phone,
		Email:       st.Email,
		Website:     st.Website,
		Address:     st.Address,
		Hours:       st.Hours,
		Palette:     string(st.Palette),
		Locale:      string(st.Locale),
		Currency:    string(st.Currency),
		LogoKey:     st.LogoKey,
		CoverKey:    st.CoverKey,
		MapURL:      st.MapURL,
		Intro:       st.Intro,
		Legal:       st.Legal,
		AboutText:   st.AboutText,
		AboutKey:    st.AboutKey,
		WelcomeBody: st.WelcomeBody,
		SignerName:  st.SignerName,
		SignerTitle: st.SignerTitle,
		Highlights:  toPlanStats(st.Highlights),
		Doctor:      toPlanDoctor(st.Doctor),
		Lab:         toPlanLab(st.Lab),
		Awards:      planapp.ClinicAwardsDTO{Text: st.Awards.Text, BadgeKeys: st.Awards.BadgeKeys},
		Hotel:       planapp.ClinicHotelDTO{Name: st.Hotel.Name, Text: st.Hotel.Text, ImageKey: st.Hotel.ImageKey},
	}, nil
}

func toPlanStats(in []clinicdomain.Stat) []planapp.ClinicStatDTO {
	var out []planapp.ClinicStatDTO
	for _, s := range in {
		out = append(out, planapp.ClinicStatDTO{Value: s.Value, Label: s.Label})
	}
	return out
}

func toPlanDoctor(d clinicdomain.Doctor) planapp.ClinicDoctorDTO {
	return planapp.ClinicDoctorDTO{
		Name: d.Name, Title: d.Title, Subtitle: d.Subtitle, Bio: d.Bio, PhotoKey: d.PhotoKey, Stats: toPlanStats(d.Stats),
	}
}

func toPlanLab(l clinicdomain.Lab) planapp.ClinicLabDTO {
	return planapp.ClinicLabDTO{Title: l.Title, Text: l.Text, ImageKey: l.ImageKey, Stats: toPlanStats(l.Stats)}
}

func toClinicStats(in []identityapp.ClinicStat) []clinicdomain.Stat {
	var out []clinicdomain.Stat
	for _, s := range in {
		out = append(out, clinicdomain.Stat{Value: s.Value, Label: s.Label})
	}
	return out
}

func provisionClinic(svc *clinicapp.Service) identityapp.ProfileFunc {
	return func(ctx context.Context, wsID id.ID, profile identityapp.ClinicProfile) error {
		_, err := svc.Onboard(ctx, wsID, clinicapp.OnboardInput{
			Name:        profile.Name,
			Slogan:      profile.Slogan,
			Phone:       profile.Phone,
			Email:       profile.Email,
			Website:     profile.Website,
			Address:     profile.Address,
			Hours:       profile.Hours,
			Locale:      profile.Locale,
			Currency:    profile.Currency,
			Palette:     profile.Palette,
			LogoKey:     profile.LogoKey,
			CoverKey:    profile.CoverKey,
			MapURL:      profile.MapURL,
			Intro:       profile.Intro,
			Legal:       profile.Legal,
			AboutText:   profile.AboutText,
			AboutKey:    profile.AboutKey,
			WelcomeBody: profile.WelcomeBody,
			SignerName:  profile.SignerName,
			SignerTitle: profile.SignerTitle,
			Highlights:  toClinicStats(profile.Highlights),
			Doctor: clinicdomain.Doctor{
				Name: profile.Doctor.Name, Title: profile.Doctor.Title, Subtitle: profile.Doctor.Subtitle,
				Bio: profile.Doctor.Bio, PhotoKey: profile.Doctor.PhotoKey, Stats: toClinicStats(profile.Doctor.Stats),
			},
			Lab: clinicdomain.Lab{
				Title: profile.Lab.Title, Text: profile.Lab.Text, ImageKey: profile.Lab.ImageKey, Stats: toClinicStats(profile.Lab.Stats),
			},
			Awards:       clinicdomain.Awards{Text: profile.Awards.Text, BadgeKeys: profile.Awards.BadgeKeys},
			Hotel:        clinicdomain.Hotel{Name: profile.Hotel.Name, Text: profile.Hotel.Text, ImageKey: profile.Hotel.ImageKey},
			RequireBrand: profile.RequireBrand,
		})
		return err
	}
}

func NewAPI(ctx context.Context, cfg config.Config, log *slog.Logger) (*API, error) {
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}

	rdb, err := redisx.New(cfg.Redis)
	if err != nil {
		pool.Close()
		return nil, err
	}
	if err := rdb.Ping(ctx).Err(); err != nil {
		pool.Close()
		_ = rdb.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	clk := clock.System{}
	tokens, err := authn.NewJWT(cfg.Auth, clk)
	if err != nil {
		pool.Close()
		_ = rdb.Close()
		return nil, err
	}

	identityStore := identitypg.New(pool)
	clinicStore := clinicpg.New(pool)
	clinicSvc := clinicapp.NewService(clinicStore)

	identitySvc, err := identityapp.NewService(
		identityStore,
		identityStore,
		identityStore,
		identityStore,
		authn.DefaultArgon2id(),
		tokens,
		clk,
		cfg.Auth.RefreshTTL,
		provisionClinic(clinicSvc),
	)
	if err != nil {
		pool.Close()
		_ = rdb.Close()
		return nil, err
	}

	catalogStore := catalogpg.New(pool)
	catalogSvc := catalogapp.NewService(catalogStore)

	encrypter := crypto.NewAESGCM(cfg.Auth.KMSKey)
	planStore := planpg.New(pool)
	planSvc := planapp.NewService(
		planStore,
		planStore,
		&catalogAdapter{svc: catalogSvc},
		&clinicSettingsAdapter{svc: clinicSvc},
		encrypter,
	).WithLogger(log)

	mediaStore := mediapg.New(pool)
	mediaSvc := mediaapp.NewService(mediaStore)
	outputStore := outputpg.New(pool)
	outputSvc := outputapp.NewService(outputStore, outputpdf.NewRenderer(), mediaSvc, outputStore, outputpdf.AwardJPEG())
	vizSvc := vizapp.NewService(&planVizAdapter{svc: planSvc})

	shareStore := sharepg.New(pool)
	shareSvc := shareapp.NewService(
		shareStore,
		&sharePlanAdapter{svc: planSvc},
		&shareClinicAdapter{svc: clinicSvc},
		&shareVizAdapter{svc: vizSvc},
		&sharePDFAdapter{svc: outputSvc},
		cfg.App.URL,
	)

	srv := httpx.NewServer(httpx.Options{
		Config: cfg,
		Log:    log,
		Ready: []httpx.Readiness{
			postgres.Checker{Pool: pool},
			redisx.Checker{Client: rdb},
		},
		Registrars: []httpx.RouteRegistrar{
			identityhttp.NewHandler(identitySvc, tokens),
			clinichttp.NewHandler(clinicSvc, tokens),
			cataloghttp.NewHandler(catalogSvc, tokens),
			planhttp.NewHandler(planSvc, tokens),
			mediahttp.NewHandler(mediaSvc, tokens),
			outputhttp.NewHandler(outputSvc, tokens),
			vizhttp.NewHandler(vizSvc, tokens),
			sharehttp.NewHandler(shareSvc, tokens),
			aihttp.NewHandler(),
		},
	})

	return &API{Server: srv, Pool: pool, Redis: rdb, Log: log}, nil
}

type planVizAdapter struct {
	svc *planapp.Service
}

func (a *planVizAdapter) GetReadyItems(ctx context.Context, workspaceID, planID id.ID) (string, string, []vizapp.ItemRef, error) {
	p, err := a.svc.GetPlan(ctx, workspaceID, planID)
	if err != nil {
		return "", "", nil, err
	}
	if !p.IsReadyOrPresented() {
		return "", "", nil, errx.New("plan_not_ready", "plan must be ready before 3D playlist", errx.ErrInvalid)
	}
	items := make([]vizapp.ItemRef, 0, len(p.Items))
	for _, it := range p.Items {
		items = append(items, vizapp.ItemRef{
			AnimationCode: it.AnimationCode,
			TreatmentName: it.TreatmentName,
			Teeth:         it.Teeth,
		})
	}
	return string(p.Locale), p.ID.String(), items, nil
}

type sharePlanAdapter struct {
	svc *planapp.Service
}

func (a *sharePlanAdapter) GetPlanForShare(ctx context.Context, clinicID, planID id.ID) (shareapp.SharedPlanData, error) {
	p, err := a.svc.GetPlan(ctx, clinicID, planID)
	if err != nil {
		return shareapp.SharedPlanData{}, err
	}
	patName := ""
	if p.Patient != nil {
		patName = p.Patient.Name
	}

	visits := make([]shareapp.SharedVisit, 0, len(p.Visits))
	items := make([]shareapp.SharedItem, 0, len(p.Items))

	itemsByVisit := make(map[string][]shareapp.SharedItem)
	for _, it := range p.Items {
		var vIDStr *string
		if it.VisitID != nil {
			s := it.VisitID.String()
			vIDStr = &s
		}
		itemDTO := shareapp.SharedItem{
			ID:            it.ID.String(),
			VisitID:       vIDStr,
			TreatmentName: it.TreatmentName,
			AnimationCode: it.AnimationCode,
			ToothNumber:   it.ToothNumber,
			Teeth:         it.Teeth,
			Description:   it.Description,
			Quantity:      it.Quantity,
			UnitPrice:     it.UnitPrice,
			IsPackage:     it.IsPackage,
		}
		items = append(items, itemDTO)
		if vIDStr != nil {
			itemsByVisit[*vIDStr] = append(itemsByVisit[*vIDStr], itemDTO)
		}
	}

	for _, v := range p.Visits {
		vIDStr := v.ID.String()
		visits = append(visits, shareapp.SharedVisit{
			ID:          vIDStr,
			OrderNum:    v.OrderNum,
			Title:       v.Title,
			PhaseLabel:  v.PhaseLabel,
			HealingNote: v.HealingNote,
			ChartTeeth:  v.ChartTeeth,
			Items:       itemsByVisit[vIDStr],
		})
	}

	discounts := make([]shareapp.SharedDiscount, 0, len(p.Discounts))
	for _, d := range p.Discounts {
		discounts = append(discounts, shareapp.SharedDiscount{
			ID:     d.ID.String(),
			Label:  d.Label,
			Amount: d.Amount,
		})
	}

	findings := make([]shareapp.SharedToothFinding, 0, len(p.Situation.ToothFindings))
	for _, f := range p.Situation.ToothFindings {
		findings = append(findings, shareapp.SharedToothFinding{
			Tooth:  f.Tooth,
			Status: f.Status,
			Note:   f.Note,
		})
	}

	return shareapp.SharedPlanData{
		ID:                 p.ID,
		ClinicID:           p.WorkspaceID,
		PatientName:        patName,
		Title:              p.Title,
		Kind:               string(p.Kind),
		Status:             string(p.Status),
		Currency:           string(p.Currency),
		Palette:            string(p.Palette),
		Locale:             string(p.Locale),
		WelcomeLetter:      p.WelcomeLetter,
		VideoURL:           p.VideoURL,
		BeforeAfterCaseIDs: p.BeforeAfterCaseIDs,
		Diagnoses:          p.Diagnoses,
		Situation: shareapp.SharedSituation{
			XrayKey:          p.Situation.XrayKey,
			ChartTeeth:       p.Situation.ChartTeeth,
			ToothFindings:    findings,
			GeneralDiagnoses: p.Situation.GeneralDiagnoses,
		},
		Visits:         visits,
		Items:          items,
		Discounts:      discounts,
		TotalsSubtotal: p.TotalsSubtotal,
		TotalsDiscount: p.TotalsDiscount,
		TotalsGrand:    p.TotalsGrand,
		CreatedAt:      p.CreatedAt,
	}, nil
}

type shareClinicAdapter struct {
	svc *clinicapp.Service
}

func (a *shareClinicAdapter) GetClinicBrand(ctx context.Context, clinicID id.ID) (shareapp.SharedClinicData, error) {
	st, err := a.svc.GetSettings(ctx, clinicID)
	if err != nil {
		return shareapp.SharedClinicData{}, err
	}

	highlights := make([]shareapp.SharedStat, 0, len(st.Highlights))
	for _, h := range st.Highlights {
		highlights = append(highlights, shareapp.SharedStat{Value: h.Value, Label: h.Label})
	}

	docStats := make([]shareapp.SharedStat, 0, len(st.Doctor.Stats))
	for _, s := range st.Doctor.Stats {
		docStats = append(docStats, shareapp.SharedStat{Value: s.Value, Label: s.Label})
	}

	labStats := make([]shareapp.SharedStat, 0, len(st.Lab.Stats))
	for _, s := range st.Lab.Stats {
		labStats = append(labStats, shareapp.SharedStat{Value: s.Value, Label: s.Label})
	}

	team := make([]shareapp.SharedDoctor, 0, len(st.Team))
	for _, d := range st.Team {
		dStats := make([]shareapp.SharedStat, 0, len(d.Stats))
		for _, s := range d.Stats {
			dStats = append(dStats, shareapp.SharedStat{Value: s.Value, Label: s.Label})
		}
		team = append(team, shareapp.SharedDoctor{
			Name:     d.Name,
			Title:    d.Title,
			Subtitle: d.Subtitle,
			Bio:      d.Bio,
			PhotoKey: d.PhotoKey,
			Stats:    dStats,
		})
	}

	reviews := make([]shareapp.SharedReview, 0, len(st.Reviews))
	for _, r := range st.Reviews {
		reviews = append(reviews, shareapp.SharedReview{
			Author:   r.Author,
			Rating:   r.Rating,
			Text:     r.Text,
			Source:   r.Source,
			Date:     r.Date,
			Verified: r.Verified,
		})
	}

	ba := make([]shareapp.SharedBeforeAfterCase, 0, len(st.BeforeAfter))
	for _, b := range st.BeforeAfter {
		ba = append(ba, shareapp.SharedBeforeAfterCase{
			ID:             b.ID,
			Title:          b.Title,
			Tag:            b.Tag,
			Description:    b.Description,
			BeforeImageKey: b.BeforeImageKey,
			AfterImageKey:  b.AfterImageKey,
		})
	}

	return shareapp.SharedClinicData{
		Name:        st.Name,
		Slogan:      st.Slogan,
		Phone:       st.Phone,
		Email:       st.Email,
		Website:     st.Website,
		Address:     st.Address,
		Hours:       st.Hours,
		LogoKey:     st.LogoKey,
		CoverKey:    st.CoverKey,
		MapURL:      st.MapURL,
		Intro:       st.Intro,
		Legal:       st.Legal,
		AboutText:   st.AboutText,
		AboutKey:    st.AboutKey,
		WelcomeBody: st.WelcomeBody,
		SignerName:  st.SignerName,
		SignerTitle: st.SignerTitle,
		Highlights:  highlights,
		Doctor: shareapp.SharedDoctor{
			Name:     st.Doctor.Name,
			Title:    st.Doctor.Title,
			Subtitle: st.Doctor.Subtitle,
			Bio:      st.Doctor.Bio,
			PhotoKey: st.Doctor.PhotoKey,
			Stats:    docStats,
		},
		Lab: shareapp.SharedLab{
			Title:    st.Lab.Title,
			Text:     st.Lab.Text,
			ImageKey: st.Lab.ImageKey,
			Stats:    labStats,
		},
		Awards: shareapp.SharedAwards{
			Text:      st.Awards.Text,
			BadgeKeys: st.Awards.BadgeKeys,
		},
		Hotel: shareapp.SharedHotel{
			Name:     st.Hotel.Name,
			Text:     st.Hotel.Text,
			ImageKey: st.Hotel.ImageKey,
		},
		Team:        team,
		Reviews:     reviews,
		BeforeAfter: ba,
		CTA: shareapp.SharedCTAConfig{
			WhatsAppNumber:   st.CTA.WhatsAppNumber,
			WhatsAppMessage:  st.CTA.WhatsAppMessage,
			PhoneNumber:      st.CTA.PhoneNumber,
			BookingURL:       st.CTA.BookingURL,
			EnableAcceptance: st.CTA.EnableAcceptance,
		},
	}, nil
}

type shareVizAdapter struct {
	svc *vizapp.Service
}

func (a *shareVizAdapter) GetVizPlaylist(ctx context.Context, clinicID, planID id.ID) (shareapp.SharedVizPlaylist, error) {
	pl, err := a.svc.Playlist(ctx, clinicID, planID)
	if err != nil {
		return shareapp.SharedVizPlaylist{}, err
	}
	clips := make([]shareapp.SharedVizClip, 0, len(pl.Clips))
	for _, c := range pl.Clips {
		clips = append(clips, shareapp.SharedVizClip{
			Code:  string(c.Code),
			Teeth: c.Teeth,
			Title: c.Title,
			Moves: c.Moves,
		})
	}
	return shareapp.SharedVizPlaylist{
		PlanID: pl.PlanID,
		Locale: pl.Locale,
		Clips:  clips,
	}, nil
}

type sharePDFAdapter struct {
	svc *outputapp.Service
}

func (a *sharePDFAdapter) GetPlanPDF(ctx context.Context, clinicID, planID id.ID) ([]byte, error) {
	doc, err := a.svc.Get(ctx, clinicID, planID)
	if err != nil {
		return nil, err
	}
	return doc.Bytes, nil
}

func (a *API) Close() {
	if a.Redis != nil {
		_ = a.Redis.Close()
	}
	if a.Pool != nil {
		a.Pool.Close()
	}
}
