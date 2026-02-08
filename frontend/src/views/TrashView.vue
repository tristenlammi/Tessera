<script setup lang="ts">
import { onMounted } from 'vue'
import { useFilesStore } from '@/stores/files'
import FileList from '@/components/FileList.vue'

const filesStore = useFilesStore()

onMounted(() => {
  filesStore.fetchTrash()
})

async function handleEmptyTrash() {
  if (confirm('Are you sure you want to permanently delete all items in trash?')) {
    await filesStore.emptyTrash()
  }
}

async function handleRestore(file: any) {
  await filesStore.restoreFile(file.id)
}
</script>

<template>
  <div class="h-full flex flex-col">
    <div class="flex items-center justify-between p-4 border-b dark:border-gray-700">
      <h1 class="text-xl font-semibold dark:text-white">Trash</h1>
      <button
        v-if="filesStore.files.length > 0"
        @click="handleEmptyTrash"
        class="px-3 py-1.5 text-sm text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg"
      >
        Empty Trash
      </button>
    </div>

    <div class="flex-1 overflow-auto">
      <div v-if="filesStore.loading" class="flex items-center justify-center h-64">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>

      <div v-else-if="filesStore.files.length === 0" class="flex flex-col items-center justify-center h-64 text-gray-500 dark:text-gray-400">
        <svg class="w-16 h-16 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
        <p class="text-lg font-medium">Trash is empty</p>
      </div>

      <FileList
        v-else
        :files="filesStore.files"
        show-restore
        @restore="handleRestore"
      />
    </div>
  </div>
</template>
