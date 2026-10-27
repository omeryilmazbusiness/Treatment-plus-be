# treatment-plus-be

Tedavi Planı Platformu API. Go modular monolith, hexagonal (ports & adapters), SOLID.

Task 1: klinik hesabı (tek user tipi). Register / login / refresh / logout / me. Cookie yok.

## Çalıştırma

```bash
cp .env.example .env
make up
```

- API: `http://localhost:8080/healthz` (proses)
- Hazır: `http://localhost:8080/readyz` (Postgres + Redis)
- Auth: `POST /v1/auth/register|login|refresh|logout`, `GET /v1/auth/me`, `GET /v1/clinics/{id}`
- Local compose: Postgres `localhost:5434`, Redis `localhost:6380` (5432/6379 çakışmasın diye)

Ayrı komutlar:

```bash
make docker-up      # Postgres 16 + Redis
make migrate-up     # golang-migrate
make run            # cmd/api
make worker         # cmd/worker (Faz 0 boş kuyruk)
make test           # unit + arch-lint
```

## Süreçler

| Process | Görev |
|---|---|
| `cmd/api` | Senkron komut / sorgu |
| `cmd/worker` | PDF, AI, e-posta, outbox (Faz 7+) |
| `cmd/migrate` | Şema (advisory lock) |

Tarayıcı bu API’ye cookie atmaz. Cookie Next BFF’de kalır (`treatment-plus-fe`).

## Mimari

Bağımlılık tek yön: `domain ← app ← adapters`. Adapter başka adapter import etmez.

```
cmd/            composition root (sinyal, dinleme)
internal/
  bootstrap/    wiring — yalnızca cmd import eder
  shared/       id, money, clock, errx, ctxkey
  platform/     httpx, postgres, redis, rls, otel, authn, authz, outbox
  identity/     domain app http pg mem
  clinic/       …
  catalog/
  plan/
  media/
  output/
  share/
  ai/
  arch/         import kuralı testi
```

Sekiz bounded context. Domain paketi `net/http`, `pgx` veya vendor SDK görmez. CI `go test ./internal/arch` ile kırılırsa build düşer.

## Env

Zorunlu: `DATABASE_URL`, `REDIS_URL`.

Local API: `DATABASE_APP_URL` (`treatment_app`, NOBYPASSRLS). Migrate owner `DATABASE_URL`. Production tek non-superuser URL yeter.

Sonra: `JWT_*`, `INTERNAL_TOKEN`, `KMS_KEY`, `S3_*`, `OPENAI_*`, `APP_URL`. Sırlar image’a gömülmez; Railway variables.

## Railway

EU, private network. Servisler: `api`, `worker`, Postgres, Redis.

- API: `railway.toml` — pre-deploy migrate, health `/healthz`
- Worker: `railway.worker.toml`

## Sıra (bozulmaz)

0 iskelet → 1 kimlik + RLS → 2 marka → 3 katalog → 4 plan → 5–6 hasta web + link → 7 medya/PDF → 8 AI

Yasak başlangıç: AI, PDF, Stripe.
