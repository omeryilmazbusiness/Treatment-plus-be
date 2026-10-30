package app

import (
	"context"
	"strings"

	"github.com/treatment-plus/treatment-plus-be/internal/clinic/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Service struct {
	store SettingsStore
	sub   SubscriptionReader
}

func NewService(store SettingsStore) *Service {
	return &Service{store: store}
}

func (s *Service) WithSubscription(sub SubscriptionReader) *Service {
	s.sub = sub
	return s
}

func (s *Service) GetSettings(ctx context.Context, workspaceID id.ID) (domain.Settings, error) {
	if workspaceID.IsZero() {
		return domain.Settings{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}
	settings, err := s.store.GetSettings(ctx, workspaceID)
	if err != nil {
		if errx.Is(err, errx.ErrNotFound) {
			return domain.DefaultSettings(workspaceID, ""), nil
		}
		return domain.Settings{}, err
	}
	return settings, nil
}

type UpdateSettingsInput struct {
	Name        *string
	Slogan      *string
	Locale      *string
	Currency    *string
	Palette     *string
	LogoKey     *string
	CoverKey    *string
	Phone       *string
	Email       *string
	Website     *string
	Address     *string
	Hours       *string
	MapURL      *string
	Intro       *string
	Legal       *string
	AboutText   *string
	AboutKey    *string
	WelcomeBody *string
	SignerName  *string
	SignerTitle *string
	Highlights  *[]domain.Stat
	Doctor      *domain.Doctor
	Lab         *domain.Lab
	Awards      *domain.Awards
	Hotel       *domain.Hotel
	Team        *[]domain.Doctor
	Reviews     *[]domain.Review
	BeforeAfter *[]domain.BeforeAfterCase
	CTA         *domain.CTAConfig
}

func (s *Service) UpdateSettings(ctx context.Context, workspaceID id.ID, input UpdateSettingsInput) (domain.Settings, error) {
	if workspaceID.IsZero() {
		return domain.Settings{}, errx.New("invalid_workspace_id", "workspace id is required", errx.ErrInvalid)
	}

	current, err := s.store.GetSettings(ctx, workspaceID)
	if err != nil {
		if errx.Is(err, errx.ErrNotFound) {
			current = domain.DefaultSettings(workspaceID, "")
		} else {
			return domain.Settings{}, err
		}
	}

	if input.Locale != nil {
		loc, err := domain.ParseLocale(*input.Locale)
		if err != nil {
			return domain.Settings{}, err
		}
		current.Locale = loc
	}
	if input.Currency != nil {
		curr, err := domain.ParseCurrency(*input.Currency)
		if err != nil {
			return domain.Settings{}, err
		}
		current.Currency = curr
	}
	if input.Palette != nil {
		pal, err := domain.ParsePalette(*input.Palette)
		if err != nil {
			return domain.Settings{}, err
		}
		current.Palette = pal
	}

	set := func(dst *string, src *string) {
		if src != nil {
			*dst = strings.TrimSpace(*src)
		}
	}
	set(&current.Name, input.Name)
	set(&current.Slogan, input.Slogan)
	set(&current.LogoKey, input.LogoKey)
	set(&current.CoverKey, input.CoverKey)
	set(&current.Phone, input.Phone)
	set(&current.Email, input.Email)
	set(&current.Website, input.Website)
	set(&current.Address, input.Address)
	set(&current.Hours, input.Hours)
	set(&current.MapURL, input.MapURL)
	set(&current.Intro, input.Intro)
	set(&current.Legal, input.Legal)
	set(&current.AboutText, input.AboutText)
	set(&current.AboutKey, input.AboutKey)
	set(&current.WelcomeBody, input.WelcomeBody)
	set(&current.SignerName, input.SignerName)
	set(&current.SignerTitle, input.SignerTitle)

	if input.Highlights != nil {
		current.Highlights = nil
		for _, h := range *input.Highlights {
			if n := h.Normalize(); n.Value != "" || n.Label != "" {
				current.Highlights = append(current.Highlights, n)
			}
		}
	}
	if input.Doctor != nil {
		current.Doctor = input.Doctor.Normalize()
	}
	if input.Lab != nil {
		current.Lab = input.Lab.Normalize()
	}
	if input.Awards != nil {
		current.Awards = input.Awards.Normalize()
	}
	if input.Hotel != nil {
		current.Hotel = input.Hotel.Normalize()
	}
	if input.Team != nil {
		current.Team = nil
		for _, doc := range *input.Team {
			if n := doc.Normalize(); n.Name != "" {
				current.Team = append(current.Team, n)
			}
		}
	}
	if input.Reviews != nil {
		current.Reviews = nil
		for _, rev := range *input.Reviews {
			if n := rev.Normalize(); n.Author != "" || n.Text != "" {
				current.Reviews = append(current.Reviews, n)
			}
		}
	}
	if input.BeforeAfter != nil {
		current.BeforeAfter = nil
		for _, ba := range *input.BeforeAfter {
			if n := ba.Normalize(); n.Title != "" || n.BeforeImageKey != "" || n.AfterImageKey != "" {
				current.BeforeAfter = append(current.BeforeAfter, n)
			}
		}
	}
	if input.CTA != nil {
		current.CTA = input.CTA.Normalize()
	}

	if err := current.Validate(); err != nil {
		return domain.Settings{}, err
	}
	if err := s.store.SaveSettings(ctx, current); err != nil {
		return domain.Settings{}, err
	}
	return current, nil
}

func (s *Service) GetProducts(ctx context.Context) []domain.Product {
	return s.GetProductsForWorkspace(ctx, id.ID{})
}

func (s *Service) GetProductsForWorkspace(ctx context.Context, wsID id.ID) []domain.Product {
	if s.sub != nil && !wsID.IsZero() {
		_, allowsUltimate, err := s.sub.GetActiveTier(ctx, wsID)
		if err == nil {
			return []domain.Product{
				{ID: "classic", Open: true},
				{ID: "ultimate", Open: allowsUltimate},
			}
		}
	}
	return domain.AvailableProducts()
}

type OnboardInput struct {
	Name         string
	Slogan       string
	Phone        string
	Email        string
	Website      string
	Address      string
	Hours        string
	Locale       string
	Currency     string
	Palette      string
	LogoKey      string
	CoverKey     string
	MapURL       string
	Intro        string
	Legal        string
	AboutText    string
	AboutKey     string
	WelcomeBody  string
	SignerName   string
	SignerTitle  string
	Highlights   []domain.Stat
	Doctor       domain.Doctor
	Lab          domain.Lab
	Awards       domain.Awards
	Hotel        domain.Hotel
	Team         []domain.Doctor
	Reviews      []domain.Review
	BeforeAfter  []domain.BeforeAfterCase
	CTA          domain.CTAConfig
	RequireBrand bool
}

func (s *Service) Onboard(ctx context.Context, workspaceID id.ID, input OnboardInput) (domain.Settings, error) {
	settings, err := domain.Onboard(workspaceID, domain.OnboardInput{
		Name:         input.Name,
		Slogan:       input.Slogan,
		Phone:        input.Phone,
		Email:        input.Email,
		Website:      input.Website,
		Address:      input.Address,
		Hours:        input.Hours,
		Locale:       input.Locale,
		Currency:     input.Currency,
		Palette:      input.Palette,
		LogoKey:      input.LogoKey,
		CoverKey:     input.CoverKey,
		MapURL:       input.MapURL,
		Intro:        input.Intro,
		Legal:        input.Legal,
		AboutText:    input.AboutText,
		AboutKey:     input.AboutKey,
		WelcomeBody:  input.WelcomeBody,
		SignerName:   input.SignerName,
		SignerTitle:  input.SignerTitle,
		Highlights:   input.Highlights,
		Doctor:       input.Doctor,
		Lab:          input.Lab,
		Awards:       input.Awards,
		Hotel:        input.Hotel,
		Team:         input.Team,
		Reviews:      input.Reviews,
		BeforeAfter:  input.BeforeAfter,
		CTA:          input.CTA,
		RequireBrand: input.RequireBrand,
	})
	if err != nil {
		return domain.Settings{}, err
	}
	if err := s.store.SaveSettings(ctx, settings); err != nil {
		return domain.Settings{}, err
	}
	return settings, nil
}
