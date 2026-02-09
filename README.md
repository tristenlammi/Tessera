# Tessera

A self-hosted productivity platform that combines file management, documents, tasks, calendar, contacts, and email into one cohesive app. Built with Go and Vue 3.

## Features

- **File Management** — Upload, organize, version, share, and access files via WebDAV
- **Documents** — Rich text editor with real-time collaboration
- **Tasks** — Kanban board with groups, checklists, and recurring tasks
- **Calendar** — Events with reminders and recurring schedules
- **Contacts** — Contact management with notes and custom fields
- **Email** — IMAP/SMTP email client with threading and labels
- **Security** — Two-factor authentication (TOTP), AES-256 encryption, JWT auth
- **Admin** — User management, module toggles, system settings, audit logs
- **Mobile-friendly** — Responsive design with dark mode and PWA support

## Deployment

Tessera offers two deployment options:

### Option 1: All-in-One Container (Recommended for Unraid / Homelab)

A single Docker container that includes everything — PostgreSQL, Redis, MinIO, the Go backend, and Nginx. No external dependencies. Secrets are auto-generated on first run.

```bash
docker run -d \
  --name tessera \
  -p 8080:8080 \
  -v /path/to/data:/data \
  --restart unless-stopped \
  ghcr.io/tessera/tessera:latest
```

Open `http://localhost:8080` and create your admin account.

**Unraid users**: Install from Community Applications using the Tessera template, or use the Docker Compose plugin.

#### Environment Variables (Optional)

All secrets are auto-generated on first run and saved to `/data/.tessera-secrets`. You only need to set these if you want to override the defaults:

| Variable | Description | Default |
|----------|-------------|---------|
| `FRONTEND_URL` | Public URL for CORS and links | `http://localhost:8080` |
| `JWT_SECRET` | JWT signing secret | Auto-generated |
| `ENCRYPTION_KEY` | AES-256 key (base64) | Auto-generated |
| `DB_PASSWORD` | PostgreSQL password | Auto-generated |
| `REDIS_PASSWORD` | Redis password | Auto-generated |
| `MINIO_ACCESS_KEY` | MinIO access key | `tessera` |
| `MINIO_SECRET_KEY` | MinIO secret key | Auto-generated |
| `MAX_UPLOAD_SIZE` | Max upload in bytes | `10737418240` (10 GB) |

#### Backups

The database, file storage, and configuration are all stored in `/data`. Back up this directory regularly. Database backups are stored in `/data/backups`.

---

### Option 2: Docker Compose (Advanced)

For users who want to run each service separately — useful if you already have a PostgreSQL or Redis instance, or want more control over resource allocation.

```bash
# 1. Copy and edit the environment file
cp .env.production.example .env.production

# 2. Generate secrets (Linux/macOS)
sed -i "s|CHANGE_ME_JWT|$(openssl rand -hex 32)|" .env.production
sed -i "s|CHANGE_ME_ENCRYPTION|$(openssl rand -base64 32)|" .env.production
sed -i "s|CHANGE_ME_DB|$(openssl rand -hex 24)|" .env.production
sed -i "s|CHANGE_ME_REDIS|$(openssl rand -hex 24)|" .env.production
sed -i "s|CHANGE_ME_MINIO|$(openssl rand -hex 24)|" .env.production

# 3. Start everything
docker compose -f docker-compose.prod.yml --env-file .env.production up -d --build
```

The frontend (Nginx) is exposed on port 8080 by default. Point your reverse proxy (Caddy, Traefik, Cloudflare Tunnel, etc.) at it.

See `.env.production.example` for all available configuration options.

---

## Development

### Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/)

That's it. Everything runs in containers.

### Getting Started

```bash
git clone https://github.com/tessera/tessera.git
cd tessera
docker compose up --build -d
```

- **Frontend**: http://localhost:3000 (Vite dev server with hot reload)
- **Backend API**: http://localhost:8080
- **MinIO Console**: http://localhost:9001 (user: `tessera_access` / pass: `tessera_secret_dev`)

The backend uses [Air](https://github.com/air-verse/air) for hot-reloading Go code. The frontend uses Vite's HMR.

### Project Structure

```
tessera/
├── backend/                 # Go backend (Fiber)
│   ├── cmd/server/          # Server entrypoint
│   ├── internal/
│   │   ├── config/          # Configuration loading
│   │   ├── database/        # PostgreSQL & Redis connections, migrations
│   │   ├── handlers/        # HTTP route handlers
│   │   ├── middleware/       # Auth, rate limiting, logging
│   │   ├── models/          # Data models
│   │   ├── repository/      # Database access layer
│   │   ├── server/          # Server setup & routing
│   │   ├── services/        # Business logic
│   │   ├── storage/         # MinIO file storage
│   │   ├── webdav/          # WebDAV server
│   │   └── websocket/       # WebSocket hub
│   └── Dockerfile           # Production build
├── frontend/                # Vue 3 frontend
│   ├── src/
│   │   ├── api/             # API client (Axios)
│   │   ├── components/      # Reusable components
│   │   ├── composables/     # Vue composables
│   │   ├── layouts/         # Page layouts
│   │   ├── router/          # Vue Router
│   │   ├── stores/          # Pinia stores
│   │   └── views/           # Page views
│   └── Dockerfile           # Production build (Nginx)
├── migrations/              # SQL database migrations
├── docker/
│   ├── aio/                 # All-in-one container files
│   ├── backup/              # Backup container
│   ├── postgres/            # PostgreSQL init script
│   └── traefik/             # Dev reverse proxy config
├── docker-compose.yml       # Development environment
├── docker-compose.prod.yml  # Production (multi-container)
└── tessera.xml              # Unraid template
```

### Tech Stack

- **Backend**: Go 1.23, Fiber, pgx (PostgreSQL), go-redis, minio-go
- **Frontend**: Vue 3, TypeScript, Tailwind CSS, Pinia, Tiptap
- **Database**: PostgreSQL 16 (with pgvector), Redis 7
- **Storage**: MinIO (S3-compatible)
- **Migrations**: golang-migrate

## License

MIT
