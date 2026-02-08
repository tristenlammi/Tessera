<script setup lang="ts">
import { computed } from 'vue'
import type { FileItem } from '@/stores/files'

const props = defineProps<{
  file: FileItem
}>()

const emit = defineEmits<{
  close: []
}>()

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const fileType = computed(() => {
  if (props.file.is_folder) return 'Folder'
  const mime = props.file.mime_type || ''
  if (mime.startsWith('image/')) return 'Image'
  if (mime.startsWith('video/')) return 'Video'
  if (mime.startsWith('audio/')) return 'Audio'
  if (mime === 'application/pdf') return 'PDF Document'
  if (mime.includes('word') || mime.includes('document')) return 'Document'
  if (mime.includes('sheet') || mime.includes('excel')) return 'Spreadsheet'
  if (mime.includes('presentation') || mime.includes('powerpoint')) return 'Presentation'
  if (mime.startsWith('text/')) return 'Text File'
  return 'File'
})

const iconColor = computed(() => {
  if (props.file.is_folder) return 'text-blue-500'
  const mime = props.file.mime_type || ''
  if (mime.startsWith('image/')) return 'text-green-500'
  if (mime.startsWith('video/')) return 'text-purple-500'
  if (mime.startsWith('audio/')) return 'text-pink-500'
  if (mime === 'application/pdf') return 'text-red-500'
  return 'text-gray-400'
})
</script>

<template>
  <div class="w-80 border-l dark:border-gray-700 bg-white dark:bg-gray-800 flex flex-col h-full">
    <!-- Header -->
    <div class="flex items-center justify-between px-4 py-3 border-b dark:border-gray-700">
      <h3 class="font-medium dark:text-white">Details</h3>
      <button
        @click="emit('close')"
        class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
      >
        <svg class="w-5 h-5 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <!-- Preview -->
    <div class="p-6 flex flex-col items-center border-b dark:border-gray-700">
      <!-- File Icon -->
      <div :class="['w-20 h-20 flex items-center justify-center mb-3', iconColor]">
        <svg v-if="file.is_folder" class="w-16 h-16" fill="currentColor" viewBox="0 0 20 20">
          <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
        </svg>
        <svg v-else-if="file.mime_type?.startsWith('image/')" class="w-16 h-16" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z" clip-rule="evenodd" />
        </svg>
        <svg v-else-if="file.mime_type?.startsWith('video/')" class="w-16 h-16" fill="currentColor" viewBox="0 0 20 20">
          <path d="M2 6a2 2 0 012-2h6a2 2 0 012 2v8a2 2 0 01-2 2H4a2 2 0 01-2-2V6zM14.553 7.106A1 1 0 0014 8v4a1 1 0 00.553.894l2 1A1 1 0 0018 13V7a1 1 0 00-1.447-.894l-2 1z" />
        </svg>
        <svg v-else class="w-16 h-16" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z" clip-rule="evenodd" />
        </svg>
      </div>

      <!-- File Name -->
      <h4 class="font-medium text-center break-all px-2 dark:text-white">{{ file.name }}</h4>
      <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{{ fileType }}</p>
    </div>

    <!-- Details -->
    <div class="flex-1 overflow-auto p-4 space-y-4">
      <!-- Size -->
      <div v-if="!file.is_folder">
        <label class="text-xs font-medium text-gray-500 uppercase">Size</label>
        <p class="text-sm mt-1">{{ formatBytes(file.size) }}</p>
      </div>

      <!-- Type -->
      <div v-if="file.mime_type">
        <label class="text-xs font-medium text-gray-500 uppercase">Type</label>
        <p class="text-sm mt-1">{{ file.mime_type }}</p>
      </div>

      <!-- Created -->
      <div>
        <label class="text-xs font-medium text-gray-500 uppercase">Created</label>
        <p class="text-sm mt-1">{{ formatDate(file.created_at) }}</p>
      </div>

      <!-- Modified -->
      <div>
        <label class="text-xs font-medium text-gray-500 uppercase">Modified</label>
        <p class="text-sm mt-1">{{ formatDate(file.updated_at) }}</p>
      </div>

      <!-- Starred -->
      <div>
        <label class="text-xs font-medium text-gray-500 uppercase">Starred</label>
        <p class="text-sm mt-1 flex items-center gap-1">
          <svg 
            :class="['w-4 h-4', file.is_starred ? 'text-yellow-400' : 'text-gray-300']" 
            fill="currentColor" 
            viewBox="0 0 20 20"
          >
            <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
          </svg>
          {{ file.is_starred ? 'Yes' : 'No' }}
        </p>
      </div>

      <!-- File ID (for debugging) -->
      <div class="pt-4 border-t dark:border-gray-700">
        <label class="text-xs font-medium text-gray-500 uppercase">File ID</label>
        <p class="text-xs mt-1 text-gray-400 font-mono break-all">{{ file.id }}</p>
      </div>
    </div>

    <!-- Actions -->
    <div class="p-4 border-t dark:border-gray-700 space-y-2">
      <a
        v-if="!file.is_folder"
        :href="`/api/files/${file.id}/download`"
        download
        class="flex items-center justify-center gap-2 w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
        Download
      </a>
    </div>
  </div>
</template>
