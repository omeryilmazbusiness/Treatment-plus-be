-- Phase 3 & 4: Catalog & Plan (Classic Form)

-- 1. Diagnoses
CREATE TABLE diagnoses (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    code text NOT NULL,
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT diagnoses_workspace_code_key UNIQUE (workspace_id, code)
);

CREATE TRIGGER diagnoses_set_updated_at
    BEFORE UPDATE ON diagnoses
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE diagnoses ENABLE ROW LEVEL SECURITY;
ALTER TABLE diagnoses FORCE ROW LEVEL SECURITY;

CREATE POLICY diagnoses_tenant ON diagnoses
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- 2. Treatments
CREATE TABLE treatments (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    code text NOT NULL,
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT treatments_workspace_code_key UNIQUE (workspace_id, code)
);

CREATE TRIGGER treatments_set_updated_at
    BEFORE UPDATE ON treatments
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE treatments ENABLE ROW LEVEL SECURITY;
ALTER TABLE treatments FORCE ROW LEVEL SECURITY;

CREATE POLICY treatments_tenant ON treatments
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- 3. Price Lists & Items (integer minor units)
CREATE TABLE price_lists (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    name text NOT NULL,
    currency text NOT NULL DEFAULT 'TRY',
    is_default boolean NOT NULL DEFAULT false,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER price_lists_set_updated_at
    BEFORE UPDATE ON price_lists
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE price_lists ENABLE ROW LEVEL SECURITY;
ALTER TABLE price_lists FORCE ROW LEVEL SECURITY;

CREATE POLICY price_lists_tenant ON price_lists
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

CREATE TABLE price_items (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    price_list_id uuid NOT NULL REFERENCES price_lists (id) ON DELETE CASCADE,
    treatment_id uuid NOT NULL REFERENCES treatments (id) ON DELETE CASCADE,
    amount bigint NOT NULL CHECK (amount >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT price_items_list_treatment_key UNIQUE (price_list_id, treatment_id)
);

CREATE TRIGGER price_items_set_updated_at
    BEFORE UPDATE ON price_items
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE price_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE price_items FORCE ROW LEVEL SECURITY;

CREATE POLICY price_items_tenant ON price_items
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- 4. Patients (encrypted contact)
CREATE TABLE patients (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    name text NOT NULL,
    contact_encrypted text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER patients_set_updated_at
    BEFORE UPDATE ON patients
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE patients ENABLE ROW LEVEL SECURITY;
ALTER TABLE patients FORCE ROW LEVEL SECURITY;

CREATE POLICY patients_tenant ON patients
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- 5. Plans (kind=classic, status draft|ready, totals, snapshot_json)
CREATE TABLE plans (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    patient_id uuid NOT NULL REFERENCES patients (id) ON DELETE RESTRICT,
    title text NOT NULL DEFAULT '',
    kind text NOT NULL DEFAULT 'classic' CHECK (kind = 'classic'),
    status text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'ready')),
    currency text NOT NULL DEFAULT 'TRY',
    diagnoses jsonb NOT NULL DEFAULT '[]'::jsonb,
    totals_subtotal bigint NOT NULL DEFAULT 0,
    totals_discount bigint NOT NULL DEFAULT 0,
    totals_grand bigint NOT NULL DEFAULT 0,
    snapshot_json jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX plans_workspace_created_idx ON plans (workspace_id, created_at DESC);
CREATE INDEX plans_patient_idx ON plans (patient_id);

CREATE TRIGGER plans_set_updated_at
    BEFORE UPDATE ON plans
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE plans FORCE ROW LEVEL SECURITY;

CREATE POLICY plans_tenant ON plans
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- 6. Plan Visits
CREATE TABLE plan_visits (
    id uuid PRIMARY KEY,
    plan_id uuid NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    order_num int NOT NULL DEFAULT 0,
    title text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX plan_visits_plan_order_idx ON plan_visits (plan_id, order_num);

CREATE TRIGGER plan_visits_set_updated_at
    BEFORE UPDATE ON plan_visits
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE plan_visits ENABLE ROW LEVEL SECURITY;
ALTER TABLE plan_visits FORCE ROW LEVEL SECURITY;

CREATE POLICY plan_visits_tenant ON plan_visits
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- 7. Plan Items (treatments, tooth_number, qty, unit_price minor unit, is_package)
CREATE TABLE plan_items (
    id uuid PRIMARY KEY,
    plan_id uuid NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    visit_id uuid REFERENCES plan_visits (id) ON DELETE SET NULL,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    treatment_id uuid REFERENCES treatments (id) ON DELETE SET NULL,
    treatment_name text NOT NULL DEFAULT '',
    tooth_number text,
    quantity int NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit_price bigint NOT NULL DEFAULT 0 CHECK (unit_price >= 0),
    is_package boolean NOT NULL DEFAULT false,
    order_num int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX plan_items_plan_order_idx ON plan_items (plan_id, order_num);

CREATE TRIGGER plan_items_set_updated_at
    BEFORE UPDATE ON plan_items
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE plan_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE plan_items FORCE ROW LEVEL SECURITY;

CREATE POLICY plan_items_tenant ON plan_items
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- 8. Plan Discounts
CREATE TABLE plan_discounts (
    id uuid PRIMARY KEY,
    plan_id uuid NOT NULL REFERENCES plans (id) ON DELETE CASCADE,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    label text NOT NULL,
    amount bigint NOT NULL DEFAULT 0 CHECK (amount >= 0),
    order_num int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX plan_discounts_plan_order_idx ON plan_discounts (plan_id, order_num);

CREATE TRIGGER plan_discounts_set_updated_at
    BEFORE UPDATE ON plan_discounts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE plan_discounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE plan_discounts FORCE ROW LEVEL SECURITY;

CREATE POLICY plan_discounts_tenant ON plan_discounts
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- Grants to treatment_app
GRANT SELECT, INSERT, UPDATE, DELETE ON diagnoses, treatments, price_lists, price_items, patients, plans, plan_visits, plan_items, plan_discounts TO treatment_app;
