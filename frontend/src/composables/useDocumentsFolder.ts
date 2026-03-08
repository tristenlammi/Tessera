import { ref, readonly } from 'vue'
import api from '@/api'

export interface DocumentsFolderFile {
  id: string
  parent_id: string | null
  owner_id: string
  name: string
  is_folder: boolean
  size: number
  mime_type: string
  is_starred: boolean
  is_trashed: boolean
  created_at: string
  updated_at: string
}

const documentsFolderId = ref<string | null>(null)
const documentsFolderLoading = ref(false)
const documentsFolderError = ref<string | null>(null)

async function ensureDocumentsFolder(): Promise<string | null> {
  if (documentsFolderId.value) return documentsFolderId.value
  documentsFolderLoading.value = true
  documentsFolderError.value = null
  try {
    const response = await api.get('/files/documents-folder')
    documentsFolderId.value = response.data.id
    return documentsFolderId.value
  } catch (err: any) {
    documentsFolderError.value = err.response?.data?.error ?? 'Failed to get Documents folder'
    return null
  } finally {
    documentsFolderLoading.value = false
  }
}

function isDocumentsFolder(id: string): boolean {
  return !!documentsFolderId.value && id === documentsFolderId.value
}

function resetDocumentsFolder() {
  documentsFolderId.value = null
}

export function useDocumentsFolder() {
  return {
    documentsFolderId: readonly(documentsFolderId),
    documentsFolderLoading: readonly(documentsFolderLoading),
    documentsFolderError: readonly(documentsFolderError),
    ensureDocumentsFolder,
    isDocumentsFolder,
    resetDocumentsFolder,
  }
}
