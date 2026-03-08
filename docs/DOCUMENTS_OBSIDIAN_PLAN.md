# Documents Module: Obsidian-like UI Implementation Plan

## Overview
Transform the Documents module into an Obsidian-style interface with a dedicated "Documents" sidebar tab, file-based document storage in a protected `Documents` folder, and a resizable document navigation sidebar. Documents are stored as `.tdoc`/`.md` files in the file system; only files within the Documents folder appear in this view.

---

## Phase 1: Documents Folder & Backend

### 1.1 Documents Folder Lifecycle
**Goal:** Ensure a `Documents` folder exists in My Files when the module is enabled, and protect it from deletion while enabled.

| Task | Description | Files |
|------|-------------|-------|
| 1.1.1 | Create documents folder when module is first enabled | `module_handler.go`, `file_service.go`, new `ensureDocumentsFolder` |
| 1.1.2 | Call folder creation from `UpdateModule` when `documents` is toggled on | `module_handler.go` |
| 1.1.3 | Store documents folder ID per user (e.g., settings or a convention: folder named "Documents" under root) | `settings_repository` or convention-based lookup |

**Option A (convention):** On first documents module enable, create folder "Documents" under root. Later look it up by name+parent.  
**Option B (settings):** Store `documents_folder_id` in user/settings. Simpler but requires migration.

**Recommended:** Option A — create "Documents" folder under root; resolve by listing root folders and finding one named "Documents" (case-insensitive).

### 1.2 Protect Documents Folder from Deletion
| Task | Description | Files |
|------|-------------|-------|
| 1.2.1 | Add `IsDocumentsFolder(ownerID, folderID)` check in file service | `file_service.go`, `file_repository.go` |
| 1.2.2 | In `Delete` and `PermanentDelete`, reject if folder is the Documents folder and documents module is enabled | `file_service.go` |
| 1.2.3 | Return clear error: `"Cannot delete the Documents folder while the Documents module is enabled. Disable it in Settings → Admin → Optional Modules to remove it."` | `file_handler.go` |

### 1.3 Documents Folder API
| Task | Description | Files |
|------|-------------|-------|
| 1.3.1 | Add `GET /files/documents-folder` (or `GET /files?documents_folder=1`) to return the documents folder ID for the current user | `file_handler.go`, `file_service.go` |
| 1.3.2 | If folder doesn't exist, create it and return it | — |

---

## Phase 2: Sidebar & Routing

### 2.1 Add Documents Tab to Sidebar
| Task | Description | Files |
|------|-------------|-------|
| 2.1.1 | Insert "Documents" between "My Files" and "Shared with Me" in `coreNavItems` (conditionally, when documents module enabled) | `Sidebar.vue` |
| 2.1.2 | Use route `/documents`; add document-text icon | `Sidebar.vue` |
| 2.1.3 | Refactor `coreNavItems` to support conditional items or split: always show My Files, conditionally show Documents, then Shared with Me, etc. | `Sidebar.vue` |

**Structure:**
- My Files
- **Documents** (if module enabled)
- Shared with Me
- Recent
- Starred
- Trash

### 2.2 Routing
| Task | Description | Files |
|------|-------------|-------|
| 2.2.1 | Keep existing routes: `/documents`, `/documents/:id` (folder), `/documents/doc/:fileId` (for editing a document file) | `router/index.ts` |
| 2.2.2 | Add nested routes for folder navigation: `/documents/folder/:folderId` | `router/index.ts` |

**Proposed routes:**
- `/documents` — root of Documents (shows documents folder contents)
- `/documents/folder/:folderId` — subfolder within Documents
- `/documents/doc/:fileId` — open document editor (or use query/hash)

---

## Phase 3: Documents View (Obsidian-like)

### 3.1 Data Model
**Use file system, not documents table.**  
- List files via `GET /files?parent_id=<documents_folder_id>`
- Create documents via `POST /documents/create-file` with `parentId: documentsFolderId`
- Update via `PUT /files/:id/content`

### 3.2 New Documents View Layout
| Task | Description | Files |
|------|-------------|-------|
| 3.2.1 | Replace `DocumentsView.vue` with Obsidian-style layout: resizable left sidebar + main content | `DocumentsView.vue` |
| 3.2.2 | Left sidebar: folder tree (only under Documents folder) + flat list of documents in current folder | New `DocumentsSidebar.vue` |
| 3.2.3 | Main area: either document list (when folder selected) or editor (when document opened) | `DocumentsView.vue` |
| 3.2.4 | Use `DocumentEditorModal` or inline editor for editing | Reuse `DocumentEditorModal.vue` or embed editor |

### 3.3 Documents Sidebar Component
| Task | Description | Files |
|------|-------------|-------|
| 3.3.1 | Create `DocumentsSidebar.vue` with: folder tree, document list, New Document, New Folder | `components/DocumentsSidebar.vue` |
| 3.3.2 | Collapsible folder tree (nested subfolders) | — |
| 3.3.3 | Click folder → navigate; click document → open editor | — |
| 3.3.4 | Resizable: min ~200px, max ~400px, default ~260px; persist width in localStorage | Use CSS `resize: horizontal` or custom drag handle |
| 3.3.5 | "New Document" and "New Folder" buttons in sidebar | — |

### 3.4 Documents Store / Composables
| Task | Description | Files |
|------|-------------|-------|
| 3.4.1 | Create `useDocumentsFolder` composable: fetch documents folder ID, list contents, create subfolder, create document | `composables/useDocumentsFolder.ts` |
| 3.4.2 | Use `filesStore` or new methods for listing; use `POST /documents/create-file` for new docs; use `POST /files/folder` for new folders | — |
| 3.4.3 | Consider deprecating or repurposing `documents.ts` store (currently uses `/documents` API with documents table) | `stores/documents.ts` |

**Decision:** Use files API exclusively for this view. The `documents` store/API may remain for other features (e.g. sharing) or be phased out.

---

## Phase 4: Mobile Experience

### 4.1 Documents Sidebar on Mobile
| Task | Description | Files |
|------|-------------|-------|
| 4.1.1 | On mobile: hide sidebar by default; show as overlay/drawer when user taps "Documents" or a menu icon | `DocumentsView.vue`, `DocumentsSidebar.vue` |
| 4.1.2 | Swipe or tap outside to close sidebar | — |
| 4.1.3 | When document is open, show editor full-screen; back button returns to sidebar or document list | — |

### 4.2 Touch Targets
| Task | Description | Files |
|------|-------------|-------|
| 4.2.1 | Ensure folder/document list items have min 44px height | `DocumentsSidebar.vue` |
| 4.2.2 | New Document / New Folder: adequate tap targets | — |

---

## Phase 5: Delete Warning for Documents Folder

### 5.1 Frontend Warning
| Task | Description | Files |
|------|-------------|-------|
| 5.1.1 | In FilesView delete confirmation: if file is the Documents folder and module enabled, show warning message instead of generic delete prompt | `FilesView.vue`, `ConfirmModal.vue` or equivalent |
| 5.1.2 | Message: "This folder can't be deleted while the Documents module is enabled. Disable it in Settings → Admin → Optional Modules to remove it." | — |
| 5.1.3 | Optionally disable Delete in context menu for Documents folder when module enabled (gray out or hide) | `FileGrid.vue`, `FolderContextMenu.vue` |
| 5.1.4 | If user attempts delete, show warning modal with single "OK" (no delete) | — |

### 5.2 Backend Enforcement
| Task | Description | Files |
|------|-------------|-------|
| 5.2.1 | Ensure backend rejects delete even if frontend is bypassed | `file_service.go` |

---

## Phase 6: Remove New Document from Files View

| Task | Description | Files |
|------|-------------|-------|
| 6.1 | Remove "New Document" button from FilesView toolbar (documents are created only from Documents tab) | `FilesView.vue` |
| 6.2 | Keep ability to open .tdoc/.md files from FilesView for preview/quick edit, or open in Documents view | Optional — can keep "Open" to open in Documents editor |

---

## Implementation Order

1. **Phase 1** — Backend: folder creation, protection, API  
2. **Phase 5** — Delete warning (frontend + backend) — quick win  
3. **Phase 2** — Sidebar tab  
4. **Phase 3** — Documents view + sidebar + composable  
5. **Phase 4** — Mobile polish  
6. **Phase 6** — Cleanup (remove New Document from FilesView if desired)

---

## File Checklist

| File | Changes |
|------|---------|
| `backend/internal/handlers/module_handler.go` | Create documents folder on enable |
| `backend/internal/handlers/file_handler.go` | Protect delete, add documents-folder endpoint |
| `backend/internal/services/file_service.go` | Documents folder logic, delete guard |
| `backend/internal/repository/settings_repository.go` | Optional: store folder ID |
| `frontend/src/components/Sidebar.vue` | Documents tab between My Files and Shared |
| `frontend/src/views/DocumentsView.vue` | Rewrite as Obsidian layout |
| `frontend/src/components/DocumentsSidebar.vue` | **New** — folder tree, doc list, resizable |
| `frontend/src/composables/useDocumentsFolder.ts` | **New** — folder ID, list, create |
| `frontend/src/views/FilesView.vue` | Delete warning for Documents folder |
| `frontend/src/router/index.ts` | Documents routes for folder/doc |
| `frontend/src/components/FolderContextMenu.vue` | Disable/handle delete for Documents folder |
| `frontend/src/components/FileGrid.vue` | Optional: context menu handling |

---

## Open Questions
- Should we keep the existing `documents` table/API for collaboration, or migrate fully to file-based?
- Should FilesView retain "Open" for .tdoc/.md to open in Documents editor, or always navigate to Documents tab?
