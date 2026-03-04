<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from 'vue'
import type { SlashCommandItem } from '@/extensions/SlashCommands'

const props = defineProps<{
  items: SlashCommandItem[]
  command: (item: SlashCommandItem) => void
}>()

const selectedIndex = ref(0)

watch(() => props.items, () => {
  selectedIndex.value = 0
})

function onKeyDown(event: KeyboardEvent): boolean {
  if (event.key === 'ArrowUp') {
    event.preventDefault()
    selectedIndex.value = (selectedIndex.value + props.items.length - 1) % props.items.length
    scrollToSelected()
    return true
  }
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    selectedIndex.value = (selectedIndex.value + 1) % props.items.length
    scrollToSelected()
    return true
  }
  if (event.key === 'Enter') {
    event.preventDefault()
    selectItem(selectedIndex.value)
    return true
  }
  return false
}

function selectItem(index: number) {
  const item = props.items[index]
  if (item) {
    props.command(item)
  }
}

function scrollToSelected() {
  const el = document.querySelector('.slash-menu-item.is-selected')
  el?.scrollIntoView({ block: 'nearest' })
}

defineExpose({ onKeyDown })
</script>

<template>
  <div
    v-if="items.length > 0"
    class="slash-command-menu bg-white dark:bg-neutral-800 border border-stone-200 dark:border-neutral-700 rounded-lg shadow-lg overflow-hidden w-64 max-h-80 overflow-y-auto"
  >
    <button
      v-for="(item, index) in items"
      :key="item.title"
      class="slash-menu-item w-full flex items-center gap-3 px-3 py-2 text-left transition-colors"
      :class="index === selectedIndex ? 'is-selected bg-stone-100 dark:bg-neutral-700' : 'hover:bg-stone-50 dark:hover:bg-neutral-700/50'"
      @click="selectItem(index)"
      @mouseenter="selectedIndex = index"
    >
      <span class="flex-shrink-0 w-8 h-8 flex items-center justify-center rounded bg-stone-100 dark:bg-neutral-700 text-stone-600 dark:text-stone-300 text-sm font-mono font-bold">
        {{ item.icon }}
      </span>
      <div class="min-w-0">
        <div class="text-sm font-medium text-stone-900 dark:text-stone-100">{{ item.title }}</div>
        <div class="text-xs text-stone-500 dark:text-stone-400 truncate">{{ item.description }}</div>
      </div>
    </button>
  </div>
  <div
    v-else
    class="slash-command-menu bg-white dark:bg-neutral-800 border border-stone-200 dark:border-neutral-700 rounded-lg shadow-lg p-3 w-64"
  >
    <span class="text-sm text-stone-500 dark:text-stone-400">No results</span>
  </div>
</template>
