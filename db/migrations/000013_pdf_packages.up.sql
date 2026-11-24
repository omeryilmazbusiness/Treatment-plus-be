-- Phase: PDF-based packages (Communication / Professional) with a monthly PDF allowance.
-- Tier ids stay (classic = Communication, ultimate = Professional); only the catalog changes.
-- Yearly plans bill 10 months (2 months free) but keep the same *monthly* PDF allowance,
-- so usage is counted per calendar-month window (usage_period_start), not per billing period.

ALTER TABLE clinic_subscriptions
    ADD COLUMN usage_period_start timestamptz NOT NULL DEFAULT now();

UPDATE subscription_plans SET
    name = 'Communication (Monthly)',
    description = 'Premium PDF treatment plans for patient communication. 30 PDFs per month.',
    price_cents = 7500,
    currency = 'USD',
    plan_quota = 30,
    features = '["30 PDFs per month", "24-Page Premium PDF Presentation", "Clinic Brand & Doctor Bio", "Clinical Findings & Odontogram", "Cost & Procedure Tables", "Extra PDF packs available monthly"]'::jsonb
WHERE id = 'classic_monthly';

UPDATE subscription_plans SET
    name = 'Communication (Yearly)',
    description = 'Annual Communication subscription, billed as 10 months. 30 PDFs per month.',
    price_cents = 75000,
    currency = 'USD',
    plan_quota = 30,
    features = '["Everything in Communication Monthly", "2 Months Free", "30 PDFs per month", "Priority Clinic Onboarding", "Extra PDF packs available monthly"]'::jsonb
WHERE id = 'classic_yearly';

UPDATE subscription_plans SET
    name = 'Professional (Monthly)',
    description = 'Everything in Communication plus interactive mobile experience, 3D simulations and analytics. 60 PDFs per month.',
    price_cents = 10000,
    currency = 'USD',
    plan_quota = 60,
    features = '["60 PDFs per month", "Everything in Communication", "Interactive Mobile Web Experience (/p/{token})", "3D Dental Simulation Clips", "Personalized Video & Welcome Letter", "Before & After Interactive Slider", "Real-Time Patient Engagement Telemetry", "One-Click Digital Acceptance & Signature", "Direct WhatsApp & Call Hub", "Extra PDF packs available monthly"]'::jsonb
WHERE id = 'ultimate_monthly';

UPDATE subscription_plans SET
    name = 'Professional (Yearly)',
    description = 'Complete Professional experience billed as 10 months, with a dedicated onboarding session. 60 PDFs per month.',
    price_cents = 100000,
    currency = 'USD',
    plan_quota = 60,
    features = '["Everything in Professional Monthly", "2 Months Free", "60 PDFs per month", "2-Hour Dedicated 1-on-1 Onboarding Session", "Custom Template & Pricing List Setup", "VIP Support", "Extra PDF packs available monthly"]'::jsonb
WHERE id = 'ultimate_yearly';
