import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '../auth'

// Mock the API module
vi.mock('@/api', () => ({
  default: {
    post: vi.fn(),
    get: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

import api from '@/api'

describe('Auth Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    localStorage.clear()
  })

  describe('initialization', () => {
    it('initializes with null user and tokens', () => {
      const store = useAuthStore()
      expect(store.user).toBeNull()
      expect(store.tokens).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })

    it('loads tokens from localStorage', () => {
      const tokens = { access_token: 'test-token', refresh_token: 'refresh', expires_at: '2024-12-31' }
      const user = { id: '1', email: 'test@example.com', name: 'Test', role: 'user', timezone: 'UTC', storage_used: 0, storage_limit: 1000 }
      
      localStorage.setItem('tokens', JSON.stringify(tokens))
      localStorage.setItem('user', JSON.stringify(user))
      
      const store = useAuthStore()
      expect(store.tokens).toEqual(tokens)
      expect(store.user).toEqual(user)
      expect(store.isAuthenticated).toBe(true)
    })
  })

  describe('login', () => {
    it('logs in successfully', async () => {
      const mockResponse = {
        status: 200,
        data: {
          user: { id: '1', email: 'test@example.com', name: 'Test', role: 'user' },
          tokens: { access_token: 'token', refresh_token: 'refresh', expires_at: '2024-12-31' }
        }
      }
      vi.mocked(api.post).mockResolvedValueOnce(mockResponse)

      const store = useAuthStore()
      await store.login('test@example.com', 'password123')

      expect(store.user).toEqual(mockResponse.data.user)
      expect(store.tokens).toEqual(mockResponse.data.tokens)
      expect(store.isAuthenticated).toBe(true)
      expect(localStorage.getItem('tokens')).toBeTruthy()
      expect(localStorage.getItem('user')).toBeTruthy()
    })

    it('handles 2FA required response', async () => {
      const mockResponse = {
        status: 428,
        data: { totp_required: true }
      }
      vi.mocked(api.post).mockResolvedValueOnce(mockResponse)

      const store = useAuthStore()
      
      await expect(store.login('test@example.com', 'password123')).rejects.toThrow('TOTP_REQUIRED')
      expect(store.totpRequired).toBe(true)
      expect(store.pendingLoginEmail).toBe('test@example.com')
    })

    it('completes TOTP login', async () => {
      const initialResponse = {
        status: 428,
        data: { totp_required: true }
      }
      const successResponse = {
        status: 200,
        data: {
          user: { id: '1', email: 'test@example.com', name: 'Test', role: 'user' },
          tokens: { access_token: 'token', refresh_token: 'refresh', expires_at: '2024-12-31' }
        }
      }
      vi.mocked(api.post)
        .mockResolvedValueOnce(initialResponse)
        .mockResolvedValueOnce(successResponse)

      const store = useAuthStore()
      
      // Initial login triggers 2FA
      await expect(store.login('test@example.com', 'password123')).rejects.toThrow('TOTP_REQUIRED')
      
      // Complete with TOTP code
      await store.completeTOTPLogin('123456')
      
      expect(store.isAuthenticated).toBe(true)
      expect(store.totpRequired).toBe(false)
    })

    it('cancels TOTP login', async () => {
      const mockResponse = {
        status: 428,
        data: { totp_required: true }
      }
      vi.mocked(api.post).mockResolvedValueOnce(mockResponse)

      const store = useAuthStore()
      await expect(store.login('test@example.com', 'password123')).rejects.toThrow('TOTP_REQUIRED')
      
      store.cancelTOTPLogin()
      
      expect(store.totpRequired).toBe(false)
      expect(store.pendingLoginEmail).toBe('')
      expect(store.pendingLoginPassword).toBe('')
    })
  })

  describe('logout', () => {
    it('clears user data and tokens', async () => {
      vi.mocked(api.post).mockResolvedValueOnce({})

      const store = useAuthStore()
      store.user = { id: '1', email: 'test@example.com', name: 'Test', role: 'user', timezone: 'UTC', storage_used: 0, storage_limit: 1000 }
      store.tokens = { access_token: 'token', refresh_token: 'refresh', expires_at: '2024-12-31' }
      localStorage.setItem('tokens', JSON.stringify(store.tokens))
      localStorage.setItem('user', JSON.stringify(store.user))

      await store.logout()

      expect(store.user).toBeNull()
      expect(store.tokens).toBeNull()
      expect(store.isAuthenticated).toBe(false)
      expect(localStorage.getItem('tokens')).toBeNull()
      expect(localStorage.getItem('user')).toBeNull()
    })

    it('logs out even if API call fails', async () => {
      vi.mocked(api.post).mockRejectedValueOnce(new Error('Network error'))

      const store = useAuthStore()
      store.user = { id: '1', email: 'test@example.com', name: 'Test', role: 'user', timezone: 'UTC', storage_used: 0, storage_limit: 1000 }
      store.tokens = { access_token: 'token', refresh_token: 'refresh', expires_at: '2024-12-31' }

      await store.logout()

      expect(store.user).toBeNull()
      expect(store.tokens).toBeNull()
    })
  })

  describe('register', () => {
    it('registers successfully', async () => {
      vi.mocked(api.post).mockResolvedValueOnce({ data: { message: 'Success' } })

      const store = useAuthStore()
      await store.register('test@example.com', 'Password123!', 'Test User')

      expect(api.post).toHaveBeenCalledWith('/auth/register', {
        email: 'test@example.com',
        password: 'Password123!',
        name: 'Test User'
      })
    })
  })

  describe('refreshTokens', () => {
    it('refreshes tokens successfully', async () => {
      const newTokens = { access_token: 'new-token', refresh_token: 'new-refresh', expires_at: '2024-12-31' }
      vi.mocked(api.post).mockResolvedValueOnce({ data: { tokens: newTokens } })

      const store = useAuthStore()
      store.tokens = { access_token: 'old-token', refresh_token: 'old-refresh', expires_at: '2024-01-01' }

      const result = await store.refreshTokens()

      expect(result).toBe(true)
      expect(store.tokens).toEqual(newTokens)
    })

    it('returns false and logs out if refresh fails', async () => {
      vi.mocked(api.post).mockRejectedValueOnce(new Error('Invalid token'))

      const store = useAuthStore()
      store.tokens = { access_token: 'token', refresh_token: 'refresh', expires_at: '2024-01-01' }

      const result = await store.refreshTokens()

      expect(result).toBe(false)
      expect(store.tokens).toBeNull()
    })

    it('returns false if no refresh token', async () => {
      const store = useAuthStore()

      const result = await store.refreshTokens()

      expect(result).toBe(false)
      expect(api.post).not.toHaveBeenCalled()
    })
  })

  describe('fetchUser', () => {
    it('fetches and stores user data', async () => {
      const userData = { id: '1', email: 'test@example.com', name: 'Test', role: 'user', timezone: 'UTC', storage_used: 500, storage_limit: 1000 }
      vi.mocked(api.get).mockResolvedValueOnce({ data: userData })

      const store = useAuthStore()
      await store.fetchUser()

      expect(store.user).toEqual(userData)
      expect(JSON.parse(localStorage.getItem('user') || '{}')).toEqual(userData)
    })
  })

  describe('2FA methods', () => {
    it('gets TOTP status', async () => {
      const status = { enabled: true, backup_codes_remaining: 10 }
      vi.mocked(api.get).mockResolvedValueOnce({ data: status })

      const store = useAuthStore()
      const result = await store.getTOTPStatus()

      expect(result).toEqual(status)
      expect(api.get).toHaveBeenCalledWith('/auth/totp/status')
    })

    it('initiates TOTP setup', async () => {
      const setup = { secret: 'JBSWY3DPEHPK3PXP', qrcode_url: 'https://example.com/qr' }
      vi.mocked(api.post).mockResolvedValueOnce({ data: setup })

      const store = useAuthStore()
      const result = await store.initiateTOTPSetup()

      expect(result).toEqual(setup)
      expect(api.post).toHaveBeenCalledWith('/auth/totp/setup')
    })

    it('confirms TOTP setup', async () => {
      const backupCodes = ['code1', 'code2', 'code3']
      vi.mocked(api.post).mockResolvedValueOnce({ data: { backup_codes: backupCodes } })

      const store = useAuthStore()
      store.user = { id: '1', email: 'test@example.com', name: 'Test', role: 'user', timezone: 'UTC', storage_used: 0, storage_limit: 1000 }
      
      const result = await store.confirmTOTPSetup('123456')

      expect(result).toEqual(backupCodes)
      expect(store.user?.totp_enabled).toBe(true)
      expect(api.post).toHaveBeenCalledWith('/auth/totp/confirm', { code: '123456' })
    })

    it('disables TOTP', async () => {
      vi.mocked(api.delete).mockResolvedValueOnce({})

      const store = useAuthStore()
      store.user = { id: '1', email: 'test@example.com', name: 'Test', role: 'user', timezone: 'UTC', storage_used: 0, storage_limit: 1000, totp_enabled: true }
      
      await store.disableTOTP('123456')

      expect(store.user?.totp_enabled).toBe(false)
      expect(api.delete).toHaveBeenCalledWith('/auth/totp', { data: { code: '123456' } })
    })

    it('regenerates backup codes', async () => {
      const newCodes = ['newcode1', 'newcode2']
      vi.mocked(api.post).mockResolvedValueOnce({ data: { backup_codes: newCodes } })

      const store = useAuthStore()
      const result = await store.regenerateBackupCodes('123456')

      expect(result).toEqual(newCodes)
      expect(api.post).toHaveBeenCalledWith('/auth/totp/backup-codes', { code: '123456' })
    })
  })
})
