# RMS Backend (bootstrap)

HTTP API bootstrap for a **Rent Management System**: **Gin**, **GORM + PostgreSQL** (`DATABASE_URL`), structured logging, shared `internal/api` helpers, **Swagger** (swaggo), and **health** endpoints. Domain modules are **not** implemented yet—only **route group placeholders** under `/api/v1`.

## Configuration

Environment variables are loaded in `internal/config/config.go` (see `.env.example`). Required for the API:

- `DATABASE_URL`
- `JWT_SECRET`, `JWT_REFRESH_SECRET`

## Run locally

```bash
cp .env.example .env
make docker-up
make run
```

## Run with Docker (API + Postgres)

```bash
cp .env.example .env
docker compose up --build -d
```

- API: `http://localhost:8080`
- Swagger: `http://localhost:8080/swagger/index.html`

Optional worker:

```bash
docker compose --profile worker up -d
```

Stop services:

```bash
docker compose down
```

- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Liveness**: `GET /health/live` — returns `503` with `shutting_down` once graceful shutdown starts.
- **Readiness**: `GET /health/ready` — PostgreSQL ping.
- **Internal** (optional): if `INTERNAL_JOB_SECRET` is set, `GET /internal/status` with header `X-Internal-Key: <secret>`.

On API startup, **`internal/app.AutoMigrate`** runs via **`NewDependencies`**, syncing GORM models to Postgres (development-friendly). Production should still apply **versioned SQL migrations** for constraints GORM does not create (EXCLUDE overlaps, allocation totals, triggers).

## Make targets

| Target | Description |
|--------|-------------|
| `make build` | Build `bin/rms-api` and `bin/rms-worker` |
| `make run` | Run API |
| `make run-worker` | Run worker (config load only for now) |
| `make test` | `go test -short ./...` |
| `make itest` | Postgres integration test (Docker) |
| `make swagger` | Regenerate `docs/` |

## Layout

- `internal/config` — env-based configuration.
- `internal/database/postgres.go` — connection pool + ping/health.
- `internal/api/*` — errors, response envelope, pagination, validator (go-playground), security (HS256 JWT + password placeholder), logger abstraction, clock, uuid.
- `internal/middleware` — request ID (Gin + `context.Context`), structured logging, panic recovery, JWT auth + RBAC roles, org scoping, auth rate limit, internal job secret.
- `internal/app` — router, dependencies, server, **`migrate.go`** (`AutoMigrate` all module models); module route placeholders in `module_routes.go`.

### Middleware quick reference

| Middleware | Purpose |
|------------|---------|
| `RequestID` | `X-Request-ID` on response; Gin context + `Request.Context()` |
| `StructuredLogger` | method, path, status, `duration_ms`, request ID |
| `Recovery` | panic → `response.Error(ErrInternal)` |
| `JWTAuth` | Bearer access JWT → `UserClaims` in context |
| `RequireRoles` | allow-list `admin` / `landlord` / `manager` |
| `RequireOrganizationParam` | non-admins must match `:org_id` or `X-Organization-ID` to JWT `org_id` |
| `AuthLoginRateLimiter` | per-IP limit on `/api/v1/auth/login` and `/register` (`RATE_LIMIT_AUTH_RPM`) |
| `InternalSecretAuth` | `X-Internal-Key` must equal `INTERNAL_JOB_SECRET` |

## Go version

Match the `go` directive in `go.mod` when installing the toolchain.
