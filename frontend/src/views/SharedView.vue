<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '@/api'
import FilePreview from '@/components/FilePreview.vue'

interface SharedFile {
  id: string
  name: string
  is_folder: boolean
  size: number
  mime_type: string
  permission: string
  owner_id: string
  owner_name: string
  owner_email: string
  shared_at: string
}

const files = ref<SharedFile[]>([])
const loading = ref(true)
const previewFile = ref<SharedFile | null>(null)

onMounted(async () => {
  await fetchSharedFiles()
})

async function fetchSharedFiles() {
  loading.value = true
  try {
    const response = await api.get('/shared')
    files.value = response.data.files || []
  } finally {
    loading.value = false
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

function getFileIcon(file: SharedFile): string {
  if (file.is_folder) return '📁'
  if (file.mime_type.startsWith('image/')) return '🖼️'
  if (file.mime_type.startsWith('video/')) return '🎬'
  if (file.mime_type.startsWith('audio/')) return '🎵'
  if (file.mime_type === 'application/pdf') return '📄'
  if (file.mime_type.includes('zip') || file.mime_type.includes('archive')) return '📦'
  return '📄'
}

function getPermissionBadge(permission: string) {
  const badges: Record<string, { text: string; class: string }> = {
    view: { text: 'View', class: 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300' },
    edit: { text: 'Edit', class: 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300' },
    admin: { text: 'Full Access', class: 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300' }
  }
  return badges[permission] || badges.view
}

function openFile(file: SharedFile) {
  if (file.is_folder) {
    // TODO: Navigate into shared folder
  } else {
    previewFile.value = file as any
  }
}
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Header -->
    <div class="flex items-center justify-between p-4 border-b dark:border-gray-700">
      <h1 class="text-xl font-semibold dark:text-white">Shared with Me</h1>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-4">
      <div v-if="loading" class="flex items-center justify-center h-64">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>

      <div v-else-if="files.length === 0" class="flex flex-col items-center justify-center h-64 text-gray-500 dark:text-gray-400">
        <svg class="w-16 h-16 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
        <p class="text-lg font-medium">No shared files</p>
        <p class="text-sm">Files shared with you will appear here.</p>
      </div>

      <div v-else class="space-y-2">
        <div
          v-for="file in files"
          :key="file.id"
          @click="openFile(file)"
          class="flex items-center gap-4 p-3 rounded-lg border dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer transition-colors"
        >
          <!-- Icon -->
          <div class="text-2xl">{{ getFileIcon(file) }}</div>

          <!-- Details -->
          <div class="flex-1 min-w-0">
            <div class="font-medium truncate dark:text-white">{{ file.name }}</div>
            <div class="text-sm text-gray-500 dark:text-gray-400">
              Shared by {{ file.owner_name || file.owner_email }} · {{ formatDate(file.shared_at) }}
            </div>
          </div>

          <!-- Permission badge -->
          <span
            class="px-2 py-1 text-xs font-medium rounded-full"
            :class="getPermissionBadge(file.permission).class"
          >
            {{ getPermissionBadge(file.permission).text }}
          </span>

          <!-- Size -->
          <div v-if="!file.is_folder" class="text-sm text-gray-500 dark:text-gray-400 w-20 text-right">
            {{ formatBytes(file.size) }}
          </div>
        </div>
      </div>
    </div>

    <!-- File Preview Modal -->
    <FilePreview
      v-if="previewFile"
      :file="previewFile"
      @close="previewFile = null"
    />
  </div>
</template>