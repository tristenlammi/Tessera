import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

interface User {
  id: string
  email: string
  name: string
  role: string
  timezone: string
  storage_used: number
  storage_limit: number
  totp_enabled?: boolean
}

interface Tokens {
  access_token: string
  refresh_token: string
  expires_at: string
}

interface TOTPStatus {
  enabled: boolean
  backup_codes_remaining: number
}

interface TOTPSetup {
  secret: string
  qrcode_url: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const tokens = ref<Tokens | null>(null)
  const totpRequired = ref(false)
  const pendingLoginEmail = ref('')
  const pendingAuthToken = ref('') // Secure token instead of storing password
  let pendingAuthTimeout: ReturnType<typeof setTimeout> | null = null

  const isAuthenticated = computed(() => !!tokens.value?.access_token)

  // Clear pending auth after timeout (matches server-side 5 minute expiry)
  function startPendingAuthTimeout() {
    clearPendingAuthTimeout()
    pendingAuthTimeout = setTimeout(() => {
      if (pendingAuthToken.value) {
        console.log('[Auth] Pending auth token expired')
        cancelTOTPLogin()
      }
    }, 5 * 60 * 1000) // 5 minutes
  }

  function clearPendingAuthTimeout() {
    if (pendingAuthTimeout) {
      clearTimeout(pendingAuthTimeout)
      pendingAuthTimeout = null
    }
  }

  // Load from localStorage on init
  const storedTokens = localStorage.getItem('tokens')
  const storedUser = localStorage.getItem('user')
  if (storedTokens) tokens.value = JSON.parse(storedTokens)
  if (storedUser) user.value = JSON.parse(storedUser)

  async function login(email: string, password: string, totpCode?: string) {
    try {
      const response = await api.post('/auth/login', { 
        email, 
        password,
        totp_code: totpCode 
      })
      
      // Check if 2FA is required (shouldn't happen here since 428 throws)
      if (response.data.totp_required) {
        totpRequired.value = true
        pendingLoginEmail.value = email
        pendingAuthToken.value = response.data.pending_auth_token || ''
        startPendingAuthTimeout()
        throw new Error('TOTP_REQUIRED')
      }
      
      user.value = response.data.user
      tokens.value = response.data.tokens
      totpRequired.value = false
      pendingLoginEmail.value = ''
      pendingAuthToken.value = ''
      clearPendingAuthTimeout()
      
      localStorage.setItem('tokens', JSON.stringify(tokens.value))
      localStorage.setItem('user', JSON.stringify(user.value))
    } catch (err: any) {
      // Handle 428 Precondition Required (2FA required)
      if (err.response?.status === 428 || err.response?.data?.totp_required) {
        totpRequired.value = true
        pendingLoginEmail.value = email
        // Store the secure pending auth token instead of password
        pendingAuthToken.value = err.response?.data?.pending_auth_token || ''
        startPendingAuthTimeout()
        throw new Error('TOTP_REQUIRED')
      }
      // Re-throw other errors
      throw err
    }
  }

  async function completeTOTPLogin(totpCode: string) {
    if (!pendingAuthToken.value) {
      throw new Error('No pending login')
    }
    
    // Use the secure pending auth token endpoint
    const response = await api.post('/auth/login/totp', {
      pending_auth_token: pendingAuthToken.value,
      totp_code: totpCode
    })
    
    user.value = response.data.user
    tokens.value = response.data.tokens
    totpRequired.value = false
    pendingLoginEmail.value = ''
    pendingAuthToken.value = ''
    clearPendingAuthTimeout()
    
    localStorage.setItem('tokens', JSON.stringify(tokens.value))
    localStorage.setItem('user', JSON.stringify(user.value))
  }

  function cancelTOTPLogin() {
    totpRequired.value = false
    pendingLoginEmail.value = ''
    pendingAuthToken.value = ''
    clearPendingAuthTimeout()
  }

  async function register(email: string, password: string, name: string) {
    await api.post('/auth/register', { email, password, name })
  }

  async function logout() {
    try {
      await api.post('/auth/logout')
    } catch {
      // Ignore errors
    }
    
    user.value = null
    tokens.value = null
    totpRequired.value = false
    pendingLoginEmail.value = ''
    pendingAuthToken.value = ''
    localStorage.removeItem('tokens')
    localStorage.removeItem('user')
  }

  async function refreshTokens() {
    if (!tokens.value?.refresh_token) return false
    
    try {
      const response = await api.post('/auth/refresh', {
        refresh_token: tokens.value.refresh_token
      })
      tokens.value = response.data.tokens
      localStorage.setItem('tokens', JSON.stringify(tokens.value))
      return true
    } catch {
      await logout()
      return false
    }
  }

  async function fetchUser() {
    const response = await api.get('/auth/me')
    user.value = response.data
    localStorage.setItem('user', JSON.stringify(user.value))
  }

  async function updateSettings(settings: { timezone?: string }) {
    const response = await api.put('/auth/settings', settings)
    user.value = response.data
    localStorage.setItem('user', JSON.stringify(user.value))
  }

  // 2FA Methods
  async function getTOTPStatus(): Promise<TOTPStatus> {
    const response = await api.get('/auth/totp/status')
    return response.data
  }

  async function initiateTOTPSetup(): Promise<TOTPSetup> {
    const response = await api.post('/auth/totp/setup')
    return response.data
  }

  async function confirmTOTPSetup(code: string): Promise<string[]> {
    const response = await api.post('/auth/totp/confirm', { code })
    if (user.value) {
      user.value.totp_enabled = true
      localStorage.setItem('user', JSON.stringify(user.value))
    }
    return response.data.backup_codes
  }

  async function disableTOTP(password: string): Promise<void> {
    await api.delete('/auth/totp', { data: { password } })
    if (user.value) {
      user.value.totp_enabled = false
      localStorage.setItem('user', JSON.stringify(user.value))
    }
  }

  async function regenerateBackupCodes(password: string): Promise<string[]> {
    const response = await api.post('/auth/totp/backup-codes', { password })
    return response.data.backup_codes
  }

  return {
    user,
    tokens,
    totpRequired,
    isAuthenticated,
    login,
    completeTOTPLogin,
    cancelTOTPLogin,
    register,
    logout,
    refreshTokens,
    fetchUser,
    updateSettings,
    getTOTPStatus,
    initiateTOTPSetup,
    confirmTOTPSetup,
    disableTOTP,
    regenerateBackupCodes
  }
})
