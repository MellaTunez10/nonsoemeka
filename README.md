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

### 1. Configuration
Copy the `.env.example` file to `.env` in the root directory and fill in the required secure passwords for the initial admin and staff accounts:
```bash
cp .env.example .env
# Edit .env and set SEED_ADMIN_PASSWORD and SEED_STAFF_PASSWORD
```

### 2. Run Application (Docker Compose)
The entire stack (Frontend, Backend API, PostgreSQL) can be started using the root Docker Compose file:
```bash
docker compose up -d --build
```

### 3. Access the Application
Open `http://localhost` (or `http://localhost:5173` if running the frontend separately via `npm run dev`).

> **Note on Credentials:** The default hardcoded passwords have been removed for security. Use the `SEED_ADMIN_PASSWORD` and `SEED_STAFF_PASSWORD` you defined in your `.env` file to log in as `admin` or `staff`.
