# Nonsoemeka Pharmacy Full-Stack POS System

Production-grade hybrid Point-of-Sale (POS) and inventory management system for **Nonsoemeka Pharmacy**, supporting pharmaceutical batch tracking and FEFO (First-Expired, First-Out) stock dispatching across Admin and Staff dashboards.

---

## 📁 Repository Layout

```
nonsoemeka/
├── nonsoemeka-backend/         # Go REST API backend (Chi, pgx, decimal, Prometheus)
│   ├── cmd/api/main.go         # Application entrypoint & server lifecycle
│   ├── internal/               # Core packages (models, dto, auth, repository, services, handlers, middleware)
│   ├── migrations/             # Relational Postgres migrations
│   ├── docs/openapi.yaml       # OpenAPI 3.0 API Specification
│   ├── Dockerfile & compose    # Containerization & test stack
│   └── .github/workflows/      # CI/CD pipeline
└── nonsoemeka-frontend/        # Vite + React + TypeScript + Tailwind CSS Frontend
    ├── src/
    │   ├── components/         # UI & printable receipt modal (react-to-print)
    │   ├── pages/              # Staff POS, Admin Inventory, Expiry, Financials, Staff, Settings
    │   ├── lib/                # Money (decimal.js), api-client, auth context
    │   └── hooks/              # TanStack Query hooks
    └── package.json
```

## 🚀 Getting Started

### 1. Database
```bash
cd nonsoemeka-backend
docker compose up -d db
```

### 2. Run Backend API
```bash
cd nonsoemeka-backend
cp .env.example .env
go run ./cmd/api/main.go
```

### 3. Run Frontend
```bash
cd nonsoemeka-frontend
npm install
npm run dev
```

Open `http://localhost:5173`. Default credentials:
- **Admin User**: `username: admin` | `password: AdminPass123!`
- **Staff User**: `username: staff` | `password: StaffPass123!`
