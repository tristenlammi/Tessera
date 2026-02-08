<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useEmailStore } from '@/stores/email'

const emit = defineEmits<{
  close: []
  saved: []
}>()

const emailStore = useEmailStore()

// View mode: 'overview' | 'add' | 'edit'
const viewMode = ref<'overview' | 'add' | 'edit'>('overview')

// Sync confirmation modal
const showSyncConfirmation = ref(false)

const form = reactive({
  name: '',
  email_address: '',
  imap_host: '',
  imap_port: 993,
  imap_username: '',
  imap_password: '',
  imap_use_tls: true,
  smtp_host: '',
  smtp_port: 587,
  smtp_username: '',
  smtp_password: '',
  smtp_use_tls: true
})

const loading = ref(false)
const error = ref<string | null>(null)
const showAdvanced = ref(false)

// Common email provider presets
const presets = [
  {
    name: 'Gmail',
    imap_host: 'imap.gmail.com',
    imap_port: 993,
    smtp_host: 'smtp.gmail.com',
    smtp_port: 587
  },
  {
    name: 'Outlook/Hotmail',
    imap_host: 'outlook.office365.com',
    imap_port: 993,
    smtp_host: 'smtp.office365.com',
    smtp_port: 587
  },
  {
    name: 'Yahoo',
    imap_host: 'imap.mail.yahoo.com',
    imap_port: 993,
    smtp_host: 'smtp.mail.yahoo.com',
    smtp_port: 587
  },
  {
    name: 'iCloud',
    imap_host: 'imap.mail.me.com',
    imap_port: 993,
    smtp_host: 'smtp.mail.me.com',
    smtp_port: 587
  }
]

function applyPreset(preset: typeof presets[0]) {
  form.imap_host = preset.imap_host
  form.imap_port = preset.imap_port
  form.smtp_host = preset.smtp_host
  form.smtp_port = preset.smtp_port
  form.imap_use_tls = true
  form.smtp_use_tls = true
}

// Sync SMTP username with IMAP username by default
function syncUsernames() {
  if (!form.smtp_username) {
    form.smtp_username = form.imap_username
  }
}

// Sync SMTP password with IMAP password by default
function syncPasswords() {
  if (!form.smtp_password) {
    form.smtp_password = form.imap_password
  }
}

// Format date for display
function formatDate(dateString: string | null) {
  if (!dateString) return 'Never'
  const date = new Date(dateString)
  return date.toLocaleString()
}

// Switch to add account view
function showAddAccount() {
  resetForm()
  viewMode.value = 'add'
}

// Reset form
function resetForm() {
  form.name = ''
  form.email_address = ''
  form.imap_host = ''
  form.imap_port = 993
  form.imap_username = ''
  form.imap_password = ''
  form.imap_use_tls = true
  form.smtp_host = ''
  form.smtp_port = 587
  form.smtp_username = ''
  form.smtp_password = ''
  form.smtp_use_tls = true
}

// Go back to overview
function goBack() {
  viewMode.value = 'overview'
  error.value = null
}

// Handle sync with confirmation
function requestSync() {
  showSyncConfirmation.value = true
}

async function confirmSync() {
  showSyncConfirmation.value = false
  await emailStore.syncAccount()
}

function cancelSync() {
  showSyncConfirmation.value = false
}

async function handleSubmit() {
  error.value = null
  
  // Basic validation
  if (!form.name.trim()) {
    error.value = 'Account name is required'
    return
  }
  if (!form.email_address.trim()) {
    error.value = 'Email address is required'
    return
  }
  if (!form.imap_host.trim()) {
    error.value = 'IMAP server is required'
    return
  }
  if (!form.smtp_host.trim()) {
    error.value = 'SMTP server is required'
    return
  }
  if (!form.imap_username.trim()) {
    error.value = 'Username is required'
    return
  }
  if (!form.imap_password) {
    error.value = 'Password is required'
    return
  }

  // Default SMTP credentials to IMAP if not set
  if (!form.smtp_username) {
    form.smtp_username = form.imap_username
  }
  if (!form.smtp_password) {
    form.smtp_password = form.imap_password
  }

  loading.value = true
  try {
    await emailStore.createAccount(form)
    emit('saved')
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to add account'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
      <!-- Header -->
      <div class="flex items-center justify-between p-4 border-b dark:border-gray-700">
        <div class="flex items-center gap-2">
          <button
            v-if="viewMode !== 'overview'"
            @click="goBack"
            class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
          </button>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ viewMode === 'overview' ? 'Settings' : viewMode === 'add' ? 'Add Email Account' : 'Edit Account' }}
          </h2>
        </div>
        <button
          @click="emit('close')"
          class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Overview View -->
      <div v-if="viewMode === 'overview'" class="p-4 space-y-6">
        <!-- Connected Accounts Section -->
        <div>
          <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">Connected Accounts</h3>
          
          <div v-if="emailStore.accounts.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
            <svg class="w-12 h-12 mx-auto mb-3 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
            <p class="mb-2">No email accounts connected</p>
            <button
              @click="showAddAccount"
              class="text-blue-600 dark:text-blue-400 hover:underline text-sm font-medium"
            >
              Add your first account
            </button>
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="account in emailStore.accounts"
              :key="account.id"
              class="p-4 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
            >
              <div class="flex items-start justify-between">
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-2">
                    <h4 class="font-medium text-gray-900 dark:text-white truncate">{{ account.name }}</h4>
                    <span v-if="account.is_default" class="px-2 py-0.5 text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-full">Default</span>
                  </div>
                  <p class="text-sm text-gray-500 dark:text-gray-400 truncate">{{ account.email_address }}</p>
                  <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-400 dark:text-gray-500">
                    <span>IMAP: {{ account.imap_host }}:{{ account.imap_port }}</span>
                    <span>SMTP: {{ account.smtp_host }}:{{ account.smtp_port }}</span>
                  </div>
                  <div class="mt-1 text-xs text-gray-400 dark:text-gray-500">
                    <span v-if="account.last_sync_at">Last synced: {{ formatDate(account.last_sync_at) }}</span>
                    <span v-else>Never synced</span>
                  </div>
                  <div v-if="account.sync_error" class="mt-1 text-xs text-red-500 dark:text-red-400">
                    Error: {{ account.sync_error }}
                  </div>
                </div>
              </div>
            </div>
            
            <button
              @click="showAddAccount"
              class="w-full flex items-center justify-center gap-2 px-4 py-2 text-sm text-blue-600 dark:text-blue-400 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg transition-colors"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
              Add Another Account
            </button>
          </div>
        </div>

        <!-- Sync Section -->
        <div v-if="emailStore.accounts.length > 0" class="border-t dark:border-gray-700 pt-6">
          <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">Email Sync</h3>
          
          <!-- Sync progress (if syncing) -->
          <div v-if="emailStore.syncing" class="mb-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
            <div class="flex items-center justify-between text-sm text-blue-700 dark:text-blue-300 mb-2">
              <span class="truncate">{{ emailStore.syncProgress.message || 'Syncing...' }}</span>
              <span v-if="emailStore.syncProgress.totalFolders > 0">
                {{ emailStore.syncProgress.completedFolders }}/{{ emailStore.syncProgress.totalFolders }}
              </span>
            </div>
            <div class="w-full bg-blue-200 dark:bg-blue-800 rounded-full h-2">
              <div 
                class="bg-blue-600 h-2 rounded-full transition-all duration-300"
                :style="{ width: emailStore.syncProgress.totalFolders > 0 ? `${(emailStore.syncProgress.completedFolders / emailStore.syncProgress.totalFolders) * 100}%` : '0%' }"
              ></div>
            </div>
          </div>

          <button
            @click="requestSync"
            :disabled="emailStore.syncing"
            class="w-full flex items-center justify-center gap-2 px-4 py-2.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <svg
              :class="['w-4 h-4', emailStore.syncing && 'animate-spin']"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            {{ emailStore.syncing ? 'Syncing...' : 'Sync Emails Now' }}
          </button>
          <p class="mt-2 text-xs text-gray-500 dark:text-gray-400 text-center">
            This will sync all emails from your connected accounts. This may take several minutes.
          </p>
        </div>
      </div>

      <!-- Add Account Form -->
      <form v-else @submit.prevent="handleSubmit" class="p-4 space-y-4">
        <!-- Error message -->
        <div v-if="error" class="p-3 bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 rounded-lg text-sm">
          {{ error }}
        </div>

        <!-- Quick setup presets -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Quick Setup
          </label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="preset in presets"
              :key="preset.name"
              type="button"
              @click="applyPreset(preset)"
              class="px-3 py-1.5 text-sm bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded-lg transition-colors"
            >
              {{ preset.name }}
            </button>
          </div>
        </div>

        <!-- Account name -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Account Name
          </label>
          <input
            v-model="form.name"
            type="text"
            placeholder="e.g., Personal, Work"
            class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700"
          />
        </div>

        <!-- Email address -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Email Address
          </label>
          <input
            v-model="form.email_address"
            type="email"
            placeholder="you@example.com"
            class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700"
          />
        </div>

        <!-- Username and Password -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Username
            </label>
            <input
              v-model="form.imap_username"
              @blur="syncUsernames"
              type="text"
              placeholder="Username or email"
              class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Password
            </label>
            <input
              v-model="form.imap_password"
              @blur="syncPasswords"
              type="password"
              placeholder="App password"
              class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700"
            />
          </div>
        </div>

        <p class="text-xs text-gray-500 dark:text-gray-400">
          For Gmail, Outlook, and other providers with 2FA, use an app-specific password.
        </p>

        <!-- IMAP Settings -->
        <div class="grid grid-cols-3 gap-4">
          <div class="col-span-2">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              IMAP Server
            </label>
            <input
              v-model="form.imap_host"
              type="text"
              placeholder="imap.example.com"
              class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Port
            </label>
            <input
              v-model.number="form.imap_port"
              type="number"
              class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700"
            />
          </div>
        </div>

        <!-- SMTP Settings -->
        <div class="grid grid-cols-3 gap-4">
          <div class="col-span-2">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              SMTP Server
            </label>
            <input
              v-model="form.smtp_host"
              type="text"
              placeholder="smtp.example.com"
              class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Port
            </label>
            <input
              v-model.number="form.smtp_port"
              type="number"
              class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700"
            />
          </div>
        </div>

        <!-- Advanced settings toggle -->
        <button
          type="button"
          @click="showAdvanced = !showAdvanced"
          class="flex items-center gap-2 text-sm text-blue-600 dark:text-blue-400 hover:underline"
        >
          <svg
            :class="['w-4 h-4 transition-transform', showAdvanced && 'rotate-90']"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
          </svg>
          Advanced Settings
        </button>

        <!-- Advanced settings -->
        <div v-if="showAdvanced" class="space-y-4 pt-2 border-t dark:border-gray-700">
          <!-- SMTP username/password (if different from IMAP) -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                SMTP Username
              </label>
              <input
                v-model="form.smtp_username"
                type="text"
                placeholder="Same as IMAP"
                class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                SMTP Password
              </label>
              <input
                v-model="form.smtp_password"
                type="password"
                placeholder="Same as IMAP"
                class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700"
              />
            </div>
          </div>

          <!-- TLS options -->
          <div class="flex gap-6">
            <label class="flex items-center gap-2">
              <input
                v-model="form.imap_use_tls"
                type="checkbox"
                class="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">IMAP Use TLS/SSL</span>
            </label>
            <label class="flex items-center gap-2">
              <input
                v-model="form.smtp_use_tls"
                type="checkbox"
                class="w-4 h-4 text-blue-600 rounded focus:ring-blue-500"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">SMTP Use STARTTLS</span>
            </label>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex justify-end gap-3 pt-4">
          <button
            type="button"
            @click="goBack"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            :disabled="loading"
            class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
          >
            <svg v-if="loading" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
            {{ loading ? 'Adding...' : 'Add Account' }}
          </button>
        </div>
      </form>
    </div>

    <!-- Sync Confirmation Modal -->
    <div v-if="showSyncConfirmation" class="fixed inset-0 bg-black/50 flex items-center justify-center z-60">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md p-6 m-4">
        <div class="flex items-center gap-3 mb-4">
          <div class="p-3 bg-blue-100 dark:bg-blue-900/30 rounded-full">
            <svg class="w-6 h-6 text-blue-600 dark:text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Sync Emails?</h3>
        </div>
        <p class="text-gray-600 dark:text-gray-400 mb-6">
          This will sync all emails from your connected accounts. Depending on the number of emails, this process may take several minutes to complete. Are you sure you want to continue?
        </p>
        <div class="flex justify-end gap-3">
          <button
            @click="cancelSync"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            Cancel
          </button>
          <button
            @click="confirmSync"
            class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors"
          >
            Yes, Sync Now
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
