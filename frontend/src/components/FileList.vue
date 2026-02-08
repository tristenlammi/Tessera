<script setup lang="ts">
import type { FileItem } from '@/stores/files'
import { useFilesStore } from '@/stores/files'

const props = defineProps<{
  files: FileItem[]
  showRestore?: boolean
}>()

const emit = defineEmits<{
  open: [file: FileItem]
  restore: [file: FileItem]
  contextmenu: [file: FileItem, event: MouseEvent]
}>()

const filesStore = useFilesStore()

function formatBytes(bytes: number): string {
  if (bytes === 0) return '—'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

function handleContextMenu(file: FileItem, event: MouseEvent) {
  event.preventDefault()
  emit('contextmenu', file, event)
}
</script>

<template>
  <table class="w-full">
    <thead class="bg-gray-50 dark:bg-gray-800 text-xs text-gray-500 dark:text-gray-400 uppercase">
      <tr>
        <th class="px-4 py-3 text-left font-medium">Name</th>
        <th class="px-4 py-3 text-left font-medium w-32">Size</th>
        <th class="px-4 py-3 text-left font-medium w-40">Modified</th>
        <th v-if="showRestore" class="px-4 py-3 text-left font-medium w-24">Actions</th>
      </tr>
    </thead>
    <tbody class="divide-y dark:divide-gray-700">
      <tr
        v-for="file in files"
        :key="file.id"
        @dblclick="emit('open', file)"
        @click="filesStore.selectFile(file.id, $event.ctrlKey || $event.metaKey)"
        @contextmenu="handleContextMenu(file, $event)"
        :class="[
          'cursor-pointer transition-colors',
          filesStore.selectedFiles.has(file.id)
            ? 'bg-blue-50 dark:bg-blue-900/30'
            : 'hover:bg-gray-50 dark:hover:bg-gray-700'
        ]"
      >
        <td class="px-4 py-2">
          <div class="flex items-center gap-3">
            <!-- Icon -->
            <svg v-if="file.is_folder" class="w-5 h-5 text-blue-500 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
              <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
            </svg>
            <svg v-else class="w-5 h-5 text-gray-400 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z" clip-rule="evenodd" />
            </svg>
            <span class="truncate" :title="file.name">{{ file.name }}</span>
            <svg v-if="file.is_starred" class="w-4 h-4 text-yellow-400 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
              <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
            </svg>
          </div>
        </td>
        <td class="px-4 py-2 text-sm text-gray-500 dark:text-gray-400">
          {{ file.is_folder ? '—' : formatBytes(file.size) }}
        </td>
        <td class="px-4 py-2 text-sm text-gray-500 dark:text-gray-400">
          {{ formatDate(file.updated_at) }}
        </td>
        <td v-if="showRestore" class="px-4 py-2">
          <button
            @click.stop="emit('restore', file)"
            class="text-sm text-blue-600 hover:text-blue-800"
          >
            Restore
          </button>
        </td>
      </tr>
    </tbody>
  </table>
</template>
