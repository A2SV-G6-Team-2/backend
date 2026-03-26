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
Postman collection & smoke test
--------------------------------

- Postman collection (importable): `docs/postman_collection.json` — contains basic flows (register -> login -> create debt -> create expense -> list). Import into Postman and run.
- Smoke-test script: `scripts/smoke-test.sh` — a small bash script that exercises the main flows using curl. It requires `curl` and (optional but recommended) `jq` for parsing. Example:

```bash
chmod +x scripts/smoke-test.sh
BASE_URL=http://localhost:8080 ./scripts/smoke-test.sh
```

If you want a CI-friendly smoke test, I can convert the script into a containerized job or a GitHub Actions workflow.
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

Interactive Swagger UI:

```text
http://localhost:8080/api-docs
```

Raw OpenAPI YAML (served with the docs):

```text
http://localhost:8080/api-docs/openapi.yaml
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

Below is a comprehensive list of the HTTP endpoints implemented by the server, grouped by area. All protected endpoints require an `Authorization: Bearer <access_token>` header.

Authentication
- POST /auth/register — register new user (body: name, email, password)
- POST /auth/login — login (body: email, password) -> returns access_token and refresh_token
- POST /auth/refresh — refresh tokens (body: refresh_token)
- POST /auth/logout — revoke refresh token (body: refresh_token)

User
- GET /user/profile — get authenticated user's profile
- PUT /user/update — update authenticated user's profile (partial updates supported)

Expenses
- GET /expenses — list expenses (query: from_date, to_date, category_id, page, page_size)
- POST /expenses — create expense (body: CreateExpenseRequest)
- GET /expenses/{id} — get expense by id
- PUT /expenses/{id} — update expense (body: UpdateExpenseRequest)
- DELETE /expenses/{id} — delete expense

Categories
- GET /categories — list categories (page, page_size)
- POST /categories — create category (body: CreateCategoryRequest)
- GET /categories/{id} — get category
- PUT /categories/{id} — update category
- DELETE /categories/{id} — delete category

Debts
- GET /debts — list debts (page, page_size)
- POST /debts — create a debt (body: CreateDebtInput)
- GET /debts/upcoming — list upcoming debts (query: days, page, page_size)
- PUT /debts/{id} — update a debt (full update; see notes)
- PATCH /debts/{id}/pay — mark a debt as paid

Reports
- GET /reports/daily — daily report (query: date)
- GET /reports/weekly — weekly report (query: start, end)
- GET /reports/monthly — monthly report (query: month YYYY-MM)

Documentation
- GET /api-docs — Swagger UI redirect
- GET /api-docs/ — Swagger UI HTML
- GET /api-docs/openapi.yaml — raw OpenAPI specification

Notes about the Debts API
- ID generation: `POST /debts` will generate a UUID server-side if you omit `id`. If you provide `id` in the request it must be a valid UUID string (Postgres enforces uuid column type).
- PUT semantics: `PUT /debts/{id}` is implemented as a full update. The handler currently expects required fields to be present: `type`, `peer_name`, `amount`, and `due_date` (formatted YYYY-MM-DD). Omitting `due_date` will cause a validation error because the handler attempts to parse it.
- Partial updates: there is no dedicated PATCH endpoint for partial debt updates (except for the `pay` path which updates status). If you need partial updates for debts I can add a PATCH endpoint or modify the PUT handler to merge omitted fields with the existing resource.

Quick debt examples (curl)
Create (server generates id):
```bash
curl -i -X POST "http://localhost:8080/debts" \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"lent",
    "peer_name":"Alex",
    "amount":100.50,
    "due_date":"2026-03-30",
    "reminder_enabled":false,
    "note":"Movie tickets"
  }'
```

Full update (PUT) — include due_date:
```bash
curl -i -X PUT "http://localhost:8080/debts/<DEBT_ID>" \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "type":"lent",
    "peer_name":"Alex",
    "amount":150.00,
    "due_date":"2026-04-15",
    "reminder_enabled":false,
    "note":"Updated amount"
  }'
```

Mark paid:
```bash
curl -i -X PATCH "http://localhost:8080/debts/<DEBT_ID>/pay" \
  -H "Authorization: Bearer <TOKEN>"
```

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