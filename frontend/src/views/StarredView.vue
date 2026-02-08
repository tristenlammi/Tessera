<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useFilesStore, type FileItem } from '@/stores/files'
import FileGrid from '@/components/FileGrid.vue'
import FileList from '@/components/FileList.vue'
import ContextMenu from '@/components/ContextMenu.vue'

const router = useRouter()
const filesStore = useFilesStore()
const viewMode = ref<'grid' | 'list'>('grid')
const contextMenu = ref<{ file: FileItem; x: number; y: number } | null>(null)

onMounted(() => {
  filesStore.fetchStarred()
})

function handleOpen(file: FileItem) {
  if (file.is_folder) {
    router.push(`/folder/${file.id}`)
  } else {
    // Preview file
  }
}

function handleContextMenu(file: FileItem, event: MouseEvent) {
  contextMenu.value = {
    file,
    x: event.clientX,
    y: event.clientY
  }
}

function closeContextMenu() {
  contextMenu.value = null
}

async function handleContextStar() {
  if (contextMenu.value) {
    await filesStore.toggleStar(contextMenu.value.file.id)
    // Refresh starred list since file was unstarred
    filesStore.fetchStarred()
  }
  closeContextMenu()
}

async function handleContextDelete() {
  if (contextMenu.value) {
    await filesStore.deleteFile(contextMenu.value.file.id)
    filesStore.fetchStarred()
  }
  closeContextMenu()
}
</script>

<template>
  <div class="h-full flex flex-col">
    <!-- Header -->
    <div class="p-4 border-b dark:border-gray-700 flex items-center justify-between">
      <h1 class="text-xl font-semibold dark:text-white">Starred</h1>
      
      <!-- View toggle -->
      <div class="flex items-center gap-2">
        <button
          @click="viewMode = 'grid'"
          :class="['p-2 rounded', viewMode === 'grid' ? 'bg-gray-200 dark:bg-gray-700' : 'hover:bg-gray-100 dark:hover:bg-gray-800']"
        >
          <svg class="w-5 h-5 dark:text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
          </svg>
        </button>
        <button
          @click="viewMode = 'list'"
          :class="['p-2 rounded', viewMode === 'list' ? 'bg-gray-200 dark:bg-gray-700' : 'hover:bg-gray-100 dark:hover:bg-gray-800']"
        >
          <svg class="w-5 h-5 dark:text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-auto">
      <!-- Loading state -->
      <div v-if="filesStore.loading" class="flex items-center justify-center h-full">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>

      <!-- Empty state -->
      <div v-else-if="filesStore.files.length === 0" class="flex flex-col items-center justify-center h-full text-gray-500 dark:text-gray-400">
        <svg class="w-16 h-16 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
        </svg>
        <p class="text-lg font-medium">No starred items</p>
        <p class="mt-1">Files and folders you star will appear here</p>
      </div>

      <!-- Files grid/list -->
      <FileGrid
        v-else-if="viewMode === 'grid'"
        :files="filesStore.files"
        @open="handleOpen"
        @contextmenu="handleContextMenu"
      />
      <FileList
        v-else
        :files="filesStore.files"
        @open="handleOpen"
        @contextmenu="handleContextMenu"
      />
    </div>

    <!-- Context Menu -->
    <ContextMenu
      v-if="contextMenu"
      :file="contextMenu.file"
      :x="contextMenu.x"
      :y="contextMenu.y"
      @close="closeContextMenu"
      @star="handleContextStar"
      @delete="handleContextDelete"
    />
  </div>
</template>
