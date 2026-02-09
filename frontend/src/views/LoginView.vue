<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import api from '@/api'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const isFirstTimeSetup = ref(false)
const checkingSetup = ref(true)

const name = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const totpCode = ref('')
const loading = ref(false)
const error = ref('')

// 2FA state from store
const showTOTPInput = computed(() => authStore.totpRequired)

onMounted(async () => {
  try {
    const response = await api.get('/auth/setup-status')
    isFirstTimeSetup.value = response.data.needs_setup
  } catch {
    isFirstTimeSetup.value = false
  } finally {
    checkingSetup.value = false
  }
})

async function handleLogin() {
  if (!email.value || !password.value) {
    error.value = 'Please fill in all fields'
    return
  }

  loading.value = true
  error.value = ''

  try {
    await authStore.login(email.value, password.value)
    const redirect = route.query.redirect as string || '/'
    router.push(redirect)
  } catch (err: any) {
    if (err.message === 'TOTP_REQUIRED') {
      // 2FA required - show TOTP input
      error.value = ''
    } else {
      error.value = err.response?.data?.error || 'Login failed'
    }
  } finally {
    loading.value = false
  }
}

async function handleTOTPSubmit() {
  if (!totpCode.value) {
    error.value = 'Please enter your 2FA code'
    return
  }

  loading.value = true
  error.value = ''

  try {
    await authStore.completeTOTPLogin(totpCode.value)
    const redirect = route.query.redirect as string || '/'
    router.push(redirect)
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Invalid 2FA code'
    totpCode.value = ''
  } finally {
    loading.value = false
  }
}

function cancelTOTP() {
  authStore.cancelTOTPLogin()
  totpCode.value = ''
  error.value = ''
}

async function handleSetup() {
  if (!name.value || !email.value || !password.value) {
    error.value = 'Please fill in all fields'
    return
  }

  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }

  if (password.value.length < 8) {
    error.value = 'Password must be at least 8 characters'
    return
  }

  loading.value = true
  error.value = ''

  try {
    // Register the first admin account
    const response = await api.post('/auth/register', {
      name: name.value,
      email: email.value,
      password: password.value
    })

    // If the backend returned an email provider hint, save it for the email module
    if (response.data.email_provider_hint) {
      localStorage.setItem('email_provider_hint', JSON.stringify(response.data.email_provider_hint))
    }

    // Log in with the new account
    await authStore.login(email.value, password.value)
    router.push('/')
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Setup failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <!-- Loading state -->
  <div v-if="checkingSetup" class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
    <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
  </div>

  <!-- First Time Setup -->
  <div v-else-if="isFirstTimeSetup" class="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 dark:from-gray-900 dark:to-gray-800 py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full space-y-8">
      <div class="text-center">
        <div class="inline-flex items-center justify-center w-16 h-16 bg-blue-600 rounded-2xl mb-4">
          <svg class="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4" />
          </svg>
        </div>
        <h1 class="text-3xl font-bold text-gray-900 dark:text-white">Welcome to Tessera</h1>
        <p class="mt-2 text-gray-600 dark:text-gray-400">Create your administrator account to get started</p>
      </div>

      <form class="mt-8 space-y-6 bg-white dark:bg-gray-800 p-8 rounded-2xl shadow-xl" @submit.prevent="handleSetup">
        <div v-if="error" class="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-3 rounded-lg text-sm">
          {{ error }}
        </div>

        <div class="space-y-4">
          <div>
            <label for="name" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Your Name
            </label>
            <input
              id="name"
              v-model="name"
              type="text"
              autocomplete="name"
              required
              placeholder="John Doe"
              class="mt-1 block w-full px-3 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          <div>
            <label for="email" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Email Address
            </label>
            <input
              id="email"
              v-model="email"
              type="email"
              autocomplete="email"
              required
              placeholder="admin@example.com"
              class="mt-1 block w-full px-3 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          <div>
            <label for="password" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Password
            </label>
            <input
              id="password"
              v-model="password"
              type="password"
              autocomplete="new-password"
              required
              placeholder="Minimum 8 characters"
              class="mt-1 block w-full px-3 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          <div>
            <label for="confirmPassword" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Confirm Password
            </label>
            <input
              id="confirmPassword"
              v-model="confirmPassword"
              type="password"
              autocomplete="new-password"
              required
              placeholder="Confirm your password"
              class="mt-1 block w-full px-3 py-2.5 border border-gray-300 dark:border-gray-600 rounded-lg shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
        </div>

        <button
          type="submit"
          :disabled="loading"
          class="w-full flex justify-center py-3 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <svg v-if="loading" class="animate-spin -ml-1 mr-2 h-5 w-5 text-white" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
          </svg>
          {{ loading ? 'Creating Account...' : 'Create Administrator Account' }}
        </button>
      </form>

      <p class="text-center text-sm text-gray-500 dark:text-gray-400">
        This account will have full administrative access to Tessera.
      </p>
    </div>
  </div>

  <!-- Normal Login -->
  <div v-else class="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full space-y-8">
      <div>
        <h1 class="text-center text-4xl font-bold text-blue-600">Tessera</h1>
        <h2 class="mt-6 text-center text-2xl font-semibold text-gray-900 dark:text-white">
          {{ showTOTPInput ? 'Two-Factor Authentication' : 'Sign in to your account' }}
        </h2>
      </div>

      <!-- 2FA Code Input -->
      <form v-if="showTOTPInput" class="mt-8 space-y-6" @submit.prevent="handleTOTPSubmit">
        <div v-if="error" class="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-3 rounded-lg text-sm">
          {{ error }}
        </div>

        <div class="text-center text-sm text-gray-600 dark:text-gray-400 mb-4">
          <p>Enter the 6-digit code from your authenticator app</p>
          <p class="mt-1 text-xs">or use one of your backup codes</p>
        </div>

        <div>
          <label for="totp-code" class="sr-only">2FA Code</label>
          <input
            id="totp-code"
            v-model="totpCode"
            type="text"
            inputmode="numeric"
            autocomplete="one-time-code"
            maxlength="20"
            placeholder="Enter code"
            class="block w-full px-3 py-3 text-center text-2xl tracking-widest border border-gray-300 dark:border-gray-600 rounded-lg shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            autofocus
          />
        </div>

        <div class="flex gap-3">
          <button
            type="button"
            @click="cancelTOTP"
            class="flex-1 py-2.5 px-4 border border-gray-300 dark:border-gray-600 rounded-lg text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
          >
            Back
          </button>
          <button
            type="submit"
            :disabled="loading"
            class="flex-1 py-2.5 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ loading ? 'Verifying...' : 'Verify' }}
          </button>
        </div>
      </form>

      <!-- Normal Login Form -->
      <form v-else class="mt-8 space-y-6" @submit.prevent="handleLogin">
        <div v-if="error" class="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-3 rounded-lg text-sm">
          {{ error }}
        </div>

        <div class="space-y-4">
          <div>
            <label for="login-email" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Email address
            </label>
            <input
              id="login-email"
              v-model="email"
              type="email"
              autocomplete="email"
              required
              class="mt-1 block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          <div>
            <label for="login-password" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Password
            </label>
            <input
              id="login-password"
              v-model="password"
              type="password"
              autocomplete="current-password"
              required
              class="mt-1 block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg shadow-sm bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
        </div>

        <button
          type="submit"
          :disabled="loading"
          class="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <span v-if="loading">Signing in...</span>
          <span v-else>Sign in</span>
        </button>

        <p class="text-center text-sm text-gray-600 dark:text-gray-400">
          Don't have an account?
          <router-link to="/register" class="font-medium text-blue-600 hover:text-blue-500">
            Create one
          </router-link>
        </p>
      </form>
    </div>
  </div>
</template>
