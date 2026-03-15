# WebsiteGo

Aplikasi web Go untuk manajemen user dengan stack:
- Gin (web framework)
- GORM + MySQL
- Session cookie (`gin-contrib/sessions`)
- SQL migration embedded (`golang-migrate`)
- Auth register/login/logout + role (`admin`/`user`)
- Dashboard user dengan search, pagination, dan CRUD khusus admin

## Fitur Utama
- Register & login dengan hash password `bcrypt`.
- User pertama saat register otomatis menjadi `admin`.
- Fallback auto-promote: jika tidak ada admin, user yang login bisa dipromosikan jadi admin.
- Route terlindungi untuk dashboard (`/dashboard`).
- Lihat daftar user di dashboard (semua user login).
- Search dashboard berdasarkan `all`, `id`, `name`, `email`, `role`.
- Pagination dashboard (`10/25/50/100`) + lompat halaman `-100/-1000/+100/+1000`.
- Tambah / update / delete user dari dashboard (admin only).
- Cegah admin menghapus akun sendiri.
- Migrasi SQL otomatis dijalankan saat `cmd/web` start.
- Middleware untuk memfilter error client disconnect agar log lebih bersih.

## Prasyarat
- Go `1.25.x` (mengikuti `go.mod`)
- MySQL aktif (contoh: XAMPP MySQL)

## Setup Database
Buat database:

```sql
CREATE DATABASE websitego CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

## Quick Start

```bash
copy .env.example .env
go mod tidy
go run ./cmd/web
```

## Konfigurasi Environment
Project membaca `.env` via `godotenv` (jika file ada). Jika tidak diisi, fallback default dari kode akan dipakai.

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

Keterangan:
- `SESSION_SECRET` tidak boleh kosong.
- `COOKIE_SAMESITE`: `lax` (default), `strict`, atau `none`.
- `COOKIE_SECURE=true` direkomendasikan untuk HTTPS production.

## Menjalankan Aplikasi

```bash
go mod tidy
go run ./cmd/web
```

Server berjalan di `http://localhost:8080` (atau sesuai `APP_PORT`).

## Migrasi
Perintah manual migrasi:

```bash
go run ./cmd/migrate up
go run ./cmd/migrate down
go run ./cmd/migrate force <version>
```

Catatan:
- `cmd/web` otomatis menjalankan `up` migration saat startup.
- File migration berada di `internal/migrations/sql` dan di-embed ke binary.

## Seed Data
Tersedia seeder di `cmd/seed`:

```bash
go run ./cmd/seed
```

Opsional jumlah user role `user`:

```bash
go run ./cmd/seed --users=10000
```

Perilaku seed:
- Membuat/mengupdate akun admin (`admin@gmail.com` / `admin789`, role `admin`).
- Membuat user biasa dengan password default: `user12345`
- Default jumlah user adalah `9,999,999` jika flag `--users` tidak diisi.

Rekomendasi:
- Untuk development, selalu gunakan `--users` agar tidak membuat data terlalu besar tanpa sengaja.

## Alur Penggunaan
1. Buka `/register` untuk membuat akun.
2. Login di `/login`.
3. Setelah login, akses `/dashboard`.
4. Logout via `/logout`.

## Endpoint Ringkas
- `GET /` redirect ke `/login` atau `/dashboard` tergantung session.
- `GET /register`, `POST /register` (guest only)
- `GET /login`, `POST /login` (guest only)
- `GET /logout`
- `GET /dashboard` (auth required)
- `POST /dashboard/users` (admin only)
- `POST /dashboard/users/:id/update` (admin only)
- `POST /dashboard/users/:id/delete` (admin only)

## Skema dan Index Database
Migration yang tersedia:
- `000004_init_schema`

Catatan untuk database lama:
- Jika sebelumnya pernah memakai versi migrasi bertahap, sesuaikan versi dengan `force` sebelum `up` (contoh: `go run ./cmd/migrate force 4`).

Index untuk optimasi dashboard/search:
- `idx_users_dashboard_list (id, name, email, role)`
- `idx_users_name (name)`
- `idx_users_role (role)`

## Struktur Project
```text
cmd/
  web/       # web server
  migrate/   # CLI migration
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
