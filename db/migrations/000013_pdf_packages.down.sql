ALTER TABLE clinic_subscriptions DROP COLUMN IF EXISTS usage_period_start;

UPDATE subscription_plans SET name = 'BrightPlans Classic (Monthly)', price_cents = 4900, currency = 'EUR', plan_quota = 30 WHERE id = 'classic_monthly';
UPDATE subscription_plans SET name = 'BrightPlans Classic (Yearly)', price_cents = 49000, currency = 'EUR', plan_quota = 50 WHERE id = 'classic_yearly';
UPDATE subscription_plans SET name = 'BrightPlans Ultimate (Monthly)', price_cents = 9900, currency = 'EUR', plan_quota = 100 WHERE id = 'ultimate_monthly';
UPDATE subscription_plans SET name = 'BrightPlans Ultimate (Yearly)', price_cents = 99000, currency = 'EUR', plan_quota = -1 WHERE id = 'ultimate_yearly';
