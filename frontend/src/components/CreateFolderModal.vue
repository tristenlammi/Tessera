<script setup lang="ts">
import { ref } from 'vue'
import { useFilesStore } from '@/stores/files'

const props = defineProps<{
  parentId: string | null
}>()

const emit = defineEmits<{
  close: []
  created: []
}>()

const filesStore = useFilesStore()

const name = ref('')
const loading = ref(false)
const error = ref('')

async function handleCreate() {
  if (!name.value.trim()) {
    error.value = 'Please enter a folder name'
    return
  }

  loading.value = true
  error.value = ''

  try {
    await filesStore.createFolder(name.value, props.parentId)
    emit('created')
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Failed to create folder'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-md mx-4">
      <div class="px-6 py-4 border-b dark:border-gray-700">
        <h2 class="text-lg font-semibold dark:text-white">New Folder</h2>
      </div>

      <form @submit.prevent="handleCreate" class="p-6">
        <div v-if="error" class="mb-4 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-3 rounded-lg text-sm">
          {{ error }}
        </div>

        <div class="mb-4">
          <label for="folder-name" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Folder name
          </label>
          <input
            id="folder-name"
            v-model="name"
            type="text"
            autofocus
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
            placeholder="Untitled folder"
          />
        </div>

        <div class="flex justify-end gap-3">
          <button
            type="button"
            @click="emit('close')"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="loading"
            class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg disabled:opacity-50"
          >
            {{ loading ? 'Creating...' : 'Create' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
