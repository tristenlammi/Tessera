import { ref, onMounted } from 'vue'

export interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

const isInstalled = ref(false)
const canInstall = ref(false)
const isOnline = ref(navigator.onLine)
const updateAvailable = ref(false)

let deferredPrompt: BeforeInstallPromptEvent | null = null
let registration: ServiceWorkerRegistration | null = null

export function usePWA() {
  onMounted(() => {
    // Check if already installed
    if (window.matchMedia('(display-mode: standalone)').matches) {
      isInstalled.value = true
    }

    // Listen for online/offline events
    window.addEventListener('online', () => {
      isOnline.value = true
    })
    window.addEventListener('offline', () => {
      isOnline.value = false
    })

    // Listen for install prompt
    window.addEventListener('beforeinstallprompt', (e) => {
      e.preventDefault()
      deferredPrompt = e as BeforeInstallPromptEvent
      canInstall.value = true
    })

    // Listen for app installed
    window.addEventListener('appinstalled', () => {
      isInstalled.value = true
      canInstall.value = false
      deferredPrompt = null
    })

    // Register service worker
    registerServiceWorker()
  })

  async function registerServiceWorker() {
    if (!('serviceWorker' in navigator)) {
      console.log('Service workers not supported')
      return
    }

    try {
      registration = await navigator.serviceWorker.register('/sw.js', {
        scope: '/',
      })

      console.log('Service Worker registered:', registration.scope)

      // Check for updates
      registration.addEventListener('updatefound', () => {
        const newWorker = registration?.installing
        if (newWorker) {
          newWorker.addEventListener('statechange', () => {
            if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
              updateAvailable.value = true
            }
          })
        }
      })

      // Handle controller change (new SW activated)
      navigator.serviceWorker.addEventListener('controllerchange', () => {
        // Reload to use new service worker
        window.location.reload()
      })
    } catch (error) {
      console.error('Service Worker registration failed:', error)
    }
  }

  async function installApp() {
    if (!deferredPrompt) {
      console.log('No install prompt available')
      return false
    }

    try {
      await deferredPrompt.prompt()
      const { outcome } = await deferredPrompt.userChoice
      
      if (outcome === 'accepted') {
        isInstalled.value = true
        canInstall.value = false
      }
      
      deferredPrompt = null
      return outcome === 'accepted'
    } catch (error) {
      console.error('Install failed:', error)
      return false
    }
  }

  async function updateApp() {
    if (!registration?.waiting) {
      return
    }

    // Tell the waiting service worker to skip waiting
    registration.waiting.postMessage('skipWaiting')
  }

  async function checkForUpdates() {
    if (!registration) {
      return
    }

    try {
      await registration.update()
    } catch (error) {
      console.error('Update check failed:', error)
    }
  }

  // Push notification support
  async function requestNotificationPermission() {
    if (!('Notification' in window)) {
      console.log('Notifications not supported')
      return false
    }

    const permission = await Notification.requestPermission()
    return permission === 'granted'
  }

  async function subscribeToPush() {
    if (!registration) {
      console.log('No service worker registration')
      return null
    }

    try {
      // You would get this from your backend
      const vapidPublicKey = import.meta.env.VITE_VAPID_PUBLIC_KEY

      if (!vapidPublicKey) {
        console.log('VAPID key not configured')
        return null
      }

      const subscription = await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapidPublicKey),
      })

      return subscription
    } catch (error) {
      console.error('Push subscription failed:', error)
      return null
    }
  }

  return {
    isInstalled,
    canInstall,
    isOnline,
    updateAvailable,
    installApp,
    updateApp,
    checkForUpdates,
    requestNotificationPermission,
    subscribeToPush,
  }
}

// Helper to convert VAPID key
function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = '='.repeat((4 - (base64String.length % 4)) % 4)
  const base64 = (base64String + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = window.atob(base64)
  const outputArray = new Uint8Array(rawData.length)

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i)
  }

  return outputArray
}
