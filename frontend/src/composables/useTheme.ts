import { ref, watch, computed } from 'vue'

type Theme = 'light' | 'dark'

const theme = ref<Theme>('light')
let initialized = false

// Initialize theme from localStorage or system preference
function initTheme() {
  if (initialized) return
  initialized = true
  
  const stored = localStorage.getItem('theme') as Theme | null
  
  if (stored) {
    theme.value = stored
  } else if (typeof window !== 'undefined' && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    theme.value = 'dark'
  }
  
  applyTheme(theme.value)
}

function applyTheme(newTheme: Theme) {
  if (typeof document === 'undefined') return
  
  if (newTheme === 'dark') {
    document.documentElement.classList.add('dark')
  } else {
    document.documentElement.classList.remove('dark')
  }
}

function toggleTheme() {
  theme.value = theme.value === 'light' ? 'dark' : 'light'
  applyTheme(theme.value)
  localStorage.setItem('theme', theme.value)
}

function setTheme(newTheme: Theme) {
  theme.value = newTheme
  applyTheme(newTheme)
  localStorage.setItem('theme', newTheme)
}

const isDark = computed(() => theme.value === 'dark')

export function useTheme() {
  // Ensure initialized when composable is used
  initTheme()
  
  return {
    theme,
    isDark,
    toggleTheme,
    setTheme
  }
}
