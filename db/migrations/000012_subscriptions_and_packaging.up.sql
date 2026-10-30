-- Phase: Subscriptions, Tiers (Classic vs Ultimate), Usage Quotas & Demo Requests

-- 1. Subscription Plans catalog (system-wide pricing tiers)
CREATE TABLE subscription_plans (
    id text PRIMARY KEY,
    tier text NOT NULL CHECK (tier IN ('classic', 'ultimate', 'enterprise')),
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    billing_interval text NOT NULL CHECK (billing_interval IN ('month', 'year')),
    price_cents bigint NOT NULL,
    currency text NOT NULL DEFAULT 'EUR',
    plan_quota integer NOT NULL DEFAULT -1, -- -1 means unlimited
    has_3d_viz boolean NOT NULL DEFAULT false,
    has_patient_portal boolean NOT NULL DEFAULT false,
    has_telemetry boolean NOT NULL DEFAULT false,
    has_digital_signature boolean NOT NULL DEFAULT false,
    has_onboarding_session boolean NOT NULL DEFAULT false,
    features jsonb NOT NULL DEFAULT '[]'::jsonb,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Seed default plans based on bright-plans.com
INSERT INTO subscription_plans (
    id, tier, name, description, billing_interval, price_cents, currency, plan_quota,
    has_3d_viz, has_patient_portal, has_telemetry, has_digital_signature, has_onboarding_session,
    features
) VALUES
(
    'classic_monthly',
    'classic',
    'BrightPlans Classic (Monthly)',
    'Standard treatment planning with 24-page high-definition PDF outputs for documentation and print.',
    'month',
    4900,
    'EUR',
    30,
    false,
    false,
    false,
    false,
    false,
    '["24-Page Premium PDF Presentation", "Clinic Brand & Doctor Bio", "Clinical Findings & Odontogram", "Cost & Procedure Tables", "30 Plans per month"]'::jsonb
),
(
    'classic_yearly',
    'classic',
    'BrightPlans Classic (Yearly)',
    'Annual Classic subscription with 2 months free.',
    'year',
    49000,
    'EUR',
    50,
    false,
    false,
    false,
    false,
    true,
    '["Everything in Classic Monthly", "2 Months Free", "50 Plans per month", "Priority Clinic Onboarding"]'::jsonb
),
(
    'ultimate_monthly',
    'ultimate',
    'BrightPlans Ultimate (Monthly)',
    'Everything in Classic plus full mobile-first interactive experience, 3D simulations and instant analytics.',
    'month',
    9900,
    'EUR',
    100,
    true,
    true,
    true,
    true,
    false,
    '["Everything in Classic", "Interactive Mobile Web Experience (/p/{token})", "3D Dental Simulation Clips", "Personalized Video & Welcome Letter", "Before & After Interactive Slider", "Real-Time Patient Engagement Telemetry", "One-Click Digital Acceptance & Signature", "Direct WhatsApp & Call Hub", "100 Plans per month"]'::jsonb
),
(
    'ultimate_yearly',
    'ultimate',
    'BrightPlans Ultimate (Yearly)',
    'Complete Ultimate experience with 2 months free, 2-hour dedicated onboarding session and unlimited plans.',
    'year',
    99000,
    'EUR',
    -1,
    true,
    true,
    true,
    true,
    true,
    '["Everything in Ultimate Monthly", "2 Months Free", "Unlimited Treatment Plans", "2-Hour Dedicated 1-on-1 Onboarding Session", "Custom Template & Pricing List Setup", "VIP Support"]'::jsonb
);

-- 2. Clinic Subscriptions (Tenant-isolated subscription state)
CREATE TABLE clinic_subscriptions (
    id uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES clinics (id) ON DELETE CASCADE,
    plan_id text NOT NULL REFERENCES subscription_plans (id),
    status text NOT NULL CHECK (status IN ('trialing', 'active', 'past_due', 'canceled', 'paused')),
    current_period_start timestamptz NOT NULL DEFAULT now(),
    current_period_end timestamptz NOT NULL,
    cancel_at_period_end boolean NOT NULL DEFAULT false,
    canceled_at timestamptz,
    plans_created_count integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT clinic_subscriptions_workspace_unique UNIQUE (workspace_id)
);

CREATE INDEX clinic_subscriptions_workspace_idx ON clinic_subscriptions (workspace_id);
CREATE INDEX clinic_subscriptions_plan_idx ON clinic_subscriptions (plan_id);

CREATE TRIGGER clinic_subscriptions_set_updated_at
    BEFORE UPDATE ON clinic_subscriptions
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

ALTER TABLE clinic_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE clinic_subscriptions FORCE ROW LEVEL SECURITY;

CREATE POLICY clinic_subscriptions_tenant ON clinic_subscriptions
    USING (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid)
    WITH CHECK (workspace_id = NULLIF(current_setting('app.clinic_id', true), '')::uuid);

-- 3. Demo Requests (Lead Funnel from "Please Design My Demo")
CREATE TABLE demo_requests (
    id uuid PRIMARY KEY,
    website_url text NOT NULL,
    clinic_name text NOT NULL DEFAULT '',
    contact_name text NOT NULL DEFAULT '',
    email text NOT NULL DEFAULT '',
    phone text NOT NULL DEFAULT '',
    notes text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'contacted', 'completed', 'rejected')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX demo_requests_created_at_idx ON demo_requests (created_at DESC);

-- 4. Permissions
GRANT SELECT ON subscription_plans TO treatment_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON clinic_subscriptions TO treatment_app;
GRANT SELECT, INSERT, UPDATE ON demo_requests TO treatment_app;

-- 5. Auto-provision existing clinics with active Ultimate Yearly subscription
INSERT INTO clinic_subscriptions (
    id, workspace_id, plan_id, status, current_period_start, current_period_end, cancel_at_period_end, plans_created_count
)
SELECT
    gen_random_uuid(),
    id,
    'ultimate_yearly',
    'active',
    now(),
    now() + interval '1 year',
    false,
    0
FROM clinics
ON CONFLICT (workspace_id) DO NOTHING;
