package app_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/treatment-plus/treatment-plus-be/internal/output/app"
	"github.com/treatment-plus/treatment-plus-be/internal/output/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/output/mem"
	"github.com/treatment-plus/treatment-plus-be/internal/output/pdf"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type stubPlans struct {
	view domain.PlanView
	err  error
}

func (s stubPlans) GetView(context.Context, id.ID, id.ID) (domain.PlanView, error) {
	return s.view, s.err
}

func TestGeneratePersistsAndGetRoundtrip(t *testing.T) {
	ws, planID := id.New(), id.New()
	snap := domain.Snapshot{PlanID: planID.String(), Title: "Plan", Currency: "TRY", Palette: "navy", Locale: "tr", Patient: domain.Patient{Name: "Ada"}}
	payload, _ := json.Marshal(snap)
	docs := mem.New()
	svc := app.NewService(stubPlans{view: domain.PlanView{PlanID: planID, ClinicID: ws, Payload: payload}}, pdf.NewRenderer(), nil, docs, pdf.AwardJPEG())

	doc, err := svc.Generate(context.Background(), ws, planID)
	if err != nil {
		t.Fatal(err)
	}
	if doc.PageCount != 24 || doc.ContentType != domain.ContentTypePDF {
		t.Fatalf("%+v", doc)
	}
	got, err := svc.Get(context.Background(), ws, planID)
	if err != nil || got.PageCount != 24 {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestGenerateRequiresReadySnapshot(t *testing.T) {
	ws, planID := id.New(), id.New()
	svc := app.NewService(stubPlans{view: domain.PlanView{Payload: []byte(`{}`)}}, pdf.NewRenderer(), nil, mem.New(), pdf.AwardJPEG())
	_, err := svc.Generate(context.Background(), ws, planID)
	if err == nil || !errors.Is(err, errx.ErrInvalid) {
		t.Fatalf("got %v", err)
	}
}
