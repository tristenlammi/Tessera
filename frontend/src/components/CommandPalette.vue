<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useFilesStore, type FileItem } from '@/stores/files'

const emit = defineEmits<{
  close: []
  openFile: [file: FileItem]
}>()

const router = useRouter()
const filesStore = useFilesStore()

const query = ref('')
const inputRef = ref<HTMLInputElement | null>(null)
const selectedIndex = ref(0)
const loading = ref(false)
const searchResults = ref<FileItem[]>([])

// Commands available in the palette
const commands = [
  { id: 'new-folder', name: 'New Folder', icon: 'folder-plus', shortcut: 'N', action: 'createFolder' },
  { id: 'upload', name: 'Upload Files', icon: 'upload', shortcut: 'U', action: 'upload' },
  { id: 'my-files', name: 'Go to My Files', icon: 'home', action: 'navigate', path: '/files' },
  { id: 'starred', name: 'Go to Starred', icon: 'star', action: 'navigate', path: '/starred' },
  { id: 'trash', name: 'Go to Trash', icon: 'trash', action: 'navigate', path: '/trash' },
  { id: 'settings', name: 'Settings', icon: 'settings', action: 'navigate', path: '/settings' },
]

const filteredCommands = computed(() => {
  if (!query.value.trim()) return commands
  const q = query.value.toLowerCase()
  return commands.filter(cmd => cmd.name.toLowerCase().includes(q))
})

const allResults = computed(() => {
  const items: Array<{ type: 'command' | 'file'; data: any }> = []
  
  // Add commands first
  filteredCommands.value.forEach(cmd => {
    items.push({ type: 'command', data: cmd })
  })
  
  // Add file results
  searchResults.value.forEach(file => {
    items.push({ type: 'file', data: file })
  })
  
  return items
})

let searchTimeout: ReturnType<typeof setTimeout> | null = null

watch(query, async (newQuery) => {
  if (searchTimeout) clearTimeout(searchTimeout)
  
  if (!newQuery.trim()) {
    searchResults.value = []
    selectedIndex.value = 0
    return
  }
  
  searchTimeout = setTimeout(async () => {
    loading.value = true
    try {
      searchResults.value = await filesStore.search(newQuery)
    } finally {
      loading.value = false
    }
  }, 200)
})

watch(allResults, () => {
  selectedIndex.value = 0
})

function handleKeydown(e: KeyboardEvent) {
  switch (e.key) {
    case 'ArrowDown':
      e.preventDefault()
      selectedIndex.value = Math.min(selectedIndex.value + 1, allResults.value.length - 1)
      break
    case 'ArrowUp':
      e.preventDefault()
      selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
      break
    case 'Enter':
      e.preventDefault()
      if (allResults.value[selectedIndex.value]) {
        selectItem(allResults.value[selectedIndex.value])
      }
      break
    case 'Escape':
      emit('close')
      break
  }
}

function selectItem(item: { type: 'command' | 'file'; data: any }) {
  if (item.type === 'command') {
    executeCommand(item.data)
  } else {
    selectFile(item.data)
  }
}

function executeCommand(cmd: any) {
  switch (cmd.action) {
    case 'navigate':
      router.push(cmd.path)
      break
    case 'createFolder':
      // Emit event or handle folder creation
      break
    case 'upload':
      document.getElementById('file-upload')?.click()
      break
  }
  emit('close')
}

function selectFile(file: FileItem) {
  if (file.is_folder) {
    router.push({ name: 'folder', params: { id: file.id } })
  } else {
    emit('openFile', file)
  }
  emit('close')
}

function getFileIcon(file: FileItem): string {
  if (file.is_folder) return 'folder'
  if (file.mime_type?.startsWith('image/')) return 'image'
  if (file.mime_type?.startsWith('video/')) return 'video'
  if (file.mime_type?.startsWith('audio/')) return 'audio'
  if (file.mime_type === 'application/pdf') return 'pdf'
  return 'file'
}

onMounted(() => {
  nextTick(() => inputRef.value?.focus())
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  if (searchTimeout) clearTimeout(searchTimeout)
})
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] bg-black/50"
    @click.self="emit('close')"
  >
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-2xl w-full max-w-xl mx-4 overflow-hidden">
      <!-- Search Input -->
      <div class="flex items-center gap-3 px-4 py-3 border-b dark:border-gray-700">
        <svg class="w-5 h-5 text-gray-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <input
          ref="inputRef"
          v-model="query"
          type="text"
          placeholder="Search files or type a command..."
          class="flex-1 text-lg outline-none placeholder-gray-400 bg-transparent dark:text-white"
        />
        <kbd class="hidden sm:inline-flex px-2 py-1 text-xs font-medium text-gray-500 bg-gray-100 dark:bg-gray-700 dark:text-gray-400 rounded">ESC</kbd>
      </div>

      <!-- Results -->
      <div class="max-h-96 overflow-auto">
        <!-- Loading -->
        <div v-if="loading" class="flex items-center justify-center py-8">
          <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
        </div>

        <!-- Results List -->
        <div v-else-if="allResults.length > 0" class="py-2">
          <!-- Commands Section -->
          <div v-if="filteredCommands.length > 0 && !query.trim()">
            <div class="px-4 py-1 text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">Quick Actions</div>
          </div>

          <template v-for="(item, index) in allResults" :key="item.type + '-' + (item.data.id || index)">
            <!-- Command Item -->
            <button
              v-if="item.type === 'command'"
              @click="selectItem(item)"
              @mouseenter="selectedIndex = index"
              :class="[
                'w-full flex items-center gap-3 px-4 py-2 text-left',
                selectedIndex === index ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300' : 'hover:bg-gray-50 dark:hover:bg-gray-700 dark:text-gray-200'
              ]"
            >
              <!-- Command Icons -->
              <svg v-if="item.data.icon === 'folder-plus'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
              </svg>
              <svg v-else-if="item.data.icon === 'upload'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
              </svg>
              <svg v-else-if="item.data.icon === 'home'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
              </svg>
              <svg v-else-if="item.data.icon === 'star'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
              </svg>
              <svg v-else-if="item.data.icon === 'trash'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
              <svg v-else-if="item.data.icon === 'settings'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              
              <span class="flex-1">{{ item.data.name }}</span>
              <kbd v-if="item.data.shortcut" class="px-2 py-0.5 text-xs text-gray-400 bg-gray-100 dark:bg-gray-700 rounded">
                {{ item.data.shortcut }}
              </kbd>
            </button>

            <!-- File Item -->
            <button
              v-else
              @click="selectItem(item)"
              @mouseenter="selectedIndex = index"
              :class="[
                'w-full flex items-center gap-3 px-4 py-2 text-left',
                selectedIndex === index ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300' : 'hover:bg-gray-50 dark:hover:bg-gray-700 dark:text-gray-200'
              ]"
            >
              <!-- File Icons -->
              <svg v-if="getFileIcon(item.data) === 'folder'" class="w-5 h-5 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
                <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
              </svg>
              <svg v-else-if="getFileIcon(item.data) === 'image'" class="w-5 h-5 text-green-500" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l4-8 3 6 2-4 3 6z" clip-rule="evenodd" />
              </svg>
              <svg v-else-if="getFileIcon(item.data) === 'video'" class="w-5 h-5 text-purple-500" fill="currentColor" viewBox="0 0 20 20">
                <path d="M2 6a2 2 0 012-2h6a2 2 0 012 2v8a2 2 0 01-2 2H4a2 2 0 01-2-2V6zM14.553 7.106A1 1 0 0014 8v4a1 1 0 00.553.894l2 1A1 1 0 0018 13V7a1 1 0 00-1.447-.894l-2 1z" />
              </svg>
              <svg v-else-if="getFileIcon(item.data) === 'pdf'" class="w-5 h-5 text-red-500" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z" clip-rule="evenodd" />
              </svg>
              <svg v-else class="w-5 h-5 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z" clip-rule="evenodd" />
              </svg>

              <div class="flex-1 min-w-0">
                <div class="truncate font-medium">{{ item.data.name }}</div>
              </div>
            </button>
          </template>
        </div>

        <!-- Empty State -->
        <div v-else-if="query.trim()" class="py-8 text-center text-gray-500 dark:text-gray-400">
          No results found for "{{ query }}"
        </div>

        <!-- Initial State -->
        <div v-else class="py-6 text-center text-gray-400 text-sm">
          Start typing to search files or commands
        </div>
      </div>

      <!-- Footer -->
      <div class="flex items-center gap-4 px-4 py-2 border-t dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 text-xs text-gray-500 dark:text-gray-400">
        <span class="flex items-center gap-1">
          <kbd class="px-1.5 py-0.5 bg-white dark:bg-gray-700 border dark:border-gray-600 rounded shadow-sm">↑</kbd>
          <kbd class="px-1.5 py-0.5 bg-white dark:bg-gray-700 border dark:border-gray-600 rounded shadow-sm">↓</kbd>
          to navigate
        </span>
        <span class="flex items-center gap-1">
          <kbd class="px-1.5 py-0.5 bg-white dark:bg-gray-700 border dark:border-gray-600 rounded shadow-sm">↵</kbd>
          to select
        </span>
        <span class="flex items-center gap-1">
          <kbd class="px-1.5 py-0.5 bg-white dark:bg-gray-700 border dark:border-gray-600 rounded shadow-sm">esc</kbd>
          to close
        </span>
      </div>
    </div>
  </div>
</template>
