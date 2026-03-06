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
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)]">
    <div class="modal-panel bg-white dark:bg-neutral-800 shadow-xl w-full">
      <div class="px-6 py-4 border-b dark:border-neutral-700">
        <h2 class="text-lg font-semibold dark:text-stone-100">New Folder</h2>
      </div>

      <form @submit.prevent="handleCreate" class="p-6">
        <div v-if="error" class="mb-4 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-3 rounded-lg text-sm">
          {{ error }}
        </div>

        <div class="mb-4">
          <label for="folder-name" class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">
            Folder name
          </label>
          <input
            id="folder-name"
            v-model="name"
            type="text"
            autofocus
            class="w-full min-h-[44px] px-3 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg focus:outline-none focus:ring-2 focus:ring-stone-400 focus:border-stone-400 dark:bg-neutral-700 dark:text-stone-100"
            placeholder="Untitled folder"
          />
        </div>

        <div class="flex justify-end gap-3">
          <button
            type="button"
            @click="emit('close')"
            class="min-h-[44px] px-4 py-2 text-sm font-medium text-stone-700 dark:text-stone-300 hover:bg-stone-100 dark:hover:bg-neutral-700 rounded-lg"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="loading"
            class="min-h-[44px] px-4 py-2 text-sm font-medium text-white bg-neutral-800 dark:bg-neutral-200 dark:text-neutral-800 hover:bg-neutral-700 dark:hover:bg-neutral-300 rounded-lg disabled:opacity-50"
          >
            {{ loading ? 'Creating...' : 'Create' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
