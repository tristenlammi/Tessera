import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { TOOLBAR_EXTENSIONS, getDefaultEnabledIds } from '@/extensions/toolbarRegistry'
import type { ToolbarExtensionMeta, ToolbarGroup } from '@/extensions/toolbarRegistry'

const STORAGE_KEY = 'tessera-editor-toolbar'

export const useEditorPrefsStore = defineStore('editorPrefs', () => {
  const enabledIds = ref<string[]>(loadFromStorage())

  function loadFromStorage(): string[] {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (raw) {
        const parsed = JSON.parse(raw)
        if (Array.isArray(parsed) && parsed.every((v: unknown) => typeof v === 'string')) {
          return parsed
        }
      }
    } catch {
      // ignore
    }
    return getDefaultEnabledIds()
  }

  function persist() {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(enabledIds.value))
  }

  function isEnabled(id: string): boolean {
    return enabledIds.value.includes(id)
  }

  function toggle(id: string, enabled: boolean) {
    if (enabled && !enabledIds.value.includes(id)) {
      enabledIds.value.push(id)
    } else if (!enabled) {
      enabledIds.value = enabledIds.value.filter((e) => e !== id)
    }
    persist()
  }

  function resetToDefaults() {
    enabledIds.value = getDefaultEnabledIds()
    persist()
  }

  const groupedExtensions = computed(() => {
    const groups = new Map<ToolbarGroup, ToolbarExtensionMeta[]>()
    for (const ext of TOOLBAR_EXTENSIONS) {
      const list = groups.get(ext.group) || []
      list.push(ext)
      groups.set(ext.group, list)
    }
    return groups
  })

  return {
    enabledIds,
    isEnabled,
    toggle,
    resetToDefaults,
    groupedExtensions,
  }
})
