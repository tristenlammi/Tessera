<template>
  <button
    :class="[
      'min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center flex-shrink-0 p-2 rounded text-stone-600 dark:text-stone-400 transition-colors',
      active
        ? 'bg-stone-200 dark:bg-neutral-600'
        : 'hover:bg-stone-100 dark:hover:bg-neutral-700'
    ]"
    :title="title"
    @click="$emit('click')"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
defineProps<{
  active?: boolean
  title?: string
}>()

defineEmits<{
  click: []
}>()
</script>
