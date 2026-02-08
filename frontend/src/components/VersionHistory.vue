<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '@/api'

interface FileVersion {
  id: string
  file_id: string
  version: number
  size: number
  created_at: string
  created_by: string
}

const props = defineProps<{
  fileId: string
  fileName: string
}>()

const emit = defineEmits<{
  close: []
}>()

const versions = ref<FileVersion[]>([])
const loading = ref(true)
const error = ref('')

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins} minute${diffMins > 1 ? 's' : ''} ago`
  if (diffHours < 24) return `${diffHours} hour${diffHours > 1 ? 's' : ''} ago`
  if (diffDays < 7) return `${diffDays} day${diffDays > 1 ? 's' : ''} ago`
  return formatDate(dateStr)
}

async function loadVersions() {
  loading.value = true
  error.value = ''
  try {
    const response = await api.get(`/files/${props.fileId}/versions`)
    versions.value = response.data.versions || []
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Failed to load versions'
  } finally {
    loading.value = false
  }
}

function downloadVersion(version: FileVersion) {
  window.open(`/api/files/${props.fileId}/versions/${version.version}/download`, '_blank')
}

async function restoreVersion(version: FileVersion) {
  if (!confirm(`Restore to version ${version.version}?`)) return
  
  try {
    await api.post(`/files/${props.fileId}/versions/${version.version}/restore`)
    emit('close')
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Failed to restore version'
  }
}

onMounted(() => {
  loadVersions()
})
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    @click.self="emit('close')"
  >
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-lg mx-4 max-h-[80vh] flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b dark:border-gray-700">
        <div>
          <h3 class="font-medium dark:text-white">Version History</h3>
          <p class="text-sm text-gray-500 dark:text-gray-400 truncate">{{ fileName }}</p>
        </div>
        <button
          @click="emit('close')"
          class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
        >
          <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Content -->
      <div class="flex-1 overflow-auto">
        <!-- Loading -->
        <div v-if="loading" class="flex items-center justify-center py-12">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>

        <!-- Error -->
        <div v-else-if="error" class="p-4 text-center text-red-600">
          {{ error }}
        </div>

        <!-- Empty -->
        <div v-else-if="versions.length === 0" class="p-8 text-center text-gray-500 dark:text-gray-400">
          <svg class="w-12 h-12 mx-auto text-gray-300 dark:text-gray-600 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p>No previous versions available</p>
          <p class="text-sm mt-1">Versions are created when files are modified</p>
        </div>

        <!-- Version List -->
        <div v-else class="divide-y dark:divide-gray-700">
          <div
            v-for="(version, index) in versions"
            :key="version.id"
            class="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50"
          >
            <div class="flex items-start justify-between gap-4">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-medium dark:text-white">Version {{ version.version }}</span>
                  <span v-if="index === 0" class="px-2 py-0.5 text-xs bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300 rounded-full">
                    Current
                  </span>
                </div>
                <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                  {{ formatRelativeTime(version.created_at) }} · {{ formatBytes(version.size) }}
                </p>
              </div>
              <div class="flex items-center gap-1">
                <button
                  @click="downloadVersion(version)"
                  class="p-2 text-gray-500 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                  title="Download this version"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                  </svg>
                </button>
                <button
                  v-if="index > 0"
                  @click="restoreVersion(version)"
                  class="p-2 text-gray-500 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded"
                  title="Restore this version"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-4 py-3 border-t dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 text-xs text-gray-500 dark:text-gray-400">
        Versions are kept for 30 days
      </div>
    </div>
  </div>
</template>
