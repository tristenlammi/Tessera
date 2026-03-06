# Tessera PWA – Mobile Implementation Plan

This plan outlines the work required to make the Tessera PWA mobile-friendly while keeping desktop behavior unchanged. Work is organized into phases with clear dependencies.

---

## Principles

- **Breakpoint**: Use `md` (768px) as the mobile/desktop boundary. Below `md`, apply mobile-specific layout and behavior.
- **Desktop preservation**: No changes to layout, components, or behavior at `md` and above.
- **Touch-first on mobile**: Minimum 44×44px tap targets, no hover-only interactions.
- **Safe areas**: Account for notches and home indicators on all fixed/overlay UI.

---

## Phase 1: Shell & Foundation

**Goal**: Mobile-friendly app shell (sidebar, header, safe areas).

### 1.1 Sidebar – Drawer on Mobile

| Task | Description | Files |
|------|-------------|-------|
| 1.1.1 | On `< md`, hide sidebar by default; show as full-width overlay drawer when hamburger is tapped | `MainLayout.vue`, `Sidebar.vue` |
| 1.1.2 | Add overlay backdrop when sidebar is open on mobile; tap outside or route change closes it | `MainLayout.vue` |
| 1.1.3 | On navigation (RouterLink click), close sidebar when on mobile | `Sidebar.vue` or `MainLayout.vue` |
| 1.1.4 | Sidebar: `fixed` or `absolute` on mobile, `relative`/normal flow on desktop | `Sidebar.vue` |
| 1.1.5 | Add transition for open/close (e.g. slide from left) | `Sidebar.vue` |

### 1.2 Header – Mobile Adaptations

| Task | Description | Files |
|------|-------------|-------|
| 1.2.1 | On mobile, hide or shorten storage text (e.g. `hidden md:block` or truncated) | `MainLayout.vue` |
| 1.2.2 | Ensure hamburger and user menu buttons meet 44px min tap target | `MainLayout.vue` |
| 1.2.3 | User dropdown: ensure it doesn’t overflow viewport on small heights | `MainLayout.vue` |

### 1.3 Safe Areas & Viewport

| Task | Description | Files |
|------|-------------|-------|
| 1.3.1 | Add `env(safe-area-inset-*)` or Tailwind `pb-safe`/`pt-safe` for fixed header, sidebar, bottom bars | `main.css`, layout components |
| 1.3.2 | Add `viewport-fit=cover` to viewport meta if using full-screen PWA | `index.html` |
| 1.3.3 | Create utility classes or Tailwind config for safe-area padding where needed | `tailwind.config.js`, `main.css` |

**Phase 1 Deliverables**: Sidebar works as drawer on mobile, closes on navigate/backdrop; header fits small screens; safe areas applied to shell.

---

## Phase 2: Files View

**Goal**: Files page usable on mobile (toolbar, breadcrumbs, actions, modals).

### 2.1 Toolbar & Breadcrumbs

| Task | Description | Files |
|------|-------------|-------|
| 2.1.1 | Make toolbar wrap or collapse on mobile: move Search, Upload, New Folder, New Document, view toggle into a responsive layout (e.g. flex-wrap, or overflow menu) | `FilesView.vue` |
| 2.1.2 | Breadcrumbs: truncate long paths (e.g. “My Files / … / Current”) or horizontal scroll | `FilesView.vue` |
| 2.1.3 | On mobile, move secondary actions (e.g. New Folder, New Document, icon size) into a “⋮” overflow menu if needed | `FilesView.vue` |
| 2.1.4 | Ensure all toolbar buttons meet 44px min tap target on mobile | `FilesView.vue` |

### 2.2 File Actions – Touch Support

| Task | Description | Files |
|------|-------------|-------|
| 2.2.1 | Add long-press handler to file items to open context menu on touch (or expose “⋮” on each row/card) | `FileGrid.vue`, `FileList.vue` |
| 2.2.2 | ContextMenu: ensure it stays on-screen on mobile (reposition or use bottom sheet pattern) | `ContextMenu.vue`, `FolderContextMenu.vue` |
| 2.2.3 | FolderContextMenu: same as above for folder long-press or “⋮” | `FolderContextMenu.vue` |

### 2.3 Modals (Files Context)

| Task | Description | Files |
|------|-------------|-------|
| 2.3.1 | CreateFolderModal, ShareModal, RenameModal, MoveDialog, ConfirmModal: full-width on mobile, internal scroll, safe-area | Shared modal styles or per-component |
| 2.3.2 | FilePreview: ensure close and download buttons have 44px tap target and safe-area | `FilePreview.vue` |
| 2.3.3 | VersionHistory: full-width on mobile, scrollable content | `VersionHistory.vue` |

### 2.4 BulkActionsBar

| Task | Description | Files |
|------|-------------|-------|
| 2.4.1 | Add bottom safe-area padding | `BulkActionsBar.vue` |
| 2.4.2 | Ensure buttons meet 44px tap target; consider horizontal scroll if many actions | `BulkActionsBar.vue` |

**Phase 2 Deliverables**: Files page toolbar and breadcrumbs usable on mobile; file actions work via touch; modals and bulk bar are mobile-friendly.

---

## Phase 3: Documents

**Goal**: Document editor and list work well on mobile.

### 3.1 Documents List View

| Task | Description | Files |
|------|-------------|-------|
| 3.1.1 | Ensure document cards/list have adequate tap targets and spacing | `DocumentsView.vue` |
| 3.1.2 | New Document modal: full-width on mobile | `DocumentsView.vue` |

### 3.2 Document Editor – Toolbar

| Task | Description | Files |
|------|-------------|-------|
| 3.2.1 | On mobile, make EditorToolbar scrollable horizontally with min tap size 44px, or group into overflow menu(s) | `EditorToolbar.vue` |
| 3.2.2 | Option A: Single scrollable row with `overflow-x-auto` and `min-w` per button. Option B: “Format”, “Insert”, etc. dropdowns. | `EditorToolbar.vue` |
| 3.2.3 | Slash command popup: ensure it stays visible and doesn’t overlap virtual keyboard | `slashCommandsRender.ts`, tippy config |

### 3.3 DocumentEditorModal

| Task | Description | Files |
|------|-------------|-------|
| 3.3.1 | Modal: full-width on mobile (`max-w-[100vw]` or similar), safe-area padding | `DocumentEditorModal.vue` |
| 3.3.2 | Header: stack title/status and actions on two rows on mobile if needed | `DocumentEditorModal.vue` |
| 3.3.3 | Editor content area: ensure it scrolls and avoids keyboard overlap where possible | `DocumentEditorModal.vue` |

### 3.4 Documents Full-Page Editor

| Task | Description | Files |
|------|-------------|-------|
| 3.4.1 | Same toolbar treatment as modal | `DocumentsView.vue`, `EditorToolbar.vue` |
| 3.4.2 | Top bar (Back, title, Share, Save): responsive layout, adequate tap targets | `DocumentsView.vue` |

**Phase 3 Deliverables**: Document list and editor usable on mobile; toolbar is tap-friendly; modals and editor scale correctly.

---

## Phase 4: Shared, Recent, Starred, Trash

**Goal**: Other file views are mobile-friendly.

### 4.1 SharedView, RecentView, StarredView, TrashView

| Task | Description | Files |
|------|-------------|-------|
| 4.1.1 | Apply same toolbar/action patterns as FilesView (wrap, overflow menu, touch targets) | `SharedView.vue`, `RecentView.vue`, `StarredView.vue`, `TrashView.vue` |
| 4.1.2 | If list + preview are side-by-side, stack vertically on mobile | Per view |
| 4.1.3 | SharedView: fix FilePreview integration for shared files; ensure preview works on mobile | `SharedView.vue`, `FilePreview.vue` |

**Phase 4 Deliverables**: All file-related views behave consistently on mobile.

---

## Phase 5: Email

**Goal**: Email works as a single-panel flow on mobile.

### 5.1 Three-Panel Layout

| Task | Description | Files |
|------|-------------|-------|
| 5.1.1 | Below `lg`, show only one of: folders | list | reading pane at a time | `EmailView.vue` |
| 5.1.2 | Folders: full-width panel with back button to return to list when coming from list/message | `EmailView.vue` |
| 5.1.3 | List: full-width; tap opens message (full-width reading pane with back to list) | `EmailView.vue` |
| 5.1.4 | Use `lg:flex` / `hidden` so panels swap; add back button when showing list or message | `EmailView.vue` |
| 5.1.5 | Folders sidebar: on mobile, either first screen or slide-out drawer | `EmailView.vue` |

### 5.2 Compose & Modals

| Task | Description | Files |
|------|-------------|-------|
| 5.2.1 | EmailCompose: full-screen on mobile, safe-area, scrollable; ensure keyboard doesn’t obscure Send | `EmailCompose.vue` |
| 5.2.2 | EmailAccountSetup, EmailLabelManager, EmailRuleManager: full-width, scrollable, touch-friendly | Per component |
| 5.2.3 | Folder create/rename, label menu, etc.: full-width or well-positioned on mobile | Per component |

**Phase 5 Deliverables**: Email is usable on mobile with clear navigation between folders, list, and message.

---

## Phase 6: Admin

**Goal**: Admin dashboard usable on mobile.

### 6.1 Tabs

| Task | Description | Files |
|------|-------------|-------|
| 6.1.1 | Make tabs horizontally scrollable on mobile, or replace with `<select>` for tab switch | `AdminDashboard.vue` |
| 6.1.2 | Ensure tab buttons meet 44px tap target | `AdminDashboard.vue` |

### 6.2 Tables

| Task | Description | Files |
|------|-------------|-------|
| 6.2.1 | Users table: keep `overflow-x-auto`; ensure row height and action buttons ≥ 44px | `AdminDashboard.vue` |
| 6.2.2 | Activity Logs table: same treatment | `AdminDashboard.vue` |
| 6.2.3 | Option: card layout for user rows on mobile instead of table | `AdminDashboard.vue` |

### 6.3 Admin Modals

| Task | Description | Files |
|------|-------------|-------|
| 6.3.1 | Edit user, Create user modals: full-width on mobile, scrollable | `AdminDashboard.vue` |
| 6.3.2 | All admin form fields and buttons: adequate tap targets | `AdminDashboard.vue` |

**Phase 6 Deliverables**: Admin tabs, tables, and modals work on mobile.

---

## Phase 7: Tasks, Calendar, Contacts

**Goal**: Module-specific mobile layouts.

### 7.1 Tasks

| Task | Description | Files |
|------|-------------|-------|
| 7.1.1 | Kanban: on mobile, use horizontal scroll with visible scroll hint, or switch to single-column list view with status filter | `TasksView.vue` |
| 7.1.2 | Task create/edit modal: full-width, scrollable; date/time and checklist touch-friendly | `TasksView.vue` |
| 7.1.3 | Group modal and delete confirm: full-width on mobile | `TasksView.vue` |

### 7.2 Calendar

| Task | Description | Files |
|------|-------------|-------|
| 7.2.1 | Month grid: ensure day cells are at least 44px or use list view for month on very small screens | `CalendarView.vue` |
| 7.2.2 | View switcher (month/week/day) and nav: adequate tap targets | `CalendarView.vue` |
| 7.2.3 | Event modal: full-width, scrollable, touch-friendly date/time | `CalendarView.vue` |

### 7.3 Contacts

| Task | Description | Files |
|------|-------------|-------|
| 7.3.1 | Master/detail: stack vertically on mobile; list full-width, detail below or in slide-up panel | `ContactsView.vue` |
| 7.3.2 | Contact create/edit and delete modals: full-width, scrollable | `ContactsView.vue` |
| 7.3.3 | Search and list: adequate tap targets | `ContactsView.vue` |

**Phase 7 Deliverables**: Tasks, Calendar, and Contacts are usable on mobile.

---

## Phase 8: Settings, Auth, Shared Modals

**Goal**: Polish remaining pages and shared components.

### 8.1 Settings

| Task | Description | Files |
|------|-------------|-------|
| 8.1.1 | Add safe-area padding; ensure form controls and toolbar toggles meet 44px tap target | `SettingsView.vue` |
| 8.1.2 | 2FA modals: full-width on mobile, scrollable | `SettingsView.vue` |

### 8.2 Auth & Public

| Task | Description | Files |
|------|-------------|-------|
| 8.2.1 | Login, Register, Reset, Public Share: ensure buttons and inputs are 44px min height; add safe-area padding | `LoginView.vue`, `RegisterView.vue`, `ResetPasswordView.vue`, `PublicShareView.vue` |

### 8.3 Command Palette

| Task | Description | Files |
|------|-------------|-------|
| 8.3.1 | Full-width on mobile; ensure search stays above keyboard; results scrollable | `CommandPalette.vue` |
| 8.3.2 | Result items: 44px min tap target | `CommandPalette.vue` |

### 8.4 Global Modal Styles

| Task | Description | Files |
|------|-------------|-------|
| 8.4.1 | Define shared modal styles: `max-w-[min(theme,max(100vw,100%))]` on mobile, `max-h-[100dvh]` with overflow, safe-area | `main.css` or composable |
| 8.4.2 | Apply to ConfirmModal, MoveDialog, SaveToFolderPicker, and any remaining modals | Per component or via shared wrapper |

**Phase 8 Deliverables**: Settings, auth, command palette, and shared modals are mobile-friendly.

---

## Phase 9: Global Touch Targets & Polish

**Goal**: Consistent touch experience and accessibility.

### 9.1 Touch Targets

| Task | Description | Files |
|------|-------------|-------|
| 9.1.1 | Audit all interactive elements: buttons, links, icon buttons, table row actions | All |
| 9.1.2 | Add `min-h-[44px] min-w-[44px]` or equivalent for primary actions on mobile | Per component |
| 9.1.3 | Ensure icon-only buttons have sufficient padding to meet 44px | Per component |

### 9.2 Slash Command Menu & Tippy

| Task | Description | Files |
|------|-------------|-------|
| 9.2.1 | SlashCommandMenu / tippy: ensure popup stays on-screen, doesn’t overlap keyboard | `slashCommandsRender.ts`, `SlashCommandMenu.vue` |
| 9.2.2 | Menu items: 44px min height | `SlashCommandMenu.vue` |

### 9.3 PDF Viewer (if used on mobile)

| Task | Description | Files |
|------|-------------|-------|
| 9.3.1 | PDFViewer: ensure controls and navigation are touch-friendly; consider full-screen on mobile | `PDFViewer.vue` |

**Phase 9 Deliverables**: Touch targets are consistent; popups and viewers work on mobile.

---

## Implementation Order Summary

| Phase | Focus | Depends On |
|-------|-------|------------|
| 1 | Shell, sidebar drawer, safe areas | — |
| 2 | Files view | Phase 1 |
| 3 | Documents | Phase 1 |
| 4 | Shared, Recent, Starred, Trash | Phase 2 |
| 5 | Email | Phase 1 |
| 6 | Admin | Phase 1 |
| 7 | Tasks, Calendar, Contacts | Phase 1 |
| 8 | Settings, Auth, Command Palette, modals | Phase 1 |
| 9 | Touch targets, slash menu, PDF | Phases 2–8 |

**Suggested execution**: Phase 1 first, then Phases 2 and 3 in parallel (or 2 then 3). Phases 4–8 can be parallelized after Phase 1. Phase 9 is a final pass.

---

## Testing Checklist (per phase)

- [ ] Layout at 375px, 414px, 390px width (iPhone SE, Pro, 14)
- [ ] Layout at 360px, 412px (common Android)
- [ ] Safe area with notch (e.g. iPhone 14 Pro simulator)
- [ ] Portrait and landscape
- [ ] Virtual keyboard doesn’t hide critical inputs or buttons
- [ ] All primary actions have ≥ 44px tap target
- [ ] No horizontal scroll on main content (except intentional, e.g. tables)
- [ ] Desktop (`md`+) unchanged

---

## File Change Overview

| Category | Files |
|----------|-------|
| Layout | `MainLayout.vue`, `Sidebar.vue` |
| Global | `index.html`, `main.css`, `tailwind.config.js` |
| Files | `FilesView.vue`, `FileGrid.vue`, `FileList.vue`, `ContextMenu.vue`, `FolderContextMenu.vue`, `BulkActionsBar.vue`, `FilePreview.vue`, `VersionHistory.vue`, CreateFolderModal, ShareModal, RenameModal, MoveDialog, ConfirmModal |
| Documents | `DocumentsView.vue`, `DocumentEditorModal.vue`, `EditorToolbar.vue`, `SlashCommandMenu.vue`, `slashCommandsRender.ts` |
| Other file views | `SharedView.vue`, `RecentView.vue`, `StarredView.vue`, `TrashView.vue` |
| Email | `EmailView.vue`, `EmailCompose.vue`, EmailAccountSetup, EmailLabelManager, EmailRuleManager |
| Admin | `AdminDashboard.vue` |
| Modules | `TasksView.vue`, `CalendarView.vue`, `ContactsView.vue` |
| Auth/Settings | `LoginView.vue`, `RegisterView.vue`, `ResetPasswordView.vue`, `PublicShareView.vue`, `SettingsView.vue` |
| Shared | `CommandPalette.vue`, `PDFViewer.vue`, SaveToFolderPicker, others as needed |
