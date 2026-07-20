# Nonsoemeka Pharmacy Backend & Frontend System

Production-grade hybrid Point-of-Sale (POS) and inventory management system for **Nonsoemeka Pharmacy**, supporting pharmaceutical batch tracking and FEFO (First-Expired, First-Out) stock dispatching across Admin and Staff dashboards.

---

## 🏛 Architectural Principles & Layer Restrictions

This project follows a strict layered architecture:

```
handlers  ──>  services  ──>  repositories  ──>  database
```

- **Handlers (`internal/handlers`)**: HTTP layer only. Parse and validate DTOs, call service methods, shape JSON responses. No raw SQL, no business logic.
- **Services (`internal/services`)**: Business logic & transaction boundaries (`pgx.Tx`). Return domain models or typed domain errors (`internal/apperrors`). Services are transport-agnostic and do NOT import router packages like `chi`.
- **Repositories (`internal/repository`)**: Data access layer. The ONLY place raw SQL appears, behind Go interfaces. Accepts `DBTX` (`pgxpool.Pool` or `pgx.Tx`). Converts Postgres errors into domain sentinels. No business logic.
- **Money Handling**: `github.com/shopspring/decimal` used exclusively across Go money types. Serialized in JSON as fixed decimal strings (e.g. `"312.50"`), never floating-point numbers.
- **Auth & Security**: JWT access tokens + rotating, httpOnly, `Secure`, `SameSite=Strict` refresh tokens with automatic token reuse detection (revokes all user tokens if an already-revoked refresh token is presented).

---

## 🚀 Quick Start (Local Development)

### Prerequisites
- Go 1.22+
- Node.js 18+ & npm
- Docker & Docker Compose
- PostgreSQL 15+

### 1. Database Setup
Start the local PostgreSQL container:
```bash
docker compose up -d db
```

### 2. Backend Server
Copy `.env.example` to `.env` and start the API server:
```bash
cd nonsoemeka-backend
cp .env.example .env
go run ./cmd/api/main.go
```

The server automatically runs database migrations from `migrations/` and seeds initial default users:
- **Admin User**: `username: admin` | `password: AdminPass123!`
- **Staff User**: `username: staff` | `password: StaffPass123!`

### 3. Frontend Client
Start the Vite React frontend:
```bash
cd nonsoemeka-frontend
npm install
npm run dev
```

Visit `http://localhost:5173` in your browser.

---

## 🧪 Backend Testing & CI

Run unit tests and integration tests with the race detector enabled:

```bash
cd nonsoemeka-backend

# Start ephemeral test database
docker compose -f docker-compose.test.yml up -d

# Run all tests with race detector
DB_HOST=localhost DB_PORT=5434 DB_USER=postgres DB_PASSWORD=postgres DB_NAME=nonsoemeka_test_db \
JWT_ACCESS_SECRET=test-access-secret-key-32chars \
JWT_REFRESH_SECRET=test-refresh-secret-key-32chars \
go test -race -v ./...

# Tear down test database
docker compose -f docker-compose.test.yml down -v
```

---

## 🔐 Authentication & Security Contract

### Token Transport Architecture
- **Access Token**: Delivered in JSON response body (`access_token`). Kept strictly in frontend memory (`lib/auth.tsx`), never written to `localStorage`.
- **Refresh Token**: Delivered exclusively via `Set-Cookie` as an `httpOnly`, `Secure`, `SameSite=Strict` cookie (`refresh_token`). JavaScript cannot access it, preventing persistent account takeovers via XSS.
- All frontend API calls use `credentials: 'include'` to pass/receive authentication cookies automatically.

---

## 🔮 Section 19: Architectural Extensibility Conventions

When extending the application with new domain features, adhere strictly to the package-per-domain pattern:

Reserved future domain package names:
- `suppliers/`: Wholesale drug suppliers and vendor management.
- `purchases/`: Purchase orders and inbound batch procurement.
- `prescriptions/`: Electronic prescription verification and doctor notes.
- `notifications/`: Stock alert webhooks and SMS notifications.
- `backups/`: Database snapshot and cloud backup utilities.

**Extraction Rule**: As concerns in `inventory/` or `sales/` grow, extract them following the `handlers/service/repository` triad rather than inflating single monolithic files.

---

## 📊 Observability & Metrics

- `GET /healthz`: Liveness check
- `GET /readyz`: DB connectivity readiness check
- `GET /metrics`: Prometheus formatted metrics (`http_requests_total`, `http_request_duration_seconds`, etc.)

> **Note on Deployment Security**: `/metrics` does not require JWT authentication so infrastructure scrapers (e.g. Prometheus) can reach it. In production, restrict `/metrics` at the ingress/network level so it is not exposed publicly.
