-- Clinic-owned images and generated Classic PDFs.

CREATE TABLE media_assets (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    object_key text NOT NULL,
    kind text NOT NULL,
    content_type text NOT NULL,
    bytes bytea NOT NULL,
    byte_size integer NOT NULL CHECK (byte_size >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT media_assets_workspace_key UNIQUE (workspace_id, object_key)
);

CREATE INDEX media_assets_workspace_idx ON media_assets (workspace_id, created_at DESC);

ALTER TABLE media_assets ENABLE ROW LEVEL SECURITY;
ALTER TABLE media_assets FORCE ROW LEVEL SECURITY;

CREATE POLICY media_assets_tenant ON media_assets
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

CREATE TABLE plan_documents (
    plan_id uuid PRIMARY KEY REFERENCES plans (id) ON DELETE CASCADE,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    content_type text NOT NULL DEFAULT 'application/pdf',
    page_count integer NOT NULL CHECK (page_count > 0),
    byte_size integer NOT NULL CHECK (byte_size > 0),
    bytes bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX plan_documents_workspace_idx ON plan_documents (workspace_id);

CREATE TRIGGER plan_documents_set_updated_at
    BEFORE UPDATE ON plan_documents
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE plan_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE plan_documents FORCE ROW LEVEL SECURITY;

CREATE POLICY plan_documents_tenant ON plan_documents
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

GRANT SELECT, INSERT, UPDATE, DELETE ON media_assets, plan_documents TO treatment_app;
