<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch } from 'vue'
import type { FileItem } from '@/stores/files'
import api from '@/api'

const props = defineProps<{
  file: FileItem
}>()

const emit = defineEmits<{
  close: []
}>()

const blobUrl = ref<string | null>(null)
const streamUrl = ref<string | null>(null)
const textContent = ref<string | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const previewType = computed(() => {
  if (props.file.is_folder) return 'folder'
  const mime = props.file.mime_type || ''
  if (mime.startsWith('image/')) return 'image'
  if (mime === 'application/pdf') return 'pdf'
  if (mime.startsWith('text/') || mime === 'application/json' || mime === 'application/javascript') return 'text'
  if (mime.startsWith('video/')) return 'video'
  if (mime.startsWith('audio/')) return 'audio'
  return 'unknown'
})

const isStreamable = computed(() => previewType.value === 'video' || previewType.value === 'audio')

// Fetch file with auth and create blob URL for preview
async function fetchFileForPreview() {
  loading.value = true
  error.value = null
  
  // Clean up previous blob URL
  if (blobUrl.value) {
    URL.revokeObjectURL(blobUrl.value)
    blobUrl.value = null
  }
  streamUrl.value = null
  textContent.value = null
  
  try {
    // For video/audio, use streaming URL with short-lived token
    if (isStreamable.value) {
      const tokenRes = await api.get(`/files/${props.file.id}/stream-token`)
      const token = tokenRes.data.token
      streamUrl.value = `/api/files/${props.file.id}/stream?token=${encodeURIComponent(token)}`
    } else {
      const response = await api.get(`/files/${props.file.id}/download`, {
        responseType: previewType.value === 'text' ? 'text' : 'blob'
      })
      
      // For text files, read as text
      if (previewType.value === 'text') {
        textContent.value = response.data
      } else {
        // For other types, create blob URL
        const mimeType = props.file.mime_type || response.headers?.['content-type'] || ''
        const blob = new Blob([response.data], { type: mimeType })
        blobUrl.value = URL.createObjectURL(blob)
      }
    }
  } catch (err: any) {
    console.error('Failed to load file for preview:', err)
    if (err.response?.data instanceof Blob) {
      try {
        const text = await err.response.data.text()
        const parsed = JSON.parse(text)
        error.value = parsed.error || err.message || 'Failed to load file'
      } catch {
        error.value = err.message || 'Failed to load file'
      }
    } else {
      error.value = err.response?.data?.error || err.message || 'Failed to load file'
    }
  } finally {
    loading.value = false
  }
}

// Trigger download with auth
async function handleDownload() {
  try {
    const response = await api.get(`/files/${props.file.id}/download`, {
      responseType: 'blob'
    })
    
    const url = URL.createObjectURL(response.data)
    const a = document.createElement('a')
    a.href = url
    a.download = props.file.name
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (err) {
    console.error('Failed to download file:', err)
  }
}

onMounted(() => {
  fetchFileForPreview()
})

onUnmounted(() => {
  // Clean up blob URL when component is destroyed
  if (blobUrl.value) {
    URL.revokeObjectURL(blobUrl.value)
  }
})

// Re-fetch if file changes
watch(() => props.file.id, () => {
  fetchFileForPreview()
})

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/70"
    @click.self="emit('close')"
    @keydown="handleKeydown"
    tabindex="0"
  >
    <div class="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl w-[95vw] h-[95vh] max-w-[1800px] flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b dark:border-gray-700 flex-shrink-0">
        <div class="flex items-center gap-3 min-w-0">
          <h3 class="font-medium truncate dark:text-white">{{ file.name }}</h3>
          <span class="text-sm text-gray-500 dark:text-gray-400 flex-shrink-0">{{ formatBytes(file.size) }}</span>
        </div>
        <div class="flex items-center gap-2">
          <button
            @click="handleDownload"
            class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg text-gray-600 dark:text-gray-400"
            title="Download"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
          </button>
          <button
            @click="emit('close')"
            class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg text-gray-600 dark:text-gray-400"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Preview content -->
      <div class="flex-1 overflow-auto p-4 flex items-center justify-center bg-gray-50 dark:bg-gray-900 min-h-0">
        <!-- Loading state -->
        <div v-if="loading" class="text-center">
          <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p class="text-gray-500 dark:text-gray-400">Loading preview...</p>
        </div>

        <!-- Error state -->
        <div v-else-if="error" class="text-center">
          <svg class="w-24 h-24 mx-auto text-red-300 mb-4" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
          </svg>
          <p class="text-red-500 mb-4">{{ error }}</p>
          <button
            @click="handleDownload"
            class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            Download Instead
          </button>
        </div>

        <!-- Image preview -->
        <img
          v-else-if="previewType === 'image' && blobUrl"
          :src="blobUrl"
          :alt="file.name"
          class="max-w-full max-h-full object-contain"
        />

        <!-- PDF preview -->
        <iframe
          v-else-if="previewType === 'pdf' && blobUrl"
          :src="blobUrl"
          type="application/pdf"
          class="w-full h-full border-0"
        />

        <!-- Video preview -->
        <video
          v-else-if="previewType === 'video' && streamUrl"
          :src="streamUrl"
          controls
          preload="metadata"
          class="max-w-full max-h-full"
        >
          Your browser does not support video playback.
        </video>

        <!-- Audio preview -->
        <div v-else-if="previewType === 'audio' && streamUrl" class="text-center">
          <svg class="w-24 h-24 mx-auto text-purple-500 mb-4" fill="currentColor" viewBox="0 0 20 20">
            <path d="M18 3a1 1 0 00-1.196-.98l-10 2A1 1 0 006 5v9.114A4.369 4.369 0 005 14c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V7.82l8-1.6v5.894A4.37 4.37 0 0015 12c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V3z" />
          </svg>
          <audio :src="streamUrl" controls preload="metadata" class="mx-auto" />
        </div>

        <!-- Text preview -->
        <div v-else-if="previewType === 'text' && textContent !== null" class="w-full h-full bg-white dark:bg-gray-800 rounded border dark:border-gray-700 p-4 overflow-auto">
          <pre class="text-sm font-mono whitespace-pre-wrap break-words dark:text-gray-200">{{ textContent }}</pre>
        </div>

        <!-- Unknown/no preview -->
        <div v-else class="text-center">
          <svg class="w-24 h-24 mx-auto text-gray-300 mb-4" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z" clip-rule="evenodd" />
          </svg>
          <p class="text-gray-500 dark:text-gray-400 mb-4">Preview not available for this file type</p>
          <button
            @click="handleDownload"
            class="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            Download
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
