<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useFilesStore } from '@/stores/files'
import Sidebar from '@/components/Sidebar.vue'

const router = useRouter()
const authStore = useAuthStore()
const filesStore = useFilesStore()

const sidebarOpen = ref(true)
const userMenuOpen = ref(false)

onMounted(() => {
  filesStore.fetchStorageStats()
})

async function handleLogout() {
  userMenuOpen.value = false
  await authStore.logout()
  router.push({ name: 'login' })
}

function goToSettings() {
  userMenuOpen.value = false
  router.push({ name: 'settings' })
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}
</script>

<template>
  <div class="h-screen flex bg-gray-50 dark:bg-gray-900">
    <!-- Sidebar -->
    <Sidebar v-if="sidebarOpen" />

    <!-- Main content -->
    <div class="flex-1 flex flex-col min-w-0">
      <!-- Top bar -->
      <header class="h-14 flex items-center justify-between px-4 bg-white dark:bg-gray-800 border-b dark:border-gray-700">
        <div class="flex items-center gap-4">
          <button
            @click="sidebarOpen = !sidebarOpen"
            class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
        </div>

        <div class="flex items-center gap-4">
          <!-- Storage indicator -->
          <div v-if="filesStore.storageStats" class="text-sm text-gray-600 dark:text-gray-400">
            {{ formatBytes(filesStore.storageStats.used) }} of {{ formatBytes(filesStore.storageStats.limit) }} used
          </div>

          <!-- User menu -->
          <div class="relative">
            <button
              class="flex items-center gap-2 p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
              @click="userMenuOpen = !userMenuOpen"
            >
              <div class="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center text-white text-sm font-medium">
                {{ authStore.user?.name?.charAt(0).toUpperCase() }}
              </div>
              <svg class="w-4 h-4 text-gray-500 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>
            
            <!-- Dropdown menu -->
            <div
              v-if="userMenuOpen"
              class="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border dark:border-gray-700 py-1 z-50"
              @click.stop
            >
              <div class="px-4 py-2 border-b dark:border-gray-700">
                <p class="text-sm font-medium text-gray-900 dark:text-white">{{ authStore.user?.name }}</p>
                <p class="text-xs text-gray-500 dark:text-gray-400">{{ authStore.user?.email }}</p>
              </div>
              <button
                @click="goToSettings"
                class="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                </svg>
                Settings
              </button>
              <button
                @click="handleLogout"
                class="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
                </svg>
                Sign out
              </button>
            </div>
            
            <!-- Click outside to close -->
            <div
              v-if="userMenuOpen"
              class="fixed inset-0 z-40"
              @click="userMenuOpen = false"
            ></div>
          </div>
        </div>
      </header>

      <!-- Page content -->
      <main class="flex-1 overflow-hidden">
        <RouterView />
      </main>
    </div>
  </div>
</template>
