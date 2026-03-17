# WebsiteGo

Go web application for user management with the following stack:
- Gin (web framework)
- GORM + MySQL
- Session cookie (`gin-contrib/sessions`)
- Embedded SQL migrations (`golang-migrate`)
- Register/login/logout auth + roles (`admin`/`user`)
- User dashboard with search, pagination, and admin-only CRUD

## Main Features
- Register & login with `bcrypt` password hashing.
- The first registered user is automatically assigned as `admin`.
- Auto-promote fallback: if no admin exists, a logging-in user can be promoted to admin.
- Protected dashboard route (`/dashboard`).
- View the user list on the dashboard (all logged-in users).
- Dashboard search by `all`, `id`, `name`, `email`, `role`.
- Dashboard pagination (`10/25/50/100`) + page jumps `-100/-1000/+100/+1000`.
- Indexed search/list queries to reduce scan time on large user tables.
- Configurable DB connection pooling for better throughput under concurrent traffic.
- Automatic migrations at startup to keep schema/indexes aligned with expected query paths.
- Cleaner logs by filtering client-disconnect noise, helping faster performance troubleshooting.
- Add / update / delete users from dashboard (admin only).
- Prevent admins from deleting their own account.

## Prerequisites
- Go `1.25.x` (based on `go.mod`)
- Running MySQL server (example: XAMPP MySQL)

## Database Setup
Create the database:

```sql
CREATE DATABASE websitego CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

## Quick Start

```bash
copy .env.example .env
go mod tidy
go run ./cmd/web
```

## Environment Configuration
The project reads `.env` through `godotenv` (if the file exists). If values are missing, code defaults will be used.

```env
APP_PORT=8080

DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=websitego

DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=25
DB_CONN_MAX_LIFETIME_MIN=10

SESSION_SECRET=change-this-secret
COOKIE_SECURE=false
COOKIE_SAMESITE=lax
```

Notes:
- `SESSION_SECRET` must not be empty.
- `COOKIE_SAMESITE`: `lax` (default), `strict`, or `none`.
- `COOKIE_SECURE=true` is recommended for HTTPS production.

## Running the Application

```bash
go mod tidy
go run ./cmd/web
```

The server runs at `http://localhost:8080` (or based on `APP_PORT`).

## Migrations
Manual migration commands:

```bash
go run ./cmd/migrate up
go run ./cmd/migrate down
go run ./cmd/migrate force <version>
```

Notes:
- `cmd/web` automatically runs `up` migration on startup.
- Migration files are in `internal/migrations/sql` and embedded into the binary.

## Seed Data
A seeder is available in `cmd/seed`:

```bash
go run ./cmd/seed
```

Optional number of `user` role records:

```bash
go run ./cmd/seed --users=10000
```

Seed behavior:
- Creates/updates the admin account (`admin@gmail.com` / `admin789`, role `admin`).
- Creates regular users with default password: `user12345`.
- Default number of users is `9,999,999` if `--users` is not provided.

Recommendation:
- For development, always use `--users` to avoid creating very large datasets unintentionally.

## Usage Flow
1. Open `/register` to create an account.
2. Sign in at `/login`.
3. After login, access `/dashboard`.
4. Logout via `/logout`.

## Endpoint Summary
- `GET /` redirects to `/login` or `/dashboard` depending on session state.
- `GET /register`, `POST /register` (guest only)
- `GET /login`, `POST /login` (guest only)
- `GET /logout`
- `GET /dashboard` (auth required)
- `POST /dashboard/users` (admin only)
- `POST /dashboard/users/:id/update` (admin only)
- `POST /dashboard/users/:id/delete` (admin only)

## Database Schema and Indexes
Available migration:
- `000004_init_schema`

Notes for legacy databases:
- If you previously used incremental migration versions, align the version with `force` before `up` (example: `go run ./cmd/migrate force 4`).

Indexes for dashboard/search optimization:
- `idx_users_dashboard_list (id, name, email, role)`
- `idx_users_name (name)`
- `idx_users_role (role)`

## Website Access Speed Features
1. Query indexing strategy
   - Dashboard list and search use indexes to avoid full table scans as data grows.
   - Composite index `idx_users_dashboard_list (id, name, email, role)` supports common list/search patterns.
   - Additional single-column indexes (`name`, `role`) help targeted filters.

2. Pagination for controlled query cost
   - Dashboard limits rows per request (`10/25/50/100`) so each response remains predictable.
   - Page jump controls help navigation without requesting excessively large payloads.
   - This keeps response time and memory usage more stable on big datasets.

3. Database connection pool tuning
   - `DB_MAX_OPEN_CONNS` controls max concurrent active DB connections.
   - `DB_MAX_IDLE_CONNS` keeps warm idle connections to reduce reconnect overhead.
   - `DB_CONN_MAX_LIFETIME_MIN` refreshes long-lived connections to reduce stale-connection risk.
   - These settings help balance throughput, latency, and DB server load.

4. Runtime consistency for performance
   - `cmd/web` runs migrations automatically so required indexes/schema stay present.
   - This reduces risk of performance drops caused by schema mismatch across environments.

5. Operational observability
   - Middleware filters expected client-disconnect errors from logs.
   - Cleaner logs make real latency/errors easier to detect and fix quickly.

## Project Structure
```text
cmd/
  web/       # web server
  migrate/   # migration CLI
  seed/      # seed data

internal/
  config/
  database/
  handlers/
  middleware/
  migrations/
  models/

templates/
assets/
```
