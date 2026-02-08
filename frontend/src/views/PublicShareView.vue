<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import api from '@/api'

const route = useRoute()
const token = computed(() => route.params.token as string)

const loading = ref(true)
const error = ref('')
const shareInfo = ref<{
  file_name: string
  file_size: number
  is_folder: boolean
  allow_download: boolean
  has_password: boolean
  max_downloads: number | null
  downloads_left: number | null
} | null>(null)

const password = ref('')
const downloading = ref(false)
const passwordError = ref('')

async function fetchShareInfo() {
  loading.value = true
  error.value = ''
  try {
    const response = await api.get(`/share/${token.value}`)
    shareInfo.value = response.data
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Share not found or expired'
  } finally {
    loading.value = false
  }
}

async function downloadFile() {
  if (!shareInfo.value?.allow_download) return
  
  downloading.value = true
  passwordError.value = ''
  
  try {
    const params = new URLSearchParams()
    if (password.value) {
      params.set('password', password.value)
    }
    
    // Open download in new window
    const url = `/api/share/${token.value}/download${password.value ? `?password=${encodeURIComponent(password.value)}` : ''}`
    window.open(url, '_blank')
    
    // Refresh share info to update download count
    setTimeout(() => fetchShareInfo(), 1000)
  } catch (err: any) {
    passwordError.value = err.response?.data?.error || 'Download failed'
  } finally {
    downloading.value = false
  }
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

onMounted(fetchShareInfo)
</script>

<template>
  <div class="min-h-screen bg-gray-100 dark:bg-gray-900 flex items-center justify-center p-4">
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-lg max-w-md w-full p-6">
      <!-- Loading -->
      <div v-if="loading" class="flex flex-col items-center py-8">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mb-4"></div>
        <p class="text-gray-500 dark:text-gray-400">Loading share...</p>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="text-center py-8">
        <div class="w-16 h-16 mx-auto mb-4 text-red-400">
          <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h2 class="text-xl font-semibold text-gray-800 dark:text-white mb-2">Share Unavailable</h2>
        <p class="text-gray-500 dark:text-gray-400">{{ error }}</p>
      </div>

      <!-- Share Info -->
      <div v-else-if="shareInfo" class="space-y-6">
        <!-- Header -->
        <div class="text-center">
            <div class="w-16 h-16 mx-auto mb-4 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
            <svg v-if="!shareInfo.is_folder" class="w-8 h-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
            </svg>
            <svg v-else class="w-8 h-8 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
          </div>
          <h1 class="text-xl font-semibold text-gray-800 dark:text-white break-all">{{ shareInfo.file_name }}</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{{ formatSize(shareInfo.file_size) }}</p>
        </div>

        <!-- Download Limit Warning -->
        <div v-if="shareInfo.downloads_left !== null && shareInfo.downloads_left <= 3" 
             class="bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-700 rounded-lg p-3 text-center">
          <p class="text-sm text-amber-700 dark:text-amber-300">
            <span class="font-medium">{{ shareInfo.downloads_left }}</span> 
            download{{ shareInfo.downloads_left !== 1 ? 's' : '' }} remaining
          </p>
        </div>

        <!-- Password Input -->
        <div v-if="shareInfo.has_password" class="space-y-2">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
            This file is password protected
          </label>
          <input
            v-model="password"
            type="password"
            placeholder="Enter password"
            class="w-full px-4 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
            @keyup.enter="downloadFile"
          />
          <p v-if="passwordError" class="text-sm text-red-600 dark:text-red-400">{{ passwordError }}</p>
        </div>

        <!-- Download Button -->
        <div v-if="shareInfo.allow_download">
          <button
            @click="downloadFile"
            :disabled="downloading || (shareInfo.downloads_left !== null && shareInfo.downloads_left <= 0)"
            class="w-full py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2 transition-colors"
          >
            <svg v-if="!downloading" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            <div v-else class="animate-spin rounded-full h-5 w-5 border-b-2 border-white"></div>
            {{ shareInfo.downloads_left !== null && shareInfo.downloads_left <= 0 ? 'Download Limit Reached' : 'Download' }}
          </button>
        </div>

        <!-- View Only Notice -->
        <div v-else class="text-center py-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg">
          <svg class="w-8 h-8 mx-auto text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
          </svg>
          <p class="text-gray-600 dark:text-gray-300">This file is view-only</p>
          <p class="text-sm text-gray-500 dark:text-gray-400">Downloads are not allowed</p>
        </div>

        <!-- Branding -->
        <div class="text-center pt-4 border-t dark:border-gray-700">
          <p class="text-xs text-gray-400">
            Shared via <span class="font-medium">Tessera</span>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
