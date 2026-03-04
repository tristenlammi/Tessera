<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useModulesStore } from '@/stores/modules'

defineProps<{
  x: number
  y: number
}>()

const emit = defineEmits<{
  close: []
  newFolder: []
  newDocument: []
  upload: []
  uploadFolder: []
  paste: []
}>()

const menuRef = ref<HTMLElement | null>(null)
const modulesStore = useModulesStore()

function handleClickOutside(e: MouseEvent) {
  if (menuRef.value && !menuRef.value.contains(e.target as Node)) {
    emit('close')
  }
}

onMounted(() => {
  nextTick(() => {
    setTimeout(() => {
      document.addEventListener('click', handleClickOutside)
      document.addEventListener('contextmenu', handleClickOutside)
    }, 10)
  })
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('contextmenu', handleClickOutside)
})

function handleAction(action: string) {
  emit(action as any)
  emit('close')
}
</script>

<template>
  <div
    ref="menuRef"
    class="fixed bg-white dark:bg-neutral-800 rounded-lg shadow-lg border border-stone-200 dark:border-neutral-700 py-1 min-w-48 z-50"
    :style="{ left: `${x}px`, top: `${y}px` }"
  >
    <button
      @click="handleAction('newFolder')"
      class="w-full px-4 py-2 text-left text-sm hover:bg-stone-100 dark:hover:bg-neutral-700 flex items-center gap-3 text-stone-700 dark:text-stone-300"
    >
      <svg class="w-4 h-4 text-stone-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
      </svg>
      New Folder
    </button>

    <button
      v-if="modulesStore.isModuleEnabled('documents')"
      @click="handleAction('newDocument')"
      class="w-full px-4 py-2 text-left text-sm hover:bg-stone-100 dark:hover:bg-neutral-700 flex items-center gap-3 text-stone-700 dark:text-stone-300"
    >
      <svg class="w-4 h-4 text-stone-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
      </svg>
      New Document
    </button>

    <hr class="my-1 border-stone-200 dark:border-neutral-700" />

    <button
      @click="handleAction('upload')"
      class="w-full px-4 py-2 text-left text-sm hover:bg-stone-100 dark:hover:bg-neutral-700 flex items-center gap-3 text-stone-700 dark:text-stone-300"
    >
      <svg class="w-4 h-4 text-stone-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
      </svg>
      Upload Files
    </button>

    <button
      @click="handleAction('uploadFolder')"
      class="w-full px-4 py-2 text-left text-sm hover:bg-stone-100 dark:hover:bg-neutral-700 flex items-center gap-3 text-stone-700 dark:text-stone-300"
    >
      <svg class="w-4 h-4 text-stone-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
      </svg>
      Upload Folder
    </button>

    <button
      @click="handleAction('paste')"
      class="w-full px-4 py-2 text-left text-sm hover:bg-stone-100 dark:hover:bg-neutral-700 flex items-center gap-3 text-stone-500 dark:text-stone-500"
      disabled
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
      </svg>
      Paste
    </button>
  </div>
</template>
