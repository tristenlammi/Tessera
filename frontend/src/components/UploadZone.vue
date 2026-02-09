<script setup lang="ts">
import { ref, computed } from 'vue'
import { useFilesStore } from '@/stores/files'
import api from '@/api'

const props = defineProps<{
  folderId: string | null
}>()

const emit = defineEmits<{
  (e: 'upload'): void
  (e: 'uploadFolder'): void
}>()

const filesStore = useFilesStore()

const isDragging = ref(false)
const dragCounter = ref(0)

// --- Per-upload progress tracking ---
interface UploadEntry {
  id: string
  fileName: string
  progress: number       // 0-100
  status: string         // 'uploading', 'creating-folders', 'done', 'error'
  totalFiles: number
  completedFiles: number
}

const activeUploads = ref<Map<string, UploadEntry>>(new Map())
const hasActiveUploads = computed(() => activeUploads.value.size > 0)

let uploadIdCounter = 0
function createUploadId(): string {
  return `upload-${++uploadIdCounter}-${Date.now()}`
}

function setUploadEntry(entry: UploadEntry) {
  // Trigger reactivity by creating a new Map
  const newMap = new Map(activeUploads.value)
  newMap.set(entry.id, entry)
  activeUploads.value = newMap
}

function removeUploadEntry(id: string) {
  const newMap = new Map(activeUploads.value)
  newMap.delete(id)
  activeUploads.value = newMap
}

// Interface for file with path info
interface FileWithPath {
  file: File
  relativePath: string
}

// Check if drag contains external files (not internal file moves)
function hasExternalFiles(e: DragEvent): boolean {
  if (!e.dataTransfer) return false
  // If it has our internal JSON data, it's a file move
  if (e.dataTransfer.types.includes('application/json')) return false
  // Check for files
  return e.dataTransfer.types.includes('Files')
}

function handleDragEnter(e: DragEvent) {
  e.preventDefault()
  if (!hasExternalFiles(e)) return
  dragCounter.value++
  isDragging.value = true
}

function handleDragOver(e: DragEvent) {
  e.preventDefault()
  if (!hasExternalFiles(e)) return
  isDragging.value = true
}

function handleDragLeave(e: DragEvent) {
  if (!hasExternalFiles(e)) return
  dragCounter.value--
  if (dragCounter.value <= 0) {
    dragCounter.value = 0
    isDragging.value = false
  }
}

async function handleDrop(e: DragEvent) {
  e.preventDefault()
  dragCounter.value = 0
  isDragging.value = false

  // Only handle external file drops
  if (!hasExternalFiles(e)) return

  const items = e.dataTransfer?.items
  if (items && items.length > 0) {
    // Check if any item is a directory using webkitGetAsEntry
    const entries: FileSystemEntry[] = []
    for (let i = 0; i < items.length; i++) {
      const entry = items[i].webkitGetAsEntry?.()
      if (entry) {
        entries.push(entry)
      }
    }

    if (entries.length > 0) {
      // Process entries (files and folders)
      const filesWithPaths = await processEntries(entries)
      if (filesWithPaths.length > 0) {
        await uploadFilesWithPaths(filesWithPaths)
      }
      return
    }
  }

  // Fallback to regular file handling
  const files = e.dataTransfer?.files
  if (files && files.length > 0) {
    await uploadFiles(files)
  }
}

// Process FileSystemEntry items (supports folders)
async function processEntries(entries: FileSystemEntry[]): Promise<FileWithPath[]> {
  const filesWithPaths: FileWithPath[] = []

  for (const entry of entries) {
    await processEntry(entry, '', filesWithPaths)
  }

  return filesWithPaths
}

async function processEntry(entry: FileSystemEntry, basePath: string, result: FileWithPath[]): Promise<void> {
  if (entry.isFile) {
    const fileEntry = entry as FileSystemFileEntry
    const file = await getFileFromEntry(fileEntry)
    if (file) {
      result.push({
        file,
        relativePath: basePath ? `${basePath}/${entry.name}` : entry.name
      })
    }
  } else if (entry.isDirectory) {
    const dirEntry = entry as FileSystemDirectoryEntry
    const dirPath = basePath ? `${basePath}/${entry.name}` : entry.name
    const reader = dirEntry.createReader()
    const entries = await readAllDirectoryEntries(reader)
    
    for (const childEntry of entries) {
      await processEntry(childEntry, dirPath, result)
    }
  }
}

function getFileFromEntry(fileEntry: FileSystemFileEntry): Promise<File | null> {
  return new Promise((resolve) => {
    fileEntry.file(
      (file) => resolve(file),
      () => resolve(null)
    )
  })
}

function readAllDirectoryEntries(reader: FileSystemDirectoryReader): Promise<FileSystemEntry[]> {
  return new Promise((resolve) => {
    const entries: FileSystemEntry[] = []
    
    function readEntries() {
      reader.readEntries(
        (results) => {
          if (results.length === 0) {
            resolve(entries)
          } else {
            entries.push(...results)
            readEntries() // Continue reading (directory might have many entries)
          }
        },
        () => resolve(entries)
      )
    }
    
    readEntries()
  })
}

async function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    // Check if files have webkitRelativePath (folder upload)
    const hasRelativePaths = Array.from(input.files).some(f => (f as any).webkitRelativePath)
    
    if (hasRelativePaths) {
      const filesWithPaths: FileWithPath[] = Array.from(input.files).map(file => ({
        file,
        relativePath: (file as any).webkitRelativePath || file.name
      }))
      await uploadFilesWithPaths(filesWithPaths)
    } else {
      await uploadFiles(input.files)
    }
    input.value = ''
  }
}

// Upload files with folder structure preservation
async function uploadFilesWithPaths(filesWithPaths: FileWithPath[]) {
  const uid = createUploadId()
  const totalFiles = filesWithPaths.length
  const label = totalFiles === 1
    ? filesWithPaths[0].file.name
    : `${totalFiles} files (folder upload)`

  setUploadEntry({
    id: uid,
    fileName: label,
    progress: 0,
    status: 'creating-folders',
    totalFiles,
    completedFiles: 0
  })

  // Extract unique folder paths and sort by depth
  const folderPaths = new Set<string>()
  for (const { relativePath } of filesWithPaths) {
    const parts = relativePath.split('/')
    // Add all parent folders
    for (let i = 1; i < parts.length; i++) {
      folderPaths.add(parts.slice(0, i).join('/'))
    }
  }
  
  // Sort folders by depth (create parent folders first)
  const sortedFolders = Array.from(folderPaths).sort((a, b) => {
    const depthA = a.split('/').length
    const depthB = b.split('/').length
    return depthA - depthB
  })

  // Map to store created folder IDs
  const folderIdMap = new Map<string, string>()
  
  // Create folders
  for (const folderPath of sortedFolders) {
    const parts = folderPath.split('/')
    const folderName = parts[parts.length - 1]
    const parentPath = parts.slice(0, -1).join('/')
    
    // Determine parent ID
    let parentId: string | null = props.folderId
    if (parentPath) {
      parentId = folderIdMap.get(parentPath) || props.folderId
    }
    
    try {
      const response = await api.post('/files/folder', {
        name: folderName,
        parent_id: parentId
      })
      folderIdMap.set(folderPath, response.data.id)
    } catch (err) {
      console.error(`Failed to create folder ${folderPath}:`, err)
    }
  }

  // Upload files
  setUploadEntry({
    id: uid,
    fileName: label,
    progress: 0,
    status: 'uploading',
    totalFiles,
    completedFiles: 0
  })

  for (let i = 0; i < totalFiles; i++) {
    const { file, relativePath } = filesWithPaths[i]
    
    // Get parent folder ID
    const parts = relativePath.split('/')
    const parentPath = parts.slice(0, -1).join('/')
    let parentId: string | null = props.folderId
    if (parentPath) {
      parentId = folderIdMap.get(parentPath) || props.folderId
    }
    
    const formData = new FormData()
    formData.append('file', file)
    if (parentId) {
      formData.append('parent_id', parentId)
    }

    try {
      await api.post('/files/upload', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        },
        onUploadProgress: (progressEvent) => {
          if (progressEvent.total) {
            const progress = Math.round(
              ((i + progressEvent.loaded / progressEvent.total) / totalFiles) * 100
            )
            setUploadEntry({
              id: uid,
              fileName: label,
              progress,
              status: 'uploading',
              totalFiles,
              completedFiles: i
            })
          }
        }
      })
    } catch (err) {
      console.error(`Upload failed for ${relativePath}:`, err)
    }
  }

  removeUploadEntry(uid)
  filesStore.fetchFiles(props.folderId)
  filesStore.fetchStorageStats()
}

async function uploadFiles(files: FileList) {
  const uid = createUploadId()
  const totalFiles = files.length
  const label = totalFiles === 1
    ? files[0].name
    : `${totalFiles} files`

  setUploadEntry({
    id: uid,
    fileName: label,
    progress: 0,
    status: 'uploading',
    totalFiles,
    completedFiles: 0
  })

  for (let i = 0; i < totalFiles; i++) {
    const file = files[i]
    const formData = new FormData()
    formData.append('file', file)
    if (props.folderId) {
      formData.append('parent_id', props.folderId)
    }

    try {
      await api.post('/files/upload', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        },
        onUploadProgress: (progressEvent) => {
          if (progressEvent.total) {
            const progress = Math.round(
              ((i + progressEvent.loaded / progressEvent.total) / totalFiles) * 100
            )
            setUploadEntry({
              id: uid,
              fileName: label,
              progress,
              status: 'uploading',
              totalFiles,
              completedFiles: i
            })
          }
        }
      })
    } catch (err) {
      console.error('Upload failed:', err)
    }
  }

  removeUploadEntry(uid)
  filesStore.fetchFiles(props.folderId)
  filesStore.fetchStorageStats()
}
</script>

<template>
  <div
    @dragenter="handleDragEnter"
    @dragover="handleDragOver"
    @dragleave="handleDragLeave"
    @drop="handleDrop"
    :class="[
      'relative h-full',
      isDragging && 'bg-blue-50 dark:bg-blue-900/20'
    ]"
  >
    <!-- Drop overlay -->
    <div
      v-if="isDragging"
      class="absolute inset-0 flex items-center justify-center bg-blue-50 dark:bg-blue-900/30 bg-opacity-90 z-10 border-2 border-dashed border-blue-400 rounded-lg m-4 pointer-events-none"
    >
      <div class="text-center">
        <svg class="w-12 h-12 mx-auto text-blue-500 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
        </svg>
        <p class="text-lg font-medium text-blue-700">Drop files or folders here to upload</p>
      </div>
    </div>

    <!-- Upload progress panels -->
    <div
      v-if="hasActiveUploads"
      class="absolute bottom-4 right-4 z-20 flex flex-col gap-2 w-72"
    >
      <div
        v-for="[id, entry] in activeUploads"
        :key="id"
        class="bg-white dark:bg-gray-800 rounded-lg shadow-lg border dark:border-gray-700 p-4"
      >
        <div class="flex items-center justify-between mb-1">
          <span class="text-sm font-medium truncate mr-2" :title="entry.fileName">{{ entry.fileName }}</span>
          <span class="text-sm text-gray-500 dark:text-gray-400 shrink-0">{{ entry.progress }}%</span>
        </div>
        <div class="text-xs text-gray-400 dark:text-gray-500 mb-2">
          <template v-if="entry.status === 'creating-folders'">Creating folders…</template>
          <template v-else>{{ entry.completedFiles }} / {{ entry.totalFiles }} files</template>
        </div>
        <div class="h-2 bg-gray-200 dark:bg-gray-600 rounded-full overflow-hidden">
          <div
            class="h-full bg-blue-600 transition-all"
            :style="{ width: `${entry.progress}%` }"
          ></div>
        </div>
      </div>
    </div>

    <!-- Hidden file input for files -->
    <input
      type="file"
      multiple
      class="hidden"
      id="file-upload"
      @change="handleFileSelect"
    />

    <!-- Hidden file input for folders -->
    <input
      type="file"
      webkitdirectory
      directory
      multiple
      class="hidden"
      id="folder-upload"
      @change="handleFileSelect"
    />

    <slot />
  </div>
</template>
