<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useFilesStore } from '@/stores/files'
import { useWebSocket } from '@/composables/useWebSocket'
import { useModulesStore } from '@/stores/modules'
import { useAuthStore } from '@/stores/auth'
import { useTheme } from '@/composables/useTheme'

const route = useRoute()
const filesStore = useFilesStore()
const modulesStore = useModulesStore()
const authStore = useAuthStore()
const { isConnected } = useWebSocket()
const { theme, toggleTheme } = useTheme()

const coreNavItems = [
  { name: 'My Files', to: '/', icon: 'folder' },
  { name: 'Shared with Me', to: '/shared', icon: 'users' },
  { name: 'Recent', to: '/recent', icon: 'clock' },
  { name: 'Starred', to: '/starred', icon: 'star' },
  { name: 'Trash', to: '/trash', icon: 'trash' }
]

const moduleNavItems = computed(() => {
  const items = []
  // Documents are now integrated into My Files, not shown as separate nav item
  if (modulesStore.isModuleEnabled('tasks')) {
    items.push({ name: 'Tasks', to: '/tasks', icon: 'clipboard-list' })
  }
  if (modulesStore.isModuleEnabled('calendar')) {
    items.push({ name: 'Calendar', to: '/calendar', icon: 'calendar' })
  }
  if (modulesStore.isModuleEnabled('contacts')) {
    items.push({ name: 'Contacts', to: '/contacts', icon: 'user-group' })
  }
  if (modulesStore.isModuleEnabled('email')) {
    items.push({ name: 'Email', to: '/email', icon: 'envelope' })
  }
  return items
})

const isAdmin = computed(() => authStore.user?.role === 'admin')

function isActive(to: string): boolean {
  if (to === '/') {
    return route.path === '/' || route.path.startsWith('/folder')
  }
  return route.path === to || route.path.startsWith(to + '/')
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

onMounted(() => {
  modulesStore.fetchModuleSettings()
})
</script>

<template>
  <aside class="w-64 bg-white dark:bg-gray-800 border-r dark:border-gray-700 flex flex-col">
    <!-- Logo -->
    <div class="h-14 flex items-center px-4 border-b dark:border-gray-700">
      <h1 class="text-xl font-bold text-blue-600">Tessera</h1>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 p-2 space-y-1 overflow-y-auto">
      <!-- Core navigation -->
      <RouterLink
        v-for="item in coreNavItems"
        :key="item.to"
        :to="item.to"
        :class="[
          'flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
          isActive(item.to)
            ? 'bg-blue-50 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
            : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
        ]"
      >
        <svg v-if="item.icon === 'folder'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <svg v-else-if="item.icon === 'users'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
        </svg>
        <svg v-else-if="item.icon === 'clock'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <svg v-else-if="item.icon === 'star'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
        </svg>
        <svg v-else-if="item.icon === 'trash'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
        {{ item.name }}
      </RouterLink>

      <!-- Module navigation (only if modules enabled) -->
      <template v-if="moduleNavItems.length > 0">
        <div class="my-3 px-3"><div class="h-px bg-gray-200 dark:bg-gray-700"></div></div>
        <RouterLink
          v-for="item in moduleNavItems"
          :key="item.to"
          :to="item.to"
          :class="[
            'flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
            isActive(item.to)
              ? 'bg-blue-50 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
          ]"
        >
          <svg v-if="item.icon === 'document-text'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <svg v-else-if="item.icon === 'clipboard-list'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" />
          </svg>
          <svg v-else-if="item.icon === 'calendar'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <svg v-else-if="item.icon === 'user-group'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          <svg v-else-if="item.icon === 'envelope'" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
          {{ item.name }}
        </RouterLink>
      </template>
    </nav>

    <!-- Bottom section -->
    <div class="mt-auto">
      <!-- Settings link (for admins) -->
      <div v-if="isAdmin" class="px-3 py-2">
        <RouterLink
          to="/admin"
          :class="[
            'flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors',
            isActive('/admin')
              ? 'bg-blue-50 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
              : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
          ]"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          Settings
        </RouterLink>
      </div>

      <!-- Storage bar -->
      <div v-if="filesStore.storageStats" class="p-4 border-t dark:border-gray-700">
        <div class="text-xs text-gray-500 dark:text-gray-400 mb-2">Storage</div>
        <div class="h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
          <div
            class="h-full bg-blue-600 transition-all"
            :style="{ width: `${Math.min(filesStore.storageStats.used_pct, 100)}%` }"
          ></div>
        </div>
        <div class="text-xs text-gray-500 dark:text-gray-400 mt-1">
          {{ formatBytes(filesStore.storageStats.used) }} of {{ formatBytes(filesStore.storageStats.limit) }}
        </div>
      </div>

      <!-- Theme toggle and connection status -->
      <div class="px-4 py-2 border-t dark:border-gray-700 flex items-center justify-between">
        <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
          <span
            class="w-2 h-2 rounded-full"
            :class="isConnected ? 'bg-green-500' : 'bg-red-500'"
          ></span>
          {{ isConnected ? 'Connected' : 'Disconnected' }}
        </div>
        <button
          @click="toggleTheme"
          class="p-1.5 rounded-lg text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
          :title="theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode'"
        >
          <!-- Sun icon (show when dark mode) -->
          <svg v-if="theme === 'dark'" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
          </svg>
          <!-- Moon icon (show when light mode) -->
          <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
          </svg>
        </button>
      </div>
    </div>
  </aside>
</template>
