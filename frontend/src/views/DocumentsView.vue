<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import { createEditorExtensions } from '@/extensions/editorConfig'
import { useEditorPrefsStore } from '@/stores/editorPrefs'
import { useEditorPasteImage } from '@/composables/useEditorPasteImage'
import { useDocumentsFolder } from '@/composables/useDocumentsFolder'
import DocumentsSidebar from '@/components/DocumentsSidebar.vue'
import EditorToolbar from '@/components/EditorToolbar.vue'
import api from '@/api'

const route = useRoute()
const editorPrefsStore = useEditorPrefsStore()
const { documentsFolderId, ensureDocumentsFolder } = useDocumentsFolder()

// --- Mobile sidebar ---
const sidebarOpen = ref(window.innerWidth >= 768)

// --- Current open file ---
const currentFileId = computed(() => route.params.fileId as string | undefined)
const documentTitle = ref('Untitled Document')
const saving = ref(false)
const lastSaved = ref<Date | null>(null)
const isDirty = ref(false)
const saveError = ref<string | null>(null)
const currentFormat = ref<string>('markdown')
let autoSaveTimeout: ReturnType<typeof setTimeout> | null = null

// --- Editor ---
const editor = useEditor({
  extensions: createEditorExtensions({
    placeholder: "Start writing... (type '/' for commands)",
    enabledToolbarIds: editorPrefsStore.enabledIds,
  }),
  content: '',
  editorProps: {
    handlePaste: useEditorPasteImage(),
  },
  onUpdate: () => {
    if (currentFileId.value) {
      isDirty.value = true
      triggerAutoSave()
    }
  },
})

function triggerAutoSave() {
  if (autoSaveTimeout) clearTimeout(autoSaveTimeout)
  autoSaveTimeout = setTimeout(() => {
    if (isDirty.value && !saving.value && currentFileId.value) saveDocument()
  }, 2000)
}

async function loadDocument(fileId: string) {
  saveError.value = null
  isDirty.value = false
  lastSaved.value = null
  try {
    const response = await api.get(`/files/${fileId}/content`)
    const data = response.data
    documentTitle.value = data.title || 'Untitled Document'
    currentFormat.value = data.format || 'markdown'
    const content = data.content || ''
    if (currentFormat.value === 'markdown') {
      editor.value?.commands.setContent(content, false)
    } else if (currentFormat.value === 'html') {
      editor.value?.commands.setContent(content)
    } else {
      try {
        editor.value?.commands.setContent(JSON.parse(content))
      } catch {
        editor.value?.commands.setContent(content)
      }
    }
  } catch (err: any) {
    console.error('Failed to load document:', err)
  }
}

async function saveDocument() {
  if (!editor.value || !currentFileId.value) return
  if (saving.value) return
  saving.value = true
  saveError.value = null
  try {
    let content: string
    if (currentFormat.value === 'markdown') {
      content = typeof (editor.value as any).getMarkdown === 'function'
        ? (editor.value as any).getMarkdown()
        : editor.value.getHTML()
    } else if (currentFormat.value === 'html') {
      content = editor.value.getHTML()
    } else {
      content = JSON.stringify(editor.value.getJSON())
    }
    await api.put(`/files/${currentFileId.value}/content`, {
      title: documentTitle.value,
      content,
      format: currentFormat.value,
    })
    lastSaved.value = new Date()
    isDirty.value = false
  } catch (err: any) {
    saveError.value = err.response?.data?.error ?? err.message ?? 'Failed to save'
  } finally {
    saving.value = false
  }
}

function handleTitleChange() {
  isDirty.value = true
  triggerAutoSave()
}

function formatRelativeTime(date: Date): string {
  const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000)
  if (seconds < 5) return 'just now'
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  return `${Math.floor(minutes / 60)}h ago`
}

// Ctrl+S save
function handleKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 's') {
    e.preventDefault()
    if (currentFileId.value) saveDocument()
  }
}

// Load doc on route change
watch(currentFileId, async (id) => {
  if (autoSaveTimeout) {
    clearTimeout(autoSaveTimeout)
    autoSaveTimeout = null
  }
  if (isDirty.value && currentFileId.value) await saveDocument()
  if (id) {
    await loadDocument(id)
    // On mobile: close sidebar when document opens
    if (window.innerWidth < 768) sidebarOpen.value = false
  } else {
    editor.value?.commands.clearContent()
    documentTitle.value = ''
    isDirty.value = false
    lastSaved.value = null
  }
}, { immediate: true })

onMounted(async () => {
  await ensureDocumentsFolder()
  window.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  if (autoSaveTimeout) clearTimeout(autoSaveTimeout)
  if (isDirty.value && currentFileId.value) saveDocument()
  window.removeEventListener('keydown', handleKeydown)
  editor.value?.destroy()
})

function handleOpenFile(_fileId: string) {
  if (window.innerWidth < 768) sidebarOpen.value = false
}
</script>

<template>
  <div class="h-full flex overflow-hidden relative">
    <!-- Mobile backdrop -->
    <div
      v-if="sidebarOpen"
      class="fixed inset-0 z-30 bg-black/40 md:hidden"
      @click="sidebarOpen = false"
    />

    <!-- Sidebar: overlay on mobile, inline on desktop -->
    <div
      :class="[
        'h-full z-40 transition-transform duration-200',
        'fixed md:relative md:translate-x-0',
        sidebarOpen ? 'translate-x-0' : '-translate-x-full md:translate-x-0',
        !sidebarOpen ? 'md:flex' : 'flex',
      ]"
    >
      <DocumentsSidebar
        :documents-folder-id="documentsFolderId"
        :current-file-id="currentFileId"
        @open-file="handleOpenFile"
        @close="sidebarOpen = false"
      />
    </div>

    <!-- Main content -->
    <div class="flex-1 flex flex-col min-w-0 h-full overflow-hidden">

      <!-- No document open: empty state -->
      <div v-if="!currentFileId" class="flex-1 flex flex-col items-center justify-center text-stone-400 dark:text-stone-500 gap-3">
        <!-- Mobile: toggle sidebar button -->
        <button
          @click="sidebarOpen = true"
          class="md:hidden mb-4 flex items-center gap-2 px-3 py-2 text-sm bg-stone-100 dark:bg-neutral-800 rounded-lg"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
          </svg>
          Browse Documents
        </button>
        <svg class="w-16 h-16 opacity-30" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
        <p class="text-sm">Select a document or create a new one</p>
      </div>

      <!-- Document editor -->
      <template v-else>
        <!-- Editor top bar -->
        <div class="flex-shrink-0 bg-white dark:bg-neutral-800 border-b border-stone-200 dark:border-neutral-700">
          <div class="flex items-center gap-2 px-3 py-2">
            <!-- Mobile sidebar toggle -->
            <button
              @click="sidebarOpen = !sidebarOpen"
              class="md:hidden flex items-center justify-center w-8 h-8 rounded hover:bg-stone-100 dark:hover:bg-neutral-700 flex-shrink-0"
              aria-label="Toggle sidebar"
            >
              <svg class="w-4 h-4 text-stone-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>

            <!-- Title -->
            <input
              v-model="documentTitle"
              @input="handleTitleChange"
              class="flex-1 min-w-0 text-base font-semibold bg-transparent border-none outline-none focus:ring-1 focus:ring-stone-300 dark:focus:ring-neutral-600 rounded px-1 dark:text-stone-100 placeholder-stone-400"
              placeholder="Untitled Document"
            />

            <!-- Save state -->
            <span v-if="saving" class="text-xs text-stone-400 flex-shrink-0">Saving...</span>
            <span v-else-if="saveError" class="text-xs text-red-500 flex-shrink-0 max-w-[160px] truncate" :title="saveError">{{ saveError }}</span>
            <span v-else-if="lastSaved" class="text-xs text-stone-400 flex-shrink-0">Saved {{ formatRelativeTime(lastSaved) }}</span>
            <span v-else-if="isDirty" class="text-xs text-amber-500 flex-shrink-0">Unsaved</span>

            <!-- Save button -->
            <button
              @click="saveDocument"
              :disabled="saving || !isDirty"
              class="flex-shrink-0 flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-md hover:bg-neutral-700 dark:hover:bg-neutral-300 disabled:opacity-40 disabled:cursor-not-allowed"
            >
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7H5a2 2 0 00-2 2v9a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-3m-1 4l-3 3m0 0l-3-3m3 3V4" />
              </svg>
              Save
            </button>
          </div>

          <!-- Formatting toolbar -->
          <div v-if="editor" class="px-3 pb-1.5 border-t border-stone-100 dark:border-neutral-700">
            <EditorToolbar :editor="editor" />
          </div>
        </div>

        <!-- Editor body -->
        <div class="flex-1 overflow-auto bg-stone-50 dark:bg-neutral-900 min-h-0">
          <div class="max-w-4xl mx-auto py-8 px-4 md:px-8 pb-[max(2rem,env(safe-area-inset-bottom))]">
            <EditorContent
              :editor="editor"
              class="prose prose-stone prose-lg dark:prose-invert max-w-none min-h-[60vh] bg-white dark:bg-neutral-800 rounded-xl shadow-sm ring-1 ring-stone-200 dark:ring-neutral-700 p-6 md:p-10 [&_.ProseMirror]:outline-none"
            />
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style>
.ProseMirror p.is-editor-empty:first-child::before {
  color: #adb5bd;
  content: attr(data-placeholder);
  float: left;
  height: 0;
  pointer-events: none;
}
</style>
