<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue'

const hasError = ref(false)
const errorMessage = ref('')

onErrorCaptured((err: Error) => {
  hasError.value = true
  errorMessage.value = err.message || 'An unexpected error occurred'
  console.error('[ErrorBoundary] Caught error:', err)
  return false // prevent propagation
})

function reload() {
  window.location.reload()
}

function goHome() {
  window.location.href = '/'
}
</script>

<template>
  <div v-if="hasError" class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
    <div class="text-center max-w-md">
      <div class="mx-auto flex items-center justify-center h-16 w-16 rounded-full bg-red-100 dark:bg-red-900/30 mb-6">
        <svg class="h-8 w-8 text-red-600 dark:text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
        </svg>
      </div>
      <h1 class="text-xl font-bold text-gray-900 dark:text-white mb-2">Something went wrong</h1>
      <p class="text-sm text-gray-500 dark:text-gray-400 mb-6">
        {{ errorMessage }}
      </p>
      <div class="flex gap-3 justify-center">
        <button
          @click="reload"
          class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm font-medium transition-colors"
        >
          Reload Page
        </button>
        <button
          @click="goHome"
          class="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 text-sm font-medium transition-colors"
        >
          Go Home
        </button>
      </div>
    </div>
  </div>
  <slot v-else />
</template>
