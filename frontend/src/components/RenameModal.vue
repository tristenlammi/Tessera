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
    <div class="bg-white dark:bg-neutral-800 rounded-lg shadow-xl w-full max-w-sm mx-4">
      <form @submit.prevent="handleSubmit">
        <div class="px-4 py-3 border-b dark:border-neutral-700">
          <h3 class="font-medium dark:text-stone-100">Rename</h3>
        </div>

        <div class="p-4">
          <input
            ref="inputRef"
            v-model="newName"
            type="text"
            class="w-full px-3 py-2 border dark:border-neutral-700 rounded-lg focus:ring-2 focus:ring-stone-400 focus:border-stone-400 dark:bg-neutral-700 dark:text-stone-100"
            :class="{ 'border-red-500': error }"
            @keydown.esc="emit('close')"
          />
          <p v-if="error" class="mt-1 text-sm text-red-600">{{ error }}</p>
        </div>

        <div class="px-4 py-3 border-t dark:border-neutral-700 bg-stone-50 dark:bg-neutral-800/50 rounded-b-lg flex justify-end gap-2">
          <button
            type="button"
            @click="emit('close')"
            class="px-4 py-2 text-stone-700 dark:text-stone-300 hover:bg-stone-200 dark:hover:bg-neutral-700 rounded-lg text-sm"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="loading"
            class="px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 text-sm disabled:opacity-50"
          >
            {{ loading ? 'Renaming...' : 'Rename' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
