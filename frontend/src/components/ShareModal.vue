<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import api from '@/api'

const props = defineProps<{
  fileId: string
  fileName: string
}>()

const emit = defineEmits<{
  close: []
}>()

// Tab state
const activeTab = ref<'link' | 'people'>('link')

// Link sharing state
const shareLink = ref('')
const loading = ref(false)
const copied = ref(false)
const expiresIn = ref('7')
const password = ref('')
const allowDownload = ref(true)
const maxDownloads = ref('')

// Analytics state
const analytics = ref<{
  view_count: number
  download_count: number
  max_downloads: number | null
  last_accessed_at: string | null
  created_at: string
} | null>(null)
const loadingAnalytics = ref(false)

// People sharing state
const email = ref('')
const permission = ref('view')
const sharingWithUser = ref(false)
const shareError = ref('')
const shareSuccess = ref('')
const existingShares = ref<any[]>([])
const loadingShares = ref(false)

const fullShareUrl = computed(() => {
  if (!shareLink.value) return ''
  return `${window.location.origin}/s/${shareLink.value}`
})

async function createShare() {
  loading.value = true
  try {
    const response = await api.post(`/files/${props.fileId}/share`, {
      expires_in_days: parseInt(expiresIn.value),
      password: password.value || undefined,
      allow_download: allowDownload.value,
      max_downloads: maxDownloads.value ? parseInt(maxDownloads.value) : undefined
    })
    shareLink.value = response.data.token
    // Fetch analytics after creating share
    await fetchAnalytics()
  } catch (err) {
    console.error('Failed to create share:', err)
  } finally {
    loading.value = false
  }
}

async function fetchAnalytics() {
  loadingAnalytics.value = true
  try {
    const response = await api.get(`/files/${props.fileId}/share/analytics`)
    analytics.value = response.data
  } catch (err) {
    // No analytics yet (share doesn't exist)
    analytics.value = null
  } finally {
    loadingAnalytics.value = false
  }
}

function formatDate(dateStr: string | null) {
  if (!dateStr) return 'Never'
  return new Date(dateStr).toLocaleString()
}

async function copyToClipboard() {
  try {
    await navigator.clipboard.writeText(fullShareUrl.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

async function shareWithUser() {
  if (!email.value.trim()) {
    shareError.value = 'Please enter an email address'
    return
  }

  sharingWithUser.value = true
  shareError.value = ''
  shareSuccess.value = ''

  try {
    const response = await api.post(`/files/${props.fileId}/share/user`, {
      email: email.value.trim(),
      permission: permission.value
    })
    shareSuccess.value = `Shared with ${response.data.shared_email || email.value}`
    email.value = ''
    await fetchExistingShares()
  } catch (err: any) {
    shareError.value = err.response?.data?.error || 'Failed to share'
  } finally {
    sharingWithUser.value = false
  }
}

async function fetchExistingShares() {
  loadingShares.value = true
  try {
    const response = await api.get(`/files/${props.fileId}/shares`)
    existingShares.value = (response.data.shares || []).filter((s: any) => s.shared_with)
  } catch (err) {
    console.error('Failed to fetch shares:', err)
  } finally {
    loadingShares.value = false
  }
}

async function revokeShare(shareId: string) {
  try {
    await api.delete(`/files/shares/${shareId}`)
    existingShares.value = existingShares.value.filter(s => s.id !== shareId)
  } catch (err) {
    console.error('Failed to revoke share:', err)
  }
}

function getPermissionLabel(perm: string) {
  const labels: Record<string, string> = {
    view: 'Can view',
    edit: 'Can edit',
    admin: 'Full access'
  }
  return labels[perm] || perm
}

watch(() => props.fileId, () => {
  if (props.fileId) {
    createShare()
    fetchExistingShares()
    fetchAnalytics()
  }
}, { immediate: true })
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)]"
    @click.self="emit('close')"
  >
    <div class="modal-panel bg-white dark:bg-neutral-800 shadow-xl w-full mx-auto">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b dark:border-neutral-700">
        <h3 class="font-medium dark:text-stone-100">Share "{{ fileName }}"</h3>
        <button
          @click="emit('close')"
          class="p-1 hover:bg-stone-100 dark:hover:bg-neutral-700 rounded"
        >
          <svg class="w-5 h-5 text-stone-500 dark:text-stone-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Tabs -->
      <div class="flex border-b dark:border-neutral-700">
        <button
          @click="activeTab = 'link'"
          :class="[
            'flex-1 py-2 text-sm font-medium text-center border-b-2 transition-colors',
            activeTab === 'link' ? 'border-neutral-800 dark:border-neutral-200 text-stone-800 dark:text-stone-200' : 'border-transparent text-stone-500 hover:text-stone-700 dark:hover:text-stone-300'
          ]"
        >
          Link Sharing
        </button>
        <button
          @click="activeTab = 'people'"
          :class="[
            'flex-1 py-2 text-sm font-medium text-center border-b-2 transition-colors',
            activeTab === 'people' ? 'border-neutral-800 dark:border-neutral-200 text-stone-800 dark:text-stone-200' : 'border-transparent text-stone-500 hover:text-stone-700 dark:hover:text-stone-300'
          ]"
        >
          Share with People
        </button>
      </div>

      <!-- Content -->
      <div class="p-4 flex-1 overflow-auto min-h-0">
        <!-- Link Tab -->
        <div v-if="activeTab === 'link'" class="space-y-4">
          <div v-if="loading" class="flex items-center justify-center py-8">
            <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-neutral-800 dark:border-neutral-200"></div>
          </div>

          <template v-else>
            <div>
              <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">Share link</label>
              <div class="flex gap-2">
                <input
                  type="text"
                  :value="fullShareUrl"
                  readonly
                  class="flex-1 px-3 py-2 border dark:border-neutral-700 rounded-lg bg-stone-50 dark:bg-neutral-700 text-sm dark:text-stone-100"
                />
                <button
                  @click="copyToClipboard"
                  class="px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-900 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 text-sm flex items-center gap-2"
                >
                  <svg v-if="!copied" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                  </svg>
                  <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                  </svg>
                  {{ copied ? 'Copied!' : 'Copy' }}
                </button>
              </div>
            </div>

            <div class="space-y-3 pt-2 border-t dark:border-neutral-700">
              <h4 class="text-sm font-medium text-stone-700 dark:text-stone-300">Options</h4>

              <div class="flex items-center justify-between">
                <label class="text-sm text-stone-600 dark:text-stone-400">Expires in</label>
                <select
                  v-model="expiresIn"
                  @change="createShare"
                  class="px-3 py-1.5 border dark:border-neutral-700 rounded-lg text-sm dark:bg-neutral-700 dark:text-stone-100"
                >
                  <option value="1">1 day</option>
                  <option value="7">7 days</option>
                  <option value="30">30 days</option>
                  <option value="90">90 days</option>
                  <option value="0">Never</option>
                </select>
              </div>

              <div class="flex items-center justify-between">
                <label class="text-sm text-stone-600 dark:text-stone-400">Allow download</label>
                <button
                  @click="allowDownload = !allowDownload; createShare()"
                  :class="[
                    'relative w-11 h-6 rounded-full transition-colors',
                    allowDownload ? 'bg-neutral-800 dark:bg-neutral-200' : 'bg-stone-300 dark:bg-neutral-600'
                  ]"
                >
                  <span
                    :class="[
                      'absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full transition-transform',
                      allowDownload ? 'translate-x-5' : ''
                    ]"
                  />
                </button>
              </div>

              <div>
                <label class="block text-sm text-stone-600 dark:text-stone-400 mb-1">Password (optional)</label>
                <input
                  v-model="password"
                  type="password"
                  placeholder="Set a password"
                  @blur="createShare"
                  class="w-full px-3 py-2 border dark:border-neutral-700 rounded-lg text-sm dark:bg-neutral-700 dark:text-stone-100"
                />
              </div>

              <div>
                <label class="block text-sm text-stone-600 dark:text-stone-400 mb-1">Max downloads (optional)</label>
                <input
                  v-model="maxDownloads"
                  type="number"
                  min="1"
                  placeholder="Unlimited"
                  @blur="createShare"
                  class="w-full px-3 py-2 border dark:border-neutral-700 rounded-lg text-sm dark:bg-neutral-700 dark:text-stone-100"
                />
              </div>
            </div>

            <!-- Analytics Section -->
            <div v-if="analytics" class="space-y-2 pt-3 border-t dark:border-neutral-700">
              <h4 class="text-sm font-medium text-stone-700 dark:text-stone-300 flex items-center gap-2">
                <svg class="w-4 h-4 text-stone-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
                Link Analytics
              </h4>
              <div class="grid grid-cols-2 gap-3">
                <div class="bg-stone-50 dark:bg-neutral-700 rounded-lg p-3 text-center">
                  <div class="text-2xl font-semibold text-stone-800 dark:text-stone-200">{{ analytics.view_count }}</div>
                  <div class="text-xs text-stone-500 dark:text-stone-400">Views</div>
                </div>
                <div class="bg-stone-50 dark:bg-neutral-700 rounded-lg p-3 text-center">
                  <div class="text-2xl font-semibold text-green-600">{{ analytics.download_count }}</div>
                  <div class="text-xs text-stone-500 dark:text-stone-400">
                    Downloads
                    <span v-if="analytics.max_downloads" class="text-stone-400">
                      / {{ analytics.max_downloads }}
                    </span>
                  </div>
                </div>
              </div>
              <div class="text-xs text-stone-500 dark:text-stone-400 text-center">
                Last accessed: {{ formatDate(analytics.last_accessed_at) }}
              </div>
            </div>
          </template>
        </div>

        <!-- People Tab -->
        <div v-if="activeTab === 'people'" class="space-y-4">
          <!-- Add person form -->
          <div class="flex gap-2">
            <input
              v-model="email"
              type="email"
              placeholder="Enter email address"
              @keyup.enter="shareWithUser"
              class="flex-1 px-3 py-2 border dark:border-neutral-700 rounded-lg text-sm dark:bg-neutral-700 dark:text-stone-100"
            />
            <select v-model="permission" class="px-3 py-2 border dark:border-neutral-700 rounded-lg text-sm dark:bg-neutral-700 dark:text-stone-100">
              <option value="view">View</option>
              <option value="edit">Edit</option>
              <option value="admin">Full access</option>
            </select>
            <button
              @click="shareWithUser"
              :disabled="sharingWithUser"
              class="px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-900 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 text-sm disabled:opacity-50"
            >
              {{ sharingWithUser ? '...' : 'Share' }}
            </button>
          </div>

          <!-- Error/Success messages -->
          <div v-if="shareError" class="text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 px-3 py-2 rounded">
            {{ shareError }}
          </div>
          <div v-if="shareSuccess" class="text-sm text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20 px-3 py-2 rounded">
            {{ shareSuccess }}
          </div>

          <!-- Existing shares -->
          <div v-if="loadingShares" class="flex justify-center py-4">
            <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-neutral-800 dark:border-neutral-200"></div>
          </div>

          <div v-else-if="existingShares.length > 0" class="space-y-2">
            <h4 class="text-sm font-medium text-stone-700 dark:text-stone-300">Shared with</h4>
            <div
              v-for="share in existingShares"
              :key="share.id"
              class="flex items-center justify-between p-2 bg-stone-50 dark:bg-neutral-700 rounded-lg"
            >
              <div class="flex items-center gap-2">
                <div class="w-8 h-8 bg-stone-100 dark:bg-neutral-700/30 rounded-full flex items-center justify-center text-sm font-medium text-stone-800 dark:text-stone-200">
                  {{ (share.shared_email || 'U')[0].toUpperCase() }}
                </div>
                <div>
                  <div class="text-sm font-medium dark:text-stone-100">{{ share.shared_name || share.shared_email || 'User' }}</div>
                  <div class="text-xs text-stone-500 dark:text-stone-400">{{ getPermissionLabel(share.permission) }}</div>
                </div>
              </div>
              <button
                @click="revokeShare(share.id)"
                class="text-sm text-red-600 hover:text-red-700"
              >
                Remove
              </button>
            </div>
          </div>

          <div v-else class="text-center py-4 text-sm text-stone-500 dark:text-stone-400">
            Not shared with anyone yet
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-4 py-3 border-t dark:border-neutral-700 bg-stone-50 dark:bg-neutral-800/50 rounded-b-lg">
        <button
          @click="emit('close')"
          class="w-full px-4 py-2 bg-stone-200 dark:bg-neutral-700 text-stone-700 dark:text-stone-300 rounded-lg hover:bg-stone-300 dark:hover:bg-neutral-600 text-sm"
        >
          Done
        </button>
      </div>
    </div>
  </div>
</template>
