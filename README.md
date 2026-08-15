# ExpenseTrackerAPI

A simple Go backend API for managing user wallets and transactions, built with [Gin](https://github.com/gin-gonic/gin) and [GORM](https://gorm.io/).

## Features

- User signup and login with JWT-based authentication
- Role-based access (`user` / `admin`) — admins can view transactions across all wallets, regular users are scoped to their own wallet
- Transaction listing with filtering by category and date range, plus pagination
- PostgreSQL persistence via GORM
- Integration tests using [testcontainers-go](https://golang.testcontainers.org/) against a real Postgres instance, and fast unit tests against an in-memory SQLite database

## Tech Stack

- **Language:** Go
- **Web framework:** [gin-gonic/gin](https://github.com/gin-gonic/gin)
- **ORM:** [gorm.io/gorm](https://gorm.io/), with the `postgres` and `sqlite` drivers
- **Auth:** JWT
- **Testing:** Go's standard `testing` package, `testcontainers-go/modules/postgres`

## Project Structure

```
.
├── auth/          # JWT creation/validation, auth middleware, role checks (IsAdmin, etc.)
├── database/       # DB connection/setup
├── handler/         # HTTP handlers (Gin) — request parsing, response writing
├── model/           # GORM models (User, Wallet, Transaction, ...)
├── repository/       # Data-access layer — queries against the database
├── response/         # Response DTOs/shaping
├── service/           # Business logic sitting between handlers and repositories
├── main.go
├── go.mod
└── go.sum
```

## Getting Started

### Prerequisites

- Go (check `go.mod` for the exact version required)
- A running PostgreSQL instance
- Docker (only needed if you want to run the integration test suite, which spins up Postgres via testcontainers)

### Installation

```bash
git clone https://github.com/HareedyBasha/ExpenseTrackerAPI.git
cd ExpenseTrackerAPI
go mod download
```

### Configuration

The API connects to PostgreSQL and uses a JWT signing secret. Set these via environment variables (or a `.env` file, if the project loads one) before running:

```bash
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=expense_tracker
export JWT_SECRET=your-secret-here
```

> **Note:** Confirm the exact variable names expected by `database/` and `auth/` in the source — adjust the above to match.

### Running the API

```bash
go run main.go
```

## API Reference

All endpoints are prefixed as defined in `main.go`'s router setup.

### Auth

| Method | Endpoint    | Description                          | Auth required |
|--------|-------------|---------------------------------------|----------------|
| POST   | `/signup`   | Register a new user                   | No             |
| POST   | `/login`    | Authenticate and receive a JWT         | No             |

**Signup / Login request body:**

```json
{
  "username": "hareedy",
  "password": "your-password"
}
```

**Login response:**

```json
{
  "token": "<jwt>"
}
```

Include the token on subsequent requests:

```
Authorization: Bearer <jwt>
```

### Wallets

*(Endpoint paths and payloads should be filled in from `handler/` — the API manages a wallet per user, with balance tracked via transactions.)*

### Transactions

| Method | Endpoint         | Description                              | Auth required |
|--------|------------------|--------------------------------------------|----------------|
| GET    | `/transactions`  | List transactions, with optional filters   | Yes            |

**Query parameters:**

| Param      | Type   | Description                                      |
|------------|--------|----------------------------------------------------|
| `category` | string | Filter by transaction category                     |
| `from`     | RFC3339 datetime | Include transactions created on/after this time |
| `to`       | RFC3339 datetime | Include transactions created before this time    |
| `page`     | int    | Page number (1-indexed)                             |
| `limit`    | int    | Page size (max 100)                                 |

Regular users only see transactions belonging to their own wallet; users with the `admin` role see transactions across all wallets.

*(Add `POST` / `PUT` / `DELETE` transaction endpoints here once confirmed from the handler package.)*

## Testing

Run the full test suite:

```bash
go test ./...
```

Some tests spin up a real PostgreSQL container via testcontainers-go and require Docker to be running locally.

## License

*(Add license information here.)*