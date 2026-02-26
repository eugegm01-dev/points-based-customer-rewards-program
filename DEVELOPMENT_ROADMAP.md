# Gophermart — Professional Development Roadmap

> Путь к production-ready накопительной системе лояльности

---

## 1. Architecture Overview

### Recommended: Clean / Hexagonal Architecture

```
cmd/gophermart/          # Entry point, wiring
internal/
├── config/              # Configuration loading (env, flags)
├── domain/              # Business entities, interfaces (pure Go, no deps)
├── handlers/            # HTTP handlers (thin layer)
├── middleware/          # Auth, logging, compression
├── repository/          # PostgreSQL implementation
├── accrual/             # External accrual service client
└── service/             # Business logic (use cases)
```

**Principles:**
- Domain layer has no external dependencies
- Handlers call services, never repositories directly
- Repositories and accrual client implement interfaces defined in domain
- Easy to test: mock interfaces

---

## 2. Implementation Phases

### Phase 0: Foundation (Day 1)
- [x] Project structure (internal/, pkg/)
- [x] Config: flags + env (stdlib: flag + os.Getenv)
- [x] Graceful shutdown (context, signal handling)
- [x] Basic HTTP server (chi)
- [x] Logging (zerolog with structured JSON + chi httplog)

### Phase 1: Auth & Users (Days 2–3)
- [x] DB migrations (goose + pgx)
- [x] User model, repository
- [x] JWT in HttpOnly cookie + auth middleware (optional Bearer)
- [x] POST /register, POST /login
- [x] Password hashing: bcrypt

### Phase 2: Orders & Balance (Days 4–5)
- [x] Orders model, repository
- [x] Balance/withdrawals tables
- [x] POST /orders (with Luhn validation)
- [ ] GET /orders, GET /balance
- [ ] Auth middleware on protected routes

### Phase 3: Accrual Integration (Days 6–7)
- [ ] Accrual HTTP client with retries/backoff
- [ ] Background worker: poll accrual for PROCESSING orders
- [ ] Update order status, credit balance
- [ ] Use context for cancellation

### Phase 4: Withdrawals (Day 8)
- [ ] POST /balance/withdraw
- [ ] GET /withdrawals
- [ ] Atomic balance update (transaction)

### Phase 5: Polish (Days 9–10)
- [ ] gzip middleware for responses
- [ ] Validation (go-playground/validator)
- [ ] Error handling: consistent API errors
- [ ] Integration tests
- [ ] Dockerfile + docker-compose for local dev

---

## 3. Tech Stack Recommendations

| Concern | Choice | Why |
|---------|--------|-----|
| Router | **chi** | Lightweight, std context, middleware |
| Config | **envconfig** or **viper** | 12-factor, env-first |
| DB | **pgx** (sqlx optional) | Native PG driver, perf |
| Migrations | **golang-migrate** | Industry standard |
| JWT | **golang-jwt/jwt** | Widely used |
| Logging | **zerolog** or **zap** | Fast, structured |
| Validation | **go-playground/validator** | Declarative, tags |

---

## 4. Database Schema (Reference)

```sql
-- users
id UUID PRIMARY KEY
login VARCHAR UNIQUE NOT NULL
password_hash VARCHAR NOT NULL
created_at TIMESTAMPTZ

-- orders
id SERIAL PRIMARY KEY
user_id UUID REFERENCES users(id)
number VARCHAR UNIQUE NOT NULL
status VARCHAR NOT NULL  -- NEW, PROCESSING, INVALID, PROCESSED
accrual DECIMAL(10,2)   -- nullable until PROCESSED
uploaded_at TIMESTAMPTZ NOT NULL

-- balances (or denormalize into users)
user_id UUID PRIMARY KEY REFERENCES users(id)
current DECIMAL(10,2) NOT NULL DEFAULT 0
withdrawn DECIMAL(10,2) NOT NULL DEFAULT 0

-- withdrawals
id SERIAL PRIMARY KEY
user_id UUID REFERENCES users(id)
order_number VARCHAR NOT NULL
sum DECIMAL(10,2) NOT NULL
processed_at TIMESTAMPTZ NOT NULL
```

---

## 5. Best Practices Checklist

### Code Quality
- [ ] `go vet`, `staticcheck`, `golangci-lint` in CI
- [ ] No `log.Fatal` in handlers — return errors
- [ ] Avoid `panic` in HTTP layer
- [ ] Use `context` for deadlines and cancellation

### Security
- [ ] bcrypt for passwords (cost ≥ 10)
- [ ] JWT: short expiry, refresh strategy or re-login
- [ ] No sensitive data in logs
- [ ] Prepared statements only (SQL injection)

### Reliability
- [ ] Retry with exponential backoff for accrual client
- [ ] Database connection pool tuning
- [ ] Graceful shutdown: drain connections
- [ ] Idempotency where possible (e.g. order re-upload)

### API Design
- [ ] Consistent error format: `{"error": "message"}`
- [ ] Correct HTTP status codes (see SPECIFICATION.md)
- [ ] Content-Type on all responses
- [ ] 204 vs 200 + empty array per spec

---

## 6. Project Structure (Target)

```
.
├── cmd/gophermart/
│   └── main.go              # config load, wire deps, start server
├── internal/
│   ├── config/config.go
│   ├── domain/
│   │   ├── user.go
│   │   ├── order.go
│   │   └── repository.go     # interfaces
│   ├── handlers/
│   │   ├── auth.go
│   │   ├── orders.go
│   │   └── balance.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── logger.go
│   │   └── gzip.go
│   ├── repository/
│   │   └── postgres/
│   │       ├── user.go
│   │       ├── order.go
│   │       └── balance.go
│   ├── accrual/
│   │   └── client.go
│   └── service/
│       ├── auth.go
│       ├── order.go
│       └── balance.go
├── migrations/
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml       # app + postgres + accrual (optional)
└── README.md
```

---

## 7. Suggested Order of Implementation

1. **Config + server skeleton** — run on `:8080`, respond to health
2. **DB + migrations** — connect, run migration
3. **User repository** — Create, GetByLogin
4. **Auth service** — Register, Login, hash passwords
5. **Auth handlers** — POST register, POST login, set cookie/header
6. **Auth middleware** — extract user from JWT/cookie
7. **Order handlers** — POST/GET orders, protected
8. **Balance handlers** — GET balance
9. **Accrual client** — GET order status
10. **Accrual worker** — goroutine, poll, update
11. **Withdraw handlers** — POST withdraw, GET withdrawals

---

## 8. Quick Wins for “Professional” Feel

- **Structured logging**: `{"level":"info","msg":"user registered","user_id":"..."}`
- **Request ID**: add `X-Request-ID` for tracing
- **Health endpoint**: `GET /health` → 200 + DB ping
- **docker-compose**: one command to run app + postgres
- **README**: how to run, env vars, API summary

---

## 9. Testing Strategy

| Layer | Tool | Focus |
|-------|------|-------|
| Unit | `testing` | Services, Luhn, validation |
| Integration | `testcontainers` or real PG | Repositories |
| API | Yandex autotests | Full flow |

Start with service + Luhn unit tests. Add integration tests for critical paths.

---

## 10. Before First PR

- [ ] All handlers from SPECIFICATION.md implemented
- [ ] `gophermart.yml` CI passes
- [ ] `statictest.yml` CI passes
- [ ] README has run instructions
- [ ] No secrets in code
- [ ] go.mod tidy, no unused deps
