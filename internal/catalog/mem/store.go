package mem

import (
	"context"
	"sync"

	"github.com/treatment-plus/treatment-plus-be/internal/catalog/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/ctxkey"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
)

type Store struct {
	mu         sync.RWMutex
	diagnoses  map[id.ID]domain.Diagnosis
	treatments map[id.ID]domain.Treatment
	priceLists map[id.ID]domain.PriceList
	priceItems map[id.ID][]domain.PriceItem
}

func New() *Store {
	return &Store{
		diagnoses:  make(map[id.ID]domain.Diagnosis),
		treatments: make(map[id.ID]domain.Treatment),
		priceLists: make(map[id.ID]domain.PriceList),
		priceItems: make(map[id.ID][]domain.PriceItem),
	}
}

func denyRead(ctx context.Context, wsID id.ID) bool {
	raw := ctxkey.ClinicID(ctx)
	return raw != "" && raw != wsID.String()
}

func denyWrite(ctx context.Context, wsID id.ID) error {
	raw := ctxkey.ClinicID(ctx)
	if raw != "" && raw != wsID.String() {
		return errx.ErrForbidden
	}
	return nil
}

func (s *Store) ListDiagnoses(ctx context.Context, wsID id.ID, includeInactive bool) ([]domain.Diagnosis, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return nil, nil
	}
	var out []domain.Diagnosis
	for _, d := range s.diagnoses {
		if d.WorkspaceID != wsID {
			continue
		}
		if !includeInactive && !d.IsActive {
			continue
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *Store) GetDiagnosis(ctx context.Context, wsID id.ID, diagID id.ID) (domain.Diagnosis, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return domain.Diagnosis{}, errx.ErrNotFound
	}
	d, ok := s.diagnoses[diagID]
	if !ok || d.WorkspaceID != wsID {
		return domain.Diagnosis{}, errx.ErrNotFound
	}
	return d, nil
}

func (s *Store) SaveDiagnosis(ctx context.Context, d domain.Diagnosis) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, d.WorkspaceID); err != nil {
		return err
	}
	for idKey, existing := range s.diagnoses {
		if existing.WorkspaceID == d.WorkspaceID && existing.Code == d.Code {
			existing.Name = d.Name
			existing.IsActive = d.IsActive
			s.diagnoses[idKey] = existing
			return nil
		}
	}
	s.diagnoses[d.ID] = d
	return nil
}

func (s *Store) ListTreatments(ctx context.Context, wsID id.ID, includeInactive bool) ([]domain.Treatment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return nil, nil
	}
	var out []domain.Treatment
	for _, t := range s.treatments {
		if t.WorkspaceID != wsID {
			continue
		}
		if !includeInactive && !t.IsActive {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *Store) GetTreatment(ctx context.Context, wsID id.ID, treatID id.ID) (domain.Treatment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return domain.Treatment{}, errx.ErrNotFound
	}
	t, ok := s.treatments[treatID]
	if !ok || t.WorkspaceID != wsID {
		return domain.Treatment{}, errx.ErrNotFound
	}
	return t, nil
}

func (s *Store) SaveTreatment(ctx context.Context, t domain.Treatment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, t.WorkspaceID); err != nil {
		return err
	}
	for idKey, existing := range s.treatments {
		if existing.WorkspaceID == t.WorkspaceID && existing.Code == t.Code {
			existing.Name = t.Name
			existing.IsActive = t.IsActive
			s.treatments[idKey] = existing
			return nil
		}
	}
	s.treatments[t.ID] = t
	return nil
}

func (s *Store) ListPriceLists(ctx context.Context, wsID id.ID) ([]domain.PriceList, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return nil, nil
	}
	var out []domain.PriceList
	for _, pl := range s.priceLists {
		if pl.WorkspaceID == wsID {
			out = append(out, pl)
		}
	}
	return out, nil
}

func (s *Store) GetPriceList(ctx context.Context, wsID id.ID, listID id.ID) (domain.PriceList, []domain.PriceItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return domain.PriceList{}, nil, errx.ErrNotFound
	}
	pl, ok := s.priceLists[listID]
	if !ok || pl.WorkspaceID != wsID {
		return domain.PriceList{}, nil, errx.ErrNotFound
	}
	items := append([]domain.PriceItem(nil), s.priceItems[listID]...)
	return pl, items, nil
}

func (s *Store) GetDefaultPriceList(ctx context.Context, wsID id.ID) (domain.PriceList, []domain.PriceItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if denyRead(ctx, wsID) {
		return domain.PriceList{}, nil, errx.ErrNotFound
	}
	for _, pl := range s.priceLists {
		if pl.WorkspaceID == wsID && pl.IsDefault {
			items := append([]domain.PriceItem(nil), s.priceItems[pl.ID]...)
			return pl, items, nil
		}
	}
	return domain.PriceList{}, nil, errx.ErrNotFound
}

func (s *Store) SavePriceList(ctx context.Context, pl domain.PriceList) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, pl.WorkspaceID); err != nil {
		return err
	}
	if pl.IsDefault {
		for idKey, existing := range s.priceLists {
			if existing.WorkspaceID == pl.WorkspaceID && existing.ID != pl.ID {
				existing.IsDefault = false
				s.priceLists[idKey] = existing
			}
		}
	}
	s.priceLists[pl.ID] = pl
	return nil
}

func (s *Store) SavePriceItems(ctx context.Context, wsID id.ID, listID id.ID, items []domain.PriceItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := denyWrite(ctx, wsID); err != nil {
		return err
	}
	pl, ok := s.priceLists[listID]
	if !ok || pl.WorkspaceID != wsID {
		return errx.ErrNotFound
	}
	merged := append([]domain.PriceItem(nil), s.priceItems[listID]...)
	for _, incoming := range items {
		replaced := false
		for i, existing := range merged {
			if existing.TreatmentID == incoming.TreatmentID {
				merged[i] = incoming
				replaced = true
				break
			}
		}
		if !replaced {
			merged = append(merged, incoming)
		}
	}
	s.priceItems[listID] = merged
	return nil
}
