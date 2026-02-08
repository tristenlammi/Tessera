# Tessera API Reference

Base URL: `/api`

All protected endpoints require a Bearer token in the `Authorization` header:
```
Authorization: Bearer <access_token>
```

---

## Table of Contents

- [Authentication](#authentication)
- [Two-Factor Authentication](#two-factor-authentication)
- [Files](#files)
- [Uploads](#uploads)
- [Sharing](#sharing)
- [Trash](#trash)
- [Search](#search)
- [Email](#email)
- [Email Batch Operations](#email-batch-operations)
- [Email Labels](#email-labels)
- [Email Rules](#email-rules)
- [Email Attachments](#email-attachments)
- [Tasks](#tasks)
- [Task Groups](#task-groups)
- [Documents](#documents)
- [Calendar](#calendar)
- [Contacts](#contacts)
- [Admin](#admin)
- [Modules](#modules)
- [WebSocket](#websocket)
- [WebDAV](#webdav)
- [Health](#health)

---

## Authentication

### `GET /auth/setup-status`
Check if the app has been set up (any users exist).

**Response** `200`
```json
{ "setup_complete": true }
```

---

### `POST /auth/register`
Create a new account. The first user gets the `admin` role.

**Body**
```json
{
  "email": "user@example.com",
  "password": "securepassword",
  "name": "John Doe"
}
```

**Response** `201`
```json
{
  "user": { "id", "email", "name", "role", "timezone", ... },
  "access_token": "...",
  "refresh_token": "..."
}
```

---

### `POST /auth/login`
Authenticate and receive tokens. If 2FA is enabled, returns a pending token instead.

**Body**
```json
{
  "email": "user@example.com",
  "password": "securepassword",
  "totp_code": ""
}
```

**Response** `200` — standard login:
```json
{
  "user": { ... },
  "access_token": "...",
  "refresh_token": "..."
}
```

**Response** `200` — 2FA required:
```json
{
  "requires_2fa": true,
  "pending_auth_token": "..."
}
```

---

### `POST /auth/login/totp`
Complete 2FA login with TOTP code.

**Body**
```json
{
  "pending_auth_token": "...",
  "totp_code": "123456"
}
```

**Response** `200`
```json
{
  "user": { ... },
  "access_token": "...",
  "refresh_token": "..."
}
```

---

### `POST /auth/refresh`
Refresh an expired access token.

**Body**
```json
{ "refresh_token": "..." }
```

**Response** `200`
```json
{
  "access_token": "...",
  "refresh_token": "..."
}
```

---

### `POST /auth/logout` 🔒
Invalidate the current session.

**Response** `200`
```json
{ "message": "Logged out successfully" }
```

---

### `GET /auth/me` 🔒
Get the authenticated user's profile.

**Response** `200`
```json
{
  "user": { "id", "email", "name", "role", "timezone", "storage_used", "storage_limit", "totp_enabled", ... }
}
```

---

### `PUT /auth/password` 🔒
Change password.

**Body**
```json
{
  "old_password": "...",
  "new_password": "..."
}
```

---

### `PUT /auth/settings` 🔒
Update user settings (e.g. timezone).

**Body**
```json
{ "timezone": "America/New_York" }
```

---

### `POST /auth/forgot-password`
Request a password reset.

**Body**
```json
{ "email": "user@example.com" }
```

---

### `POST /auth/reset-password`
Reset password with token.

**Body**
```json
{
  "token": "...",
  "new_password": "..."
}
```

---

## Two-Factor Authentication

All endpoints require 🔒 authentication.

### `GET /auth/totp/status`
Check if TOTP 2FA is enabled.

**Response** `200`
```json
{ "enabled": false }
```

---

### `POST /auth/totp/setup`
Generate TOTP secret and QR code.

**Response** `200`
```json
{
  "secret": "...",
  "qr_code": "data:image/png;base64,..."
}
```

---

### `POST /auth/totp/confirm`
Confirm TOTP setup with a code from authenticator app.

**Body**
```json
{ "code": "123456" }
```

**Response** `200`
```json
{
  "message": "2FA enabled successfully",
  "backup_codes": ["code1", "code2", ...]
}
```

---

### `DELETE /auth/totp`
Disable TOTP 2FA.

**Body**
```json
{ "password": "..." }
```

---

### `POST /auth/totp/backup-codes`
Regenerate backup codes.

**Body**
```json
{ "password": "..." }
```

**Response** `200`
```json
{ "backup_codes": ["code1", "code2", ...] }
```

---

## Files

All endpoints require 🔒 authentication. Base path: `/files`

### `GET /files?parent_id=`
List files in a folder. Omit `parent_id` for root.

**Response** `200`
```json
{
  "files": [
    {
      "id": "uuid",
      "parent_id": null,
      "owner_id": "uuid",
      "name": "photo.jpg",
      "is_folder": false,
      "size": 1024,
      "mime_type": "image/jpeg",
      "is_starred": false,
      "is_trashed": false,
      "created_at": "2026-01-01T00:00:00Z",
      "updated_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

---

### `GET /files/:id`
Get file metadata.

---

### `POST /files/folder`
Create a folder.

**Body**
```json
{
  "name": "New Folder",
  "parent_id": "uuid or null"
}
```

---

### `PUT /files/:id`
Update file/folder (rename, move, star).

**Body**
```json
{
  "name": "Renamed",
  "parent_id": "uuid",
  "is_starred": true
}
```

---

### `DELETE /files/:id`
Move file to trash.

---

### `POST /files/:id/restore`
Restore file from trash.

---

### `POST /files/:id/copy`
Copy a file.

**Body**
```json
{ "destination_id": "uuid or null" }
```

---

### `GET /files/:id/download`
Download a file. Returns the raw file with correct `Content-Type` and `Content-Disposition` headers.

---

### `GET /files/:id/versions`
Get version history for a file.

---

### `POST /files/:id/versions/:version/restore`
Restore a previous file version.

---

### `GET /files/:id/content`
Get document (`.tdoc`) content as JSON.

---

### `PUT /files/:id/content`
Update document content.

**Body**
```json
{
  "title": "My Document",
  "content": "<p>HTML content</p>",
  "format": "html"
}
```

---

## Uploads

All endpoints require 🔒 authentication.

### `POST /files/upload`
Simple multipart file upload.

**Form Fields**
- `file` — the file
- `parent_id` — (optional) destination folder ID

---

### `POST /upload`
Initiate a chunked (Tus) upload.

---

### `PATCH /upload/:uploadId`
Send a chunk.

---

### `HEAD /upload/:uploadId`
Check upload progress.

---

## Sharing

All endpoints require 🔒 authentication.

### `POST /files/:id/share`
Create a public share link.

**Body**
```json
{
  "expires_in_days": 7,
  "password": "optional",
  "allow_download": true,
  "max_downloads": 100
}
```

---

### `POST /files/:id/share/user`
Share a file with another user.

**Body**
```json
{
  "email": "other@example.com",
  "permission": "view"
}
```

---

### `GET /files/:id/shares`
List all shares for a file.

---

### `GET /files/:id/share/analytics`
Get view/download analytics for a share.

---

### `DELETE /files/shares/:shareId`
Revoke a share.

---

### `GET /shared`
List files shared with the current user.

---

### `GET /share/:token` *(public)*
Get public share metadata.

---

### `GET /share/:token/download` *(public)*
Download a publicly shared file. Accepts `?password=` query param.

---

## Trash

All endpoints require 🔒 authentication.

### `GET /trash`
List trashed files.

### `DELETE /trash`
Permanently delete all trashed files.

---

## Search

### `GET /search?q=term` 🔒
Search files by name.

---

## Starred

### `GET /starred` 🔒
List all starred files.

---

## Storage

### `GET /storage` 🔒
Get storage usage stats.

**Response** `200`
```json
{
  "used": 1073741824,
  "limit": 10737418240,
  "used_pct": 10.0,
  "by_type": { "image/jpeg": 524288, "application/pdf": 549453 }
}
```

---

## Email

All endpoints require 🔒 authentication. Base path: `/email`

### Accounts

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/accounts` | List email accounts |
| `POST` | `/accounts` | Add email account |
| `GET` | `/accounts/:accountId` | Get account details |
| `PUT` | `/accounts/:accountId` | Update account |
| `DELETE` | `/accounts/:accountId` | Delete account |
| `POST` | `/accounts/:accountId/sync` | Trigger IMAP sync |
| `GET` | `/accounts/:accountId/sync/stream` | SSE sync progress stream |
| `PUT` | `/accounts/:accountId/signature` | Update email signature |
| `PUT` | `/accounts/:accountId/send-delay` | Set undo-send delay (seconds) |

**Create Account Body**
```json
{
  "name": "Work Email",
  "email_address": "user@example.com",
  "imap_host": "imap.gmail.com",
  "imap_port": 993,
  "imap_username": "user@example.com",
  "imap_password": "app-password",
  "imap_use_tls": true,
  "smtp_host": "smtp.gmail.com",
  "smtp_port": 587,
  "smtp_username": "user@example.com",
  "smtp_password": "app-password",
  "smtp_use_tls": true,
  "is_default": true
}
```

### Folders

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/accounts/:accountId/folders` | List folders |
| `GET` | `/accounts/:accountId/folders/tree` | Get folder tree (nested) |
| `POST` | `/accounts/:accountId/folders` | Create folder |
| `POST` | `/accounts/:accountId/folders/reorder` | Reorder folders |
| `PUT` | `/folders/:folderId` | Update folder |
| `DELETE` | `/folders/:folderId` | Delete folder |
| `PATCH` | `/folders/:folderId/move` | Move folder |
| `POST` | `/folders/:folderId/read` | Mark all emails in folder as read |
| `GET` | `/accounts/:accountId/counts` | Get unread counts per folder |

### Emails & Threads

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/folders/:folderId/emails` | List emails in folder |
| `GET` | `/folders/:folderId/threads` | List threads in folder |
| `GET` | `/threads/:threadId/emails` | Get emails in thread |
| `GET` | `/threads/:threadId/conversation` | Get conversation view |
| `POST` | `/accounts/:accountId/threads/reindex` | Rebuild thread index |
| `GET` | `/emails/:emailId` | Get single email |
| `PATCH` | `/emails/:emailId/read` | Mark read/unread |
| `PATCH` | `/emails/:emailId/star` | Star/unstar |
| `PATCH` | `/emails/:emailId/move` | Move to folder |
| `DELETE` | `/emails/:emailId` | Delete email |
| `GET` | `/accounts/:accountId/search?q=` | Search emails |
| `GET` | `/accounts/:accountId/starred` | List starred emails |
| `GET` | `/accounts/:accountId/drafts` | List draft emails |

### Sending

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/send` | Send email immediately |
| `POST` | `/send/queue` | Queue email (with undo-send delay) |
| `POST` | `/send/:sendId/cancel` | Cancel a queued send |

**Send Email Body**
```json
{
  "account_id": "uuid",
  "to": ["recipient@example.com"],
  "cc": [],
  "bcc": [],
  "subject": "Hello",
  "body": "<p>Email body</p>",
  "is_html": true,
  "reply_to_id": "optional-email-uuid"
}
```

### Drafts

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/drafts` | Save or update draft |
| `GET` | `/drafts/:draftId` | Get draft |
| `DELETE` | `/drafts/:draftId` | Delete draft |
| `GET` | `/accounts/:accountId/compose-drafts` | List compose drafts |

---

## Email Batch Operations

All require 🔒 authentication.

| Method | Endpoint | Body | Description |
|---|---|---|---|
| `POST` | `/batch/read` | `{ email_ids, is_read }` | Batch mark read/unread |
| `POST` | `/batch/star` | `{ email_ids, is_starred }` | Batch star/unstar |
| `POST` | `/batch/move` | `{ email_ids, folder_id }` | Batch move |
| `POST` | `/batch/delete` | `{ email_ids }` | Batch delete |
| `POST` | `/batch/label` | `{ email_ids, label_id }` | Batch assign label |

---

## Email Labels

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/accounts/:accountId/labels` | List labels |
| `POST` | `/accounts/:accountId/labels` | Create label `{ name, color }` |
| `PUT` | `/labels/:labelId` | Update label |
| `DELETE` | `/labels/:labelId` | Delete label |
| `GET` | `/labels/:labelId/emails` | Get emails with label |
| `GET` | `/emails/:emailId/labels` | Get labels on email |
| `POST` | `/emails/:emailId/labels/:labelId` | Assign label to email |
| `DELETE` | `/emails/:emailId/labels/:labelId` | Remove label from email |

---

## Email Rules

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/accounts/:accountId/rules` | List rules |
| `POST` | `/accounts/:accountId/rules` | Create rule |
| `PUT` | `/rules/:ruleId` | Update rule |
| `POST` | `/rules/:ruleId/run` | Run rule against existing emails |
| `DELETE` | `/rules/:ruleId` | Delete rule |

**Create Rule Body**
```json
{
  "name": "Auto-label",
  "is_enabled": true,
  "priority": 1,
  "match_type": "all",
  "conditions": [
    { "field": "from", "operator": "contains", "value": "github.com" }
  ],
  "actions": [
    { "type": "add_label", "value": "label-uuid" }
  ],
  "stop_processing": false
}
```

---

## Email Attachments

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/attachments/:attachmentId` | Get attachment metadata |
| `GET` | `/attachments/:attachmentId/download` | Download attachment file |

---

## Tasks

All endpoints require 🔒 authentication. Base path: `/tasks`

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/` | List all tasks |
| `POST` | `/` | Create task |
| `GET` | `/:id` | Get task |
| `PUT` | `/:id` | Update task |
| `PUT` | `/:id/move` | Move task (change status + order) |
| `DELETE` | `/:id` | Delete task |
| `PUT` | `/reorder` | Reorder tasks in a column |

**Create Task Body**
```json
{
  "title": "Fix bug",
  "description": "Details...",
  "status": "todo",
  "priority": "high",
  "due_date": "2026-02-15",
  "group_id": "optional-uuid",
  "tags": ["backend"],
  "linked_email_id": "optional-uuid",
  "linked_email_subject": "optional"
}
```

**Move Task Body**
```json
{
  "status": "in_progress",
  "order": 0
}
```

**Reorder Body**
```json
{
  "status": "todo",
  "task_ids": ["uuid1", "uuid2", "uuid3"]
}
```

Status values: `todo`, `in_progress`, `done`
Priority values: `low`, `medium`, `high`

---

## Task Groups

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/tasks/groups` | List groups |
| `POST` | `/tasks/groups` | Create group |
| `PUT` | `/tasks/groups/:id` | Update group |
| `DELETE` | `/tasks/groups/:id` | Delete group |

**Body**
```json
{
  "name": "Sprint 1",
  "color": "#3b82f6"
}
```

---

## Documents

All endpoints require 🔒 authentication. Base path: `/documents`

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/` | List documents |
| `POST` | `/` | Create document |
| `POST` | `/create-file` | Create document as a `.tdoc` file |
| `GET` | `/:id` | Get document |
| `PUT` | `/:id` | Update document (with optimistic locking) |
| `DELETE` | `/:id` | Delete document |
| `POST` | `/:id/share` | Share with user |
| `DELETE` | `/:id/share/:userId` | Remove collaborator |

**Create Document Body**
```json
{
  "title": "Meeting Notes",
  "content": "<p>Content here</p>",
  "format": "html"
}
```

**Update Document Body**
```json
{
  "title": "Updated Title",
  "content": "<p>Updated content</p>",
  "version": 1
}
```
Returns `409 Conflict` if the version doesn't match (another user edited).

---

## Calendar

All endpoints require 🔒 authentication. Base path: `/calendar`

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/events?start=&end=` | List events in date range |
| `POST` | `/events` | Create event |
| `GET` | `/events/:id` | Get event |
| `PUT` | `/events/:id` | Update event |
| `DELETE` | `/events/:id` | Delete event |
| `DELETE` | `/events/by-task/:taskId` | Delete event linked to a task |

**Create Event Body**
```json
{
  "title": "Team Standup",
  "description": "Daily sync",
  "start_date": "2026-02-10T09:00:00Z",
  "end_date": "2026-02-10T09:30:00Z",
  "all_day": false,
  "color": "#3b82f6",
  "linked_task_id": "optional-uuid"
}
```

---

## Contacts

All endpoints require 🔒 authentication. Base path: `/contacts`

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/` | List contacts |
| `POST` | `/` | Create contact |
| `GET` | `/:id` | Get contact |
| `PUT` | `/:id` | Update contact |
| `PATCH` | `/:id/favorite` | Toggle favorite |
| `DELETE` | `/:id` | Delete contact |

**Create Contact Body**
```json
{
  "first_name": "Jane",
  "last_name": "Doe",
  "email": "jane@example.com",
  "phone": "+1234567890",
  "company": "Acme Corp",
  "job_title": "Engineer",
  "birthday": "1990-05-15",
  "notes": "Met at conference"
}
```

---

## Admin

All endpoints require 🔒 authentication with `admin` role. Base path: `/admin`

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/stats` | System statistics |
| `GET` | `/settings` | Get global settings |
| `PATCH` | `/settings` | Update global settings |
| `GET` | `/users` | List all users |
| `POST` | `/users` | Create user |
| `GET` | `/users/:id` | Get user details |
| `PATCH` | `/users/:id` | Update user |
| `DELETE` | `/users/:id` | Delete user |
| `GET` | `/logs` | Get audit/activity logs |
| `POST` | `/cache/clear` | Clear Redis cache |
| `POST` | `/cleanup` | Run cleanup job |
| `GET` | `/modules` | Get module configuration |
| `PUT` | `/modules/:id` | Enable/disable a module |
| `PUT` | `/modules` | Update all modules |

---

## Modules

### `GET /modules` 🔒
Get enabled/disabled state of optional modules.

**Response** `200`
```json
{
  "modules": [
    { "id": "email", "name": "Email", "enabled": true },
    { "id": "documents", "name": "Documents", "enabled": true },
    { "id": "tasks", "name": "Tasks", "enabled": true },
    { "id": "calendar", "name": "Calendar", "enabled": true },
    { "id": "contacts", "name": "Contacts", "enabled": true }
  ]
}
```

---

## WebSocket

### `GET /ws?ticket=`
Connect to the real-time WebSocket endpoint. Requires a short-lived ticket obtained from `GET /auth/ws-ticket`.

**Get ticket** 🔒
```
GET /auth/ws-ticket → { "ticket": "..." }
```

**Events received:**
- `file:created` — a file was created
- `file:updated` — a file was updated
- `file:deleted` — a file was deleted
- `file:moved` — a file was moved
- `file:restored` — a file was restored from trash

---

## WebDAV

WebDAV is available at `/webdav/`. Authenticate with email and password. Compatible with:

- Windows: Map Network Drive → `http://localhost:8080/webdav/`
- macOS: Finder → Connect to Server → `http://localhost:8080/webdav/`
- Linux: `davfs2` or any WebDAV client
- Cyberduck, WinSCP, etc.

---

## Health

### `GET /health`
Liveness check. Returns `200` if the server is running.

### `GET /ready`
Readiness check. Verifies database, Redis, and MinIO connectivity.

### `GET /metrics`
Prometheus metrics endpoint.

### `GET /jobs/stats`
Background job queue statistics.

---

## Error Format

All errors return a consistent JSON format:

```json
{
  "error": "Description of the error"
}
```

Common HTTP status codes:
- `400` — Bad request / validation error
- `401` — Unauthorized / invalid token
- `403` — Forbidden / insufficient permissions
- `404` — Resource not found
- `409` — Conflict (e.g. document version mismatch)
- `429` — Rate limited
- `500` — Internal server error

---

## Rate Limiting

All endpoints are rate limited to **100 requests per minute** per IP address using a sliding window algorithm. When exceeded, the API returns `429 Too Many Requests`.
