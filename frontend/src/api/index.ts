import axios from 'axios'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json'
  }
})

// Refresh lock to prevent race conditions
let isRefreshing = false
let refreshSubscribers: { resolve: (token: string) => void; reject: (error: Error) => void }[] = []

function subscribeTokenRefresh(resolve: (token: string) => void, reject: (error: Error) => void) {
  refreshSubscribers.push({ resolve, reject })
}

function onTokenRefreshed(token: string) {
  refreshSubscribers.forEach(({ resolve }) => resolve(token))
  refreshSubscribers = []
}

function onTokenRefreshFailed(error: Error) {
  refreshSubscribers.forEach(({ reject }) => reject(error))
  refreshSubscribers = []
}

// Request interceptor - add auth token
api.interceptors.request.use(
  (config) => {
    const authStore = useAuthStore()
    if (authStore.tokens?.access_token) {
      config.headers.Authorization = `Bearer ${authStore.tokens.access_token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// URLs that should never trigger a token refresh (to prevent infinite loops)
const AUTH_URLS = ['/auth/refresh', '/auth/login', '/auth/register', '/auth/login/totp']

// Response interceptor - handle token refresh with lock to prevent race conditions
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    // Never try to refresh for auth endpoints themselves — prevents infinite loop
    if (AUTH_URLS.some(url => originalRequest.url?.includes(url))) {
      return Promise.reject(error)
    }
    
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        // Another request is already refreshing, wait for it
        return new Promise((resolve, reject) => {
          subscribeTokenRefresh(
            (token: string) => {
              originalRequest.headers.Authorization = `Bearer ${token}`
              resolve(api(originalRequest))
            },
            (err: Error) => reject(err)
          )
        })
      }
      
      originalRequest._retry = true
      isRefreshing = true
      
      const authStore = useAuthStore()
      
      try {
        const refreshed = await authStore.refreshTokens()
        
        isRefreshing = false
        
        if (refreshed && authStore.tokens?.access_token) {
          onTokenRefreshed(authStore.tokens.access_token)
          originalRequest.headers.Authorization = `Bearer ${authStore.tokens.access_token}`
          return api(originalRequest)
        } else {
          const refreshError = new Error('Token refresh failed')
          onTokenRefreshFailed(refreshError)
          router.push({ name: 'login' })
          return Promise.reject(refreshError)
        }
      } catch (refreshError) {
        isRefreshing = false
        onTokenRefreshFailed(refreshError as Error)
        router.push({ name: 'login' })
        return Promise.reject(refreshError)
      }
    }
    
    return Promise.reject(error)
  }
)

export default api
