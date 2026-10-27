# Mimari kilit — Faz 0

Kaynak: Tedavi Planı Platformu teknik mimari v2.0 (26 Ekim 2026).

- Modular monolith. Tek API, tek worker, tek veritabanı.
- Hexagonal: port interface, adapter dışarıda.
- SOLID: bir paket = bir bounded context; fake ve gerçek adapter aynı sözleşme; god repository yok; composition root `cmd` + `internal/bootstrap`.
- Para: `money.Money` integer minor unit. Float yok.
- Kimlik: `clinic_id` body/query’den alınmaz. RLS `SET LOCAL app.clinic_id`.
- SQL: yalnızca sqlc. ORM yok.
- Ağır iş: worker + outbox. HTTP beklemez.
- Cache: hasta sayfasında Redis/CDN yok (PHI).
- PK: ULIDv2, Postgres’te uuid.

İş tabloları Task 1: `clinics`, `sessions`. Tek aktör klinik hesabıdır; `users` / `clinic_members` / rol yoktur.
`clinic_id` body/query’den alınmaz. RLS `SET LOCAL app.clinic_id` (UUID text). Auth bootstrap SECURITY DEFINER + `row_security=off`.
API `treatment_app` (NOBYPASSRLS) ile bağlanır; migrate owner superuser RLS’i ezer.
