# Finance Backend — Greenpark Financial Control Dashboard API

A small, dependency-free Go HTTP API that serves the data for the **Dashboard
Finance Greenpark** (CEO financial control / cash & margin guard dashboard).

It is built with a clean, layered architecture so the file-backed data source can
later be swapped for a real accounting/ERP source without touching the transport
or business logic. Master data is **editable** (login + admin CRUD) and persisted
to a JSON file, so input flows straight back into the dashboard reads.

## Architecture

```
cmd/server            composition root — wires everything and runs the server
internal/
  config              env-based runtime configuration (with defaults)
  domain              core entities (Project, Receivable, Payable, Facility, User, …) — no deps
  passwd              salted SHA-256 password hashing helpers
  auth                token login + in-memory bearer-token sessions
  repository          storage boundary (interface) + file-backed seeded store (CRUD + users)
  service             business logic — composes data, derives the summary, write use-cases
  transport/http      router, handlers (auth + reads + writes), middleware, JSON helpers
```

## Auth & roles

- `POST /api/auth/login` → `{ token, user }`. Send `Authorization: Bearer <token>` on every other call.
- **admin** (`admin / admin123`) — full master-data write access.
- **viewer** (`viewer / viewer123`) — read-only; writes return `403`.

Master-data edits (`POST`/`PUT`/`DELETE`) persist to the JSON store
(`FINANCE_DATA_PATH`, default `data/finance-data.json`) and immediately change the
derived `summary` returned by the dashboard.

Dependency direction points inward: `transport → service → repository → domain`.
Each layer depends only on the interfaces of the one beneath it.

All monetary values are expressed in **millions of Rupiah (Rp juta)**.

## Run

```bash
cd backend/finance
go run ./cmd/server
# finance API listening on http://localhost:8084
```

Configuration via environment variables:

| Variable                | Default | Description           |
| ----------------------- | ------- | --------------------- |
| `FINANCE_PORT`          | `8084`  | HTTP port             |
| `FINANCE_ALLOW_ORIGIN`  | `*`     | CORS allowed origin   |
| `FINANCE_DATA_PATH`     | `data/finance-data.json` | JSON file the master data persists to |

## Test

```bash
go test ./...
```

## API

All responses are JSON. Read-only `GET` endpoints under `/api`:

| Method · Path                 | Description                                  |
| ----------------------------- | -------------------------------------------- |
| `GET /api/health`             | Liveness probe                               |
| `GET /api/dashboard`          | Full payload (all sections + derived summary)|
| `GET /api/summary`            | Executive KPI summary (derived)              |
| `GET /api/projects`           | Project P&L list                             |
| `GET /api/projects/{id}`      | Single project (404 if unknown)              |
| `GET /api/receivables`        | Receivables (AR) with aging                  |
| `GET /api/payables`           | Payables (AP) with due/priority              |
| `GET /api/facilities`         | Funding facilities (bank / KPR / equity)     |
| `GET /api/cost-structure`     | Budget-vs-actual cost categories             |
| `GET /api/treasury`           | Cash position figures                        |
| `GET /api/ai-insights`        | Generated insights for the war-room          |
| `GET /api/decisions`          | Critical decisions per role                  |
| `GET /api/kpis`               | KPI reference table (15 indicators)          |
| `GET /api/triggers`           | Early-warning trigger rules                  |

The `summary` is **derived** in the service layer from projects, receivables,
payables and the treasury position (collection rate, net margin, runway,
outstanding AR/AP, overdue risk, …).
