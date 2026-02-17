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
