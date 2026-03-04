<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const name = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')

async function handleRegister() {
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
    await authStore.register(email.value, password.value, name.value)
    router.push({ name: 'login', query: { registered: 'true' } })
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Registration failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-stone-50 dark:bg-neutral-900 py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full space-y-8">
      <div>
        <h1 class="text-center text-4xl font-bold text-stone-800 dark:text-stone-200">Tessera</h1>
        <h2 class="mt-6 text-center text-2xl font-semibold text-stone-900 dark:text-stone-100">
          Create your account
        </h2>
      </div>

      <form class="mt-8 space-y-6" @submit.prevent="handleRegister">
        <div v-if="error" class="bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 p-3 rounded-lg text-sm">
          {{ error }}
        </div>

        <div class="space-y-4">
          <div>
            <label for="name" class="block text-sm font-medium text-stone-700 dark:text-stone-300">
              Full name
            </label>
            <input
              id="name"
              v-model="name"
              type="text"
              autocomplete="name"
              required
              class="mt-1 block w-full px-3 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-stone-400 focus:border-neutral-700 dark:border-neutral-300 dark:bg-neutral-800 dark:text-stone-100"
            />
          </div>

          <div>
            <label for="email" class="block text-sm font-medium text-stone-700 dark:text-stone-300">
              Email address
            </label>
            <input
              id="email"
              v-model="email"
              type="email"
              autocomplete="email"
              required
              class="mt-1 block w-full px-3 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-stone-400 focus:border-neutral-700 dark:border-neutral-300 dark:bg-neutral-800 dark:text-stone-100"
            />
          </div>

          <div>
            <label for="password" class="block text-sm font-medium text-stone-700 dark:text-stone-300">
              Password
            </label>
            <input
              id="password"
              v-model="password"
              type="password"
              autocomplete="new-password"
              required
              class="mt-1 block w-full px-3 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-stone-400 focus:border-neutral-700 dark:border-neutral-300 dark:bg-neutral-800 dark:text-stone-100"
            />
          </div>

          <div>
            <label for="confirmPassword" class="block text-sm font-medium text-stone-700 dark:text-stone-300">
              Confirm password
            </label>
            <input
              id="confirmPassword"
              v-model="confirmPassword"
              type="password"
              autocomplete="new-password"
              required
              class="mt-1 block w-full px-3 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-stone-400 focus:border-neutral-700 dark:border-neutral-300 dark:bg-neutral-800 dark:text-stone-100"
            />
          </div>
        </div>

        <button
          type="submit"
          :disabled="loading"
          class="w-full flex justify-center py-2.5 px-4 border border-transparent rounded-lg shadow-sm text-sm font-medium text-white bg-neutral-800 dark:bg-neutral-200 hover:bg-neutral-700 dark:hover:bg-neutral-300 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-stone-400 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <span v-if="loading">Creating account...</span>
          <span v-else>Create account</span>
        </button>

        <p class="text-center text-sm text-stone-600 dark:text-stone-400">
          Already have an account?
          <router-link to="/login" class="font-medium text-stone-800 dark:text-stone-200 hover:text-stone-700 dark:text-stone-300">
            Sign in
          </router-link>
        </p>
      </form>
    </div>
  </div>
</template>
