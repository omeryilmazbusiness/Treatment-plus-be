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
}

func NewService(store SettingsStore) *Service {
	return &Service{store: store}
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
	Name     *string
	Locale   *string
	Currency *string
	Palette  *string
	LogoKey  *string
	Phone    *string
	MapURL   *string
	Intro    *string
	Legal    *string
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

	if input.Name != nil {
		current.Name = strings.TrimSpace(*input.Name)
	}
	if input.LogoKey != nil {
		current.LogoKey = strings.TrimSpace(*input.LogoKey)
	}
	if input.Phone != nil {
		current.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.MapURL != nil {
		current.MapURL = strings.TrimSpace(*input.MapURL)
	}
	if input.Intro != nil {
		current.Intro = strings.TrimSpace(*input.Intro)
	}
	if input.Legal != nil {
		current.Legal = strings.TrimSpace(*input.Legal)
	}

	if err := s.store.SaveSettings(ctx, current); err != nil {
		return domain.Settings{}, err
	}

	return current, nil
}

func (s *Service) GetProducts(ctx context.Context) []domain.Product {
	return domain.AvailableProducts()
}
