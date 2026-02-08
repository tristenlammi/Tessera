import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export interface Document {
  id: string
  fileId: string
  title: string
  content: string  // JSON or HTML content
  format: 'tiptap' | 'markdown' | 'html'
  createdAt: string
  updatedAt: string
  ownerId: string
  ownerName: string
  collaborators: DocumentCollaborator[]
  isPublic: boolean
  version: number
}

export interface DocumentCollaborator {
  userId: string
  userName: string
  userEmail: string
  permission: 'view' | 'edit'
  color: string
  cursor?: { from: number; to: number }
  online: boolean
}

export const useDocumentsStore = defineStore('documents', () => {
  const documents = ref<Document[]>([])
  const currentDocument = ref<Document | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const error = ref<string | null>(null)
  const collaborators = ref<DocumentCollaborator[]>([])
  const unsavedChanges = ref(false)

  const recentDocuments = computed(() => 
    [...documents.value]
      .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
      .slice(0, 10)
  )

  async function fetchDocuments() {
    loading.value = true
    error.value = null

    try {
      const response = await api.get('/documents')
      documents.value = response.data.documents || []
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to fetch documents'
    } finally {
      loading.value = false
    }
  }

  async function fetchDocument(id: string) {
    loading.value = true
    error.value = null

    try {
      const response = await api.get(`/documents/${id}`)
      currentDocument.value = response.data
      collaborators.value = response.data.collaborators || []
      return response.data
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to fetch document'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createDocument(doc: Partial<Document>) {
    loading.value = true
    error.value = null

    try {
      const response = await api.post('/documents', doc)
      documents.value.push(response.data)
      return response.data
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to create document'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function saveDocument(id: string, content: string, version?: number) {
    saving.value = true
    error.value = null

    try {
      const response = await api.put(`/documents/${id}`, {
        content,
        version
      })
      
      if (currentDocument.value?.id === id) {
        currentDocument.value = response.data
      }
      
      const index = documents.value.findIndex(d => d.id === id)
      if (index !== -1) {
        documents.value[index] = response.data
      }
      
      unsavedChanges.value = false
      return response.data
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to save document'
      throw err
    } finally {
      saving.value = false
    }
  }

  async function deleteDocument(id: string) {
    loading.value = true
    error.value = null

    try {
      await api.delete(`/documents/${id}`)
      documents.value = documents.value.filter(d => d.id !== id)
      if (currentDocument.value?.id === id) {
        currentDocument.value = null
      }
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to delete document'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function shareDocument(id: string, userEmail: string, permission: 'view' | 'edit') {
    try {
      const response = await api.post(`/documents/${id}/share`, {
        email: userEmail,
        permission
      })
      
      if (currentDocument.value?.id === id) {
        currentDocument.value.collaborators = response.data.collaborators
      }
      
      return response.data
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to share document'
      throw err
    }
  }

  async function removeCollaborator(docId: string, userId: string) {
    try {
      await api.delete(`/documents/${docId}/share/${userId}`)
      
      if (currentDocument.value?.id === docId) {
        currentDocument.value.collaborators = 
          currentDocument.value.collaborators.filter(c => c.userId !== userId)
      }
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to remove collaborator'
      throw err
    }
  }

  function updateCollaboratorCursor(userId: string, cursor: { from: number; to: number }) {
    const collab = collaborators.value.find(c => c.userId === userId)
    if (collab) {
      collab.cursor = cursor
    }
  }

  function setCollaboratorOnline(userId: string, online: boolean) {
    const collab = collaborators.value.find(c => c.userId === userId)
    if (collab) {
      collab.online = online
    }
  }

  function markUnsavedChanges() {
    unsavedChanges.value = true
  }

  function clearCurrentDocument() {
    currentDocument.value = null
    collaborators.value = []
    unsavedChanges.value = false
  }

  return {
    documents,
    currentDocument,
    loading,
    saving,
    error,
    collaborators,
    unsavedChanges,
    recentDocuments,
    fetchDocuments,
    fetchDocument,
    createDocument,
    saveDocument,
    deleteDocument,
    shareDocument,
    removeCollaborator,
    updateCollaboratorCursor,
    setCollaboratorOnline,
    markUnsavedChanges,
    clearCurrentDocument
  }
})
