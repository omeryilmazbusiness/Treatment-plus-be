package app

import (
	"context"
	"strings"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/viz/domain"
)

type PlanReader interface {
	GetReadyItems(ctx context.Context, workspaceID, planID id.ID) (locale string, planIDStr string, items []ItemRef, err error)
}

type ItemRef struct {
	AnimationCode string
	TreatmentName string
	Teeth         []string
}

type Service struct {
	plans PlanReader
}

func NewService(plans PlanReader) *Service {
	return &Service{plans: plans}
}

func (s *Service) Recipes() []domain.Recipe {
	return domain.Library()
}

func (s *Service) Demo(codeRaw, locale, teethCSV, itemsRaw string) (domain.Playlist, error) {
	if locale == "" {
		locale = "tr"
	}
	out := domain.Playlist{PlanID: "demo", Locale: locale}
	if strings.TrimSpace(itemsRaw) != "" {
		for _, part := range splitCSV(itemsRaw) {
			codeStr, teethStr, _ := strings.Cut(part, ":")
			c, err := domain.ParseCode(codeStr)
			if err != nil {
				return domain.Playlist{}, err
			}
			rec, ok := domain.RecipeByCode(c)
			if !ok {
				continue
			}
			teeth := splitTeeth(teethStr)
			if len(teeth) == 0 {
				teeth = domain.DefaultTeeth(c)
			}
			out.Clips = append(out.Clips, domain.Clip{
				Code: rec.Code, Teeth: teeth, Title: rec.Title(locale), Moves: rec.Moves,
			})
		}
	} else {
		codes := domain.AllCodes()
		if strings.TrimSpace(codeRaw) != "" {
			var parsed []domain.Code
			for _, part := range splitCSV(codeRaw) {
				c, err := domain.ParseCode(part)
				if err != nil {
					return domain.Playlist{}, err
				}
				parsed = append(parsed, c)
			}
			codes = parsed
		}
		filter := splitCSV(teethCSV)
		for _, c := range codes {
			rec, ok := domain.RecipeByCode(c)
			if !ok {
				continue
			}
			teeth := filter
			if len(teeth) == 0 {
				teeth = domain.DefaultTeeth(c)
			}
			out.Clips = append(out.Clips, domain.Clip{
				Code: rec.Code, Teeth: teeth, Title: rec.Title(locale), Moves: rec.Moves,
			})
		}
	}
	if len(out.Clips) == 0 {
		return domain.Playlist{}, errx.New("viz_empty", "no animation clips", errx.ErrInvalid)
	}
	return out, nil
}

func splitTeeth(s string) []string {
	s = strings.ReplaceAll(s, "+", ",")
	s = strings.ReplaceAll(s, "|", ",")
	s = strings.ReplaceAll(s, " ", ",")
	return splitCSV(s)
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (s *Service) Playlist(ctx context.Context, workspaceID, planID id.ID) (domain.Playlist, error) {
	if workspaceID.IsZero() || planID.IsZero() {
		return domain.Playlist{}, errx.New("invalid_id", "workspace and plan id required", errx.ErrInvalid)
	}
	if s.plans == nil {
		return domain.Playlist{}, errx.New("viz_unavailable", "plan reader is not configured", errx.ErrUnavailable)
	}
	locale, planKey, items, err := s.plans.GetReadyItems(ctx, workspaceID, planID)
	if err != nil {
		return domain.Playlist{}, err
	}
	out := domain.Playlist{PlanID: planKey, Locale: locale}
	for _, it := range items {
		code, err := resolveCode(it.AnimationCode, it.TreatmentName)
		if err != nil {
			continue
		}
		rec, ok := domain.RecipeByCode(code)
		if !ok {
			continue
		}
		teeth := it.Teeth
		if len(teeth) == 0 {
			continue
		}
		out.Clips = append(out.Clips, domain.Clip{
			Code:  rec.Code,
			Teeth: teeth,
			Title: rec.Title(locale),
			Moves: rec.Moves,
		})
	}
	if len(out.Clips) == 0 {
		return domain.Playlist{}, errx.New("viz_empty", "no animatable treatments on this plan", errx.ErrInvalid)
	}
	return out, nil
}

func resolveCode(explicit, name string) (domain.Code, error) {
	if strings.TrimSpace(explicit) != "" {
		return domain.ParseCode(explicit)
	}
	return domain.ParseCode(name)
}
