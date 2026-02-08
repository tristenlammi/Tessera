import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'
import { wsClient, type WebSocketEvent } from '@/composables/useWebSocket'

export interface FileItem {
  id: string
  parent_id: string | null
  owner_id: string
  name: string
  is_folder: boolean
  size: number
  mime_type: string
  is_starred: boolean
  is_trashed: boolean
  trashed_at: string | null
  created_at: string
  updated_at: string
}

export interface BreadcrumbItem {
  id: string | null
  name: string
}

export interface StorageStats {
  used: number
  limit: number
  by_type: Record<string, number>
  used_pct: number
}

export const useFilesStore = defineStore('files', () => {
  const files = ref<FileItem[]>([])
  const currentFolder = ref<string | null>(null)
  const breadcrumbs = ref<BreadcrumbItem[]>([{ id: null, name: 'My Files' }])
  const loading = ref(false)
  const selectedFiles = ref<Set<string>>(new Set())
  const storageStats = ref<StorageStats | null>(null)

  const folders = computed(() => files.value.filter(f => f.is_folder))
  const regularFiles = computed(() => files.value.filter(f => !f.is_folder))

  // Set up WebSocket event listeners
  function initWebSocketListeners() {
    // File created - add to list if in current folder
    wsClient.on('file:created', (event: WebSocketEvent) => {
      const file = event.payload as FileItem
      const eventFolderId = event.folder_id ?? null
      
      // Only add if we're viewing the folder where the file was created
      if (eventFolderId === currentFolder.value) {
        // Check if file already exists (avoid duplicates)
        if (!files.value.find(f => f.id === file.id)) {
          files.value.push(file)
        }
      }
    })

    // File updated - update in list
    wsClient.on('file:updated', (event: WebSocketEvent) => {
      const file = event.payload as FileItem
      const index = files.value.findIndex(f => f.id === file.id)
      if (index !== -1) {
        files.value[index] = file
      }
    })

    // File deleted - remove from list
    wsClient.on('file:deleted', (event: WebSocketEvent) => {
      const { id } = event.payload as { id: string }
      files.value = files.value.filter(f => f.id !== id)
    })

    // File moved - remove from current view if moved away
    wsClient.on('file:moved', (event: WebSocketEvent) => {
      const file = event.payload as FileItem
      const eventFolderId = event.folder_id ?? null
      
      // If moved to a different folder, remove from view
      if (eventFolderId !== currentFolder.value) {
        files.value = files.value.filter(f => f.id !== file.id)
      } else {
        // If moved into current folder, add it
        if (!files.value.find(f => f.id === file.id)) {
          files.value.push(file)
        }
      }
    })

    // File restored - add to list if in current folder
    wsClient.on('file:restored', (event: WebSocketEvent) => {
      const file = event.payload as FileItem
      const eventFolderId = event.folder_id ?? null
      
      if (eventFolderId === currentFolder.value) {
        if (!files.value.find(f => f.id === file.id)) {
          files.value.push(file)
        }
      }
    })

    // Storage updated - refresh stats
    wsClient.on('storage:updated', () => {
      fetchStorageStats()
    })
  }

  // Initialize listeners on store creation
  initWebSocketListeners()

  async function fetchFiles(parentId: string | null = null) {
    loading.value = true
    try {
      // Unsubscribe from old folder, subscribe to new one
      if (currentFolder.value !== parentId) {
        wsClient.unsubscribe(currentFolder.value)
        wsClient.subscribe(parentId)
      }

      const params = parentId ? { parent_id: parentId } : {}
      const response = await api.get('/files', { params })
      files.value = response.data.files || []
      currentFolder.value = parentId
      
      // Update breadcrumbs from response if available
      if (response.data.breadcrumbs) {
        breadcrumbs.value = [{ id: null, name: 'My Files' }, ...response.data.breadcrumbs]
      }
    } finally {
      loading.value = false
    }
  }

  // Navigate into a folder
  function navigateToFolder(folder: FileItem) {
    breadcrumbs.value.push({ id: folder.id, name: folder.name })
    fetchFiles(folder.id)
  }

  // Navigate to a breadcrumb
  function navigateToBreadcrumb(index: number) {
    const crumb = breadcrumbs.value[index]
    breadcrumbs.value = breadcrumbs.value.slice(0, index + 1)
    fetchFiles(crumb.id)
  }

  async function createFolder(name: string, parentId: string | null = null) {
    const response = await api.post('/files/folder', {
      name,
      parent_id: parentId
    })
    files.value.push(response.data)
    return response.data
  }

  async function renameFile(id: string, name: string) {
    const response = await api.put(`/files/${id}`, { name })
    const index = files.value.findIndex(f => f.id === id)
    if (index !== -1) {
      files.value[index] = response.data
    }
    return response.data
  }

  async function moveFile(id: string, parentId: string | null) {
    const response = await api.put(`/files/${id}`, { parent_id: parentId })
    // Remove from current view if moved to different folder
    if (parentId !== currentFolder.value) {
      files.value = files.value.filter(f => f.id !== id)
    }
    return response.data
  }

  async function deleteFile(id: string, permanent = false) {
    await api.delete(`/files/${id}`, { params: { permanent } })
    files.value = files.value.filter(f => f.id !== id)
  }

  async function toggleStar(id: string) {
    const file = files.value.find(f => f.id === id)
    if (!file) return
    
    const response = await api.put(`/files/${id}`, { is_starred: !file.is_starred })
    const index = files.value.findIndex(f => f.id === id)
    if (index !== -1) {
      files.value[index] = response.data
    }
  }

  async function restoreFile(id: string) {
    const response = await api.post(`/files/${id}/restore`)
    files.value = files.value.filter(f => f.id !== id)
    return response.data
  }

  async function fetchTrash() {
    loading.value = true
    try {
      const response = await api.get('/trash')
      files.value = response.data.files
    } finally {
      loading.value = false
    }
  }

  async function fetchStarred() {
    loading.value = true
    try {
      const response = await api.get('/starred')
      files.value = response.data.files
    } finally {
      loading.value = false
    }
  }

  async function emptyTrash() {
    await api.delete('/trash')
    files.value = []
  }

  async function fetchStorageStats() {
    const response = await api.get('/storage')
    storageStats.value = response.data
  }

  async function uploadFile(file: File, parentId: string | null = null) {
    const formData = new FormData()
    formData.append('file', file)
    if (parentId) {
      formData.append('parent_id', parentId)
    }
    
    const response = await api.post('/files/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    })
    return response.data as FileItem
  }

  async function search(query: string) {
    loading.value = true
    try {
      const response = await api.get('/search', { params: { q: query } })
      return response.data.files as FileItem[]
    } finally {
      loading.value = false
    }
  }

  function selectFile(id: string, multi = false) {
    if (multi) {
      if (selectedFiles.value.has(id)) {
        selectedFiles.value.delete(id)
      } else {
        selectedFiles.value.add(id)
      }
    } else {
      selectedFiles.value.clear()
      selectedFiles.value.add(id)
    }
  }

  function clearSelection() {
    selectedFiles.value.clear()
  }

  return {
    files,
    folders,
    regularFiles,
    currentFolder,
    breadcrumbs,
    loading,
    selectedFiles,
    storageStats,
    fetchFiles,
    navigateToFolder,
    navigateToBreadcrumb,
    createFolder,
    renameFile,
    moveFile,
    deleteFile,
    toggleStar,
    restoreFile,
    fetchTrash,
    fetchStarred,
    emptyTrash,
    fetchStorageStats,
    uploadFile,
    search,
    selectFile,
    clearSelection
  }
})
