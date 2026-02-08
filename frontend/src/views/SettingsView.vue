<template>
  <div class="max-w-2xl mx-auto px-4 py-8">
    <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-8">Settings</h1>

    <!-- General Settings -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6 mb-6">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">General</h2>
      
      <!-- Timezone -->
      <div class="mb-6">
        <label for="timezone" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Timezone
        </label>
        <p class="text-sm text-gray-500 dark:text-gray-400 mb-2">
          Choose your timezone to display dates and times correctly.
        </p>
        <select
          id="timezone"
          v-model="selectedTimezone"
          @change="saveTimezone"
          class="w-full border border-gray-300 dark:border-gray-600 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
        >
          <optgroup v-for="group in groupedTimezones" :key="group.label" :label="group.label">
            <option v-for="tz in group.timezones" :key="tz.value" :value="tz.value">
              {{ tz.label }}
            </option>
          </optgroup>
        </select>
      </div>
    </div>

    <!-- Two-Factor Authentication -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6 mb-6">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Two-Factor Authentication</h2>
      
      <div v-if="loadingTOTP" class="flex items-center justify-center py-8">
        <div class="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600"></div>
      </div>

      <!-- 2FA Enabled State -->
      <div v-else-if="totpStatus?.enabled" class="space-y-4">
        <div class="flex items-center gap-2 text-green-600">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
          <span class="font-medium">2FA is enabled</span>
        </div>
        
        <p class="text-sm text-gray-600 dark:text-gray-400">
          Backup codes remaining: {{ totpStatus.backup_codes_remaining }} / 10
        </p>
        
        <div class="flex gap-3 pt-2">
          <button
            @click="showRegenerateModal = true"
            class="px-4 py-2 text-sm font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100"
          >
            Regenerate backup codes
          </button>
          <button
            @click="showDisableModal = true"
            class="px-4 py-2 text-sm font-medium text-red-600 bg-red-50 rounded-lg hover:bg-red-100"
          >
            Disable 2FA
          </button>
        </div>
      </div>

      <!-- 2FA Disabled State -->
      <div v-else class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          Add an extra layer of security to your account by enabling two-factor authentication.
        </p>
        <button
          @click="startTOTPSetup"
          :disabled="settingUp2FA"
          class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50"
        >
          {{ settingUp2FA ? 'Loading...' : 'Enable 2FA' }}
        </button>
      </div>
    </div>

    <!-- Account Info -->
    <div class="bg-white dark:bg-gray-800 rounded-lg shadow p-6">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Account</h2>
      
      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Email</label>
          <p class="mt-1 text-sm text-gray-900 dark:text-white">{{ authStore.user?.email }}</p>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Name</label>
          <p class="mt-1 text-sm text-gray-900 dark:text-white">{{ authStore.user?.name }}</p>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Storage Used</label>
          <p class="mt-1 text-sm text-gray-900 dark:text-white">
            {{ formatBytes(authStore.user?.storage_used || 0) }} of {{ formatBytes(authStore.user?.storage_limit || 0) }}
          </p>
        </div>
      </div>
    </div>

    <!-- 2FA Setup Modal -->
    <div v-if="showSetupModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-md w-full mx-4 p-6">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Set Up Two-Factor Authentication</h3>
        
        <!-- Step 1: Show QR/Secret -->
        <div v-if="setupStep === 1" class="space-y-4">
          <p class="text-sm text-gray-600 dark:text-gray-400">
            Scan the QR code with your authenticator app (Google Authenticator, Authy, Bitwarden, etc.)
          </p>
          
          <!-- QR Code Placeholder - In production, generate QR from qrcodeUrl -->
          <div class="bg-gray-100 dark:bg-gray-700 p-4 rounded-lg text-center">
            <div class="bg-white dark:bg-gray-600 p-4 inline-block rounded-lg">
              <!-- QR Code would be rendered here -->
              <img 
                :src="`https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(totpSetup?.qrcode_url || '')}`"
                alt="QR Code"
                class="w-48 h-48"
              />
            </div>
          </div>
          
          <div>
            <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Or enter this key manually in your app:
            </p>
            <div class="bg-gray-100 dark:bg-gray-700 p-3 rounded-lg font-mono text-sm text-center select-all break-all dark:text-gray-200">
              {{ totpSetup?.secret }}
            </div>
          </div>
          
          <div class="flex justify-end gap-3 pt-4">
            <button
              @click="cancelSetup"
              class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600"
            >
              Cancel
            </button>
            <button
              @click="setupStep = 2"
              class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700"
            >
              Next
            </button>
          </div>
        </div>
        
        <!-- Step 2: Verify Code -->
        <div v-else-if="setupStep === 2" class="space-y-4">
          <p class="text-sm text-gray-600 dark:text-gray-400">
            Enter the 6-digit code from your authenticator app to verify setup.
          </p>
          
          <div v-if="setupError" class="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-3 rounded-lg text-sm">
            {{ setupError }}
          </div>
          
          <div>
            <label for="verify-code" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Verification Code
            </label>
            <input
              id="verify-code"
              v-model="verifyCode"
              type="text"
              inputmode="numeric"
              maxlength="6"
              placeholder="000000"
              class="w-full px-3 py-2 text-center text-lg tracking-widest border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
            />
          </div>
          
          <div class="flex justify-end gap-3 pt-4">
            <button
              @click="setupStep = 1"
              class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600"
            >
              Back
            </button>
            <button
              @click="verifyAndEnable"
              :disabled="verifying || verifyCode.length !== 6"
              class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              {{ verifying ? 'Verifying...' : 'Enable 2FA' }}
            </button>
          </div>
        </div>
        
        <!-- Step 3: Show Backup Codes -->
        <div v-else-if="setupStep === 3" class="space-y-4">
          <div class="flex items-center gap-2 text-green-600 mb-2">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
            <span class="font-medium">2FA Enabled Successfully!</span>
          </div>
          
          <div class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg p-4">
            <p class="text-sm font-medium text-yellow-800 dark:text-yellow-300 mb-2">
              ⚠️ Save these backup codes in a safe place
            </p>
            <p class="text-xs text-yellow-700 dark:text-yellow-400">
              You can use these codes to access your account if you lose your authenticator device. Each code can only be used once.
            </p>
          </div>
          
          <div class="bg-gray-100 dark:bg-gray-700 p-4 rounded-lg">
            <div class="grid grid-cols-2 gap-2 font-mono text-sm dark:text-gray-200">
              <div v-for="code in backupCodes" :key="code" class="bg-white dark:bg-gray-600 px-3 py-2 rounded text-center">
                {{ code }}
              </div>
            </div>
          </div>
          
          <div class="flex justify-end pt-4">
            <button
              @click="finishSetup"
              class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700"
            >
              I've saved my backup codes
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Disable 2FA Modal -->
    <div v-if="showDisableModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-md w-full mx-4 p-6">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Disable Two-Factor Authentication</h3>
        
        <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
          Enter your password to disable 2FA. This will make your account less secure.
        </p>
        
        <div v-if="disableError" class="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-3 rounded-lg text-sm mb-4">
          {{ disableError }}
        </div>
        
        <div class="mb-4">
          <label for="disable-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
            Password
          </label>
          <input
            id="disable-password"
            v-model="disablePassword"
            type="password"
            class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
          />
        </div>
        
        <div class="flex justify-end gap-3">
          <button
            @click="closeDisableModal"
            class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600"
          >
            Cancel
          </button>
          <button
            @click="disableTOTP"
            :disabled="disabling || !disablePassword"
            class="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 disabled:opacity-50"
          >
            {{ disabling ? 'Disabling...' : 'Disable 2FA' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Regenerate Backup Codes Modal -->
    <div v-if="showRegenerateModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-md w-full mx-4 p-6">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Regenerate Backup Codes</h3>
        
        <div v-if="!newBackupCodes">
          <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
            This will invalidate all your existing backup codes and generate new ones.
          </p>
          
          <div v-if="regenerateError" class="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-3 rounded-lg text-sm mb-4">
            {{ regenerateError }}
          </div>
          
          <div class="mb-4">
            <label for="regenerate-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
              Password
            </label>
            <input
              id="regenerate-password"
              v-model="regeneratePassword"
              type="password"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-gray-700 dark:text-white"
            />
          </div>
          
          <div class="flex justify-end gap-3">
            <button
              @click="closeRegenerateModal"
              class="px-4 py-2 text-sm font-medium text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-700 rounded-lg hover:bg-gray-200 dark:hover:bg-gray-600"
            >
              Cancel
            </button>
            <button
              @click="regenerateBackupCodes"
              :disabled="regenerating || !regeneratePassword"
              class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              {{ regenerating ? 'Regenerating...' : 'Regenerate' }}
            </button>
          </div>
        </div>
        
        <div v-else class="space-y-4">
          <div class="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg p-4">
            <p class="text-sm font-medium text-yellow-800 dark:text-yellow-300 mb-2">
              ⚠️ Save these new backup codes
            </p>
            <p class="text-xs text-yellow-700 dark:text-yellow-400">
              Your old backup codes are no longer valid.
            </p>
          </div>
          
          <div class="bg-gray-100 dark:bg-gray-700 p-4 rounded-lg">
            <div class="grid grid-cols-2 gap-2 font-mono text-sm dark:text-gray-200">
              <div v-for="code in newBackupCodes" :key="code" class="bg-white dark:bg-gray-600 px-3 py-2 rounded text-center">
                {{ code }}
              </div>
            </div>
          </div>
          
          <div class="flex justify-end">
            <button
              @click="closeRegenerateModal"
              class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700"
            >
              Done
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Save indicator -->
    <div v-if="saving" class="fixed bottom-4 right-4 bg-blue-600 text-white px-4 py-2 rounded-lg shadow-lg">
      Saving...
    </div>
    <div v-if="saved" class="fixed bottom-4 right-4 bg-green-600 text-white px-4 py-2 rounded-lg shadow-lg">
      Settings saved!
    </div>
    <div v-if="error" class="fixed bottom-4 right-4 bg-red-600 text-white px-4 py-2 rounded-lg shadow-lg">
      {{ error }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

const selectedTimezone = ref(authStore.user?.timezone || 'UTC')
const saving = ref(false)
const saved = ref(false)
const error = ref('')

// 2FA State
const loadingTOTP = ref(true)
const totpStatus = ref<{ enabled: boolean; backup_codes_remaining: number } | null>(null)
const settingUp2FA = ref(false)
const showSetupModal = ref(false)
const setupStep = ref(1)
const totpSetup = ref<{ secret: string; qrcode_url: string } | null>(null)
const verifyCode = ref('')
const verifying = ref(false)
const setupError = ref('')
const backupCodes = ref<string[]>([])

// Disable 2FA State
const showDisableModal = ref(false)
const disablePassword = ref('')
const disabling = ref(false)
const disableError = ref('')

// Regenerate Backup Codes State
const showRegenerateModal = ref(false)
const regeneratePassword = ref('')
const regenerating = ref(false)
const regenerateError = ref('')
const newBackupCodes = ref<string[] | null>(null)

// Common timezones grouped by region
const timezones = [
  // UTC
  { value: 'UTC', label: 'UTC (Coordinated Universal Time)', region: 'UTC' },
  
  // Americas
  { value: 'America/New_York', label: '(UTC-05:00) Eastern Time (US & Canada)', region: 'Americas' },
  { value: 'America/Chicago', label: '(UTC-06:00) Central Time (US & Canada)', region: 'Americas' },
  { value: 'America/Denver', label: '(UTC-07:00) Mountain Time (US & Canada)', region: 'Americas' },
  { value: 'America/Los_Angeles', label: '(UTC-08:00) Pacific Time (US & Canada)', region: 'Americas' },
  { value: 'America/Anchorage', label: '(UTC-09:00) Alaska', region: 'Americas' },
  { value: 'Pacific/Honolulu', label: '(UTC-10:00) Hawaii', region: 'Americas' },
  { value: 'America/Phoenix', label: '(UTC-07:00) Arizona', region: 'Americas' },
  { value: 'America/Toronto', label: '(UTC-05:00) Toronto', region: 'Americas' },
  { value: 'America/Vancouver', label: '(UTC-08:00) Vancouver', region: 'Americas' },
  { value: 'America/Mexico_City', label: '(UTC-06:00) Mexico City', region: 'Americas' },
  { value: 'America/Sao_Paulo', label: '(UTC-03:00) São Paulo', region: 'Americas' },
  { value: 'America/Argentina/Buenos_Aires', label: '(UTC-03:00) Buenos Aires', region: 'Americas' },
  
  // Europe
  { value: 'Europe/London', label: '(UTC+00:00) London', region: 'Europe' },
  { value: 'Europe/Paris', label: '(UTC+01:00) Paris', region: 'Europe' },
  { value: 'Europe/Berlin', label: '(UTC+01:00) Berlin', region: 'Europe' },
  { value: 'Europe/Madrid', label: '(UTC+01:00) Madrid', region: 'Europe' },
  { value: 'Europe/Rome', label: '(UTC+01:00) Rome', region: 'Europe' },
  { value: 'Europe/Amsterdam', label: '(UTC+01:00) Amsterdam', region: 'Europe' },
  { value: 'Europe/Brussels', label: '(UTC+01:00) Brussels', region: 'Europe' },
  { value: 'Europe/Vienna', label: '(UTC+01:00) Vienna', region: 'Europe' },
  { value: 'Europe/Stockholm', label: '(UTC+01:00) Stockholm', region: 'Europe' },
  { value: 'Europe/Zurich', label: '(UTC+01:00) Zurich', region: 'Europe' },
  { value: 'Europe/Athens', label: '(UTC+02:00) Athens', region: 'Europe' },
  { value: 'Europe/Helsinki', label: '(UTC+02:00) Helsinki', region: 'Europe' },
  { value: 'Europe/Moscow', label: '(UTC+03:00) Moscow', region: 'Europe' },
  
  // Asia
  { value: 'Asia/Dubai', label: '(UTC+04:00) Dubai', region: 'Asia' },
  { value: 'Asia/Kolkata', label: '(UTC+05:30) Mumbai, New Delhi', region: 'Asia' },
  { value: 'Asia/Dhaka', label: '(UTC+06:00) Dhaka', region: 'Asia' },
  { value: 'Asia/Bangkok', label: '(UTC+07:00) Bangkok', region: 'Asia' },
  { value: 'Asia/Singapore', label: '(UTC+08:00) Singapore', region: 'Asia' },
  { value: 'Asia/Hong_Kong', label: '(UTC+08:00) Hong Kong', region: 'Asia' },
  { value: 'Asia/Shanghai', label: '(UTC+08:00) Shanghai', region: 'Asia' },
  { value: 'Asia/Taipei', label: '(UTC+08:00) Taipei', region: 'Asia' },
  { value: 'Asia/Seoul', label: '(UTC+09:00) Seoul', region: 'Asia' },
  { value: 'Asia/Tokyo', label: '(UTC+09:00) Tokyo', region: 'Asia' },
  
  // Australia & Pacific
  { value: 'Australia/Perth', label: '(UTC+08:00) Perth', region: 'Australia & Pacific' },
  { value: 'Australia/Adelaide', label: '(UTC+09:30) Adelaide', region: 'Australia & Pacific' },
  { value: 'Australia/Sydney', label: '(UTC+10:00) Sydney', region: 'Australia & Pacific' },
  { value: 'Australia/Melbourne', label: '(UTC+10:00) Melbourne', region: 'Australia & Pacific' },
  { value: 'Australia/Brisbane', label: '(UTC+10:00) Brisbane', region: 'Australia & Pacific' },
  { value: 'Pacific/Auckland', label: '(UTC+12:00) Auckland', region: 'Australia & Pacific' },
  
  // Africa & Middle East
  { value: 'Africa/Cairo', label: '(UTC+02:00) Cairo', region: 'Africa & Middle East' },
  { value: 'Africa/Johannesburg', label: '(UTC+02:00) Johannesburg', region: 'Africa & Middle East' },
  { value: 'Africa/Lagos', label: '(UTC+01:00) Lagos', region: 'Africa & Middle East' },
  { value: 'Asia/Jerusalem', label: '(UTC+02:00) Jerusalem', region: 'Africa & Middle East' },
  { value: 'Asia/Riyadh', label: '(UTC+03:00) Riyadh', region: 'Africa & Middle East' },
]

const groupedTimezones = computed(() => {
  const groups: Record<string, typeof timezones> = {}
  
  for (const tz of timezones) {
    if (!groups[tz.region]) {
      groups[tz.region] = []
    }
    groups[tz.region].push(tz)
  }
  
  return Object.entries(groups).map(([label, timezones]) => ({
    label,
    timezones
  }))
})

async function saveTimezone() {
  saving.value = true
  saved.value = false
  error.value = ''
  
  try {
    await authStore.updateSettings({ timezone: selectedTimezone.value })
    saving.value = false
    saved.value = true
    setTimeout(() => { saved.value = false }, 2000)
  } catch (e) {
    saving.value = false
    error.value = 'Failed to save settings'
    setTimeout(() => { error.value = '' }, 3000)
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 2FA Functions
async function loadTOTPStatus() {
  loadingTOTP.value = true
  try {
    totpStatus.value = await authStore.getTOTPStatus()
  } catch (e) {
    console.error('Failed to load 2FA status:', e)
  } finally {
    loadingTOTP.value = false
  }
}

async function startTOTPSetup() {
  settingUp2FA.value = true
  setupError.value = ''
  
  try {
    totpSetup.value = await authStore.initiateTOTPSetup()
    setupStep.value = 1
    showSetupModal.value = true
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to start 2FA setup'
    setTimeout(() => { error.value = '' }, 3000)
  } finally {
    settingUp2FA.value = false
  }
}

function cancelSetup() {
  showSetupModal.value = false
  totpSetup.value = null
  verifyCode.value = ''
  setupStep.value = 1
  setupError.value = ''
}

async function verifyAndEnable() {
  verifying.value = true
  setupError.value = ''
  
  try {
    backupCodes.value = await authStore.confirmTOTPSetup(verifyCode.value)
    setupStep.value = 3
  } catch (e: any) {
    setupError.value = e.response?.data?.error || 'Invalid verification code'
  } finally {
    verifying.value = false
  }
}

function finishSetup() {
  showSetupModal.value = false
  totpSetup.value = null
  verifyCode.value = ''
  setupStep.value = 1
  backupCodes.value = []
  loadTOTPStatus()
}

function closeDisableModal() {
  showDisableModal.value = false
  disablePassword.value = ''
  disableError.value = ''
}

async function disableTOTP() {
  disabling.value = true
  disableError.value = ''
  
  try {
    await authStore.disableTOTP(disablePassword.value)
    closeDisableModal()
    loadTOTPStatus()
  } catch (e: any) {
    disableError.value = e.response?.data?.error || 'Failed to disable 2FA'
  } finally {
    disabling.value = false
  }
}

function closeRegenerateModal() {
  showRegenerateModal.value = false
  regeneratePassword.value = ''
  regenerateError.value = ''
  newBackupCodes.value = null
  if (totpStatus.value) {
    loadTOTPStatus()
  }
}

async function regenerateBackupCodes() {
  regenerating.value = true
  regenerateError.value = ''
  
  try {
    newBackupCodes.value = await authStore.regenerateBackupCodes(regeneratePassword.value)
  } catch (e: any) {
    regenerateError.value = e.response?.data?.error || 'Failed to regenerate backup codes'
  } finally {
    regenerating.value = false
  }
}

onMounted(() => {
  // Sync with store in case it was updated
  if (authStore.user?.timezone) {
    selectedTimezone.value = authStore.user.timezone
  }
  
  // Load 2FA status
  loadTOTPStatus()
})
</script>
