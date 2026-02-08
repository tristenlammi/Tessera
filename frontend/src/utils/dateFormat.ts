import { useAuthStore } from '@/stores/auth'

/**
 * Format a date string according to the user's timezone preference
 * Shows time for today, weekday for this week, or full date
 */
export function formatDate(dateStr: string): string {
  const authStore = useAuthStore()
  const timezone = authStore.user?.timezone || 'UTC'
  
  const date = new Date(dateStr)
  const now = new Date()
  
  // Get dates in user's timezone for comparison
  const dateInTz = new Date(date.toLocaleString('en-US', { timeZone: timezone }))
  const nowInTz = new Date(now.toLocaleString('en-US', { timeZone: timezone }))
  
  const diff = nowInTz.getTime() - dateInTz.getTime()
  const oneDay = 24 * 60 * 60 * 1000

  // Same day - show time
  if (diff < oneDay && dateInTz.getDate() === nowInTz.getDate()) {
    return date.toLocaleTimeString('en-US', { 
      hour: 'numeric', 
      minute: '2-digit',
      timeZone: timezone 
    })
  }
  
  // Within last week - show weekday
  if (diff < 7 * oneDay) {
    return date.toLocaleDateString('en-US', { 
      weekday: 'short',
      timeZone: timezone 
    })
  }
  
  // Same year - show month and day
  if (dateInTz.getFullYear() === nowInTz.getFullYear()) {
    return date.toLocaleDateString('en-US', { 
      month: 'short', 
      day: 'numeric',
      timeZone: timezone 
    })
  }
  
  // Different year - show full date
  return date.toLocaleDateString('en-US', { 
    month: 'short', 
    day: 'numeric', 
    year: 'numeric',
    timeZone: timezone 
  })
}

/**
 * Format a date string with full details including time
 */
export function formatFullDate(dateStr: string): string {
  const authStore = useAuthStore()
  const timezone = authStore.user?.timezone || 'UTC'
  
  return new Date(dateStr).toLocaleString('en-US', {
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    timeZone: timezone
  })
}

/**
 * Format a date for file listings (shorter format)
 */
export function formatFileDate(dateStr: string): string {
  const authStore = useAuthStore()
  const timezone = authStore.user?.timezone || 'UTC'
  
  return new Date(dateStr).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    timeZone: timezone
  })
}

/**
 * Format a datetime for admin logs and detailed views
 */
export function formatDateTime(dateStr: string): string {
  const authStore = useAuthStore()
  const timezone = authStore.user?.timezone || 'UTC'
  
  return new Date(dateStr).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    second: '2-digit',
    timeZone: timezone
  })
}
