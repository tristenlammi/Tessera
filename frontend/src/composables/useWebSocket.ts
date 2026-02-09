import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import api from '@/api'

export type EventType =
  | 'connected'
  | 'ack'
  | 'file:created'
  | 'file:updated'
  | 'file:deleted'
  | 'file:moved'
  | 'file:restored'
  | 'upload:started'
  | 'upload:progress'
  | 'upload:complete'
  | 'share:created'
  | 'share:revoked'
  | 'storage:updated'

export interface WebSocketEvent {
  type: EventType
  payload: any
  folder_id?: string | null
  user_id: string
  timestamp: number
}

type EventHandler = (event: WebSocketEvent) => void

class WebSocketClient {
  private ws: WebSocket | null = null
  private reconnectAttempts = 0
  private baseDelay = 1000       // Start at 1s
  private maxDelay = 60000       // Cap at 60s
  private handlers: Map<EventType | '*', Set<EventHandler>> = new Map()
  private subscribedFolders: Set<string | null> = new Set()
  private isConnecting = false
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null

  public isConnected = ref(false)
  public connectionError = ref<string | null>(null)

  async connect() {
    if (this.isConnecting) return
    this.isConnecting = true
    
    try {
      await this.doConnect()
    } finally {
      this.isConnecting = false
    }
  }

  private async doConnect() {
    // Get a short-lived ticket for WebSocket authentication
    // This avoids exposing the JWT in the URL
    let ticket: string
    try {
      const response = await api.get('/auth/ws-ticket')
      ticket = response.data.ticket
    } catch (e) {
      console.error('[WS] Failed to get WebSocket ticket:', e)
      this.connectionError.value = 'Failed to authenticate WebSocket connection'
      this.scheduleReconnect()
      return
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/ws?ticket=${ticket}`

    try {
      this.ws = new WebSocket(wsUrl)

      this.ws.onopen = () => {
        console.log('[WS] Connected')
        this.isConnected.value = true
        this.connectionError.value = null
        this.reconnectAttempts = 0

        // Resubscribe to folders
        this.subscribedFolders.forEach(folderId => {
          this.subscribe(folderId)
        })
      }

      this.ws.onmessage = (event) => {
        try {
          const data: WebSocketEvent = JSON.parse(event.data)
          this.handleEvent(data)
        } catch (e) {
          console.error('[WS] Failed to parse message:', e)
        }
      }

      this.ws.onclose = (event) => {
        console.log('[WS] Disconnected:', event.code, event.reason)
        this.isConnected.value = false
        this.ws = null

        // Reconnect unless it was a clean close initiated by us
        if (event.code !== 1000) {
          this.scheduleReconnect()
        }
      }

      this.ws.onerror = (error) => {
        console.error('[WS] Error:', error)
        this.connectionError.value = 'WebSocket connection error'
      }
    } catch (e) {
      console.error('[WS] Failed to connect:', e)
      this.connectionError.value = 'Failed to establish WebSocket connection'
      this.scheduleReconnect()
    }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return // Already scheduled

    this.reconnectAttempts++
    // Exponential backoff with jitter, capped at maxDelay
    const baseWait = Math.min(this.baseDelay * Math.pow(2, this.reconnectAttempts - 1), this.maxDelay)
    const jitter = Math.random() * baseWait * 0.3 // Up to 30% jitter
    const delay = Math.round(baseWait + jitter)

    console.log(`[WS] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`)
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
    }, delay)
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.close(1000, 'User disconnect')
      this.ws = null
    }
    this.isConnected.value = false
    this.reconnectAttempts = 0
    this.subscribedFolders.clear()
  }

  subscribe(folderId: string | null) {
    this.subscribedFolders.add(folderId)
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        action: 'subscribe',
        folder_id: folderId
      }))
    }
  }

  unsubscribe(folderId: string | null) {
    this.subscribedFolders.delete(folderId)
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({
        action: 'unsubscribe',
        folder_id: folderId
      }))
    }
  }

  on(eventType: EventType | '*', handler: EventHandler) {
    if (!this.handlers.has(eventType)) {
      this.handlers.set(eventType, new Set())
    }
    this.handlers.get(eventType)!.add(handler)

    // Return unsubscribe function
    return () => {
      this.handlers.get(eventType)?.delete(handler)
    }
  }

  off(eventType: EventType | '*', handler: EventHandler) {
    this.handlers.get(eventType)?.delete(handler)
  }

  private handleEvent(event: WebSocketEvent) {
    // Call specific handlers
    this.handlers.get(event.type)?.forEach(handler => handler(event))
    
    // Call wildcard handlers
    this.handlers.get('*')?.forEach(handler => handler(event))
  }

  ping() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action: 'ping' }))
    }
  }
}

// Singleton instance
const wsClient = new WebSocketClient()

export function useWebSocket() {
  const authStore = useAuthStore()

  onMounted(() => {
    if (authStore.isAuthenticated) {
      wsClient.connect()
    }
  })

  // Watch for auth changes
  watch(() => authStore.isAuthenticated, (isAuth) => {
    if (isAuth) {
      wsClient.connect()
    } else {
      wsClient.disconnect()
    }
  })

  return {
    isConnected: wsClient.isConnected,
    connectionError: wsClient.connectionError,
    subscribe: wsClient.subscribe.bind(wsClient),
    unsubscribe: wsClient.unsubscribe.bind(wsClient),
    on: wsClient.on.bind(wsClient),
    off: wsClient.off.bind(wsClient),
    disconnect: wsClient.disconnect.bind(wsClient)
  }
}

// Export client for direct usage
export { wsClient }
