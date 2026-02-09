<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { RouterView } from 'vue-router'
import ErrorBoundary from '@/components/ErrorBoundary.vue'
import api from '@/api'

const backendReady = ref(false)
const checking = ref(true)
const retryCount = ref(0)
let retryTimer: ReturnType<typeof setTimeout> | null = null

async function checkBackend() {
  try {
    await api.get('/auth/setup-status', { timeout: 5000 })
    backendReady.value = true
    checking.value = false
  } catch {
    retryCount.value++
    checking.value = false
    // Retry every 3 seconds
    retryTimer = setTimeout(checkBackend, 3000)
  }
}

onMounted(() => {
  checkBackend()
})

onUnmounted(() => {
  if (retryTimer) clearTimeout(retryTimer)
})
</script>

<template>
  <!-- Backend not reachable -->
  <div v-if="!backendReady" class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
    <div class="text-center max-w-md px-6">
      <!-- Animated logo / icon -->
      <div class="inline-flex items-center justify-center w-20 h-20 bg-blue-600 rounded-2xl mb-6 shadow-lg">
        <svg class="w-12 h-12 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
        </svg>
      </div>

      <h1 class="text-3xl font-bold text-gray-900 dark:text-white mb-2">Tessera</h1>

      <div class="mt-6 space-y-4">
        <!-- Spinner -->
        <div class="flex justify-center">
          <div class="relative">
            <div class="w-10 h-10 border-4 border-blue-200 dark:border-blue-900 rounded-full"></div>
            <div class="w-10 h-10 border-4 border-transparent border-t-blue-600 rounded-full absolute top-0 left-0 animate-spin"></div>
          </div>
        </div>

        <p class="text-lg font-medium text-gray-700 dark:text-gray-300">
          Waiting for backend...
        </p>

        <p class="text-sm text-gray-500 dark:text-gray-400">
          The server is starting up or is currently unavailable.
          <br />
          Retrying automatically<span v-if="retryCount > 0"> (attempt {{ retryCount }})</span>...
        </p>
      </div>
    </div>
  </div>

  <!-- Backend is up, render app normally -->
  <ErrorBoundary v-else>
    <RouterView />
  </ErrorBoundary>
</template>
