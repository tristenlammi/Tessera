<script setup lang="ts">
import { ref, computed } from 'vue'
import type { FileItem } from '@/stores/files'
import { useFilesStore } from '@/stores/files'

const props = defineProps<{
  files: FileItem[]
  iconSize?: number
}>()

const emit = defineEmits<{
  open: [file: FileItem]
  contextmenu: [file: FileItem, event: MouseEvent]
  drop: [fileIds: string[], targetFolderId: string]
}>()

const filesStore = useFilesStore()
const dragOverFolder = ref<string | null>(null)

// Default icon size is 48px
const actualIconSize = computed(() => props.iconSize || 48)

// Dynamic grid columns class based on icon size
const gridColumnsClass = computed(() => {
  const size = actualIconSize.value
  if (size <= 40) {
    return 'grid-cols-6 sm:grid-cols-8 md:grid-cols-10 lg:grid-cols-12 xl:grid-cols-14'
  } else if (size <= 56) {
    return 'grid-cols-4 sm:grid-cols-6 md:grid-cols-8 lg:grid-cols-10 xl:grid-cols-12'
  } else if (size <= 72) {
    return 'grid-cols-3 sm:grid-cols-4 md:grid-cols-6 lg:grid-cols-8 xl:grid-cols-10'
  } else {
    return 'grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 xl:grid-cols-8'
  }
})

// Icon size style
const iconSizeStyle = computed(() => ({
  width: `${actualIconSize.value}px`,
  height: `${actualIconSize.value}px`
}))

function getFileIcon(file: FileItem): string {
  if (file.is_folder) return 'folder'
  if (file.mime_type?.startsWith('image/')) return 'image'
  if (file.mime_type?.startsWith('video/')) return 'video'
  if (file.mime_type?.startsWith('audio/')) return 'audio'
  if (file.mime_type === 'application/pdf') return 'pdf'
  return 'file'
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '—'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function handleContextMenu(file: FileItem, event: MouseEvent) {
  event.preventDefault()
  emit('contextmenu', file, event)
}

function handleDragStart(file: FileItem, event: DragEvent) {
  if (!event.dataTransfer) return
  
  // If file is selected, drag all selected files; otherwise just this one
  const fileIds = filesStore.selectedFiles.has(file.id)
    ? Array.from(filesStore.selectedFiles)
    : [file.id]
  
  event.dataTransfer.setData('application/json', JSON.stringify(fileIds))
  event.dataTransfer.effectAllowed = 'move'
}

function handleDragOver(file: FileItem, event: DragEvent) {
  if (!file.is_folder) return
  event.preventDefault()
  event.stopPropagation()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
  dragOverFolder.value = file.id
}

function handleDragLeave(file: FileItem, event: DragEvent) {
  // Only reset if we're leaving the folder element itself, not a child
  const relatedTarget = event.relatedTarget as HTMLElement | null
  const currentTarget = event.currentTarget as HTMLElement
  if (relatedTarget && currentTarget.contains(relatedTarget)) {
    return // Still within the same folder element
  }
  if (dragOverFolder.value === file.id) {
    dragOverFolder.value = null
  }
}

function handleDrop(file: FileItem, event: DragEvent) {
  event.preventDefault()
  event.stopPropagation()
  dragOverFolder.value = null
  
  if (!file.is_folder || !event.dataTransfer) return
  
  const data = event.dataTransfer.getData('application/json')
  if (!data) return
  
  try {
    const fileIds = JSON.parse(data) as string[]
    // Don't drop a folder into itself
    if (fileIds.includes(file.id)) return
    emit('drop', fileIds, file.id)
  } catch (e) {
    console.error('Invalid drag data', e)
  }
}
</script>

<template>
  <div class="p-4 grid gap-2" :class="gridColumnsClass">
    <div
      v-for="file in files"
      :key="file.id"
      draggable="true"
      @dragstart="handleDragStart(file, $event)"
      @dragover="handleDragOver(file, $event)"
      @dragleave="handleDragLeave(file, $event)"
      @drop="handleDrop(file, $event)"
      @dblclick="emit('open', file)"
      @click="filesStore.selectFile(file.id, $event.ctrlKey || $event.metaKey)"
      @contextmenu="handleContextMenu(file, $event)"
      :class="[
        'group p-2 rounded-lg border cursor-pointer transition-all select-none',
        filesStore.selectedFiles.has(file.id)
          ? 'border-blue-500 bg-blue-50 dark:bg-blue-900/30'
          : dragOverFolder === file.id
            ? 'border-blue-400 bg-blue-100 dark:bg-blue-900/20 scale-105'
            : 'border-transparent hover:bg-gray-100 dark:hover:bg-gray-700'
      ]"
    >
      <!-- Icon -->
      <div class="relative flex items-center justify-center mb-1 pointer-events-none">
        <!-- Star indicator -->
        <svg 
          v-if="file.is_starred" 
          class="absolute -top-1 -right-1 w-4 h-4 text-yellow-400 z-10"
          fill="currentColor" 
          viewBox="0 0 20 20"
        >
          <path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
        </svg>
        <!-- Folder icon -->
        <svg v-if="getFileIcon(file) === 'folder'" class="text-blue-500" :style="iconSizeStyle" fill="currentColor" viewBox="0 0 20 20">
          <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
        </svg>
        <!-- Image icon -->
        <svg v-else-if="getFileIcon(file) === 'image'" class="text-green-500" :style="iconSizeStyle" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z" clip-rule="evenodd" />
        </svg>
        <!-- Video icon -->
        <svg v-else-if="getFileIcon(file) === 'video'" class="text-purple-500" :style="iconSizeStyle" fill="currentColor" viewBox="0 0 20 20">
          <path d="M2 6a2 2 0 012-2h6a2 2 0 012 2v8a2 2 0 01-2 2H4a2 2 0 01-2-2V6zM14.553 7.106A1 1 0 0014 8v4a1 1 0 00.553.894l2 1A1 1 0 0018 13V7a1 1 0 00-1.447-.894l-2 1z" />
        </svg>
        <!-- PDF icon -->
        <svg v-else-if="getFileIcon(file) === 'pdf'" class="text-red-500" :style="iconSizeStyle" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z" clip-rule="evenodd" />
        </svg>
        <!-- Generic file icon -->
        <svg v-else class="text-gray-400" :style="iconSizeStyle" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z" clip-rule="evenodd" />
        </svg>
      </div>

      <!-- Name -->
      <div class="text-xs font-medium truncate text-center pointer-events-none" :title="file.name">
        {{ file.name }}
      </div>
      <div v-if="!file.is_folder" class="text-[10px] text-gray-500 dark:text-gray-400 text-center pointer-events-none">
        {{ formatBytes(file.size) }}
      </div>
    </div>
  </div>
</template>
