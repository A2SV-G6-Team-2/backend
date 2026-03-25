# Personal Expense Tracker Backend

Go backend for the A2SV Personal Expense Tracker project.

## Stack

- Go
- `net/http`
- PostgreSQL 15+
- Goose migrations
- JWT access and refresh tokens
- OpenAPI / Swagger UI

## Project Structure

```text
.
├── delivery/
│   ├── apiresponse/        # shared JSON response and pagination helpers
│   └── http/               # handlers, routes, middleware, Swagger
├── domain/                 # core entities
├── infrastructure/
│   ├── auth/               # JWT and password hashing
│   ├── db/                 # DB init and migrations
│   └── repository*/        # PostgreSQL repository implementations
├── repository/             # repository interfaces
├── tests/                  # centralized test suite
├── usecases/               # business logic
└── main.go                 # app wiring
```

## Environment

Copy the example file and adjust it for your local machine:

```bash
cp .env.example .env
```

Typical local configuration:

```env
DB_USER=postgres
DB_PASSWORD=
DB_HOST=127.0.0.1
DB_PORT=5432
DB_NAME=expense_tracker_dev
JWT_SECRET=development-secret
ACCESS_TOKEN_TTL_HOURS=10
REFRESH_TOKEN_TTL_HOURS=168
```

## Local Setup

1. Make sure PostgreSQL is installed and running.
2. Create the database:

```bash
createdb expense_tracker_dev
```

3. Start the application:

```bash
go run main.go
```

The server currently listens on `:8080`.

Important:
- startup runs Goose migrations automatically
- the app does not currently read a `PORT` env var; `main.go` binds to `:8080`

## API Documentation

Swagger UI:

```text
http://localhost:8080/api-docs
```

Raw OpenAPI YAML:

```text
http://localhost:8080/openapi.yaml
```

## Response Format

All HTTP APIs use the same top-level envelope:

```json
{
  "success": true,
  "message": "User fetched successfully",
  "data": {},
  "errors": null,
  "meta": null
}
```

Error responses always use `errors` as an array of strings:

```json
{
  "success": false,
  "message": "Validation failed",
  "data": null,
  "errors": [
    "email is required",
    "password must be at least 8 characters"
  ],
  "meta": null
}
```

List endpoints return `data.items` and pagination metadata:

```json
{
  "success": true,
  "message": "Expenses retrieved successfully",
  "data": {
    "items": []
  },
  "errors": null,
  "meta": {
    "pagination": {
      "page": 1,
      "page_size": 10,
      "total_items": 0,
      "total_pages": 0,
      "has_next": false,
      "has_previous": false
    }
  }
}
```

## Authentication

Auth flow:

1. Register with `POST /auth/register`
2. Login with `POST /auth/login`
3. Use `Authorization: Bearer <access_token>` for protected APIs
4. Rotate tokens with `POST /auth/refresh`
5. Revoke the current refresh token with `POST /auth/logout`

Current defaults:
- access token TTL: 10 hours
- refresh token TTL: 7 days

Password policy:
- minimum 8 characters
- at least one uppercase letter
- at least one lowercase letter
- at least one digit
- at least one special character

## Main Endpoints

Authentication:
- `POST /auth/register`
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`

User:
- `GET /user/profile`
- `PUT /user/update`

Expenses:
- `GET /expenses`
- `POST /expenses`
- `GET /expenses/{id}`
- `PUT /expenses/{id}`
- `DELETE /expenses/{id}`

Categories:
- `GET /categories`
- `POST /categories`
- `GET /categories/{id}`
- `PUT /categories/{id}`
- `DELETE /categories/{id}`

Debts:
- `GET /debts`
- `POST /debts`
- `GET /debts/upcoming`
- `PUT /debts/{id}`
- `PATCH /debts/{id}/pay`

Reports:
- `GET /reports/daily`
- `GET /reports/weekly`
- `GET /reports/monthly`

## Pagination

List endpoints use:

- `page`, default `1`
- `page_size`, default `10`

Validation rules:
- `page >= 1`
- `1 <= page_size <= 100`

## Example Requests

Login:

```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"Secure123!"}'
```

Refresh tokens:

```bash
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh-token>"}'
```

Logout:

```bash
curl -X POST http://localhost:8080/auth/logout \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh-token>"}'
```

List expenses:

```bash
curl -X GET "http://localhost:8080/expenses?page=1&page_size=10&from_date=2026-02-01&to_date=2026-02-28" \
  -H "Authorization: Bearer <access-token>"
```

Create category:

```bash
curl -X POST http://localhost:8080/categories \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access-token>" \
  -d '{"name":"Food"}'
```

Create expense:

```bash
curl -X POST http://localhost:8080/expenses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access-token>" \
  -d '{"amount":50,"expense_date":"2026-02-12","note":"Lunch"}'
```

## Tests

Run the full test suite:

```bash
go test ./...
```

The test files are centralized under [tests](/Volumes/Mike%20Data/Projects/A2SV/backend/tests).
