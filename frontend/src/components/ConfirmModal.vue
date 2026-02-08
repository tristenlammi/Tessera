<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'

const props = withDefaults(defineProps<{
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
}>(), {
  title: 'Confirm',
  confirmText: 'Confirm',
  cancelText: 'Cancel',
  danger: false
})

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('cancel')
  if (e.key === 'Enter') emit('confirm')
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="emit('cancel')"
  >
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-sm mx-4">
      <!-- Header -->
      <div class="px-4 py-3 border-b dark:border-gray-700">
        <h3 class="font-medium text-lg dark:text-white">{{ title }}</h3>
      </div>

      <!-- Content -->
      <div class="px-4 py-4">
        <p class="text-gray-600 dark:text-gray-300">{{ message }}</p>
      </div>

      <!-- Actions -->
      <div class="flex justify-end gap-2 px-4 py-3 border-t dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 rounded-b-lg">
        <button
          @click="emit('cancel')"
          class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600"
        >
          {{ cancelText }}
        </button>
        <button
          @click="emit('confirm')"
          :class="[
            'px-4 py-2 text-sm font-medium text-white rounded-lg',
            danger
              ? 'bg-red-600 hover:bg-red-700'
              : 'bg-blue-600 hover:bg-blue-700'
          ]"
        >
          {{ confirmText }}
        </button>
      </div>
    </div>
  </div>
</template>
