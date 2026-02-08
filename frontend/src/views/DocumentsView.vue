<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Highlight from '@tiptap/extension-highlight'
import Underline from '@tiptap/extension-underline'
import TextAlign from '@tiptap/extension-text-align'
import Image from '@tiptap/extension-image'
import Link from '@tiptap/extension-link'
import { Table, TableRow, TableCell, TableHeader } from '@tiptap/extension-table'
import { TextStyle } from '@tiptap/extension-text-style'
import Color from '@tiptap/extension-color'
import { useDocumentsStore } from '@/stores/documents'
import { useFilesStore } from '@/stores/files'

const route = useRoute()
const router = useRouter()
const documentsStore = useDocumentsStore()
const filesStore = useFilesStore()

const showNewDocModal = ref(false)
const newDocTitle = ref('')
const showShareModal = ref(false)
const shareEmail = ref('')
const sharePermission = ref<'view' | 'edit'>('view')
const autoSaveInterval = ref<number | null>(null)

const documentId = computed(() => route.params.id as string | undefined)

const editor = useEditor({
  extensions: [
    StarterKit.configure({
      heading: {
        levels: [1, 2, 3]
      }
    }),
    Placeholder.configure({
      placeholder: 'Start writing...'
    }),
    Highlight.configure({
      multicolor: true
    }),
    Underline,
    TextAlign.configure({
      types: ['heading', 'paragraph']
    }),
    Image.configure({
      inline: true,
      allowBase64: true
    }),
    Link.configure({
      openOnClick: false
    }),
    Table.configure({
      resizable: true
    }),
    TableRow,
    TableCell,
    TableHeader,
    TextStyle,
    Color
  ],
  content: '',
  onUpdate: () => {
    documentsStore.markUnsavedChanges()
  }
})

// Toolbar button states
const isActive = (type: string, attrs?: Record<string, any>) => {
  return editor.value?.isActive(type, attrs) ?? false
}

// Formatting actions
function toggleBold() {
  editor.value?.chain().focus().toggleBold().run()
}

function toggleItalic() {
  editor.value?.chain().focus().toggleItalic().run()
}

function toggleUnderline() {
  editor.value?.chain().focus().toggleUnderline().run()
}

function toggleStrike() {
  editor.value?.chain().focus().toggleStrike().run()
}

function toggleHighlight() {
  editor.value?.chain().focus().toggleHighlight().run()
}

function setHeading(level: 1 | 2 | 3) {
  editor.value?.chain().focus().toggleHeading({ level }).run()
}

function toggleBulletList() {
  editor.value?.chain().focus().toggleBulletList().run()
}

function toggleOrderedList() {
  editor.value?.chain().focus().toggleOrderedList().run()
}

function toggleBlockquote() {
  editor.value?.chain().focus().toggleBlockquote().run()
}

function toggleCodeBlock() {
  editor.value?.chain().focus().toggleCodeBlock().run()
}

function setTextAlign(align: 'left' | 'center' | 'right' | 'justify') {
  editor.value?.chain().focus().setTextAlign(align).run()
}

function insertTable() {
  editor.value?.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
}

function addLink() {
  const url = window.prompt('Enter URL:')
  if (url) {
    editor.value?.chain().focus().setLink({ href: url }).run()
  }
}

function removeLink() {
  editor.value?.chain().focus().unsetLink().run()
}

function insertImage() {
  const url = window.prompt('Enter image URL:')
  if (url) {
    editor.value?.chain().focus().setImage({ src: url }).run()
  }
}

function undo() {
  editor.value?.chain().focus().undo().run()
}

function redo() {
  editor.value?.chain().focus().redo().run()
}

// Document operations
async function loadDocument(id: string) {
  try {
    const doc = await documentsStore.fetchDocument(id)
    if (editor.value && doc.content) {
      editor.value.commands.setContent(JSON.parse(doc.content))
    }
  } catch (err) {
    console.error('Failed to load document:', err)
  }
}

async function saveDocument() {
  if (!documentsStore.currentDocument || !editor.value) return
  
  try {
    const content = JSON.stringify(editor.value.getJSON())
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
      content: JSON.stringify({ type: 'doc', content: [{ type: 'paragraph' }] }),
      format: 'tiptap'
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
  <div class="h-full flex flex-col bg-gray-50 dark:bg-gray-900">
    <!-- Document List View -->
    <div v-if="!documentId" class="flex-1 p-6">
      <div class="max-w-4xl mx-auto">
        <div class="flex items-center justify-between mb-6">
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Documents</h1>
          <button
            @click="showNewDocModal = true"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            New Document
          </button>
        </div>

        <!-- Recent Documents -->
        <div v-if="documentsStore.recentDocuments.length > 0">
          <h2 class="text-lg font-semibold text-gray-700 dark:text-gray-300 mb-4">Recent Documents</h2>
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <div
              v-for="doc in documentsStore.recentDocuments"
              :key="doc.id"
              @click="router.push(`/documents/${doc.id}`)"
              class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 cursor-pointer hover:shadow-md transition-shadow"
            >
              <div class="flex items-start gap-3">
                <svg class="w-8 h-8 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <div class="flex-1 min-w-0">
                  <h3 class="font-medium text-gray-900 dark:text-white truncate">{{ doc.title }}</h3>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ new Date(doc.updatedAt).toLocaleDateString() }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else class="text-center py-12">
          <svg class="w-16 h-16 mx-auto text-gray-300 dark:text-gray-600 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-2">No documents yet</h3>
          <p class="text-gray-500 dark:text-gray-400 mb-4">Create your first document to get started</p>
          <button
            @click="showNewDocModal = true"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Create Document
          </button>
        </div>
      </div>
    </div>

    <!-- Editor View -->
    <template v-else>
      <!-- Toolbar -->
      <div class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-4 py-2">
        <div class="flex items-center gap-2 flex-wrap">
          <!-- Back button -->
          <button
            @click="goBack"
            class="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
            title="Back to documents"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
          </button>

          <div class="h-6 w-px bg-gray-300 dark:bg-gray-600 mx-1"></div>

          <!-- Document title -->
          <span class="font-medium text-gray-900 dark:text-white truncate max-w-[200px]">
            {{ documentsStore.currentDocument?.title || 'Untitled' }}
          </span>

          <!-- Save indicator -->
          <span v-if="documentsStore.unsavedChanges" class="text-xs text-yellow-600 dark:text-yellow-400">
            (Unsaved)
          </span>
          <span v-else-if="documentsStore.saving" class="text-xs text-blue-600 dark:text-blue-400">
            Saving...
          </span>
          <span v-else class="text-xs text-green-600 dark:text-green-400">
            Saved
          </span>

          <div class="flex-1"></div>

          <!-- Undo/Redo -->
          <button @click="undo" class="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700" title="Undo (Ctrl+Z)">
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
            </svg>
          </button>
          <button @click="redo" class="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700" title="Redo (Ctrl+Y)">
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 10h-10a8 8 0 00-8 8v2M21 10l-6 6m6-6l-6-6" />
            </svg>
          </button>

          <div class="h-6 w-px bg-gray-300 dark:bg-gray-600 mx-1"></div>

          <!-- Text formatting -->
          <button 
            @click="toggleBold" 
            :class="['p-2 rounded', isActive('bold') ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Bold (Ctrl+B)"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M13.5 15.5H10V12.5H13.5A1.5 1.5 0 0115 14A1.5 1.5 0 0113.5 15.5M10 6.5H13A1.5 1.5 0 0114.5 8A1.5 1.5 0 0113 9.5H10M15.6 10.79C16.57 10.11 17.25 9 17.25 8C17.25 5.74 15.5 4 13.25 4H7V18H14.04C16.14 18 17.75 16.3 17.75 14.21C17.75 12.69 16.89 11.39 15.6 10.79Z" />
            </svg>
          </button>
          <button 
            @click="toggleItalic" 
            :class="['p-2 rounded', isActive('italic') ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Italic (Ctrl+I)"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M10,4V7H12.21L8.79,15H6V18H14V15H11.79L15.21,7H18V4H10Z" />
            </svg>
          </button>
          <button 
            @click="toggleUnderline" 
            :class="['p-2 rounded', isActive('underline') ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Underline (Ctrl+U)"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M5,21H19V19H5V21M12,17A6,6 0 0,0 18,11V3H15.5V11A3.5,3.5 0 0,1 12,14.5A3.5,3.5 0 0,1 8.5,11V3H6V11A6,6 0 0,0 12,17Z" />
            </svg>
          </button>
          <button 
            @click="toggleStrike" 
            :class="['p-2 rounded', isActive('strike') ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Strikethrough"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M3,14H21V12H3M5,4V7H10V10H14V7H19V4M10,19H14V16H10V19Z" />
            </svg>
          </button>
          <button 
            @click="toggleHighlight" 
            :class="['p-2 rounded', isActive('highlight') ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Highlight"
          >
            <svg class="w-5 h-5 text-yellow-500" viewBox="0 0 24 24" fill="currentColor">
              <path d="M15.243 3.343l5.414 5.414-1.414 1.414-5.414-5.414 1.414-1.414zm-1.414 1.414L4.1 14.486l-.707 6.364 6.364-.707 9.728-9.728-5.657-5.657zM5.686 18.313l-.465-4.187 4.652 4.652-4.187-.465z" />
            </svg>
          </button>

          <div class="h-6 w-px bg-gray-300 dark:bg-gray-600 mx-1"></div>

          <!-- Headings -->
          <button 
            @click="setHeading(1)" 
            :class="['p-2 rounded text-sm font-bold', isActive('heading', { level: 1 }) ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Heading 1"
          >
            H1
          </button>
          <button 
            @click="setHeading(2)" 
            :class="['p-2 rounded text-sm font-bold', isActive('heading', { level: 2 }) ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Heading 2"
          >
            H2
          </button>
          <button 
            @click="setHeading(3)" 
            :class="['p-2 rounded text-sm font-bold', isActive('heading', { level: 3 }) ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Heading 3"
          >
            H3
          </button>

          <div class="h-6 w-px bg-gray-300 dark:bg-gray-600 mx-1"></div>

          <!-- Lists -->
          <button 
            @click="toggleBulletList" 
            :class="['p-2 rounded', isActive('bulletList') ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Bullet List"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M7,5H21V7H7V5M7,13V11H21V13H7M4,4.5A1.5,1.5 0 0,1 5.5,6A1.5,1.5 0 0,1 4,7.5A1.5,1.5 0 0,1 2.5,6A1.5,1.5 0 0,1 4,4.5M4,10.5A1.5,1.5 0 0,1 5.5,12A1.5,1.5 0 0,1 4,13.5A1.5,1.5 0 0,1 2.5,12A1.5,1.5 0 0,1 4,10.5M7,19V17H21V19H7M4,16.5A1.5,1.5 0 0,1 5.5,18A1.5,1.5 0 0,1 4,19.5A1.5,1.5 0 0,1 2.5,18A1.5,1.5 0 0,1 4,16.5Z" />
            </svg>
          </button>
          <button 
            @click="toggleOrderedList" 
            :class="['p-2 rounded', isActive('orderedList') ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Numbered List"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M7,13V11H21V13H7M7,19V17H21V19H7M7,7V5H21V7H7M3,8V5H2V4H4V8H3M2,17V16H5V20H2V19H4V18.5H3V17.5H4V17H2M4.25,10A0.75,0.75 0 0,1 5,10.75C5,10.95 4.92,11.14 4.79,11.27L3.12,13H5V14H2V13.08L4,11H2V10H4.25Z" />
            </svg>
          </button>

          <div class="h-6 w-px bg-gray-300 dark:bg-gray-600 mx-1"></div>

          <!-- Alignment -->
          <button 
            @click="setTextAlign('left')" 
            :class="['p-2 rounded', isActive({ textAlign: 'left' }) ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Align Left"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M3,3H21V5H3V3M3,7H15V9H3V7M3,11H21V13H3V11M3,15H15V17H3V15M3,19H21V21H3V19Z" />
            </svg>
          </button>
          <button 
            @click="setTextAlign('center')" 
            :class="['p-2 rounded', isActive({ textAlign: 'center' }) ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Align Center"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M3,3H21V5H3V3M7,7H17V9H7V7M3,11H21V13H3V11M7,15H17V17H7V15M3,19H21V21H3V19Z" />
            </svg>
          </button>
          <button 
            @click="setTextAlign('right')" 
            :class="['p-2 rounded', isActive({ textAlign: 'right' }) ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Align Right"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M3,3H21V5H3V3M9,7H21V9H9V7M3,11H21V13H3V11M9,15H21V17H9V15M3,19H21V21H3V19Z" />
            </svg>
          </button>

          <div class="h-6 w-px bg-gray-300 dark:bg-gray-600 mx-1"></div>

          <!-- Quote, Code, Table -->
          <button 
            @click="toggleBlockquote" 
            :class="['p-2 rounded', isActive('blockquote') ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Quote"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M14,17H17L19,13V7H13V13H16M6,17H9L11,13V7H5V13H8L6,17Z" />
            </svg>
          </button>
          <button 
            @click="toggleCodeBlock" 
            :class="['p-2 rounded', isActive('codeBlock') ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Code Block"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M14.6,16.6L19.2,12L14.6,7.4L16,6L22,12L16,18L14.6,16.6M9.4,16.6L4.8,12L9.4,7.4L8,6L2,12L8,18L9.4,16.6Z" />
            </svg>
          </button>
          <button 
            @click="insertTable" 
            class="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
            title="Insert Table"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M5,4H19A2,2 0 0,1 21,6V18A2,2 0 0,1 19,20H5A2,2 0 0,1 3,18V6A2,2 0 0,1 5,4M5,8V12H11V8H5M13,8V12H19V8H13M5,14V18H11V14H5M13,14V18H19V14H13Z" />
            </svg>
          </button>

          <div class="h-6 w-px bg-gray-300 dark:bg-gray-600 mx-1"></div>

          <!-- Link, Image -->
          <button 
            @click="addLink" 
            :class="['p-2 rounded', isActive('link') ? 'bg-gray-200 dark:bg-gray-600' : 'hover:bg-gray-100 dark:hover:bg-gray-700']"
            title="Add Link"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M3.9,12C3.9,10.29 5.29,8.9 7,8.9H11V7H7A5,5 0 0,0 2,12A5,5 0 0,0 7,17H11V15.1H7C5.29,15.1 3.9,13.71 3.9,12M8,13H16V11H8V13M17,7H13V8.9H17C18.71,8.9 20.1,10.29 20.1,12C20.1,13.71 18.71,15.1 17,15.1H13V17H17A5,5 0 0,0 22,12A5,5 0 0,0 17,7Z" />
            </svg>
          </button>
          <button 
            @click="insertImage" 
            class="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
            title="Insert Image"
          >
            <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" viewBox="0 0 24 24" fill="currentColor">
              <path d="M21,3H3C2,3 1,4 1,5V19A2,2 0 0,0 3,21H21C22,21 23,20 23,19V5C23,4 22,3 21,3M5,17L8.5,12.5L11,15.5L14.5,11L19,17H5Z" />
            </svg>
          </button>

          <div class="flex-1"></div>

          <!-- Share button -->
          <button
            @click="showShareModal = true"
            class="px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
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
            class="px-3 py-1.5 text-sm bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 flex items-center gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
            </svg>
            Save
          </button>
        </div>
      </div>

      <!-- Editor Content -->
      <div class="flex-1 overflow-auto">
        <div class="max-w-4xl mx-auto py-8 px-4">
          <EditorContent 
            :editor="editor" 
            class="prose prose-lg dark:prose-invert max-w-none min-h-[500px] bg-white dark:bg-gray-800 rounded-lg shadow p-8"
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
          class="w-8 h-8 rounded-full border-2 border-white dark:border-gray-800 flex items-center justify-center text-white text-sm font-medium"
        >
          {{ collab.userName.charAt(0).toUpperCase() }}
        </div>
      </div>
    </template>

    <!-- New Document Modal -->
    <div v-if="showNewDocModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6 w-full max-w-md">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">New Document</h2>
        <input
          v-model="newDocTitle"
          type="text"
          placeholder="Document title"
          class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white mb-4"
          @keydown.enter="createNewDocument"
        />
        <div class="flex justify-end gap-2">
          <button
            @click="showNewDocModal = false"
            class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
          >
            Cancel
          </button>
          <button
            @click="createNewDocument"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            Create
          </button>
        </div>
      </div>
    </div>

    <!-- Share Modal -->
    <div v-if="showShareModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6 w-full max-w-md">
        <h2 class="text-xl font-bold text-gray-900 dark:text-white mb-4">Share Document</h2>
        
        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Email address</label>
          <input
            v-model="shareEmail"
            type="email"
            placeholder="user@example.com"
            class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
          />
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Permission</label>
          <select
            v-model="sharePermission"
            class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
          >
            <option value="view">Can view</option>
            <option value="edit">Can edit</option>
          </select>
        </div>

        <!-- Current collaborators -->
        <div v-if="documentsStore.currentDocument?.collaborators?.length" class="mb-4">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Current collaborators</label>
          <div class="space-y-2">
            <div
              v-for="collab in documentsStore.currentDocument.collaborators"
              :key="collab.userId"
              class="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-700 rounded"
            >
              <div class="flex items-center gap-2">
                <div
                  :style="{ backgroundColor: collab.color }"
                  class="w-6 h-6 rounded-full flex items-center justify-center text-white text-xs"
                >
                  {{ collab.userName.charAt(0).toUpperCase() }}
                </div>
                <span class="text-sm text-gray-900 dark:text-white">{{ collab.userEmail }}</span>
                <span class="text-xs text-gray-500">({{ collab.permission }})</span>
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
            class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
          >
            Close
          </button>
          <button
            @click="shareDocument"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
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
