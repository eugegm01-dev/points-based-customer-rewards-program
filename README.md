# Gophermart – Loyalty Points System

Gophermart is a HTTP API for a customer loyalty program. Users can register, upload order numbers, earn points, and withdraw them for future orders.

## Features

- User registration and authentication (JWT in HttpOnly cookie)
- Upload order numbers (Luhn validation)
- Check order processing status and earned points
- View current balance and withdrawal history
- Withdraw points (simulate paying for a new order)
- Background worker that polls an external accrual system to update order statuses and credit balances

## Technologies

- Go 1.24
- PostgreSQL (via pgx)
- Chi router
- Goose migrations
- Zerolog for structured logging
- JWT for authentication
- Gzip compression
- Docker Compose for local development

## Getting Started

### Prerequisites

- Docker and Docker Compose (for local run)
- Go 1.24+ (if running without Docker)

### Configuration

The service is configured via environment variables or command-line flags:

| Variable | Flag | Description | Default |
|----------|------|-------------|---------|
| `RUN_ADDRESS` | `-a` | Server address and port | `:8080` |
| `DATABASE_URI` | `-d` | PostgreSQL connection URI | **required** |
| `ACCRUAL_SYSTEM_ADDRESS` | `-r` | Base URL of the accrual system | `http://localhost:8080` |
| `AUTH_SECRET` | `-s` | Secret for JWT signing | `autotest-secret-key-change-in-production` |

### Running with Docker Compose

docker-compose up
This starts:

Gophermart on localhost:8080

PostgreSQL on localhost:5432

A mock accrual system on localhost:8081 (optional)

Running manually
Start a PostgreSQL instance.

Set environment variables or pass flags.

Run:

bash
go run cmd/gophermart/main.go
API Endpoints
All endpoints (except /health) return JSON. Authentication is via cookie token (set after login/register).

Method	Path	Description
POST	/api/user/register	Register a new user
POST	/api/user/login	Login
POST	/api/user/orders	Upload an order number
GET	/api/user/orders	List user's orders
GET	/api/user/balance	Get current balance
POST	/api/user/balance/withdraw	Withdraw points
GET	/api/user/withdrawals	List withdrawals
GET	/health	Health check
For detailed request/response formats, see SPECIFICATION.md.

Testing
Run unit and integration tests:

bash
go test -v ./...
Integration tests require Docker to run testcontainers (they spin up a temporary PostgreSQL).

To see test coverage:

bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
Code Structure
text
cmd/gophermart/          # Entry point
internal/
├── config/              # Configuration
├── domain/              # Business entities & interfaces
├── handlers/            # HTTP handlers
├── middleware/          # Auth, gzip
├── repository/postgres/ # Database implementation
├── service/             # Business logic
├── accrual/             # External API client
├── auth/                # JWT helpers
├── logger/              # Structured logging
└── migrate/             # Database migrations
Contributing
Feel free to open issues or pull requests.

License
MIT

03. 01. 2026 