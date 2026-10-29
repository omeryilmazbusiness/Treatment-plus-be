package pg

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/treatment-plus/treatment-plus-be/internal/plan/domain"
	"github.com/treatment-plus/treatment-plus-be/internal/platform/rls"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/errx"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/id"
	"github.com/treatment-plus/treatment-plus-be/internal/shared/money"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Patients
func (s *Store) ListPatients(ctx context.Context, wsID id.ID) ([]domain.Patient, error) {
	var out []domain.Patient
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT id, workspace_id, name, contact_encrypted, created_at, updated_at
			FROM patients
			WHERE workspace_id = $1
			ORDER BY name ASC
		`, toUUID(wsID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var p domain.Patient
			var pID, wsUUID pgtype.UUID
			if err := rows.Scan(&pID, &wsUUID, &p.Name, &p.Contact, &p.CreatedAt, &p.UpdatedAt); err != nil {
				return err
			}
			p.ID = fromUUID(pID)
			p.WorkspaceID = fromUUID(wsUUID)
			out = append(out, p)
		}
		return rows.Err()
	})
	return out, err
}

func (s *Store) GetPatient(ctx context.Context, wsID id.ID, patientID id.ID) (domain.Patient, error) {
	var out domain.Patient
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		var pID, wsUUID pgtype.UUID
		row := tx.QueryRow(ctx, `
			SELECT id, workspace_id, name, contact_encrypted, created_at, updated_at
			FROM patients
			WHERE workspace_id = $1 AND id = $2
		`, toUUID(wsID), toUUID(patientID))
		if err := row.Scan(&pID, &wsUUID, &out.Name, &out.Contact, &out.CreatedAt, &out.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}
		out.ID = fromUUID(pID)
		out.WorkspaceID = fromUUID(wsUUID)
		return nil
	})
	return out, err
}

func (s *Store) SavePatient(ctx context.Context, p domain.Patient, encryptedContact string) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO patients (id, workspace_id, name, contact_encrypted)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				contact_encrypted = EXCLUDED.contact_encrypted,
				updated_at = now()
		`, toUUID(p.ID), toUUID(p.WorkspaceID), p.Name, encryptedContact)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrForbidden
		}
		return nil
	})
}

// Plans
func (s *Store) ListPlans(ctx context.Context, wsID id.ID) ([]domain.Plan, error) {
	var out []domain.Plan
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
			SELECT p.id, p.workspace_id, p.patient_id, p.title, p.kind, p.status, p.currency,
			       p.palette, p.locale, p.diagnoses, p.situation_json,
			       p.welcome_letter, p.video_url, p.before_after_case_ids,
			       p.totals_subtotal, p.totals_discount, p.totals_grand,
			       p.snapshot_json, p.created_at, p.updated_at,
			       pat.name, pat.contact_encrypted
			FROM plans p
			LEFT JOIN patients pat ON pat.id = p.patient_id
			WHERE p.workspace_id = $1
			ORDER BY p.created_at DESC
		`, toUUID(wsID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var pl domain.Plan
			var pID, wsUUID, patID pgtype.UUID
			var kind, status, curr, palette, locale string
			var welcomeLetter, videoURL string
			var baCaseIDs []string
			var diagJSON []byte
			var sitJSON []byte
			var snapJSON []byte
			var patName, patContact *string

			if err := rows.Scan(
				&pID, &wsUUID, &patID, &pl.Title, &kind, &status, &curr,
				&palette, &locale, &diagJSON, &sitJSON,
				&welcomeLetter, &videoURL, &baCaseIDs,
				&pl.TotalsSubtotal, &pl.TotalsDiscount, &pl.TotalsGrand,
				&snapJSON, &pl.CreatedAt, &pl.UpdatedAt,
				&patName, &patContact,
			); err != nil {
				return err
			}
			pl.ID = fromUUID(pID)
			pl.WorkspaceID = fromUUID(wsUUID)
			pl.PatientID = fromUUID(patID)
			pl.Kind = domain.PlanKind(kind)
			pl.Status = domain.PlanStatus(status)
			pl.Currency = money.Currency(curr)
			pl.Palette, _ = domain.ParsePalette(palette)
			pl.Locale, _ = domain.ParseLocale(locale)
			pl.WelcomeLetter = welcomeLetter
			pl.VideoURL = videoURL
			pl.BeforeAfterCaseIDs = baCaseIDs

			if len(diagJSON) > 0 {
				_ = json.Unmarshal(diagJSON, &pl.Diagnoses)
			}
			if len(sitJSON) > 0 {
				_ = json.Unmarshal(sitJSON, &pl.Situation)
			}
			if len(snapJSON) > 0 {
				_ = json.Unmarshal(snapJSON, &pl.SnapshotJSON)
			}
			if patName != nil {
				var contactStr string
				if patContact != nil {
					contactStr = *patContact
				}
				pl.Patient = &domain.Patient{
					ID:          pl.PatientID,
					WorkspaceID: pl.WorkspaceID,
					Name:        *patName,
					Contact:     contactStr,
				}
			}
			out = append(out, pl)
		}

		if len(out) > 0 {
			vRows, err := tx.Query(ctx, `
				SELECT id, plan_id, workspace_id, order_num, title, phase_label, healing_note, chart_teeth, created_at, updated_at
				FROM plan_visits
				WHERE workspace_id = $1
				ORDER BY order_num ASC
			`, toUUID(wsID))
			if err == nil {
				defer vRows.Close()
				planMap := make(map[id.ID]*domain.Plan, len(out))
				for i := range out {
					planMap[out[i].ID] = &out[i]
				}
				for vRows.Next() {
					var v domain.Visit
					var vID, vPlanID, vWsUUID pgtype.UUID
					if err := vRows.Scan(&vID, &vPlanID, &vWsUUID, &v.OrderNum, &v.Title, &v.PhaseLabel, &v.HealingNote, &v.ChartTeeth, &v.CreatedAt, &v.UpdatedAt); err == nil {
						v.ID = fromUUID(vID)
						v.PlanID = fromUUID(vPlanID)
						v.WorkspaceID = fromUUID(vWsUUID)
						if target, ok := planMap[v.PlanID]; ok {
							target.Visits = append(target.Visits, v)
						}
					}
				}
			}

			iRows, err := tx.Query(ctx, `
				SELECT id, plan_id, visit_id, workspace_id, treatment_id, treatment_name,
				       tooth_number, teeth, description, quantity, unit_price, is_package, order_num, animation_code, created_at, updated_at
				FROM plan_items
				WHERE workspace_id = $1
				ORDER BY order_num ASC
			`, toUUID(wsID))
			if err == nil {
				defer iRows.Close()
				planMap := make(map[id.ID]*domain.Plan, len(out))
				for i := range out {
					planMap[out[i].ID] = &out[i]
				}
				for iRows.Next() {
					var it domain.Item
					var itID, itPlanID, itWsUUID pgtype.UUID
					var itVisitID, itTrID pgtype.UUID
					if err := iRows.Scan(
						&itID, &itPlanID, &itVisitID, &itWsUUID, &itTrID, &it.TreatmentName,
						&it.ToothNumber, &it.Teeth, &it.Description, &it.Quantity, &it.UnitPrice, &it.IsPackage, &it.OrderNum,
						&it.AnimationCode, &it.CreatedAt, &it.UpdatedAt,
					); err == nil {
						it.ID = fromUUID(itID)
						it.PlanID = fromUUID(itPlanID)
						it.WorkspaceID = fromUUID(itWsUUID)
						if itVisitID.Valid {
							v := fromUUID(itVisitID)
							it.VisitID = &v
						}
						if itTrID.Valid {
							tr := fromUUID(itTrID)
							it.TreatmentID = &tr
						}
						if target, ok := planMap[it.PlanID]; ok {
							target.Items = append(target.Items, it)
						}
					}
				}
			}
		}
		return rows.Err()
	})
	return out, err
}

func (s *Store) GetPlan(ctx context.Context, wsID id.ID, planID id.ID) (domain.Plan, error) {
	var pl domain.Plan
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		var pID, wsUUID, patID pgtype.UUID
		var kind, status, curr, palette, locale string
		var welcomeLetter, videoURL string
		var baCaseIDs []string
		var diagJSON []byte
		var sitJSON []byte
		var snapJSON []byte

		row := tx.QueryRow(ctx, `
			SELECT id, workspace_id, patient_id, title, kind, status, currency,
			       palette, locale, diagnoses, situation_json,
			       welcome_letter, video_url, before_after_case_ids,
			       totals_subtotal, totals_discount, totals_grand,
			       snapshot_json, created_at, updated_at
			FROM plans
			WHERE workspace_id = $1 AND id = $2
		`, toUUID(wsID), toUUID(planID))

		if err := row.Scan(
			&pID, &wsUUID, &patID, &pl.Title, &kind, &status, &curr,
			&palette, &locale, &diagJSON, &sitJSON,
			&welcomeLetter, &videoURL, &baCaseIDs,
			&pl.TotalsSubtotal, &pl.TotalsDiscount, &pl.TotalsGrand,
			&snapJSON, &pl.CreatedAt, &pl.UpdatedAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errx.ErrNotFound
			}
			return err
		}

		pl.ID = fromUUID(pID)
		pl.WorkspaceID = fromUUID(wsUUID)
		pl.PatientID = fromUUID(patID)
		pl.Kind = domain.PlanKind(kind)
		pl.Status = domain.PlanStatus(status)
		pl.Currency = money.Currency(curr)
		pl.Palette, _ = domain.ParsePalette(palette)
		pl.Locale, _ = domain.ParseLocale(locale)
		pl.WelcomeLetter = welcomeLetter
		pl.VideoURL = videoURL
		pl.BeforeAfterCaseIDs = baCaseIDs

		if len(diagJSON) > 0 {
			_ = json.Unmarshal(diagJSON, &pl.Diagnoses)
		}
		if len(sitJSON) > 0 {
			_ = json.Unmarshal(sitJSON, &pl.Situation)
		}
		if len(snapJSON) > 0 {
			_ = json.Unmarshal(snapJSON, &pl.SnapshotJSON)
		}

		// Visits
		vRows, err := tx.Query(ctx, `
			SELECT id, plan_id, workspace_id, order_num, title, phase_label, healing_note, chart_teeth, created_at, updated_at
			FROM plan_visits
			WHERE workspace_id = $1 AND plan_id = $2
			ORDER BY order_num ASC
		`, toUUID(wsID), toUUID(planID))
		if err != nil {
			return err
		}
		defer vRows.Close()

		for vRows.Next() {
			var v domain.Visit
			var vID, vPlanID, vWsUUID pgtype.UUID
			if err := vRows.Scan(&vID, &vPlanID, &vWsUUID, &v.OrderNum, &v.Title, &v.PhaseLabel, &v.HealingNote, &v.ChartTeeth, &v.CreatedAt, &v.UpdatedAt); err != nil {
				return err
			}
			v.ID = fromUUID(vID)
			v.PlanID = fromUUID(vPlanID)
			v.WorkspaceID = fromUUID(vWsUUID)
			pl.Visits = append(pl.Visits, v)
		}
		if err := vRows.Err(); err != nil {
			return err
		}

		// Items
		iRows, err := tx.Query(ctx, `
			SELECT id, plan_id, visit_id, workspace_id, treatment_id, treatment_name,
			       tooth_number, teeth, description, quantity, unit_price, is_package, order_num, animation_code, created_at, updated_at
			FROM plan_items
			WHERE workspace_id = $1 AND plan_id = $2
			ORDER BY order_num ASC
		`, toUUID(wsID), toUUID(planID))
		if err != nil {
			return err
		}
		defer iRows.Close()

		for iRows.Next() {
			var it domain.Item
			var itID, itPlanID, itWsUUID pgtype.UUID
			var itVisitID, itTrID pgtype.UUID
			if err := iRows.Scan(
				&itID, &itPlanID, &itVisitID, &itWsUUID, &itTrID, &it.TreatmentName,
				&it.ToothNumber, &it.Teeth, &it.Description, &it.Quantity, &it.UnitPrice, &it.IsPackage, &it.OrderNum,
				&it.AnimationCode, &it.CreatedAt, &it.UpdatedAt,
			); err != nil {
				return err
			}
			it.ID = fromUUID(itID)
			it.PlanID = fromUUID(itPlanID)
			it.WorkspaceID = fromUUID(itWsUUID)
			if itVisitID.Valid {
				v := fromUUID(itVisitID)
				it.VisitID = &v
			}
			if itTrID.Valid {
				tr := fromUUID(itTrID)
				it.TreatmentID = &tr
			}
			pl.Items = append(pl.Items, it)
		}
		if err := iRows.Err(); err != nil {
			return err
		}

		// Discounts
		dRows, err := tx.Query(ctx, `
			SELECT id, plan_id, workspace_id, label, amount, order_num, created_at, updated_at
			FROM plan_discounts
			WHERE workspace_id = $1 AND plan_id = $2
			ORDER BY order_num ASC
		`, toUUID(wsID), toUUID(planID))
		if err != nil {
			return err
		}
		defer dRows.Close()

		for dRows.Next() {
			var d domain.Discount
			var dID, dPlanID, dWsUUID pgtype.UUID
			if err := dRows.Scan(&dID, &dPlanID, &dWsUUID, &d.Label, &d.Amount, &d.OrderNum, &d.CreatedAt, &d.UpdatedAt); err != nil {
				return err
			}
			d.ID = fromUUID(dID)
			d.PlanID = fromUUID(dPlanID)
			d.WorkspaceID = fromUUID(dWsUUID)
			pl.Discounts = append(pl.Discounts, d)
		}
		return dRows.Err()
	})
	return pl, err
}

func (s *Store) CreatePlan(ctx context.Context, p domain.Plan) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		diagJSON, _ := json.Marshal(p.Diagnoses)
		sitJSON, _ := json.Marshal(p.Situation)
		baCaseIDs := p.BeforeAfterCaseIDs
		if baCaseIDs == nil {
			baCaseIDs = []string{}
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO plans (
				id, workspace_id, patient_id, title, kind, status, currency,
				palette, locale, diagnoses, situation_json,
				welcome_letter, video_url, before_after_case_ids,
				totals_subtotal, totals_discount, totals_grand
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		`,
			toUUID(p.ID),
			toUUID(p.WorkspaceID),
			toUUID(p.PatientID),
			p.Title,
			string(p.Kind),
			string(p.Status),
			string(p.Currency),
			string(p.Palette),
			string(p.Locale),
			diagJSON,
			sitJSON,
			p.WelcomeLetter,
			p.VideoURL,
			baCaseIDs,
			p.TotalsSubtotal,
			p.TotalsDiscount,
			p.TotalsGrand,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrForbidden
		}
		return nil
	})
}

func (s *Store) UpdatePlanHeader(ctx context.Context, p domain.Plan) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		diagJSON, _ := json.Marshal(p.Diagnoses)
		sitJSON, _ := json.Marshal(p.Situation)
		baCaseIDs := p.BeforeAfterCaseIDs
		if baCaseIDs == nil {
			baCaseIDs = []string{}
		}
		tag, err := tx.Exec(ctx, `
			UPDATE plans
			SET title = $3, diagnoses = $4, situation_json = $5, palette = $6, locale = $7, currency = $8,
			    totals_subtotal = $9, totals_discount = $10, totals_grand = $11,
			    welcome_letter = $12, video_url = $13, before_after_case_ids = $14,
			    updated_at = now()
			WHERE workspace_id = $1 AND id = $2 AND status = 'draft'
		`, toUUID(p.WorkspaceID), toUUID(p.ID), p.Title, diagJSON, sitJSON, string(p.Palette), string(p.Locale), string(p.Currency),
			p.TotalsSubtotal, p.TotalsDiscount, p.TotalsGrand,
			p.WelcomeLetter, p.VideoURL, baCaseIDs)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.ErrNotFound
		}
		return nil
	})
}

func (s *Store) ReplaceVisits(ctx context.Context, wsID id.ID, planID id.ID, visits []domain.Visit) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		// Delete existing
		_, err := tx.Exec(ctx, `
			DELETE FROM plan_visits WHERE workspace_id = $1 AND plan_id = $2
		`, toUUID(wsID), toUUID(planID))
		if err != nil {
			return err
		}

		for _, v := range visits {
			teeth := v.ChartTeeth
			if teeth == nil {
				teeth = []string{}
			}
			_, err := tx.Exec(ctx, `
				INSERT INTO plan_visits (id, plan_id, workspace_id, order_num, title, phase_label, healing_note, chart_teeth)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`, toUUID(v.ID), toUUID(planID), toUUID(wsID), v.OrderNum, v.Title, v.PhaseLabel, v.HealingNote, teeth)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) ReplaceItems(ctx context.Context, wsID id.ID, planID id.ID, items []domain.Item, totals domain.Totals) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		// Delete existing
		_, err := tx.Exec(ctx, `
			DELETE FROM plan_items WHERE workspace_id = $1 AND plan_id = $2
		`, toUUID(wsID), toUUID(planID))
		if err != nil {
			return err
		}

		for _, it := range items {
			var vUUID, trUUID *pgtype.UUID
			if it.VisitID != nil {
				u := toUUID(*it.VisitID)
				vUUID = &u
			}
			if it.TreatmentID != nil {
				u := toUUID(*it.TreatmentID)
				trUUID = &u
			}
			teeth := it.Teeth
			if teeth == nil {
				teeth = []string{}
			}
			_, err := tx.Exec(ctx, `
				INSERT INTO plan_items (
					id, plan_id, visit_id, workspace_id, treatment_id, treatment_name,
					tooth_number, teeth, description, quantity, unit_price, is_package, order_num, animation_code
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			`,
				toUUID(it.ID), toUUID(planID), vUUID, toUUID(wsID), trUUID, it.TreatmentName,
				it.ToothNumber, teeth, it.Description, it.Quantity, it.UnitPrice, it.IsPackage, it.OrderNum, it.AnimationCode,
			)
			if err != nil {
				return err
			}
		}

		// Update totals on plan
		_, err = tx.Exec(ctx, `
			UPDATE plans
			SET totals_subtotal = $3, totals_discount = $4, totals_grand = $5, updated_at = now()
			WHERE workspace_id = $1 AND id = $2 AND status = 'draft'
		`, toUUID(wsID), toUUID(planID), totals.Subtotal, totals.Discount, totals.Grand)
		return err
	})
}

func (s *Store) ReplaceDiscounts(ctx context.Context, wsID id.ID, planID id.ID, discounts []domain.Discount, totals domain.Totals) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		// Delete existing
		_, err := tx.Exec(ctx, `
			DELETE FROM plan_discounts WHERE workspace_id = $1 AND plan_id = $2
		`, toUUID(wsID), toUUID(planID))
		if err != nil {
			return err
		}

		for _, d := range discounts {
			_, err := tx.Exec(ctx, `
				INSERT INTO plan_discounts (id, plan_id, workspace_id, label, amount, order_num)
				VALUES ($1, $2, $3, $4, $5, $6)
			`, toUUID(d.ID), toUUID(planID), toUUID(wsID), d.Label, d.Amount, d.OrderNum)
			if err != nil {
				return err
			}
		}

		// Update totals on plan
		_, err = tx.Exec(ctx, `
			UPDATE plans
			SET totals_subtotal = $3, totals_discount = $4, totals_grand = $5, updated_at = now()
			WHERE workspace_id = $1 AND id = $2 AND status = 'draft'
		`, toUUID(wsID), toUUID(planID), totals.Subtotal, totals.Discount, totals.Grand)
		return err
	})
}

func (s *Store) SetReady(ctx context.Context, wsID id.ID, planID id.ID, snapshotJSON []byte) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := rls.Apply(ctx, tx); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE plans
			SET status = 'ready', snapshot_json = $3, updated_at = now()
			WHERE workspace_id = $1 AND id = $2 AND status = 'draft'
		`, toUUID(wsID), toUUID(planID), snapshotJSON)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errx.New("cannot_set_ready", "plan not found or already ready", errx.ErrNotFound)
		}
		return nil
	})
}

func toUUID(v id.ID) pgtype.UUID {
	var u pgtype.UUID
	copy(u.Bytes[:], v.Bytes())
	u.Valid = true
	return u
}

func fromUUID(u pgtype.UUID) id.ID {
	if !u.Valid {
		return id.ID{}
	}
	out, err := id.FromBytes(u.Bytes[:])
	if err != nil {
		return id.ID{}
	}
	return out
}
