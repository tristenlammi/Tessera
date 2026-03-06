<template>
  <div 
    v-if="isOpen" 
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-2 md:p-4 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)]"
  >
    <div class="bg-white dark:bg-neutral-800 rounded-lg shadow-xl w-full max-w-7xl h-full md:h-[85vh] flex flex-col md:mx-4 md:max-h-[90dvh]">
      <!-- Header -->
      <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-3 px-4 py-3 border-b dark:border-neutral-700 flex-shrink-0">
        <div class="flex items-center gap-3 min-w-0 flex-1">
          <input
            v-model="documentTitle"
            @input="handleTitleChange"
            class="text-lg font-semibold bg-transparent border-none focus:outline-none focus:ring-2 focus:ring-stone-400 rounded px-2 py-1 dark:text-stone-100"
            placeholder="Untitled Document"
          />
          <span v-if="saving" class="text-sm text-stone-500 dark:text-stone-400">
            Saving...
          </span>
          <span v-else-if="lastSaved" class="text-sm text-stone-500 dark:text-stone-400">
            Saved {{ formatRelativeTime(lastSaved) }}
          </span>
          <span v-else-if="isDirty" class="text-sm text-orange-500">
            Unsaved changes
          </span>
          <span v-if="saveError" class="text-sm text-red-500">
            {{ saveError }}
          </span>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <button
            @click="saveDocument"
            :disabled="saving || !isDirty"
            class="min-h-[44px] flex items-center px-3 py-1.5 text-sm bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 disabled:opacity-50 disabled:cursor-not-allowed gap-1"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
            </svg>
            Save
          </button>
          <button
            @click="handleClose"
            class="min-w-[44px] min-h-[44px] flex items-center justify-center p-2 hover:bg-stone-100 dark:hover:bg-neutral-700 rounded-lg transition-colors"
          >
            <svg class="w-5 h-5 text-stone-500 dark:text-stone-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Toolbar -->
      <div v-if="editor" class="px-4 py-2 border-b dark:border-neutral-700 flex-shrink-0">
        <EditorToolbar :editor="editor" />
      </div>

      <!-- Editor Content -->
      <div class="flex-1 overflow-auto p-4 min-h-0">
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
import { createEditorExtensions } from '@/extensions/editorConfig'
import { useEditorPrefsStore } from '@/stores/editorPrefs'
import { useEditorPasteImage } from '@/composables/useEditorPasteImage'
import EditorToolbar from './EditorToolbar.vue'
import api from '@/api'

const editorPrefsStore = useEditorPrefsStore()

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
const currentFormat = ref<string>('markdown')
const saveError = ref<string | null>(null)
let autoSaveTimeout: ReturnType<typeof setTimeout> | null = null

const editor = useEditor({
  extensions: createEditorExtensions({
    placeholder: "Type '/' for commands...",
    enabledToolbarIds: editorPrefsStore.enabledIds,
  }),
  content: '',
  editorProps: {
    handlePaste: useEditorPasteImage(),
  },
  onUpdate: ({ editor: ed }) => {
    if (ed && props.isOpen) {
      isDirty.value = true
      triggerAutoSave()
    }
  }
})

function triggerAutoSave() {
  // Clear any existing timeout
  if (autoSaveTimeout) {
    clearTimeout(autoSaveTimeout)
    autoSaveTimeout = null
  }
  autoSaveTimeout = setTimeout(async () => {
    if (isDirty.value && !saving.value) {
      await saveDocument()
    }
  }, 2000)
}

// Load document when opening existing file
watch(() => props.isOpen, async (isOpen) => {
  saveError.value = null
  if (isOpen && props.fileId) {
    currentFileId.value = props.fileId
    isDirty.value = false
    await loadDocument(props.fileId)
  } else if (isOpen && !props.fileId) {
    currentFileId.value = null
    documentTitle.value = 'Untitled Document'
    currentFormat.value = 'markdown'
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
    currentFormat.value = data.format || 'html'
    const content = data.content || ''
    if (currentFormat.value === 'markdown') {
      editor.value?.commands.setContent(content)
    } else if (currentFormat.value === 'html') {
      editor.value?.commands.setContent(content)
    } else {
      try {
        editor.value?.commands.setContent(JSON.parse(content))
      } catch {
        editor.value?.commands.setContent(content)
      }
    }
  } catch (err) {
    console.error('Failed to load document:', err)
  }
}

async function saveDocument(): Promise<boolean> {
  if (!editor.value) return false
  if (saving.value) return false

  saving.value = true
  saveError.value = null

  try {
    let content: string
    if (currentFormat.value === 'markdown') {
      content = (editor.value.storage.markdown as any).getMarkdown()
    } else if (currentFormat.value === 'html') {
      content = editor.value.getHTML()
    } else {
      content = JSON.stringify(editor.value.getJSON())
    }
    const documentData = {
      title: documentTitle.value,
      content: content,
      format: currentFormat.value
    }

    if (currentFileId.value) {
      await api.put(`/files/${currentFileId.value}/content`, documentData)
    } else {
      const ext = currentFormat.value === 'markdown' ? '.md' : '.tdoc'
      const name = documentTitle.value.endsWith(ext) ? documentTitle.value : `${documentTitle.value}${ext}`
      const response = await api.post('/documents/create-file', {
        ...documentData,
        parentId: props.folderId ?? null,
        name,
      })
      const file = response.data
      const fileId = file?.id ?? file?.ID
      if (fileId) {
        currentFileId.value = fileId
      }
      emit('saved', file)
    }

    lastSaved.value = new Date()
    isDirty.value = false
    return true
  } catch (err: any) {
    const message = err.response?.data?.error ?? err.message ?? 'Failed to save'
    saveError.value = message
    console.error('Failed to save document:', err)
    return false
  } finally {
    saving.value = false
  }
}

function handleTitleChange() {
  isDirty.value = true
  triggerAutoSave()
}

async function handleClose() {
  // Clear auto-save timeout
  if (autoSaveTimeout) {
    clearTimeout(autoSaveTimeout)
    autoSaveTimeout = null
  }
  // Save before closing if there are unsaved changes
  if (editor.value && isDirty.value) {
    const ok = await saveDocument()
    if (!ok) return // Keep modal open if save failed
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
