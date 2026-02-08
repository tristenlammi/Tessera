import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'user'
  storageUsed: number
  storageQuota: number
  createdAt: string
  lastLoginAt: string | null
  isActive: boolean
  emailVerified: boolean
}

export interface SystemStats {
  totalUsers: number
  activeUsers: number
  totalStorage: number
  usedStorage: number
  totalFiles: number
  totalShares: number
  uploadsToday: number
  downloadsToday: number
}

export interface SystemSettings {
  siteName: string
  siteUrl: string
  defaultQuota: number
  allowRegistration: boolean
  requireEmailVerification: boolean
  maxUploadSize: number
  allowedFileTypes: string[]
  maintenanceMode: boolean
  smtpHost: string
  smtpPort: number
  smtpUser: string
  smtpFrom: string
}

export interface ActivityLog {
  id: string
  userId: string
  userEmail: string
  action: string
  resourceType: string
  resourceId: string
  ipAddress: string
  userAgent: string
  createdAt: string
  details: Record<string, unknown>
}

export const useAdminStore = defineStore('admin', () => {
  const users = ref<User[]>([])
  const systemStats = ref<SystemStats | null>(null)
  const systemSettings = ref<SystemSettings | null>(null)
  const activityLogs = ref<ActivityLog[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Pagination
  const usersPage = ref(1)
  const usersTotal = ref(0)
  const logsPage = ref(1)
  const logsTotal = ref(0)

  const totalPages = computed(() => Math.ceil(usersTotal.value / 20))
  const logsTotalPages = computed(() => Math.ceil(logsTotal.value / 50))

  // Users management
  async function fetchUsers(page = 1, search = '') {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams({ page: String(page), limit: '20' })
      if (search) params.set('search', search)
      
      const response = await api.get(`/admin/users?${params}`)
      users.value = response.data.users
      usersTotal.value = response.data.total
      usersPage.value = page
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch users'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function getUser(userId: string) {
    loading.value = true
    error.value = null
    try {
      const response = await api.get(`/admin/users/${userId}`)
      return response.data
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch user'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function createUser(data: { email: string; password: string; name: string; role: 'admin' | 'user'; storageQuota: number }) {
    loading.value = true
    error.value = null
    try {
      const response = await api.post('/admin/users', data)
      users.value.unshift(response.data)
      usersTotal.value++
      return response.data
    } catch (err: any) {
      error.value = err.response?.data?.error || err.message || 'Failed to create user'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateUser(userId: string, data: Partial<User>) {
    loading.value = true
    error.value = null
    try {
      const response = await api.patch(`/admin/users/${userId}`, data)
      const index = users.value.findIndex(u => u.id === userId)
      if (index !== -1) {
        users.value[index] = response.data
      }
      return response.data
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to update user'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteUser(userId: string) {
    loading.value = true
    error.value = null
    try {
      await api.delete(`/admin/users/${userId}`)
      users.value = users.value.filter(u => u.id !== userId)
      usersTotal.value--
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to delete user'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function setUserQuota(userId: string, quota: number) {
    return updateUser(userId, { storageQuota: quota })
  }

  async function toggleUserStatus(userId: string, isActive: boolean) {
    return updateUser(userId, { isActive })
  }

  async function setUserRole(userId: string, role: 'admin' | 'user') {
    return updateUser(userId, { role })
  }

  // System stats
  async function fetchSystemStats() {
    loading.value = true
    error.value = null
    try {
      const response = await api.get('/admin/stats')
      systemStats.value = response.data
      return response.data
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch stats'
      throw err
    } finally {
      loading.value = false
    }
  }

  // System settings
  async function fetchSystemSettings() {
    loading.value = true
    error.value = null
    try {
      const response = await api.get('/admin/settings')
      systemSettings.value = response.data
      return response.data
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch settings'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateSystemSettings(settings: Partial<SystemSettings>) {
    loading.value = true
    error.value = null
    try {
      const response = await api.patch('/admin/settings', settings)
      systemSettings.value = response.data
      return response.data
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to update settings'
      throw err
    } finally {
      loading.value = false
    }
  }

  // Activity logs
  async function fetchActivityLogs(page = 1, filters: Record<string, string> = {}) {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams({ page: String(page), limit: '50', ...filters })
      const response = await api.get(`/admin/logs?${params}`)
      activityLogs.value = response.data.logs
      logsTotal.value = response.data.total
      logsPage.value = page
      return response.data
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to fetch logs'
      throw err
    } finally {
      loading.value = false
    }
  }

  // Maintenance
  async function toggleMaintenanceMode(enabled: boolean) {
    return updateSystemSettings({ maintenanceMode: enabled })
  }

  async function clearCache() {
    loading.value = true
    error.value = null
    try {
      await api.post('/admin/cache/clear')
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to clear cache'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function runCleanup() {
    loading.value = true
    error.value = null
    try {
      const response = await api.post('/admin/cleanup')
      return response.data
    } catch (err: unknown) {
      error.value = err instanceof Error ? err.message : 'Failed to run cleanup'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    // State
    users,
    systemStats,
    systemSettings,
    activityLogs,
    loading,
    error,
    usersPage,
    usersTotal,
    totalPages,
    logsPage,
    logsTotal,
    logsTotalPages,
    // Users
    fetchUsers,
    getUser,
    createUser,
    updateUser,
    deleteUser,
    setUserQuota,
    toggleUserStatus,
    setUserRole,
    // Stats
    fetchSystemStats,
    // Settings
    fetchSystemSettings,
    updateSystemSettings,
    // Logs
    fetchActivityLogs,
    // Maintenance
    toggleMaintenanceMode,
    clearCache,
    runCleanup,
  }
})
