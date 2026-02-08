<script setup lang="ts">
import { computed } from 'vue'
import { useFilesStore } from '@/stores/files'

const emit = defineEmits<{
  move: []
  copy: []
  delete: []
  download: []
}>()

const filesStore = useFilesStore()

const selectedCount = computed(() => filesStore.selectedFiles.size)
</script>

<template>
  <div 
    v-if="selectedCount > 0"
    class="fixed bottom-6 left-1/2 -translate-x-1/2 bg-gray-900 text-white rounded-lg shadow-xl px-4 py-3 flex items-center gap-4 z-40"
  >
    <span class="text-sm font-medium">
      {{ selectedCount }} item{{ selectedCount > 1 ? 's' : '' }} selected
    </span>

    <div class="w-px h-6 bg-gray-700"></div>

    <!-- Move -->
    <button
      @click="emit('move')"
      class="flex items-center gap-2 px-3 py-1.5 hover:bg-gray-800 rounded-lg text-sm"
      title="Move"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
      </svg>
      Move
    </button>

    <!-- Copy -->
    <button
      @click="emit('copy')"
      class="flex items-center gap-2 px-3 py-1.5 hover:bg-gray-800 rounded-lg text-sm"
      title="Copy"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
      </svg>
      Copy
    </button>

    <!-- Download -->
    <button
      @click="emit('download')"
      class="flex items-center gap-2 px-3 py-1.5 hover:bg-gray-800 rounded-lg text-sm"
      title="Download"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
      </svg>
      Download
    </button>

    <!-- Delete -->
    <button
      @click="emit('delete')"
      class="flex items-center gap-2 px-3 py-1.5 hover:bg-red-600 rounded-lg text-sm"
      title="Delete"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
      </svg>
      Delete
    </button>

    <div class="w-px h-6 bg-gray-700"></div>

    <!-- Clear Selection -->
    <button
      @click="filesStore.clearSelection()"
      class="flex items-center gap-2 px-3 py-1.5 hover:bg-gray-800 rounded-lg text-sm"
      title="Clear selection"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
      </svg>
      Clear
    </button>
  </div>
</template>
