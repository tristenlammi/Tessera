<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useFilesStore } from '@/stores/files'

const props = defineProps<{
  fileId: string
  currentName: string
}>()

const emit = defineEmits<{
  close: []
  renamed: [newName: string]
}>()

const filesStore = useFilesStore()
const newName = ref(props.currentName)
const inputRef = ref<HTMLInputElement | null>(null)
const loading = ref(false)
const error = ref('')

// Focus input on mount
nextTick(() => {
  inputRef.value?.focus()
  inputRef.value?.select()
})

async function handleSubmit() {
  if (!newName.value.trim()) {
    error.value = 'Name cannot be empty'
    return
  }

  if (newName.value === props.currentName) {
    emit('close')
    return
  }

  loading.value = true
  error.value = ''

  try {
    await filesStore.renameFile(props.fileId, newName.value.trim())
    emit('renamed', newName.value.trim())
    emit('close')
  } catch (err) {
    error.value = 'Failed to rename'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="emit('close')"
  >
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-sm mx-4">
      <form @submit.prevent="handleSubmit">
        <div class="px-4 py-3 border-b dark:border-gray-700">
          <h3 class="font-medium dark:text-white">Rename</h3>
        </div>

        <div class="p-4">
          <input
            ref="inputRef"
            v-model="newName"
            type="text"
            class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
            :class="{ 'border-red-500': error }"
            @keydown.esc="emit('close')"
          />
          <p v-if="error" class="mt-1 text-sm text-red-600">{{ error }}</p>
        </div>

        <div class="px-4 py-3 border-t dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 rounded-b-lg flex justify-end gap-2">
          <button
            type="button"
            @click="emit('close')"
            class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-700 rounded-lg text-sm"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="loading"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm disabled:opacity-50"
          >
            {{ loading ? 'Renaming...' : 'Rename' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
