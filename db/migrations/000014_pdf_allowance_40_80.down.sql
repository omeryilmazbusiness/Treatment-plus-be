UPDATE subscription_plans SET plan_quota = 30 WHERE id IN ('classic_monthly', 'classic_yearly');
UPDATE subscription_plans SET plan_quota = 60 WHERE id IN ('ultimate_monthly', 'ultimate_yearly');
