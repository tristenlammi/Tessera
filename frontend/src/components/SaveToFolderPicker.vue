<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useFilesStore } from '@/stores/files'

const props = defineProps<{
  saving?: boolean
}>()

const emit = defineEmits<{
  select: [folderId: string | null]
  cancel: []
}>()

const filesStore = useFilesStore()
const selectedFolderId = ref<string | null>(null)
const loading = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    await filesStore.fetchFiles(null) // Fetch root files
  } finally {
    loading.value = false
  }
})

// Get only folders from current view
const folders = computed(() => {
  return filesStore.files.filter(f => f.is_folder)
})

function selectFolder(folderId: string | null) {
  selectedFolderId.value = folderId
}

function confirm() {
  emit('select', selectedFolderId.value)
}

function getFolderIcon(folder: any): string {
  return '📁'
}
</script>

<template>
  <div class="space-y-4">
    <!-- Folder browser -->
    <div class="border dark:border-gray-700 rounded-lg max-h-[300px] overflow-y-auto">
      <!-- Root option -->
      <button
        @click="selectFolder(null)"
        class="w-full flex items-center gap-3 px-4 py-2.5 text-left hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
        :class="{ 'bg-blue-50 dark:bg-blue-900/30': selectedFolderId === null }"
      >
        <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
        </svg>
        <span class="font-medium text-gray-900 dark:text-white">My Files (Root)</span>
      </button>
      
      <div v-if="loading" class="px-4 py-8 text-center text-gray-500">
        <svg class="w-6 h-6 animate-spin mx-auto mb-2" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        Loading folders...
      </div>
      
      <div v-else-if="folders.length === 0" class="px-4 py-4 text-center text-sm text-gray-500 dark:text-gray-400">
        No folders available. File will be saved to root.
      </div>
      
      <template v-else>
        <button
          v-for="folder in folders"
          :key="folder.id"
          @click="selectFolder(folder.id)"
          class="w-full flex items-center gap-3 px-4 py-2.5 text-left hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors border-t dark:border-gray-700"
          :class="{ 'bg-blue-50 dark:bg-blue-900/30': selectedFolderId === folder.id }"
        >
          <span class="text-lg">📁</span>
          <span class="text-gray-700 dark:text-gray-300">{{ folder.name }}</span>
        </button>
      </template>
    </div>
    
    <!-- Selected location info -->
    <div class="text-sm text-gray-600 dark:text-gray-400">
      Save to: <span class="font-medium text-gray-900 dark:text-white">
        {{ selectedFolderId ? folders.find(f => f.id === selectedFolderId)?.name : 'My Files (Root)' }}
      </span>
    </div>
    
    <!-- Actions -->
    <div class="flex justify-end gap-3">
      <button
        @click="emit('cancel')"
        :disabled="saving"
        class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg disabled:opacity-50"
      >
        Cancel
      </button>
      <button
        @click="confirm"
        :disabled="saving"
        class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"
      >
        <svg v-if="saving" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        {{ saving ? 'Saving...' : 'Save Here' }}
      </button>
    </div>
  </div>
</template>
