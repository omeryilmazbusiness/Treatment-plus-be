-- Phase 0: extensions only. Business tables start in Phase 1 (identity + RLS).
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
