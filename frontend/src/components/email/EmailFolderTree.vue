<script setup lang="ts">
import { ref, computed } from 'vue'
import type { EmailFolder } from '@/stores/email'

const props = defineProps<{
  folder: EmailFolder
  level?: number
  currentFolderId?: string | null
  currentVirtualFolder?: string | null
  currentLabelId?: string | null
  draggable?: boolean
  isFirst?: boolean
  isLast?: boolean
}>()

const emit = defineEmits<{
  (e: 'select', folder: EmailFolder): void
  (e: 'contextmenu', folder: EmailFolder, event: MouseEvent): void
  (e: 'drop', folderId: string, targetFolderId: string | null, position: 'before' | 'after' | 'into'): void
  (e: 'emailDrop', emailId: string, targetFolderId: string): void
}>()

const level = computed(() => props.level || 0)
const isExpanded = ref(true)
const hasChildren = computed(() => props.folder.children && props.folder.children.length > 0)
const isDragOver = ref(false)
const dropPosition = ref<'before' | 'into' | 'after' | null>(null)
const isCustomFolder = computed(() => props.folder.folder_type === 'custom')

const isSelected = computed(() => {
  return props.currentFolderId === props.folder.id && 
    !props.currentVirtualFolder && 
    !props.currentLabelId
})

function toggleExpand(event: Event) {
  event.stopPropagation()
  isExpanded.value = !isExpanded.value
}

function selectFolder() {
  emit('select', props.folder)
}

function handleContextMenu(event: MouseEvent) {
  event.preventDefault()
  emit('contextmenu', props.folder, event)
}

function handleDragStart(event: DragEvent) {
  if (!isCustomFolder.value) {
    event.preventDefault()
    return
  }
  event.dataTransfer?.setData('text/plain', props.folder.id)
  event.dataTransfer!.effectAllowed = 'move'
}

function handleDragOver(event: DragEvent) {
  event.preventDefault()
  event.stopPropagation()
  event.dataTransfer!.dropEffect = 'move'
  
  // Check if this is an email drag (emails always drop "into")
  const dragType = event.dataTransfer?.types.includes('application/email-id') ? 'email' : 'folder'
  
  if (dragType === 'email') {
    // Emails always drop into the folder
    dropPosition.value = 'into'
  } else {
    // Determine drop position based on mouse position for folders
    const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
    const y = event.clientY - rect.top
    const height = rect.height
    
    // Top 25%: drop before, Middle 50%: drop into, Bottom 25%: drop after
    if (y < height * 0.25) {
      dropPosition.value = 'before'
    } else if (y > height * 0.75) {
      dropPosition.value = 'after'
    } else {
      dropPosition.value = 'into'
    }
  }
  
  isDragOver.value = true
}

function handleDragLeave(event: DragEvent) {
  // Only trigger if we're actually leaving this element
  const relatedTarget = event.relatedTarget as HTMLElement
  const currentTarget = event.currentTarget as HTMLElement
  if (!currentTarget.contains(relatedTarget)) {
    isDragOver.value = false
    dropPosition.value = null
  }
}

function handleDrop(event: DragEvent) {
  event.preventDefault()
  event.stopPropagation()
  isDragOver.value = false
  const pos = dropPosition.value
  dropPosition.value = null
  
  // Check if this is an email drop
  const emailId = event.dataTransfer?.getData('application/email-id')
  if (emailId) {
    emit('emailDrop', emailId, props.folder.id)
    return
  }
  
  // Otherwise it's a folder drop
  const draggedFolderId = event.dataTransfer?.getData('text/plain')
  if (draggedFolderId && draggedFolderId !== props.folder.id && pos) {
    emit('drop', draggedFolderId, props.folder.id, pos)
  }
}

function getFolderIcon(folderType: string | null): string {
  switch (folderType) {
    case 'inbox': return 'inbox'
    case 'sent': return 'paper-airplane'
    case 'drafts': return 'pencil'
    case 'trash': return 'trash'
    case 'spam': return 'exclamation'
    case 'archive': return 'archive'
    default: return 'folder'
  }
}
</script>

<template>
  <div>
    <!-- Drop indicator line - before -->
    <div
      v-if="isDragOver && dropPosition === 'before'"
      class="h-0.5 bg-blue-500 mx-2 rounded-full"
      :style="{ marginLeft: `${8 + level * 16}px` }"
    ></div>
    
    <div
      :class="[
        'relative transition-all',
        isDragOver && dropPosition === 'into' && 'ring-2 ring-blue-500 ring-inset rounded-lg bg-blue-50/50 dark:bg-blue-900/20'
      ]"
      @dragover="handleDragOver"
      @dragleave="handleDragLeave"
      @drop="handleDrop"
    >
      <button
        @click="selectFolder"
        @contextmenu="handleContextMenu"
        :draggable="isCustomFolder && draggable"
        @dragstart="handleDragStart"
        :class="[
          'w-full flex items-center gap-2 py-2 rounded-lg text-sm transition-colors',
          isSelected
            ? 'bg-blue-50 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
            : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700',
          isCustomFolder && draggable && 'cursor-grab active:cursor-grabbing'
        ]"
        :style="{ paddingLeft: `${12 + level * 16}px`, paddingRight: '12px' }"
      >
        <!-- Expand/collapse button for nested folders -->
        <button
          v-if="hasChildren"
          @click="toggleExpand"
          class="w-4 h-4 flex items-center justify-center -ml-1 hover:bg-gray-200 dark:hover:bg-gray-600 rounded"
        >
          <svg
            :class="['w-3 h-3 transition-transform', isExpanded ? 'rotate-90' : '']"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
        </button>
        <span v-else class="w-4"></span>

        <!-- Folder icons -->
        <svg v-if="getFolderIcon(folder.folder_type) === 'inbox'" class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
        </svg>
        <svg v-else-if="getFolderIcon(folder.folder_type) === 'paper-airplane'" class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
        </svg>
        <svg v-else-if="getFolderIcon(folder.folder_type) === 'pencil'" class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
        </svg>
        <svg v-else-if="getFolderIcon(folder.folder_type) === 'trash'" class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
        <svg v-else-if="getFolderIcon(folder.folder_type) === 'exclamation'" class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <svg v-else-if="getFolderIcon(folder.folder_type) === 'archive'" class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
        </svg>
        <svg v-else class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        
        <span class="flex-1 text-left truncate">{{ folder.name }}</span>
        
        <span
          v-if="folder.unread_count > 0"
          class="px-2 py-0.5 text-xs font-medium bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300 rounded-full"
        >
          {{ folder.unread_count }}
        </span>
      </button>
    </div>

    <!-- Drop indicator line - after (only show if no children or collapsed) -->
    <div
      v-if="isDragOver && dropPosition === 'after' && (!hasChildren || !isExpanded)"
      class="h-0.5 bg-blue-500 mx-2 rounded-full"
      :style="{ marginLeft: `${8 + level * 16}px` }"
    ></div>

    <!-- Recursive children -->
    <div v-if="hasChildren && isExpanded">
      <EmailFolderTree
        v-for="(child, index) in folder.children"
        :key="child.id"
        :folder="child"
        :level="level + 1"
        :current-folder-id="currentFolderId"
        :current-virtual-folder="currentVirtualFolder"
        :current-label-id="currentLabelId"
        :draggable="draggable"
        :is-first="index === 0"
        :is-last="index === folder.children!.length - 1"
        @select="$emit('select', $event)"
        @contextmenu="(folder, event) => $emit('contextmenu', folder, event)"
        @drop="(folderId, targetId, position) => $emit('drop', folderId, targetId, position)"
        @emailDrop="(emailId, targetFolderId) => $emit('emailDrop', emailId, targetFolderId)"
      />
    </div>
    
    <!-- Drop indicator line - after children -->
    <div
      v-if="isDragOver && dropPosition === 'after' && hasChildren && isExpanded"
      class="h-0.5 bg-blue-500 mx-2 rounded-full"
      :style="{ marginLeft: `${8 + level * 16}px` }"
    ></div>
  </div>
</template>
