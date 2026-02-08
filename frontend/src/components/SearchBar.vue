<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useFilesStore } from '@/stores/files'

const router = useRouter()
const filesStore = useFilesStore()

const query = ref('')
const showResults = ref(false)
const results = ref<any[]>([])
const loading = ref(false)

let searchTimeout: number | null = null

async function handleSearch() {
  if (!query.value.trim()) {
    results.value = []
    showResults.value = false
    return
  }

  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }

  searchTimeout = setTimeout(async () => {
    loading.value = true
    showResults.value = true
    try {
      results.value = await filesStore.search(query.value)
    } finally {
      loading.value = false
    }
  }, 300) as unknown as number
}

function selectResult(file: any) {
  showResults.value = false
  query.value = ''
  if (file.is_folder) {
    router.push({ name: 'folder', params: { id: file.id } })
  } else {
    // Navigate to parent folder
    if (file.parent_id) {
      router.push({ name: 'folder', params: { id: file.parent_id } })
    } else {
      router.push({ name: 'files' })
    }
  }
}

function handleBlur() {
  setTimeout(() => {
    showResults.value = false
  }, 200)
}
</script>

<template>
  <div class="relative">
    <div class="relative">
      <svg
        class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input
        v-model="query"
        @input="handleSearch"
        @focus="query && (showResults = true)"
        @blur="handleBlur"
        type="text"
        placeholder="Search files... (Ctrl+K)"
        class="w-full pl-10 pr-4 py-2 bg-gray-100 dark:bg-gray-700 border-0 rounded-lg focus:bg-white dark:focus:bg-gray-800 focus:ring-2 focus:ring-blue-500 text-sm dark:text-white dark:placeholder-gray-400"
      />
    </div>

    <!-- Search results dropdown -->
    <div
      v-if="showResults"
      class="absolute top-full left-0 right-0 mt-1 bg-white dark:bg-gray-800 rounded-lg shadow-lg border dark:border-gray-700 max-h-96 overflow-auto z-50"
    >
      <div v-if="loading" class="p-4 text-center text-gray-500 dark:text-gray-400">
        Searching...
      </div>
      <div v-else-if="results.length === 0" class="p-4 text-center text-gray-500 dark:text-gray-400">
        No results found
      </div>
      <div v-else>
        <button
          v-for="file in results"
          :key="file.id"
          @click="selectResult(file)"
          class="w-full flex items-center gap-3 px-4 py-2 hover:bg-gray-50 dark:hover:bg-gray-700 text-left dark:text-white"
        >
          <svg v-if="file.is_folder" class="w-5 h-5 text-blue-500" fill="currentColor" viewBox="0 0 20 20">
            <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
          </svg>
          <svg v-else class="w-5 h-5 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M4 4a2 2 0 012-2h4.586A2 2 0 0112 2.586L15.414 6A2 2 0 0116 7.414V16a2 2 0 01-2 2H6a2 2 0 01-2-2V4z" clip-rule="evenodd" />
          </svg>
          <span class="truncate">{{ file.name }}</span>
        </button>
      </div>
    </div>
  </div>
</template>
