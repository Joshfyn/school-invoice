# School Invoice SaaS

A multi-tenant invoicing system for Nigerian primary and secondary schools.

## Features

- **Multi-tenant Architecture**: Each school has isolated data
- **Student Management**: Track students, guardians, and class enrollments
- **Invoice Generation**: Create invoices for fees with class-specific pricing
- **Payment Integration**: Online payments via Paystack (cards, bank transfer, USSD)
- **SMS Notifications**: Send invoice links and reminders via Termii
- **Role-based Access Control**: Custom roles with granular permissions
- **Reports**: Dashboard summaries, outstanding fees, and collection reports

## Tech Stack

- **Backend**: Go (Golang) with Gin framework
- **Database**: PostgreSQL
- **Cache/Queue**: Redis
- **Payment Gateway**: Paystack
- **SMS Gateway**: Termii

## Getting Started

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- Make (optional)

### Quick Start with Docker

1. Clone the repository
2. Copy environment file:
   ```bash
   cp .env.example .env
   ```
3. Start all services:
   ```bash
   docker-compose up -d
   ```
4. Run migrations:
   ```bash
   make migrate-up
   ```

### Local Development

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Start PostgreSQL and Redis (using Docker):
   ```bash
   docker-compose up -d db redis
   ```

3. Run migrations:
   ```bash
   export DATABASE_URL="postgres://postgres:postgres@localhost:5432/school_invoice?sslmode=disable"
   make migrate-up
   ```

4. Run the API:
   ```bash
   make run
   ```

5. Run with hot reload (requires Air):
   ```bash
   make tools  # Install Air
   make dev
   ```

### Local Development (Windows)

These steps assume you’re using **PowerShell**.

1. Install required software:
   - **Git**: install Git for Windows and ensure `git` works in PowerShell.
   - **Go 1.21+**: install Go and ensure `go version` works.
   - **Docker Desktop**: enable WSL2 backend (recommended) and ensure Docker is running.
   - **Optional (recommended)**: install `make` (or use the non-`make` commands below).

2. Clone the repository and create your env file:
   ```powershell
   git clone <your-repo-url>
   cd school-invoice
   Copy-Item .env.example .env
   ```

3. Download Go dependencies:
   ```powershell
   go mod download
   ```

4. Start PostgreSQL and Redis:
   ```powershell
   docker-compose up -d db redis
   ```
   If `docker-compose` isn’t available, use:
   ```powershell
   docker compose up -d db redis
   ```

5. Install dev tools (migrate + air):
   - With `make`: `make tools`
   - Without `make`:

```powershell
go install github.com/air-verse/air@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

   Ensure your Go bin directory is on `PATH` (commonly `%USERPROFILE%\go\bin`) so `air` and `migrate` are available.

6. Run migrations:
   Set `DATABASE_URL`:

```powershell
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/school_invoice?sslmode=disable"
```

   - With `make`: `make migrate-up`
   - Without `make`: `migrate -path .\migrations -database "$env:DATABASE_URL" up`

7. Run the API:
   - With `make`: `make run`
   - Without `make`: `go run .\cmd\api`

8. Hot reload (optional):
   - With `make`: `make dev`
   - Without `make`: `air -c .air.toml`

## Project Structure

```
school-invoice/
├── cmd/
│   ├── api/              # Main API server
│   └── worker/           # Background jobs
├── internal/
│   ├── config/           # Configuration
│   ├── database/         # Database connections
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models
│   ├── handlers/         # HTTP handlers
│   ├── services/         # Business logic
│   └── utils/            # Utilities
├── pkg/
│   ├── paystack/         # Paystack SDK
│   └── termii/           # Termii SMS SDK
├── migrations/           # SQL migrations
├── docker-compose.yml
├── Dockerfile
└── Makefile
```

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - Register a new school
- `POST /api/v1/auth/login` - User login

### Schools
- `GET /api/v1/schools/profile` - Get school profile
- `PUT /api/v1/schools/profile` - Update school profile

### Students
- `GET /api/v1/students` - List students
- `POST /api/v1/students` - Create student
- `GET /api/v1/students/:id` - Get student details

### Invoices
- `GET /api/v1/invoices` - List invoices
- `POST /api/v1/invoices` - Create invoice
- `POST /api/v1/invoices/bulk` - Bulk create invoices
- `POST /api/v1/invoices/:id/send` - Send invoice SMS

### Payments
- `POST /api/v1/payments/initialize` - Start payment
- `POST /api/v1/payments/webhook` - Paystack webhook
- `POST /api/v1/payments/verify-bank` - Verify bank payment

See the full API documentation for more endpoints.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | Server port | 8080 |
| ENV | Environment (development/production) | development |
| DATABASE_URL | PostgreSQL connection string | - |
| REDIS_URL | Redis connection string | - |
| JWT_SECRET | JWT signing secret | - |
| PAYSTACK_SECRET_KEY | Paystack secret key | - |
| TERMII_API_KEY | Termii API key | - |

## License

MIT
