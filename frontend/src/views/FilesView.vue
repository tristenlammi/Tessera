<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useFilesStore, type FileItem } from '@/stores/files'
import { useModulesStore } from '@/stores/modules'
import FileGrid from '@/components/FileGrid.vue'
import FileList from '@/components/FileList.vue'
import UploadZone from '@/components/UploadZone.vue'
import CreateFolderModal from '@/components/CreateFolderModal.vue'
import FilePreview from '@/components/FilePreview.vue'
import ContextMenu from '@/components/ContextMenu.vue'
import ShareModal from '@/components/ShareModal.vue'
import RenameModal from '@/components/RenameModal.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import CommandPalette from '@/components/CommandPalette.vue'
import FileInfoPanel from '@/components/FileInfoPanel.vue'
import MoveDialog from '@/components/MoveDialog.vue'
import BulkActionsBar from '@/components/BulkActionsBar.vue'
import VersionHistory from '@/components/VersionHistory.vue'
import DocumentEditorModal from '@/components/DocumentEditorModal.vue'
import FolderContextMenu from '@/components/FolderContextMenu.vue'

const route = useRoute()
const router = useRouter()
const filesStore = useFilesStore()
const modulesStore = useModulesStore()

const viewMode = ref<'grid' | 'list'>('grid')
const showCreateFolder = ref(false)
const showCommandPalette = ref(false)
const showInfoPanel = ref(false)
const infoPanelFile = ref<FileItem | null>(null)
const showUploadMenu = ref(false)

// Icon size state - persisted to localStorage
const ICON_SIZE_KEY = 'tessera-files-icon-size'
const iconSize = ref<number>(parseInt(localStorage.getItem(ICON_SIZE_KEY) || '48', 10))

// Save icon size to localStorage when it changes
watch(iconSize, (newSize) => {
  localStorage.setItem(ICON_SIZE_KEY, String(newSize))
})

// Preview state
const previewFile = ref<FileItem | null>(null)

// Document editor state
const showDocumentEditor = ref(false)
const editingDocumentId = ref<string | null>(null)

// Context menu state
const contextMenu = ref<{ file: FileItem; x: number; y: number } | null>(null)
const folderContextMenu = ref<{ x: number; y: number } | null>(null)

// Modal states
const shareFile = ref<FileItem | null>(null)
const renameFile = ref<FileItem | null>(null)
const deleteConfirm = ref<FileItem | null>(null)
const moveDialog = ref<{ fileIds: string[]; mode: 'move' | 'copy' } | null>(null)
const bulkDeleteConfirm = ref(false)
const versionHistoryFile = ref<FileItem | null>(null)

const currentFolderId = computed(() => {
  return route.params.id as string || null
})

onMounted(() => {
  // Reset breadcrumbs when navigating via URL
  if (!currentFolderId.value) {
    filesStore.breadcrumbs = [{ id: null, name: 'My Files' }]
  }
  filesStore.fetchFiles(currentFolderId.value)
  filesStore.fetchStorageStats()
  
  // Global keyboard shortcut for command palette
  document.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleGlobalKeydown)
})

function handleGlobalKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
    e.preventDefault()
    showCommandPalette.value = true
  }
}

watch(currentFolderId, (newId) => {
  filesStore.fetchFiles(newId)
})

function openFile(file: FileItem) {
  if (file.is_folder) {
    filesStore.navigateToFolder(file)
    router.push({ name: 'folder', params: { id: file.id } })
  } else if (file.name.endsWith('.tdoc')) {
    // Open document in editor modal
    editingDocumentId.value = file.id
    showDocumentEditor.value = true
  } else {
    previewFile.value = file
  }
}

function createNewDocument() {
  editingDocumentId.value = null
  showDocumentEditor.value = true
}

function handleDocumentSaved(_savedFile: any) {
  filesStore.fetchFiles(currentFolderId.value)
}

function handleDocumentEditorClose() {
  showDocumentEditor.value = false
  editingDocumentId.value = null
}

function handleFolderCreated() {
  showCreateFolder.value = false
}

// Breadcrumb navigation
function navigateToBreadcrumb(index: number) {
  const crumb = filesStore.breadcrumbs[index]
  filesStore.navigateToBreadcrumb(index)
  if (crumb.id) {
    router.push({ name: 'folder', params: { id: crumb.id } })
  } else {
    router.push({ name: 'files' })
  }
}

// Context menu handlers
function showContextMenu(file: FileItem, event: MouseEvent) {
  event.preventDefault()
  event.stopPropagation()
  console.log('Context menu triggered for:', file.name, 'at', event.clientX, event.clientY)
  contextMenu.value = { file, x: event.clientX, y: event.clientY }
}

function handleContextOpen() {
  if (contextMenu.value) openFile(contextMenu.value.file)
}

function handleContextDownload() {
  if (contextMenu.value && !contextMenu.value.file.is_folder) {
    window.open(`/api/files/${contextMenu.value.file.id}/download`, '_blank')
  }
}

function handleContextShare() {
  if (contextMenu.value) shareFile.value = contextMenu.value.file
}

function handleContextStar() {
  if (contextMenu.value) filesStore.toggleStar(contextMenu.value.file.id)
}

function handleContextRename() {
  if (contextMenu.value) renameFile.value = contextMenu.value.file
}

function handleContextCopy() {
  if (contextMenu.value) {
    moveDialog.value = { fileIds: [contextMenu.value.file.id], mode: 'copy' }
  }
}

function handleContextMove() {
  if (contextMenu.value) {
    moveDialog.value = { fileIds: [contextMenu.value.file.id], mode: 'move' }
  }
}

function handleContextInfo() {
  if (contextMenu.value) {
    infoPanelFile.value = contextMenu.value.file
    showInfoPanel.value = true
  }
}

function handleContextVersions() {
  if (contextMenu.value && !contextMenu.value.file.is_folder) {
    versionHistoryFile.value = contextMenu.value.file
  }
}

function handleContextDelete() {
  if (contextMenu.value) {
    deleteConfirm.value = contextMenu.value.file
  }
}

async function confirmDelete() {
  if (deleteConfirm.value) {
    await filesStore.deleteFile(deleteConfirm.value.id)
    deleteConfirm.value = null
  }
}

// Bulk operations
function handleBulkMove() {
  moveDialog.value = { fileIds: Array.from(filesStore.selectedFiles), mode: 'move' }
}

function handleBulkCopy() {
  moveDialog.value = { fileIds: Array.from(filesStore.selectedFiles), mode: 'copy' }
}

function handleBulkDelete() {
  bulkDeleteConfirm.value = true
}

async function confirmBulkDelete() {
  const ids = Array.from(filesStore.selectedFiles)
  for (const id of ids) {
    await filesStore.deleteFile(id)
  }
  filesStore.clearSelection()
  bulkDeleteConfirm.value = false
}

function handleBulkDownload() {
  // For now, download files individually
  // TODO: Implement zip download on backend
  filesStore.selectedFiles.forEach(id => {
    window.open(`/api/files/${id}/download`, '_blank')
  })
}

function handleMoveComplete() {
  filesStore.clearSelection()
  filesStore.fetchFiles(currentFolderId.value)
}

async function handleDrop(fileIds: string[], targetFolderId: string) {
  for (const fileId of fileIds) {
    await filesStore.moveFile(fileId, targetFolderId)
  }
  filesStore.clearSelection()
  filesStore.fetchFiles(currentFolderId.value)
}

// Folder context menu (right-click on empty space)
function showFolderContextMenu(event: MouseEvent) {
  event.preventDefault()
  // Only show if we right-clicked on empty space, not on a file
  const target = event.target as HTMLElement
  if (target.closest('[data-file-item]')) return
  folderContextMenu.value = { x: event.clientX, y: event.clientY }
}

function handleFolderContextNewFolder() {
  showCreateFolder.value = true
}

function handleFolderContextNewDocument() {
  createNewDocument()
}

function triggerFileUpload() {
  const input = document.getElementById('file-upload') as HTMLInputElement
  if (input) input.click()
  showUploadMenu.value = false
}

function triggerFolderUpload() {
  const input = document.getElementById('folder-upload') as HTMLInputElement
  if (input) input.click()
  showUploadMenu.value = false
}
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between p-4 border-b dark:border-gray-700">
      <div class="flex items-center gap-2">
        <nav class="flex items-center gap-1 text-sm">
          <template v-for="(crumb, index) in filesStore.breadcrumbs" :key="crumb.id ?? 'root'">
            <span v-if="index > 0" class="text-gray-400">/</span>
            <button
              @click="navigateToBreadcrumb(index)"
              class="hover:text-blue-600 px-1"
              :class="{ 'font-medium': index === filesStore.breadcrumbs.length - 1 }"
            >
              {{ crumb.name }}
            </button>
          </template>
        </nav>
      </div>

      <div class="flex items-center gap-2">
        <!-- Search Button -->
        <button
          @click="showCommandPalette = true"
          class="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <span class="hidden sm:inline">Search</span>
          <kbd class="hidden sm:inline px-1.5 py-0.5 text-xs bg-white dark:bg-gray-600 dark:text-gray-300 rounded border dark:border-gray-500">⌘K</kbd>
        </button>

        <!-- Upload Dropdown -->
        <div class="relative">
          <button
            @click="showUploadMenu = !showUploadMenu"
            class="flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
            </svg>
            Upload
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
            </svg>
          </button>
          
          <!-- Upload dropdown menu -->
          <div
            v-if="showUploadMenu"
            class="absolute right-0 mt-1 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border dark:border-gray-700 py-1 z-50"
            @click.stop
          >
            <button
              @click="triggerFileUpload"
              class="w-full px-4 py-2 text-sm text-left text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
              Upload Files
            </button>
            <button
              @click="triggerFolderUpload"
              class="w-full px-4 py-2 text-sm text-left text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
              </svg>
              Upload Folder
            </button>
          </div>
        </div>
        
        <!-- Click outside to close upload menu -->
        <div
          v-if="showUploadMenu"
          class="fixed inset-0 z-40"
          @click="showUploadMenu = false"
        ></div>
        <button
          @click="showCreateFolder = true"
          class="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700"
        >
          New Folder
        </button>
        <button
          v-if="modulesStore.isModuleEnabled('documents')"
          @click="createNewDocument"
          class="px-3 py-1.5 text-sm border border-gray-300 rounded-lg hover:bg-gray-50 dark:border-gray-600 dark:hover:bg-gray-700 flex items-center gap-1"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          New Document
        </button>
        
        <!-- Icon Size Slider (only shown in grid view) -->
        <div v-if="viewMode === 'grid'" class="flex items-center gap-2 px-2">
          <svg class="w-3 h-3 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
            <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5z" />
          </svg>
          <input
            v-model="iconSize"
            type="range"
            min="32"
            max="96"
            step="8"
            class="w-20 h-1 bg-gray-200 dark:bg-gray-600 rounded-lg appearance-none cursor-pointer accent-blue-600"
            title="Icon size"
          />
          <svg class="w-5 h-5 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
            <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5z" />
          </svg>
        </div>
        
        <div class="flex border dark:border-gray-600 rounded-lg overflow-hidden">
          <button
            @click="viewMode = 'grid'"
            :class="['px-3 py-1.5', viewMode === 'grid' ? 'bg-gray-100 dark:bg-gray-700' : 'hover:bg-gray-50 dark:hover:bg-gray-700']"
          >
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path d="M5 3a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2V5a2 2 0 00-2-2H5zM5 11a2 2 0 00-2 2v2a2 2 0 002 2h2a2 2 0 002-2v-2a2 2 0 00-2-2H5zM11 5a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V5zM11 13a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
            </svg>
          </button>
          <button
            @click="viewMode = 'list'"
            :class="['px-3 py-1.5', viewMode === 'list' ? 'bg-gray-100 dark:bg-gray-700' : 'hover:bg-gray-50 dark:hover:bg-gray-700']"
          >
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M3 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1zm0 4a1 1 0 011-1h12a1 1 0 110 2H4a1 1 0 01-1-1z" clip-rule="evenodd" />
            </svg>
          </button>
        </div>
      </div>
    </div>

    <!-- Upload Zone & Files -->
    <UploadZone :folder-id="currentFolderId" class="flex-1 overflow-auto" @contextmenu.prevent="showFolderContextMenu">
      <div v-if="filesStore.loading" class="flex items-center justify-center h-64">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>

      <div v-else-if="filesStore.files.length === 0" class="flex flex-col items-center justify-center h-64 text-gray-500 dark:text-gray-400">
        <svg class="w-16 h-16 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <p class="text-lg font-medium">No files yet</p>
        <p class="text-sm mb-4">Drop files or folders here, or click to upload</p>
        <div class="flex gap-2">
          <label for="file-upload" class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 cursor-pointer">
            Upload Files
          </label>
          <label for="folder-upload" class="px-4 py-2 border border-blue-600 text-blue-600 rounded-lg hover:bg-blue-50 cursor-pointer">
            Upload Folder
          </label>
        </div>
      </div>

      <FileGrid
        v-else-if="viewMode === 'grid'"
        :files="filesStore.files"
        :icon-size="iconSize"
        @open="openFile"
        @contextmenu="showContextMenu"
        @drop="handleDrop"
      />

      <FileList
        v-else
        :files="filesStore.files"
        @open="openFile"
        @contextmenu="showContextMenu"
      />
    </UploadZone>

    <!-- Create Folder Modal -->
    <CreateFolderModal
      v-if="showCreateFolder"
      :parent-id="currentFolderId"
      @close="showCreateFolder = false"
      @created="handleFolderCreated"
    />

    <!-- File Preview Modal -->
    <FilePreview
      v-if="previewFile"
      :file="previewFile"
      @close="previewFile = null"
    />

    <!-- Context Menu -->
    <ContextMenu
      v-if="contextMenu"
      :file="contextMenu.file"
      :x="contextMenu.x"
      :y="contextMenu.y"
      @close="contextMenu = null"
      @open="handleContextOpen"
      @download="handleContextDownload"
      @share="handleContextShare"
      @star="handleContextStar"
      @rename="handleContextRename"
      @copy="handleContextCopy"
      @move="handleContextMove"
      @info="handleContextInfo"
      @versions="handleContextVersions"
      @delete="handleContextDelete"
    />

    <!-- Folder Context Menu (right-click on empty space) -->
    <FolderContextMenu
      v-if="folderContextMenu"
      :x="folderContextMenu.x"
      :y="folderContextMenu.y"
      @close="folderContextMenu = null"
      @newFolder="handleFolderContextNewFolder"
      @newDocument="handleFolderContextNewDocument"
      @upload="triggerFileUpload"
      @uploadFolder="triggerFolderUpload"
    />

    <!-- Share Modal -->
    <ShareModal
      v-if="shareFile"
      :file-id="shareFile.id"
      :file-name="shareFile.name"
      @close="shareFile = null"
    />

    <!-- Rename Modal -->
    <RenameModal
      v-if="renameFile"
      :file-id="renameFile.id"
      :current-name="renameFile.name"
      @close="renameFile = null"
      @renamed="renameFile = null"
    />

    <!-- Delete Confirmation Modal -->
    <ConfirmModal
      v-if="deleteConfirm"
      title="Move to Trash"
      :message="`Are you sure you want to move &quot;${deleteConfirm.name}&quot; to trash?`"
      confirm-text="Move to Trash"
      cancel-text="Cancel"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="deleteConfirm = null"
    />

    <!-- Bulk Delete Confirmation Modal -->
    <ConfirmModal
      v-if="bulkDeleteConfirm"
      title="Move to Trash"
      :message="`Are you sure you want to move ${filesStore.selectedFiles.size} item(s) to trash?`"
      confirm-text="Move to Trash"
      cancel-text="Cancel"
      :danger="true"
      @confirm="confirmBulkDelete"
      @cancel="bulkDeleteConfirm = false"
    />

    <!-- Command Palette -->
    <CommandPalette
      v-if="showCommandPalette"
      @close="showCommandPalette = false"
      @openFile="(file) => { previewFile = file }"
    />

    <!-- File Info Panel -->
    <Teleport to="body">
      <div v-if="showInfoPanel && infoPanelFile" class="fixed top-0 right-0 h-full z-40">
        <FileInfoPanel
          :file="infoPanelFile"
          @close="showInfoPanel = false; infoPanelFile = null"
        />
      </div>
    </Teleport>

    <!-- Move/Copy Dialog -->
    <MoveDialog
      v-if="moveDialog"
      :file-ids="moveDialog.fileIds"
      :mode="moveDialog.mode"
      @close="moveDialog = null"
      @complete="handleMoveComplete"
    />

    <!-- Bulk Actions Bar -->
    <BulkActionsBar
      @move="handleBulkMove"
      @copy="handleBulkCopy"
      @delete="handleBulkDelete"
      @download="handleBulkDownload"
    />

    <!-- Version History Panel -->
    <VersionHistory
      v-if="versionHistoryFile"
      :file-id="versionHistoryFile.id"
      :file-name="versionHistoryFile.name"
      @close="versionHistoryFile = null"
    />

    <!-- Document Editor Modal -->
    <DocumentEditorModal
      :is-open="showDocumentEditor"
      :file-id="editingDocumentId"
      :folder-id="currentFolderId"
      @close="handleDocumentEditorClose"
      @saved="handleDocumentSaved"
    />
  </div>
</template>
