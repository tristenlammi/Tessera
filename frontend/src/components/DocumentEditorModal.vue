<template>
  <div 
    v-if="isOpen" 
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
  >
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-5xl h-[85vh] flex flex-col mx-4">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b dark:border-gray-700">
        <div class="flex items-center gap-3">
          <input
            v-model="documentTitle"
            @input="handleTitleChange"
            class="text-lg font-semibold bg-transparent border-none focus:outline-none focus:ring-2 focus:ring-blue-500 rounded px-2 py-1 dark:text-white"
            placeholder="Untitled Document"
          />
          <span v-if="saving" class="text-sm text-gray-500 dark:text-gray-400">
            Saving...
          </span>
          <span v-else-if="lastSaved" class="text-sm text-gray-500 dark:text-gray-400">
            Saved {{ formatRelativeTime(lastSaved) }}
          </span>
          <span v-else-if="isDirty" class="text-sm text-orange-500">
            Unsaved changes
          </span>
        </div>
        <div class="flex items-center gap-2">
          <button
            @click="saveDocument"
            :disabled="saving || !isDirty"
            class="px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
            </svg>
            Save
          </button>
          <button
            @click="handleClose"
            class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Toolbar -->
      <div v-if="editor" class="flex items-center gap-1 px-4 py-2 border-b dark:border-gray-700 flex-wrap">
        <!-- Text formatting -->
        <button
          @click="toggleBold"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('bold') }]"
          title="Bold"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 4h8a4 4 0 014 4 4 4 0 01-4 4H6z M6 12h9a4 4 0 014 4 4 4 0 01-4 4H6z" />
          </svg>
        </button>
        <button
          @click="toggleItalic"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('italic') }]"
          title="Italic"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 4h4m-2 0v16m-4 0h8" transform="skewX(-10)" />
          </svg>
        </button>
        <button
          @click="toggleUnderline"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('underline') }]"
          title="Underline"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 3v7a6 6 0 0012 0V3 M4 21h16" />
          </svg>
        </button>
        <button
          @click="toggleStrike"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('strike') }]"
          title="Strikethrough"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 12h12 M7 6h10a2 2 0 012 2c0 2-2 4-6 4 M7 18h6c4 0 6-2 6-4" />
          </svg>
        </button>

        <div class="w-px h-6 bg-gray-300 dark:bg-gray-600 mx-1"></div>

        <!-- Headings -->
        <button
          @click="setHeading(1)"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700 text-sm font-bold', { 'bg-gray-200 dark:bg-gray-600': isActive('heading', { level: 1 }) }]"
          title="Heading 1"
        >
          H1
        </button>
        <button
          @click="setHeading(2)"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700 text-sm font-bold', { 'bg-gray-200 dark:bg-gray-600': isActive('heading', { level: 2 }) }]"
          title="Heading 2"
        >
          H2
        </button>
        <button
          @click="setHeading(3)"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700 text-sm font-bold', { 'bg-gray-200 dark:bg-gray-600': isActive('heading', { level: 3 }) }]"
          title="Heading 3"
        >
          H3
        </button>

        <div class="w-px h-6 bg-gray-300 dark:bg-gray-600 mx-1"></div>

        <!-- Lists -->
        <button
          @click="toggleBulletList"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('bulletList') }]"
          title="Bullet List"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h.01M8 6h12M4 12h.01M8 12h12M4 18h.01M8 18h12" />
          </svg>
        </button>
        <button
          @click="toggleOrderedList"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('orderedList') }]"
          title="Numbered List"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h.01M8 6h12M4 12h.01M8 12h12M4 18h.01M8 18h12" />
            <text x="2" y="8" font-size="6" fill="currentColor">1</text>
            <text x="2" y="14" font-size="6" fill="currentColor">2</text>
            <text x="2" y="20" font-size="6" fill="currentColor">3</text>
          </svg>
        </button>

        <div class="w-px h-6 bg-gray-300 dark:bg-gray-600 mx-1"></div>

        <!-- Quote & Code -->
        <button
          @click="toggleBlockquote"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('blockquote') }]"
          title="Quote"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
          </svg>
        </button>
        <button
          @click="toggleCodeBlock"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('codeBlock') }]"
          title="Code Block"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
          </svg>
        </button>

        <div class="w-px h-6 bg-gray-300 dark:bg-gray-600 mx-1"></div>

        <!-- Alignment -->
        <button
          @click="setTextAlign('left')"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('paragraph', { textAlign: 'left' }) }]"
          title="Align Left"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h10M4 18h14" />
          </svg>
        </button>
        <button
          @click="setTextAlign('center')"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('paragraph', { textAlign: 'center' }) }]"
          title="Align Center"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M7 12h10M5 18h14" />
          </svg>
        </button>
        <button
          @click="setTextAlign('right')"
          :class="['p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700', { 'bg-gray-200 dark:bg-gray-600': isActive('paragraph', { textAlign: 'right' }) }]"
          title="Align Right"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M10 12h10M6 18h14" />
          </svg>
        </button>

        <div class="w-px h-6 bg-gray-300 dark:bg-gray-600 mx-1"></div>

        <!-- Undo/Redo -->
        <button
          @click="undo"
          class="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50"
          title="Undo"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
          </svg>
        </button>
        <button
          @click="redo"
          class="p-2 rounded hover:bg-gray-100 dark:hover:bg-gray-700 disabled:opacity-50"
          title="Redo"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 10H11a8 8 0 00-8 8v2M21 10l-6 6m6-6l-6-6" />
          </svg>
        </button>
      </div>

      <!-- Editor Content -->
      <div class="flex-1 overflow-auto p-4">
        <editor-content 
          :editor="editor" 
          class="prose prose-lg dark:prose-invert max-w-none h-full [&_.ProseMirror]:min-h-full [&_.ProseMirror]:outline-none [&_.ProseMirror]:h-full"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Underline from '@tiptap/extension-underline'
import TextAlign from '@tiptap/extension-text-align'
import api from '@/api'

const props = defineProps<{
  isOpen: boolean
  fileId?: string | null
  folderId?: string | null
}>()

const emit = defineEmits<{
  close: []
  saved: [file: any]
}>()

const documentTitle = ref('Untitled Document')
const saving = ref(false)
const lastSaved = ref<Date | null>(null)
const currentFileId = ref<string | null>(null)
const isDirty = ref(false)
let autoSaveTimeout: ReturnType<typeof setTimeout> | null = null

const editor = useEditor({
  extensions: [
    StarterKit.configure({
      heading: {
        levels: [1, 2, 3]
      },
      bulletList: {
        keepMarks: true,
        keepAttributes: false,
      },
      orderedList: {
        keepMarks: true,
        keepAttributes: false,
      },
    }),
    Placeholder.configure({
      placeholder: 'Start writing your document...'
    }),
    Underline,
    TextAlign.configure({
      types: ['heading', 'paragraph']
    })
  ],
  content: '',
  onUpdate: ({ editor: ed }) => {
    if (ed && props.isOpen) {
      isDirty.value = true
      triggerAutoSave()
    }
  }
})

// Toolbar action functions
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

function setHeading(level: 1 | 2 | 3) {
  editor.value?.chain().focus().toggleHeading({ level }).run()
}

function toggleBulletList() {
  console.log('toggleBulletList called, editor:', !!editor.value)
  editor.value?.chain().focus().toggleBulletList().run()
}

function toggleOrderedList() {
  console.log('toggleOrderedList called, editor:', !!editor.value)
  editor.value?.chain().focus().toggleOrderedList().run()
}

function toggleBlockquote() {
  console.log('toggleBlockquote called, editor:', !!editor.value)
  editor.value?.chain().focus().toggleBlockquote().run()
}

function toggleCodeBlock() {
  editor.value?.chain().focus().toggleCodeBlock().run()
}

function setTextAlign(align: 'left' | 'center' | 'right') {
  editor.value?.chain().focus().setTextAlign(align).run()
}

function undo() {
  editor.value?.chain().focus().undo().run()
}

function redo() {
  editor.value?.chain().focus().redo().run()
}

function isActive(name: string, attributes?: Record<string, any>): boolean {
  return editor.value?.isActive(name, attributes) ?? false
}

function triggerAutoSave() {
  // Clear any existing timeout
  if (autoSaveTimeout) {
    clearTimeout(autoSaveTimeout)
    autoSaveTimeout = null
  }
  // Schedule a new save in 2 seconds
  autoSaveTimeout = setTimeout(async () => {
    console.log('Auto-save triggered, isDirty:', isDirty.value, 'saving:', saving.value)
    if (isDirty.value && !saving.value) {
      await saveDocument()
    }
  }, 2000)
}

// Load document when opening existing file
watch(() => props.isOpen, async (isOpen) => {
  if (isOpen && props.fileId) {
    currentFileId.value = props.fileId
    isDirty.value = false
    await loadDocument(props.fileId)
  } else if (isOpen && !props.fileId) {
    // New document
    currentFileId.value = null
    documentTitle.value = 'Untitled Document'
    editor.value?.commands.setContent('')
    lastSaved.value = null
    isDirty.value = false
  }
})

async function loadDocument(fileId: string) {
  try {
    const response = await api.get(`/files/${fileId}/content`)
    const data = response.data
    documentTitle.value = data.title || 'Untitled Document'
    editor.value?.commands.setContent(data.content || '')
  } catch (err) {
    console.error('Failed to load document:', err)
  }
}

async function saveDocument() {
  if (!editor.value) {
    console.log('saveDocument: no editor')
    return
  }
  if (saving.value) {
    console.log('saveDocument: already saving')
    return
  }
  
  saving.value = true
  console.log('saveDocument: starting save, currentFileId:', currentFileId.value)
  
  try {
    const content = editor.value.getHTML()
    const documentData = {
      title: documentTitle.value,
      content: content,
      format: 'html'
    }

    if (currentFileId.value) {
      // Update existing document
      console.log('saveDocument: updating existing file')
      await api.put(`/files/${currentFileId.value}/content`, documentData)
    } else {
      // Create new document file
      console.log('saveDocument: creating new file')
      const response = await api.post('/documents/create-file', {
        ...documentData,
        parentId: props.folderId || null,
        name: documentTitle.value.endsWith('.tdoc') ? documentTitle.value : `${documentTitle.value}.tdoc`
      })
      currentFileId.value = response.data.id
      console.log('saveDocument: created file with id:', currentFileId.value)
      emit('saved', response.data)
    }
    
    lastSaved.value = new Date()
    isDirty.value = false
    console.log('saveDocument: save complete')
  } catch (err: any) {
    console.error('Failed to save document:', err)
    console.error('Error response:', err?.response?.data)
  } finally {
    saving.value = false
  }
}

function handleTitleChange() {
  isDirty.value = true
  triggerAutoSave()
}

function handleClose() {
  // Clear auto-save timeout
  if (autoSaveTimeout) {
    clearTimeout(autoSaveTimeout)
    autoSaveTimeout = null
  }
  // Save before closing if there are unsaved changes
  if (editor.value && isDirty.value) {
    saveDocument()
  }
  emit('close')
}

function formatRelativeTime(date: Date): string {
  const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000)
  
  if (seconds < 5) return 'just now'
  if (seconds < 60) return `${seconds}s ago`
  
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  
  const hours = Math.floor(minutes / 60)
  return `${hours}h ago`
}

onUnmounted(() => {
  if (autoSaveTimeout) {
    clearTimeout(autoSaveTimeout)
  }
  editor.value?.destroy()
})
</script>

<style>
/* Placeholder styling */
.ProseMirror p.is-editor-empty:first-child::before {
  color: #adb5bd;
  content: attr(data-placeholder);
  float: left;
  height: 0;
  pointer-events: none;
}
</style>
