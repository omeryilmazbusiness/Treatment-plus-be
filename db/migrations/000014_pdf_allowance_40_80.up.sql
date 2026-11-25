-- Package marketing allowance: Communication 40 PDF/mo, Professional 80 PDF/mo.
UPDATE subscription_plans SET
    plan_quota = 40,
    description = 'Premium PDF treatment plans for patient communication. 40+ PDFs per month.',
    features = '["40+ PDFs per month", "24-Page Premium PDF Presentation", "Clinic Brand & Doctor Bio", "Clinical Findings & Odontogram", "Cost & Procedure Tables", "Extra PDF packs available monthly"]'::jsonb
WHERE id IN ('classic_monthly', 'classic_yearly');

UPDATE subscription_plans SET
    plan_quota = 80,
    description = CASE
        WHEN id = 'ultimate_monthly'
            THEN 'Everything in Communication plus interactive mobile experience, 3D simulations and analytics. 80+ PDFs per month.'
        ELSE 'Complete Professional experience billed as 10 months, with a dedicated onboarding session. 80+ PDFs per month.'
    END,
    features = CASE
        WHEN id = 'ultimate_monthly'
            THEN '["80+ PDFs per month", "Everything in Communication", "Interactive Mobile Web Experience (/p/{token})", "3D Dental Simulation Clips", "Personalized Video & Welcome Letter", "Before & After Interactive Slider", "Real-Time Patient Engagement Telemetry", "One-Click Digital Acceptance & Signature", "Direct WhatsApp & Call Hub", "Extra PDF packs available monthly"]'::jsonb
        ELSE '["Everything in Professional Monthly", "2 Months Free", "80+ PDFs per month", "2-Hour Dedicated 1-on-1 Onboarding Session", "Custom Template & Pricing List Setup", "VIP Support", "Extra PDF packs available monthly"]'::jsonb
    END
WHERE id IN ('ultimate_monthly', 'ultimate_yearly');
