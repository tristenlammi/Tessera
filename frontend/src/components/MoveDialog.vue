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
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="emit('close')"
  >
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4 max-h-[80vh] flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b dark:border-gray-700">
        <h3 class="font-medium dark:text-white">
          {{ mode === 'move' ? 'Move' : 'Copy' }} {{ fileIds.length }} item{{ fileIds.length > 1 ? 's' : '' }}
        </h3>
        <button
          @click="emit('close')"
          class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
        >
          <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Breadcrumb -->
      <div class="px-4 py-2 border-b dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
        <nav class="flex items-center gap-1 text-sm overflow-x-auto">
          <template v-for="(item, index) in currentPath" :key="item.id ?? 'root'">
            <span v-if="index > 0" class="text-gray-400">/</span>
            <button
              @click="navigateToBreadcrumb(index)"
              class="hover:text-blue-600 px-1 whitespace-nowrap"
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
          <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
        </div>

        <div v-else-if="folders.length === 0" class="py-8 text-center text-gray-500 dark:text-gray-400">
          No folders here
        </div>

        <div v-else class="py-2">
          <button
            v-for="folder in folders"
            :key="folder.id"
            @dblclick="navigateToFolder(folder)"
            @click="selectedFolder = folder.id"
            :class="[
              'w-full flex items-center gap-3 px-4 py-2 text-left',
              selectedFolder === folder.id ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300' : 'hover:bg-gray-50 dark:hover:bg-gray-700/50'
            ]"
          >
            <svg class="w-5 h-5 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
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
      <div class="flex items-center justify-between px-4 py-3 border-t dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ mode === 'move' ? 'Moving' : 'Copying' }} to: 
          <span class="font-medium">{{ currentPath[currentPath.length - 1].name }}</span>
        </p>
        <div class="flex gap-2">
          <button
            @click="emit('close')"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600"
          >
            Cancel
          </button>
          <button
            @click="handleSubmit"
            :disabled="loading"
            class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {{ loading ? 'Processing...' : (mode === 'move' ? 'Move Here' : 'Copy Here') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
