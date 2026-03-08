<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import { createEditorExtensions } from '@/extensions/editorConfig'
import { useEditorPrefsStore } from '@/stores/editorPrefs'
import { useDocumentsStore } from '@/stores/documents'
import { useEditorPasteImage } from '@/composables/useEditorPasteImage'
import EditorToolbar from '@/components/EditorToolbar.vue'

const route = useRoute()
const router = useRouter()
const editorPrefsStore = useEditorPrefsStore()
const documentsStore = useDocumentsStore()

const showNewDocModal = ref(false)
const newDocTitle = ref('')
const showShareModal = ref(false)
const shareEmail = ref('')
const sharePermission = ref<'view' | 'edit'>('view')
const autoSaveInterval = ref<number | null>(null)

const documentId = computed(() => route.params.id as string | undefined)

const editor = useEditor({
  extensions: createEditorExtensions({
    placeholder: "Type '/' for commands...",
    enabledToolbarIds: editorPrefsStore.enabledIds,
  }),
  content: '',
  editorProps: {
    handlePaste: useEditorPasteImage(),
  },
  onUpdate: () => {
    documentsStore.markUnsavedChanges()
  }
})

// Document operations
async function loadDocument(id: string) {
  try {
    const doc = await documentsStore.fetchDocument(id)
    if (editor.value && doc.content) {
      const format = doc.format || 'tiptap'
      if (format === 'markdown') {
        editor.value.commands.setContent(doc.content)
      } else if (format === 'html') {
        editor.value.commands.setContent(doc.content)
      } else {
        editor.value.commands.setContent(JSON.parse(doc.content))
      }
    }
  } catch (err) {
    console.error('Failed to load document:', err)
  }
}

async function saveDocument() {
  if (!documentsStore.currentDocument || !editor.value) return
  
  try {
    const format = documentsStore.currentDocument.format || 'markdown'
    let content: string
    if (format === 'markdown') {
      content = typeof editor.value.getMarkdown === 'function'
        ? editor.value.getMarkdown()
        : editor.value.getHTML()
    } else if (format === 'html') {
      content = editor.value.getHTML()
    } else {
      content = JSON.stringify(editor.value.getJSON())
    }
    await documentsStore.saveDocument(
      documentsStore.currentDocument.id,
      content,
      documentsStore.currentDocument.version
    )
  } catch (err) {
    console.error('Failed to save document:', err)
  }
}

async function createNewDocument() {
  if (!newDocTitle.value.trim()) return
  
  try {
    const doc = await documentsStore.createDocument({
      title: newDocTitle.value,
      content: '',
      format: 'markdown'
    })
    
    showNewDocModal.value = false
    newDocTitle.value = ''
    router.push(`/documents/${doc.id}`)
  } catch (err) {
    console.error('Failed to create document:', err)
  }
}

async function shareDocument() {
  if (!documentsStore.currentDocument || !shareEmail.value.trim()) return
  
  try {
    await documentsStore.shareDocument(
      documentsStore.currentDocument.id,
      shareEmail.value,
      sharePermission.value
    )
    shareEmail.value = ''
    showShareModal.value = false
  } catch (err) {
    console.error('Failed to share document:', err)
  }
}

function goBack() {
  router.push('/documents')
}

// Auto-save every 30 seconds
function startAutoSave() {
  autoSaveInterval.value = window.setInterval(() => {
    if (documentsStore.unsavedChanges && documentsStore.currentDocument) {
      saveDocument()
    }
  }, 30000)
}

function stopAutoSave() {
  if (autoSaveInterval.value) {
    clearInterval(autoSaveInterval.value)
    autoSaveInterval.value = null
  }
}

// Keyboard shortcuts
function handleKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 's') {
    e.preventDefault()
    saveDocument()
  }
}

watch(documentId, async (newId) => {
  if (newId) {
    await loadDocument(newId)
  } else {
    documentsStore.clearCurrentDocument()
    editor.value?.commands.clearContent()
  }
}, { immediate: true })

onMounted(async () => {
  await documentsStore.fetchDocuments()
  startAutoSave()
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  stopAutoSave()
  window.removeEventListener('keydown', handleKeydown)
  documentsStore.clearCurrentDocument()
})
</script>

<template>
  <div class="h-full flex flex-col bg-stone-50 dark:bg-neutral-900">
    <!-- Document List View -->
    <div v-if="!documentId" class="flex-1 p-6">
      <div class="max-w-4xl mx-auto">
        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
          <h1 class="text-2xl font-bold text-stone-900 dark:text-stone-100">Documents</h1>
          <button
            @click="showNewDocModal = true"
            class="min-h-[44px] flex items-center justify-center gap-2 px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 self-start sm:self-center"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            New Document
          </button>
        </div>

        <!-- Recent Documents -->
        <div v-if="documentsStore.recentDocuments.length > 0">
          <h2 class="text-lg font-semibold text-stone-700 dark:text-stone-300 mb-4">Recent Documents</h2>
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <div
              v-for="doc in documentsStore.recentDocuments"
              :key="doc.id"
              @click="router.push(`/documents/${doc.id}`)"
              class="bg-white dark:bg-neutral-800 rounded-lg border border-stone-200 dark:border-neutral-700 p-4 min-h-[72px] cursor-pointer hover:shadow-md transition-shadow active:bg-stone-50 dark:active:bg-neutral-700/50"
            >
              <div class="flex items-start gap-3">
                <svg class="w-8 h-8 text-stone-700 dark:text-stone-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <div class="flex-1 min-w-0">
                  <h3 class="font-medium text-stone-900 dark:text-stone-100 truncate">{{ doc.title }}</h3>
                  <p class="text-sm text-stone-500 dark:text-stone-400">
                    {{ new Date(doc.updatedAt).toLocaleDateString() }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else class="text-center py-12">
          <svg class="w-16 h-16 mx-auto text-stone-300 dark:text-stone-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <h3 class="text-lg font-medium text-stone-900 dark:text-stone-100 mb-2">No documents yet</h3>
          <p class="text-stone-500 dark:text-stone-400 mb-4">Create your first document to get started</p>
          <button
            @click="showNewDocModal = true"
            class="min-h-[44px] px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300"
          >
            Create Document
          </button>
        </div>
      </div>
    </div>

    <!-- Editor View -->
    <template v-else>
      <!-- Toolbar -->
      <div class="bg-white dark:bg-neutral-800 border-b border-stone-200 dark:border-neutral-700 px-4 py-2 pt-[max(0.5rem,env(safe-area-inset-top))]">
        <div class="flex flex-col md:flex-row md:items-center gap-2 md:flex-wrap">
          <div class="flex items-center gap-2 min-w-0">
          <!-- Back button -->
          <button
            @click="goBack"
            class="min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center p-2 rounded hover:bg-stone-100 dark:hover:bg-neutral-700"
            title="Back to documents"
          >
            <svg class="w-5 h-5 text-stone-600 dark:text-stone-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
          </button>

          <div class="h-6 w-px bg-stone-300 dark:bg-neutral-600 mx-1"></div>

          <!-- Document title -->
          <span class="font-medium text-stone-900 dark:text-stone-100 truncate max-w-[200px]">
            {{ documentsStore.currentDocument?.title || 'Untitled' }}
          </span>

          <!-- Save indicator -->
          <span v-if="documentsStore.unsavedChanges" class="text-xs text-yellow-600 dark:text-yellow-400">
            (Unsaved)
          </span>
          <span v-else-if="documentsStore.saving" class="text-xs text-stone-800 dark:text-stone-200 dark:text-stone-400">
            Saving...
          </span>
          <span v-else class="text-xs text-green-600 dark:text-green-400">
            Saved
          </span>
          </div>

          <div class="flex-1"></div>

          <div class="flex items-center gap-2">
          <!-- Share button -->
          <button
            @click="showShareModal = true"
            class="min-h-[44px] flex items-center justify-center px-3 py-1.5 text-sm bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z" />
            </svg>
            Share
          </button>

          <!-- Save button -->
          <button
            @click="saveDocument"
            :disabled="documentsStore.saving"
            class="min-h-[44px] flex items-center justify-center px-3 py-1.5 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
            </svg>
            Save
          </button>
          </div>
        </div>

        <!-- Editor Toolbar -->
        <div class="mt-2 pt-2 border-t border-stone-100 dark:border-neutral-700">
          <EditorToolbar :editor="editor" />
        </div>
      </div>

      <!-- Editor Content -->
      <div class="flex-1 overflow-auto min-h-0">
        <div class="max-w-4xl mx-auto py-6 md:py-8 px-4 pb-[env(safe-area-inset-bottom)]">
          <EditorContent 
            :editor="editor" 
            class="prose prose-lg dark:prose-invert max-w-none min-h-[500px] bg-white dark:bg-neutral-800 rounded-lg shadow p-8"
          />
        </div>
      </div>

      <!-- Collaborators -->
      <div v-if="documentsStore.collaborators.length > 0" class="fixed bottom-4 right-4 flex -space-x-2">
        <div
          v-for="collab in documentsStore.collaborators.filter(c => c.online)"
          :key="collab.userId"
          :title="collab.userName"
          :style="{ backgroundColor: collab.color }"
          class="w-8 h-8 rounded-full border-2 border-white dark:border-neutral-800 flex items-center justify-center text-white text-sm font-medium"
        >
          {{ collab.userName.charAt(0).toUpperCase() }}
        </div>
      </div>
    </template>

    <!-- New Document Modal -->
    <div v-if="showNewDocModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)]">
      <div class="modal-panel bg-white dark:bg-neutral-800 shadow-xl w-full">
        <div class="p-6">
          <h2 class="text-xl font-bold text-stone-900 dark:text-stone-100 mb-4">New Document</h2>
          <input
            v-model="newDocTitle"
            type="text"
            placeholder="Document title"
            class="w-full px-4 py-2 border rounded-lg dark:bg-neutral-700 dark:border-neutral-700 dark:text-stone-100 mb-4 min-h-[44px]"
            @keydown.enter="createNewDocument"
          />
          <div class="flex justify-end gap-2">
          <button
            @click="showNewDocModal = false"
            class="px-4 py-2 text-stone-700 dark:text-stone-300 hover:bg-stone-100 dark:hover:bg-neutral-700 rounded-lg"
          >
            Cancel
          </button>
          <button
            @click="createNewDocument"
            class="min-h-[44px] px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300"
          >
            Create
          </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Share Modal -->
    <div v-if="showShareModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)]">
      <div class="modal-panel bg-white dark:bg-neutral-800 shadow-xl w-full max-h-[90dvh] overflow-y-auto p-6">
        <h2 class="text-xl font-bold text-stone-900 dark:text-stone-100 mb-4">Share Document</h2>
        
        <div class="mb-4">
          <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">Email address</label>
          <input
            v-model="shareEmail"
            type="email"
            placeholder="user@example.com"
            class="w-full px-4 py-2 border rounded-lg dark:bg-neutral-700 dark:border-neutral-700 dark:text-stone-100"
          />
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">Permission</label>
          <select
            v-model="sharePermission"
            class="w-full px-4 py-2 border rounded-lg dark:bg-neutral-700 dark:border-neutral-700 dark:text-stone-100"
          >
            <option value="view">Can view</option>
            <option value="edit">Can edit</option>
          </select>
        </div>

        <!-- Current collaborators -->
        <div v-if="documentsStore.currentDocument?.collaborators?.length" class="mb-4">
          <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">Current collaborators</label>
          <div class="space-y-2">
            <div
              v-for="collab in documentsStore.currentDocument.collaborators"
              :key="collab.userId"
              class="flex items-center justify-between p-2 bg-stone-50 dark:bg-neutral-700 rounded"
            >
              <div class="flex items-center gap-2">
                <div
                  :style="{ backgroundColor: collab.color }"
                  class="w-6 h-6 rounded-full flex items-center justify-center text-white text-xs"
                >
                  {{ collab.userName.charAt(0).toUpperCase() }}
                </div>
                <span class="text-sm text-stone-900 dark:text-stone-100">{{ collab.userEmail }}</span>
                <span class="text-xs text-stone-500">({{ collab.permission }})</span>
              </div>
              <button
                @click="documentsStore.removeCollaborator(documentsStore.currentDocument!.id, collab.userId)"
                class="text-red-500 hover:text-red-700"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-2">
          <button
            @click="showShareModal = false"
            class="px-4 py-2 text-stone-700 dark:text-stone-300 hover:bg-stone-100 dark:hover:bg-neutral-700 rounded-lg"
          >
            Close
          </button>
          <button
            @click="shareDocument"
            class="px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300"
          >
            Share
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style>
/* Tiptap editor styles */
.ProseMirror {
  outline: none;
  min-height: 400px;
}

.ProseMirror p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  float: left;
  color: #adb5bd;
  pointer-events: none;
  height: 0;
}

.ProseMirror table {
  border-collapse: collapse;
  table-layout: fixed;
  width: 100%;
  margin: 1em 0;
}

.ProseMirror th,
.ProseMirror td {
  border: 1px solid #ddd;
  padding: 8px 12px;
  vertical-align: top;
}

.ProseMirror th {
  background-color: #f5f5f5;
  font-weight: bold;
}

.dark .ProseMirror th {
  background-color: #374151;
}

.dark .ProseMirror th,
.dark .ProseMirror td {
  border-color: #4b5563;
}

.ProseMirror blockquote {
  border-left: 3px solid #ddd;
  padding-left: 1rem;
  margin-left: 0;
}

.ProseMirror pre {
  background-color: #1e1e1e;
  color: #d4d4d4;
  padding: 1rem;
  border-radius: 0.5rem;
  overflow-x: auto;
}

.ProseMirror mark {
  background-color: #fef08a;
}

.dark .ProseMirror mark {
  background-color: #ca8a04;
}
</style>
