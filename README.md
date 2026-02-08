# Tessera

A self-hosted productivity platform combining file storage, email, documents, tasks, calendar, and contacts into a single unified application.

## Features

### File Management
- Upload, download, and organize files in nested folders
- Grid and list view modes with adjustable icon sizes
- Drag-and-drop uploads with progress tracking
- File previews (images, PDFs, video, audio, text)
- File versioning with restore support
- Star/favorite files for quick access
- Trash with restore and permanent delete
- Copy, move, and rename files/folders
- Storage quota tracking per user
- Chunked uploads (Tus protocol) for large files up to 10 GB
- File search across all user files
- Command palette (Ctrl+K) for quick actions
- Context menus for file operations
- Bulk actions (multi-select, batch delete/move/copy)

### File Sharing
- Public link sharing with optional password protection
- User-to-user sharing with view/edit permissions
- Expiring links with max download limits
- Share analytics (view count, download count, last accessed)
- Public share download page

### WebDAV & Drive Mounting
- Full WebDAV server for mounting Tessera storage as a local drive
- Accessible via Traefik at `http://localhost/webdav/` (port 80)
- **Recommended**: use [rclone](https://rclone.org/) to mount as a native drive letter
- Also compatible with Cyberduck, WinSCP, macOS Finder, and other WebDAV clients
- Authenticates with your Tessera email and password (Basic Auth)

### Email Client
- Multi-account IMAP/SMTP email support
- Gmail-style threaded conversation view
- Compose, reply, reply-all, and forward
- Rich text editor for composing emails
- Email attachments (view, download, save to files)
- Folder management (create, rename, delete, reorder, nested folders)
- Labels with custom colors
- Email rules/filters for automatic organization
- Star/flag emails
- Batch operations (mark read, star, move, delete, label)
- Search across emails
- HTML email signature per account
- Undo-send with configurable delay
- Draft auto-save
- Mark folder as read
- Unread counts per folder
- Background IMAP sync on a schedule

### Documents
- Rich text document editor (Tiptap/ProseMirror)
- Headings, bold, italic, underline, strikethrough, highlight
- Bullet lists, ordered lists, blockquotes, code blocks
- Text alignment (left, center, right, justify)
- Tables with resizable columns
- Image and link insertion
- Text color support
- Undo/redo
- Auto-save
- Documents stored as `.tdoc` files in the file system

### Tasks (Kanban Board)
- Kanban board with To Do, In Progress, and Done columns
- Drag-and-drop between columns
- Task groups for organization
- Priority levels (low, medium, high)
- Due dates
- Task descriptions
- Create tasks from emails
- Calendar integration (tasks with due dates appear on calendar)
- Done column grays out completed tasks
- Reorderable tasks within columns

### Calendar
- Monthly calendar view with event display
- Create, edit, and delete events
- All-day and timed events
- Color-coded events
- Linked task events (synced from tasks with due dates)

### Contacts
- Contact management with name, email, phone, company, notes
- Favorite/unfavorite contacts
- Search and filter contacts

### Admin Dashboard
- System statistics (users, files, storage)
- User management (create, edit, delete, activate/deactivate)
- Global settings management
- Module enable/disable (email, documents, tasks, calendar, contacts)
- Activity/audit logs
- Cache management
- Cleanup tools

### Security
- JWT authentication with short-lived access tokens and refresh tokens
- Two-factor authentication (TOTP) with backup codes
- AES-256 encryption for sensitive data (email passwords)
- Password hashing with bcrypt
- Rate limiting (100 requests/minute per IP)
- CORS protection
- Input sanitization and header injection prevention
- WebSocket authentication via short-lived tickets
- Session management

### Real-Time
- WebSocket support for live updates
- File change notifications (create, update, delete, move)
- Live folder subscriptions

---

## Tech Stack

### Backend
| Technology | Purpose |
|---|---|
| **Go 1.23** | API server |
| **Fiber v2.52** | HTTP framework |
| **PostgreSQL 16** (pgvector) | Primary database |
| **Redis 7** | Sessions, cache |
| **MinIO** | S3-compatible object storage |
| **go-imap/v2** | IMAP email sync |
| **pgx/v5** | PostgreSQL driver |
| **zerolog** | Structured logging |
| **golang-jwt/v5** | JWT authentication |
| **Prometheus** | Metrics export |

### Frontend
| Technology | Purpose |
|---|---|
| **Vue 3** | UI framework |
| **TypeScript** | Type safety |
| **Vite 5** | Build tool / dev server |
| **Pinia** | State management |
| **Vue Router** | Client-side routing |
| **Tailwind CSS 3** | Styling |
| **Tiptap** | Rich text editor |
| **Axios** | HTTP client |
| **vuedraggable** | Drag-and-drop (Kanban) |
| **PDF.js** | PDF rendering |
| **DOMPurify** | HTML sanitization |

### Infrastructure
| Technology | Purpose |
|---|---|
| **Docker Compose** | Container orchestration |
| **Traefik v3** | Reverse proxy |
| **Air** | Go hot-reload (dev) |

---

## Project Structure

```
tessera/
├── backend/
│   ├── cmd/                    # Application entrypoint
│   ├── internal/
│   │   ├── config/             # Environment configuration
│   │   ├── database/           # Database connection & migrations
│   │   ├── errors/             # Custom error types
│   │   ├── handlers/           # HTTP route handlers
│   │   ├── jobs/               # Background job queue & workers
│   │   ├── logger/             # Structured logging setup
│   │   ├── metrics/            # Prometheus metrics
│   │   ├── middleware/         # Auth, rate limiting, metrics
│   │   ├── models/             # Data models
│   │   ├── repository/         # Database queries
│   │   ├── security/           # Encryption, TOTP
│   │   ├── server/             # Server setup & routing
│   │   ├── services/           # Business logic
│   │   ├── storage/            # MinIO storage layer
│   │   ├── webdav/             # WebDAV server
│   │   └── websocket/          # WebSocket hub & handlers
│   └── migrations/             # SQL migration files
├── frontend/
│   └── src/
│       ├── api/                # Axios instance & interceptors
│       ├── components/         # Reusable Vue components
│       ├── composables/        # Vue composables
│       ├── layouts/            # Page layouts
│       ├── router/             # Vue Router configuration
│       ├── stores/             # Pinia stores
│       ├── styles/             # Global CSS
│       ├── types/              # TypeScript type definitions
│       ├── utils/              # Utility functions
│       └── views/              # Page-level components
├── migrations/                 # Database migration files
├── docker/
│   ├── grafana/                # Grafana provisioning (optional)
│   ├── postgres/               # Postgres init scripts
│   └── prometheus/             # Prometheus config (optional)
├── scripts/                    # Development scripts
├── docker-compose.yml
└── .env.example                # Environment variable template
```

---

## Getting Started

### Prerequisites
- **Docker** and **Docker Compose**

### Setup

1. **Clone the repository**
   ```bash
   git clone <repo-url> tessera
   cd tessera
   ```

2. **Create your environment file**
   ```bash
   cp .env.example .env
   ```
   For production, change `JWT_SECRET` and `ENCRYPTION_KEY` to secure random values.

3. **Start all services**
   ```bash
   docker compose up -d --build
   ```

4. **Access the application**
   | Service | URL |
   |---|---|
   | Frontend | http://localhost:3000 |
   | Backend API | http://localhost:8080 |
   | MinIO Console | http://localhost:9001 |
   | Traefik Dashboard | http://localhost:8081 |

5. **Create your account** — the first registered user automatically gets the `admin` role.

### Mount as a Drive (rclone)

Tessera includes a WebDAV server so you can mount your files as a native drive letter. The recommended client is [rclone](https://rclone.org/).

1. **Install rclone**
   - Windows: `winget install Rclone.Rclone`
   - macOS: `brew install rclone`
   - Linux: `sudo apt install rclone` or `curl https://rclone.org/install.sh | sudo bash`

2. **Configure the remote**
   ```bash
   rclone config
   ```
   | Prompt | Value |
   |---|---|
   | name | `tessera` |
   | Storage type | `webdav` |
   | url | `http://localhost/webdav/` |
   | vendor | `other` |
   | user | Your Tessera email |
   | password | Your Tessera password |

3. **Mount as a drive**

   **Windows** (mount as `T:\`):
   ```powershell
   rclone mount tessera:/ T: --vfs-cache-mode full
   ```

   **macOS / Linux** (mount to `~/tessera`):
   ```bash
   mkdir -p ~/tessera
   rclone mount tessera:/ ~/tessera --vfs-cache-mode full
   ```

   The `--vfs-cache-mode full` flag enables read/write caching for the best performance. The drive will appear in your file explorer and work like any other local drive.

4. **Auto-mount on startup (optional)**

   **Windows**: Create a shortcut in `shell:startup` that runs:
   ```
   rclone mount tessera:/ T: --vfs-cache-mode full --no-console
   ```

   **Linux (systemd)**:
   ```bash
   # ~/.config/systemd/user/rclone-tessera.service
   [Unit]
   Description=Mount Tessera via rclone
   After=network-online.target

   [Service]
   ExecStart=rclone mount tessera:/ %h/tessera --vfs-cache-mode full
   ExecStop=/bin/fusermount -u %h/tessera
   Restart=on-failure

   [Install]
   WantedBy=default.target
   ```
   ```bash
   systemctl --user enable --now rclone-tessera
   ```

> **Note**: Windows' built-in "Map Network Drive" feature has known limitations with HTTP WebDAV (file size caps, registry tweaks required). rclone avoids all of these issues and provides significantly better performance.

### Development

The backend uses [Air](https://github.com/air-verse/air) for hot-reload. The frontend uses Vite HMR. Both auto-refresh on file changes.

```bash
# Rebuild a single service
docker compose up -d --build backend
docker compose up -d --build frontend

# View logs
docker compose logs -f backend
docker compose logs -f frontend

# Access the database
docker exec -it tessera-postgres psql -U tessera -d tessera
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `APP_ENV` | `development` | `development` or `production` |
| `APP_DEBUG` | `true` | Enable debug logging |
| `SERVER_HOST` | `0.0.0.0` | Server bind address |
| `SERVER_PORT` | `8080` | Server port |
| `DB_HOST` | `postgres` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_NAME` | `tessera` | Database name |
| `DB_USER` | `tessera` | Database user |
| `DB_PASSWORD` | — | Database password |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `REDIS_HOST` | `redis` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | — | Redis password |
| `MINIO_ENDPOINT` | `minio:9000` | MinIO endpoint |
| `MINIO_ACCESS_KEY` | — | MinIO access key |
| `MINIO_SECRET_KEY` | — | MinIO secret key |
| `MINIO_BUCKET` | `tessera-files` | MinIO bucket name |
| `MINIO_USE_SSL` | `false` | Use SSL for MinIO |
| `JWT_SECRET` | — | JWT signing secret (**change in production**) |
| `JWT_EXPIRY` | `15m` | Access token lifetime |
| `JWT_REFRESH_EXPIRY` | `7d` | Refresh token lifetime |
| `ENCRYPTION_KEY` | — | 32-byte AES-256 key (**change in production**) |
| `MAX_UPLOAD_SIZE` | `10737418240` | Max upload size in bytes (10 GB) |
| `CHUNK_SIZE` | `10485760` | Chunk size for uploads (10 MB) |
| `FRONTEND_URL` | `http://localhost:3000` | Frontend URL for CORS |

---

## API Reference

Full API documentation is in [API.md](API.md).

---

## License

Private — all rights reserved.
