<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useFilesStore, type FileItem } from '@/stores/files'

const props = defineProps<{
  fileIds: string[]
  mode: 'move' | 'copy'
}>()

const emit = defineEmits<{
  close: []
  complete: []
}>()

const filesStore = useFilesStore()
const loading = ref(false)
const loadingFolders = ref(true)
const folders = ref<FileItem[]>([])
const currentPath = ref<{ id: string | null; name: string }[]>([{ id: null, name: 'My Files' }])
const selectedFolder = ref<string | null>(null)
const error = ref('')

async function loadFolders(parentId: string | null = null) {
  loadingFolders.value = true
  try {
    const response = await fetch(`/api/files?parent_id=${parentId || ''}`, {
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    const data = await response.json()
    folders.value = (data.files || []).filter((f: FileItem) => f.is_folder)
  } finally {
    loadingFolders.value = false
  }
}

function navigateToFolder(folder: FileItem) {
  currentPath.value.push({ id: folder.id, name: folder.name })
  selectedFolder.value = folder.id
  loadFolders(folder.id)
}

function navigateToBreadcrumb(index: number) {
  const item = currentPath.value[index]
  currentPath.value = currentPath.value.slice(0, index + 1)
  selectedFolder.value = item.id
  loadFolders(item.id)
}

async function handleSubmit() {
  loading.value = true
  error.value = ''
  
  try {
    for (const fileId of props.fileIds) {
      if (props.mode === 'move') {
        await filesStore.moveFile(fileId, selectedFolder.value)
      } else {
        // Copy would need a separate API endpoint
        // For now, just show error
        error.value = 'Copy functionality not yet implemented'
        return
      }
    }
    emit('complete')
    emit('close')
  } catch (err: any) {
    error.value = err.message || 'Operation failed'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadFolders(null)
})
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)] overflow-y-auto"
    @click.self="emit('close')"
  >
    <div class="modal-panel bg-white dark:bg-neutral-800 shadow-xl w-full mx-auto flex flex-col flex-shrink-0 my-auto">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b dark:border-neutral-700">
        <h3 class="font-medium dark:text-stone-100">
          {{ mode === 'move' ? 'Move' : 'Copy' }} {{ fileIds.length }} item{{ fileIds.length > 1 ? 's' : '' }}
        </h3>
        <button
          @click="emit('close')"
          class="min-w-[44px] min-h-[44px] flex items-center justify-center hover:bg-stone-100 dark:hover:bg-neutral-700 rounded -m-2"
        >
          <svg class="w-5 h-5 text-stone-500 dark:text-stone-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Breadcrumb -->
      <div class="px-4 py-2 border-b dark:border-neutral-700 bg-stone-50 dark:bg-neutral-800/50">
        <nav class="flex items-center gap-1 text-sm overflow-x-auto">
          <template v-for="(item, index) in currentPath" :key="item.id ?? 'root'">
            <span v-if="index > 0" class="text-stone-400">/</span>
            <button
              @click="navigateToBreadcrumb(index)"
              class="min-h-[44px] flex items-center hover:text-stone-800 dark:hover:text-stone-200 px-2 py-1 -my-1 whitespace-nowrap rounded"
              :class="{ 'font-medium': index === currentPath.length - 1 }"
            >
              {{ item.name }}
            </button>
          </template>
        </nav>
      </div>

      <!-- Folder List -->
      <div class="flex-1 overflow-auto min-h-[200px]">
        <div v-if="loadingFolders" class="flex items-center justify-center py-8">
          <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-neutral-800 dark:border-neutral-200"></div>
        </div>

        <div v-else-if="folders.length === 0" class="py-8 text-center text-stone-500 dark:text-stone-400">
          No folders here
        </div>

        <div v-else class="py-2">
          <button
            v-for="folder in folders"
            :key="folder.id"
            @dblclick="navigateToFolder(folder)"
            @click="selectedFolder = folder.id"
            :class="[
              'w-full flex items-center gap-3 px-4 py-2 min-h-[44px] text-left',
              selectedFolder === folder.id ? 'bg-stone-100 dark:bg-neutral-700/30 text-stone-800 dark:text-stone-300' : 'hover:bg-stone-50 dark:hover:bg-neutral-700/50'
            ]"
          >
            <svg class="w-5 h-5 text-stone-700 dark:text-stone-300" fill="currentColor" viewBox="0 0 20 20">
              <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
            </svg>
            <span class="truncate">{{ folder.name }}</span>
          </button>
        </div>
      </div>

      <!-- Error -->
      <div v-if="error" class="px-4 py-2 text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20">
        {{ error }}
      </div>

      <!-- Actions -->
      <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 px-4 py-3 border-t dark:border-neutral-700 bg-stone-50 dark:bg-neutral-800/50">
        <p class="text-sm text-stone-500 dark:text-stone-400">
          {{ mode === 'move' ? 'Moving' : 'Copying' }} to: 
          <span class="font-medium">{{ currentPath[currentPath.length - 1].name }}</span>
        </p>
        <div class="flex gap-2">
          <button
            @click="emit('close')"
            class="min-h-[44px] px-4 py-2 text-sm font-medium text-stone-700 dark:text-stone-300 bg-white dark:bg-neutral-700 border border-stone-300 dark:border-neutral-700 rounded-lg hover:bg-stone-50 dark:hover:bg-neutral-600"
          >
            Cancel
          </button>
          <button
            @click="handleSubmit"
            :disabled="loading"
            class="min-h-[44px] px-4 py-2 text-sm font-medium text-white bg-neutral-800 dark:bg-neutral-200 dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 disabled:opacity-50"
          >
            {{ loading ? 'Processing...' : (mode === 'move' ? 'Move Here' : 'Copy Here') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
