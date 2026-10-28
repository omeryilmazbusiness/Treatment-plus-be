package app_test

import (
	"context"
	"strings"
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/viz/app"
	"github.com/treatment-plus/treatment-plus-be/internal/viz/domain"
)

type fakePlans struct {
	locale string
	plan   string
	items  []app.ItemRef
	err    error
}

func (f fakePlans) GetReadyItems(context.Context, id.ID, id.ID) (string, string, []app.ItemRef, error) {
	return f.locale, f.plan, f.items, f.err
}

func TestDemoAllClips(t *testing.T) {
	pl, err := app.NewService(nil).Demo("", "tr", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(pl.Clips) != len(domain.AllCodes()) {
		t.Fatalf("clips %d", len(pl.Clips))
	}
	if pl.Clips[0].Title == "" || len(pl.Clips[0].Moves) == 0 {
		t.Fatalf("%+v", pl.Clips[0])
	}
}

func TestDemoSingleCode(t *testing.T) {
	pl, err := app.NewService(nil).Demo("Zirkonyum Kron", "en", "21", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(pl.Clips) != 1 || pl.Clips[0].Code != domain.ZirconiaCrown {
		t.Fatalf("%+v", pl.Clips)
	}
	if pl.Clips[0].Title != "Zirconia crown" || pl.Clips[0].Teeth[0] != "21" {
		t.Fatalf("%+v", pl.Clips[0])
	}
}

func TestDemoItemsPlaylist(t *testing.T) {
	pl, err := app.NewService(nil).Demo("", "tr", "", "composite_filling:16+26,emax_veneer:11+21,implant:36")
	if err != nil {
		t.Fatal(err)
	}
	if len(pl.Clips) != 3 {
		t.Fatalf("clips %d", len(pl.Clips))
	}
	if pl.Clips[0].Code != domain.CompositeFilling || strings.Join(pl.Clips[0].Teeth, ",") != "16,26" {
		t.Fatalf("%+v", pl.Clips[0])
	}
	spaced, err := app.NewService(nil).Demo("", "tr", "", "composite_filling:16 26")
	if err != nil || strings.Join(spaced.Clips[0].Teeth, ",") != "16,26" {
		t.Fatalf("space teeth %+v %v", spaced.Clips, err)
	}
	if pl.Clips[1].Code != domain.EmaxVeneer || pl.Clips[2].Code != domain.Implant || pl.Clips[2].Teeth[0] != "36" {
		t.Fatalf("%+v", pl.Clips)
	}
}

func TestPlaylistReadyItems(t *testing.T) {
	ws, plan := id.New(), id.New()
	svc := app.NewService(fakePlans{
		locale: "tr",
		plan:   plan.String(),
		items: []app.ItemRef{
			{AnimationCode: "implant", TreatmentName: "Diş İmplantı", Teeth: []string{"16"}},
			{TreatmentName: "Panoramik", Teeth: []string{"16"}},
		},
	})
	pl, err := svc.Playlist(context.Background(), ws, plan)
	if err != nil {
		t.Fatal(err)
	}
	if len(pl.Clips) != 1 || pl.Clips[0].Code != domain.Implant {
		t.Fatalf("%+v", pl)
	}
}

func TestPlaylistEmptyRejected(t *testing.T) {
	ws, plan := id.New(), id.New()
	_, err := app.NewService(fakePlans{locale: "tr", plan: plan.String()}).Playlist(context.Background(), ws, plan)
	if err == nil || !errx.Is(err, errx.ErrInvalid) {
		t.Fatalf("%v", err)
	}
}
