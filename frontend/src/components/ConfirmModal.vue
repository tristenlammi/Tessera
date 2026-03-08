<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'

const props = withDefaults(defineProps<{
  title?: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
  showCancel?: boolean
}>(), {
  title: 'Confirm',
  confirmText: 'Confirm',
  cancelText: 'Cancel',
  danger: false,
  showCancel: true
})

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('cancel')
  if (e.key === 'Enter') emit('confirm')
}

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
})
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)] overflow-y-auto"
    @click.self="emit('cancel')"
  >
    <div class="modal-panel max-w-sm bg-white dark:bg-neutral-800 shadow-xl w-full mx-auto flex-shrink-0 my-auto">
      <!-- Header -->
      <div class="px-4 py-3 border-b dark:border-neutral-700">
        <h3 class="font-medium text-lg dark:text-stone-100">{{ title }}</h3>
      </div>

      <!-- Content -->
      <div class="px-4 py-4">
        <p class="text-stone-600 dark:text-stone-300">{{ message }}</p>
      </div>

      <!-- Actions -->
      <div class="flex justify-end gap-2 px-4 py-3 border-t dark:border-neutral-700 bg-stone-50 dark:bg-neutral-800/50 rounded-b-lg">
        <button
          v-if="props.showCancel"
          @click="emit('cancel')"
          class="min-h-[44px] px-4 py-2 text-sm font-medium text-stone-700 dark:text-stone-300 bg-white dark:bg-neutral-700 border border-stone-300 dark:border-neutral-700 rounded-lg hover:bg-stone-50 dark:hover:bg-neutral-600"
        >
          {{ cancelText }}
        </button>
        <button
          @click="emit('confirm')"
          :class="[
            'min-h-[44px] px-4 py-2 text-sm font-medium text-white rounded-lg',
            danger
              ? 'bg-red-600 hover:bg-red-700'
              : 'bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-900 hover:bg-neutral-700 dark:hover:bg-neutral-300'
          ]"
        >
          {{ confirmText }}
        </button>
      </div>
    </div>
  </div>
</template>
