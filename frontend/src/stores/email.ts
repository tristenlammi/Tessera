import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export interface EmailAccount {
  id: string
  name: string
  email_address: string
  imap_host: string
  imap_port: number
  imap_use_tls: boolean
  smtp_host: string
  smtp_port: number
  smtp_use_tls: boolean
  last_sync_at: string | null
  sync_error: string | null
  is_default: boolean
  signature: string
  send_delay: number
  created_at: string
  updated_at: string
}

export interface EmailFolder {
  id: string
  account_id: string
  parent_id?: string | null
  name: string
  remote_name: string
  folder_type: string | null
  delimiter?: string | null
  sort_order?: number
  unread_count: number
  total_count: number
  children?: EmailFolder[]
  created_at: string
  updated_at: string
}

export interface EmailAddress {
  name?: string
  address: string
}

export interface EmailAttachment {
  id: string
  email_id: string
  filename: string
  content_type: string
  size: number
  content_id: string
  is_inline: boolean
  created_at: string
}

export interface Email {
  id: string
  account_id: string
  folder_id: string
  message_id: string
  uid: number
  subject: string
  from_address: string
  from_name: string
  to: EmailAddress[]
  cc: EmailAddress[]
  reply_to: string
  in_reply_to: string
  text_body: string
  html_body: string
  snippet: string
  is_read: boolean
  is_starred: boolean
  is_answered: boolean
  is_draft: boolean
  has_attachments: boolean
  date: string
  received_at: string
  created_at: string
  updated_at: string
  labels?: EmailLabel[]
  attachments?: EmailAttachment[]
}

export interface EmailListItem {
  id: string
  thread_id?: string
  subject: string
  from_address: string
  from_name: string
  snippet: string
  date: string
  is_read: boolean
  is_starred: boolean
  has_attachments: boolean
  thread_count?: number
}

// Email thread/conversation
export interface EmailThread {
  thread_id: string
  subject: string
  snippet: string
  latest_date: string
  email_count: number
  unread_count: number
  has_attachments: boolean
  is_starred: boolean
  participants?: EmailAddress[]
  latest_email?: EmailListItem
  emails?: EmailListItem[]
  expanded?: boolean
}

export interface ComposeEmail {
  to: string[]
  cc?: string[]
  bcc?: string[]
  subject: string
  body: string
  is_html?: boolean
  reply_to?: string
  attachments?: File[]
}

export interface EmailLabel {
  id: string
  account_id: string
  name: string
  color: string
  is_system: boolean
  email_count?: number
  created_at: string
  updated_at: string
}

export interface RuleCondition {
  field: 'from' | 'to' | 'subject' | 'body'
  operator: 'contains' | 'equals' | 'startswith' | 'endswith' | 'regex'
  value: string
}

export interface RuleAction {
  type: 'label' | 'move' | 'star' | 'mark_read' | 'archive' | 'delete'
  value: string // label_id, folder_id, or empty
}

export interface EmailRule {
  id: string
  account_id: string
  name: string
  is_enabled: boolean
  priority: number
  match_type: 'any' | 'all'
  conditions: RuleCondition[]
  actions: RuleAction[]
  stop_processing: boolean
  created_at: string
  updated_at: string
}

// Virtual folder types for special views
export type VirtualFolder = 'starred' | 'drafts' | null

export interface EmailDraft {
  id: string
  account_id: string
  to: EmailAddress[]
  cc: EmailAddress[]
  bcc: EmailAddress[]
  subject: string
  body: string
  is_html: boolean
  reply_to_id: string
  created_at: string
  updated_at: string
}

export interface UndoSendItem {
  sendId: string
  subject: string
  delay: number
  sentAt: number
  timer: number | null
}

export interface SyncProgress {
  status: 'idle' | 'syncing' | 'complete' | 'error'
  currentFolder: string
  totalFolders: number
  completedFolders: number
  currentFolderEmails: number
  syncedEmails: number
  message: string
}

export const useEmailStore = defineStore('email', () => {
  // State
  const accounts = ref<EmailAccount[]>([])
  const currentAccount = ref<EmailAccount | null>(null)
  const folders = ref<EmailFolder[]>([])
  const currentFolder = ref<EmailFolder | null>(null)
  const currentVirtualFolder = ref<VirtualFolder>(null)
  const emails = ref<EmailListItem[]>([])
  const currentEmail = ref<Email | null>(null)
  const labels = ref<EmailLabel[]>([])
  const currentLabel = ref<EmailLabel | null>(null)
  const rules = ref<EmailRule[]>([])
  const loading = ref(false)
  const loadingEmail = ref(false)
  const loadingMore = ref(false)
  const syncing = ref(false)
  const syncProgress = ref<SyncProgress>({
    status: 'idle',
    currentFolder: '',
    totalFolders: 0,
    completedFolders: 0,
    currentFolderEmails: 0,
    syncedEmails: 0,
    message: ''
  })
  const error = ref<string | null>(null)
  const starredCount = ref(0)
  const draftCount = ref(0)
  
  // Threading state
  const threads = ref<EmailThread[]>([])
  const threadViewEnabled = ref(true) // Enable thread view by default
  const totalThreads = ref(0)
  
  // Thread conversation state (Gmail-style full thread view)
  const threadConversation = ref<Email[]>([])
  const currentThreadId = ref<string | null>(null)
  const loadingConversation = ref(false)
  
  // Pagination state
  const currentPage = ref(1)
  const pageSize = ref(50)
  const hasMoreEmails = ref(true)
  
  // Batch selection state
  const selectedEmailIds = ref<Set<string>>(new Set())
  const selectAllMode = ref(false)
  
  // Undo-send state
  const undoSendItems = ref<UndoSendItem[]>([])
  
  // Draft auto-save state
  const currentDraft = ref<EmailDraft | null>(null)
  let draftAutoSaveTimer: number | null = null
  const DRAFT_AUTO_SAVE_INTERVAL = 10000 // 10 seconds
  
  // Auto-refresh interval
  let autoRefreshInterval: number | null = null
  const autoRefreshEnabled = ref(true)
  const AUTO_REFRESH_INTERVAL = 30000 // 30 seconds

  // Helper to find folder by type in a tree
  function findFolderByType(folderList: EmailFolder[], folderType: string): EmailFolder | undefined {
    for (const folder of folderList) {
      if (folder.folder_type === folderType) return folder
      if (folder.children && folder.children.length > 0) {
        const found = findFolderByType(folder.children, folderType)
        if (found) return found
      }
    }
    return undefined
  }

  // Computed
  const inboxFolder = computed(() => findFolderByType(folders.value, 'inbox'))
  const sentFolder = computed(() => findFolderByType(folders.value, 'sent'))
  const draftsFolder = computed(() => findFolderByType(folders.value, 'drafts'))
  const trashFolder = computed(() => findFolderByType(folders.value, 'trash'))
  const spamFolder = computed(() => findFolderByType(folders.value, 'spam'))
  const customLabels = computed(() => labels.value.filter(l => !l.is_system))
  
  const unreadCount = computed(() => {
    // Count unread across all folders (recursively)
    function countUnread(folderList: EmailFolder[]): number {
      let sum = 0
      for (const f of folderList) {
        sum += f.unread_count
        if (f.children && f.children.length > 0) {
          sum += countUnread(f.children)
        }
      }
      return sum
    }
    return countUnread(folders.value)
  })

  // Actions
  async function fetchAccounts() {
    loading.value = true
    error.value = null
    try {
      const response = await api.get('/email/accounts')
      accounts.value = response.data
      if (accounts.value.length > 0 && !currentAccount.value) {
        const defaultAccount = accounts.value.find(a => a.is_default) || accounts.value[0]
        await selectAccount(defaultAccount)
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch accounts'
    } finally {
      loading.value = false
    }
  }

  async function createAccount(data: {
    name: string
    email_address: string
    imap_host: string
    imap_port: number
    imap_username: string
    imap_password: string
    imap_use_tls: boolean
    smtp_host: string
    smtp_port: number
    smtp_username: string
    smtp_password: string
    smtp_use_tls: boolean
  }) {
    loading.value = true
    error.value = null
    try {
      const response = await api.post('/email/accounts', data)
      accounts.value.push(response.data)
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to create account'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function deleteAccount(accountId: string) {
    try {
      await api.delete(`/email/accounts/${accountId}`)
      accounts.value = accounts.value.filter(a => a.id !== accountId)
      if (currentAccount.value?.id === accountId) {
        currentAccount.value = accounts.value[0] || null
        if (currentAccount.value) {
          await selectAccount(currentAccount.value)
        } else {
          folders.value = []
          emails.value = []
          labels.value = []
          rules.value = []
        }
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to delete account'
      throw e
    }
  }

  async function selectAccount(account: EmailAccount) {
    currentAccount.value = account
    currentVirtualFolder.value = null
    currentLabel.value = null
    await Promise.all([
      fetchFolders(account.id),
      fetchLabels(account.id),
      fetchCounts(account.id)
    ])
    // Select inbox by default
    if (inboxFolder.value) {
      await selectFolder(inboxFolder.value)
    } else if (folders.value.length > 0) {
      await selectFolder(folders.value[0])
    }
  }

  async function syncAccount(accountId?: string) {
    const id = accountId || currentAccount.value?.id
    if (!id) return

    syncing.value = true
    error.value = null
    syncProgress.value = {
      status: 'syncing',
      currentFolder: '',
      totalFolders: 0,
      completedFolders: 0,
      currentFolderEmails: 0,
      syncedEmails: 0,
      message: 'Syncing emails...'
    }
    
    try {
      // Start sync and wait for completion
      await api.post(`/email/accounts/${id}/sync`)
      
      syncProgress.value.status = 'complete'
      syncProgress.value.message = 'Sync complete!'
      
      // Refresh folders, emails, and counts after sync
      await Promise.all([
        fetchFolders(id),
        fetchCounts(id)
      ])
      if (currentFolder.value) {
        await fetchEmails(currentFolder.value.id)
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to sync account'
      syncProgress.value.status = 'error'
      syncProgress.value.message = error.value || 'Sync failed'
    } finally {
      syncing.value = false
    }
  }

  async function fetchFolders(accountId: string) {
    try {
      // Use tree endpoint to get hierarchical folders
      const response = await api.get(`/email/accounts/${accountId}/folders/tree`)
      folders.value = response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch folders'
    }
  }

  // Flatten folders tree for easy lookups (e.g., trashFolder computed)
  function flattenFolders(folderList: EmailFolder[]): EmailFolder[] {
    const result: EmailFolder[] = []
    for (const folder of folderList) {
      result.push(folder)
      if (folder.children && folder.children.length > 0) {
        result.push(...flattenFolders(folder.children))
      }
    }
    return result
  }

  const allFoldersFlat = computed(() => flattenFolders(folders.value))

  async function createFolder(name: string, parentId?: string | null) {
    if (!currentAccount.value) return
    try {
      const response = await api.post(`/email/accounts/${currentAccount.value.id}/folders`, {
        name,
        parent_id: parentId || null
      })
      await fetchFolders(currentAccount.value.id)
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to create folder'
      throw e
    }
  }

  async function updateFolder(folderId: string, updates: { name?: string; parent_id?: string | null; sort_order?: number }) {
    try {
      await api.put(`/email/folders/${folderId}`, updates)
      if (currentAccount.value) {
        await fetchFolders(currentAccount.value.id)
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update folder'
      throw e
    }
  }

  async function deleteFolder(folderId: string) {
    try {
      await api.delete(`/email/folders/${folderId}`)
      if (currentAccount.value) {
        await fetchFolders(currentAccount.value.id)
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to delete folder'
      throw e
    }
  }

  async function moveFolder(folderId: string, targetId: string | null, position?: 'before' | 'after' | 'into') {
    try {
      // Default to 'into' for nesting (old behavior)
      const pos = position || 'into'
      
      if (pos === 'into') {
        // Nesting: move folder as child of target (or root if targetId is null)
        await api.patch(`/email/folders/${folderId}/move`, { parent_id: targetId })
      } else {
        // Reordering: place before or after target
        await api.patch(`/email/folders/${folderId}/move`, { 
          target_id: targetId,
          position: pos
        })
      }
      
      if (currentAccount.value) {
        await fetchFolders(currentAccount.value.id)
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to move folder'
      throw e
    }
  }

  async function reorderFolders(orders: { id: string; sort_order: number }[]) {
    if (!currentAccount.value) return
    try {
      await api.post(`/email/accounts/${currentAccount.value.id}/folders/reorder`, { orders })
      await fetchFolders(currentAccount.value.id)
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to reorder folders'
      throw e
    }
  }

  async function fetchCounts(accountId: string) {
    try {
      const response = await api.get(`/email/accounts/${accountId}/counts`)
      starredCount.value = response.data.starred || 0
      draftCount.value = response.data.drafts || 0
    } catch (e: any) {
      // Silent fail for counts
    }
  }

  async function selectFolder(folder: EmailFolder) {
    currentFolder.value = folder
    currentVirtualFolder.value = null
    currentLabel.value = null
    currentEmail.value = null
    
    // Use thread view by default, with fallback to flat list
    if (threadViewEnabled.value) {
      await fetchThreads(folder.id)
      // Also fetch flat emails as fallback/for individual email access
      await fetchEmails(folder.id)
    } else {
      await fetchEmails(folder.id)
    }
  }

  async function selectVirtualFolder(type: VirtualFolder) {
    if (!currentAccount.value) return
    currentFolder.value = null
    currentVirtualFolder.value = type
    currentLabel.value = null
    currentEmail.value = null
    threads.value = []
    currentThreadId.value = null
    threadConversation.value = []
    
    loading.value = true
    try {
      if (type === 'starred') {
        const response = await api.get(`/email/accounts/${currentAccount.value.id}/starred`)
        emails.value = response.data
      } else if (type === 'drafts') {
        const response = await api.get(`/email/accounts/${currentAccount.value.id}/drafts`)
        emails.value = response.data
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch emails'
    } finally {
      loading.value = false
    }
  }

  async function selectLabel(label: EmailLabel) {
    currentFolder.value = null
    currentVirtualFolder.value = null
    currentLabel.value = label
    currentEmail.value = null
    threads.value = []
    currentThreadId.value = null
    threadConversation.value = []
    
    loading.value = true
    try {
      const response = await api.get(`/email/labels/${label.id}/emails`)
      emails.value = response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch emails'
    } finally {
      loading.value = false
    }
  }

  async function fetchEmails(folderId: string, page = 1, limit = 50) {
    loading.value = true
    try {
      const response = await api.get(`/email/folders/${folderId}/emails`, {
        params: { page, pageSize: limit }
      })
      emails.value = response.data
      // Reset pagination state
      currentPage.value = 1
      hasMoreEmails.value = response.data.length >= limit
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch emails'
    } finally {
      loading.value = false
    }
  }

  // Thread-related functions
  async function fetchThreads(folderId: string, page = 1, limit = 50) {
    loading.value = true
    try {
      const response = await api.get(`/email/folders/${folderId}/threads`, {
        params: { page, pageSize: limit }
      })
      threads.value = response.data.threads || []
      totalThreads.value = response.data.total || 0
      // Reset pagination state
      currentPage.value = 1
      hasMoreEmails.value = threads.value.length >= limit
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch threads'
      // Fallback to flat email list if threads fail
      threads.value = []
    } finally {
      loading.value = false
    }
  }

  async function fetchThreadEmails(threadId: string): Promise<EmailListItem[]> {
    try {
      const response = await api.get(`/email/threads/${threadId}/emails`)
      return response.data || []
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch thread emails'
      return []
    }
  }

  async function expandThread(threadId: string) {
    const thread = threads.value.find(t => t.thread_id === threadId)
    if (thread) {
      if (!thread.emails || thread.emails.length === 0) {
        thread.emails = await fetchThreadEmails(threadId)
      }
      thread.expanded = true
    }
  }

  function collapseThread(threadId: string) {
    const thread = threads.value.find(t => t.thread_id === threadId)
    if (thread) {
      thread.expanded = false
    }
  }

  function toggleThread(threadId: string) {
    const thread = threads.value.find(t => t.thread_id === threadId)
    if (thread) {
      if (thread.expanded) {
        collapseThread(threadId)
      } else {
        expandThread(threadId)
      }
    }
  }

  async function reindexThreads(accountId: string) {
    try {
      await api.post(`/email/accounts/${accountId}/threads/reindex`)
      // Refresh current view
      if (currentFolder.value) {
        if (threadViewEnabled.value) {
          await fetchThreads(currentFolder.value.id)
        } else {
          await fetchEmails(currentFolder.value.id)
        }
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to reindex threads'
    }
  }

  // Open a thread conversation (Gmail-style): fetch all full emails in the thread
  async function openThread(threadId: string) {
    loadingConversation.value = true
    currentThreadId.value = threadId
    currentEmail.value = null
    threadConversation.value = []
    try {
      const response = await api.get(`/email/threads/${threadId}/conversation`)
      threadConversation.value = response.data || []
      // Update read status in the thread list
      const thread = threads.value.find(t => t.thread_id === threadId)
      if (thread) {
        thread.unread_count = 0
      }
      // Update read status in the flat email list too
      for (const email of threadConversation.value) {
        const listEmail = emails.value.find(e => e.id === email.id)
        if (listEmail) {
          listEmail.is_read = true
        }
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch thread conversation'
    } finally {
      loadingConversation.value = false
    }
  }

  // Close the thread conversation view
  function closeThread() {
    threadConversation.value = []
    currentThreadId.value = null
  }

  function setThreadViewEnabled(enabled: boolean) {
    threadViewEnabled.value = enabled
    // Refresh current view with new mode
    if (currentFolder.value) {
      if (enabled) {
        fetchThreads(currentFolder.value.id)
      } else {
        fetchEmails(currentFolder.value.id)
      }
    }
  }

  async function loadMoreEmails() {
    if (!currentFolder.value || loadingMore.value || !hasMoreEmails.value) return
    
    loadingMore.value = true
    try {
      const nextPage = currentPage.value + 1
      const response = await api.get(`/email/folders/${currentFolder.value.id}/emails`, {
        params: { page: nextPage, pageSize: pageSize.value }
      })
      
      if (response.data.length > 0) {
        emails.value = [...emails.value, ...response.data]
        currentPage.value = nextPage
        hasMoreEmails.value = response.data.length >= pageSize.value
      } else {
        hasMoreEmails.value = false
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to load more emails'
    } finally {
      loadingMore.value = false
    }
  }

  async function fetchEmail(emailId: string) {
    loadingEmail.value = true
    // Clear thread conversation when viewing a single email
    threadConversation.value = []
    currentThreadId.value = null
    try {
      const response = await api.get(`/email/emails/${emailId}`)
      currentEmail.value = response.data
      // Update local read state (backend marks as read and returns labels)
      const listEmail = emails.value.find(e => e.id === emailId)
      if (listEmail && !listEmail.is_read) {
        listEmail.is_read = true
        if (currentFolder.value) {
          currentFolder.value.unread_count = Math.max(0, currentFolder.value.unread_count - 1)
        }
      }
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch email'
      throw e
    } finally {
      loadingEmail.value = false
    }
  }

  // Prefetch an email in the background (hover cache warmup)
  const prefetchCache = new Map<string, boolean>()
  async function prefetchEmail(emailId: string) {
    if (prefetchCache.has(emailId) || currentEmail.value?.id === emailId) return
    prefetchCache.set(emailId, true)
    try {
      await api.get(`/email/emails/${emailId}`)
    } catch {
      prefetchCache.delete(emailId)
    }
  }

  async function markAsRead(emailId: string, read: boolean) {
    try {
      await api.patch(`/email/emails/${emailId}/read`, { is_read: read })
      // Update local state
      const email = emails.value.find(e => e.id === emailId)
      if (email) {
        email.is_read = read
      }
      if (currentEmail.value?.id === emailId) {
        currentEmail.value.is_read = read
      }
      // Update folder unread count
      if (currentFolder.value) {
        currentFolder.value.unread_count += read ? -1 : 1
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update read status'
    }
  }

  async function markFolderAsRead(folderId: string) {
    try {
      console.log('[markFolderAsRead] Calling API for folder:', folderId)
      const response = await api.post(`/email/folders/${folderId}/read`)
      console.log('[markFolderAsRead] API response:', response.data)
      
      // Update folder unread count
      const folder = folders.value.find(f => f.id === folderId)
      if (folder) {
        folder.unread_count = 0
      }
      if (currentFolder.value?.id === folderId) {
        currentFolder.value.unread_count = 0
        // If we're viewing this folder, update all email read states
        emails.value = emails.value.map(email => ({
          ...email,
          is_read: true
        }))
      }
      
      return response.data.updated as number
    } catch (e: any) {
      console.error('[markFolderAsRead] Error:', e)
      error.value = e.response?.data?.error || 'Failed to mark folder as read'
      throw e
    }
  }

  async function markAsStarred(emailId: string, starred: boolean) {
    try {
      await api.patch(`/email/emails/${emailId}/star`, { is_starred: starred })
      const email = emails.value.find(e => e.id === emailId)
      if (email) {
        email.is_starred = starred
      }
      if (currentEmail.value?.id === emailId) {
        currentEmail.value.is_starred = starred
      }
      // Update starred count
      if (currentAccount.value) {
        starredCount.value += starred ? 1 : -1
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update starred status'
    }
  }

  async function moveEmail(emailId: string, targetFolderId: string) {
    try {
      await api.patch(`/email/emails/${emailId}/move`, { folder_id: targetFolderId })
      // Remove from current flat list
      emails.value = emails.value.filter(e => e.id !== emailId)
      // Remove from threads list (if this email is the latest in any thread)
      threads.value = threads.value.filter(t => t.latest_email?.id !== emailId)
      // Remove from open conversation view
      if (threadConversation.value.length > 0) {
        threadConversation.value = threadConversation.value.filter(e => e.id !== emailId)
        if (threadConversation.value.length === 0) {
          currentThreadId.value = null
        }
      }
      if (currentEmail.value?.id === emailId) {
        currentEmail.value = null
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to move email'
    }
  }

  async function deleteEmail(emailId: string) {
    try {
      await api.delete(`/email/emails/${emailId}`)
      // Remove from current flat list
      emails.value = emails.value.filter(e => e.id !== emailId)
      // Remove from threads list
      threads.value = threads.value.filter(t => t.latest_email?.id !== emailId)
      // Remove from open conversation view
      if (threadConversation.value.length > 0) {
        threadConversation.value = threadConversation.value.filter(e => e.id !== emailId)
        if (threadConversation.value.length === 0) {
          currentThreadId.value = null
        }
      }
      if (currentEmail.value?.id === emailId) {
        currentEmail.value = null
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to delete email'
    }
  }

  async function searchEmails(accountId: string, query: string) {
    loading.value = true
    try {
      const response = await api.get(`/email/accounts/${accountId}/search`, {
        params: { q: query }
      })
      emails.value = response.data
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to search emails'
      return []
    } finally {
      loading.value = false
    }
  }

  // ============ Attachments ============

  async function downloadAttachment(attachmentId: string, filename: string) {
    try {
      const response = await api.get(`/email/attachments/${attachmentId}/download`, {
        responseType: 'blob'
      })
      
      // Create download link
      const url = window.URL.createObjectURL(new Blob([response.data]))
      const link = document.createElement('a')
      link.href = url
      link.setAttribute('download', filename)
      document.body.appendChild(link)
      link.click()
      link.remove()
      window.URL.revokeObjectURL(url)
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to download attachment'
      throw e
    }
  }

  async function getAttachmentBlob(attachmentId: string): Promise<Blob> {
    const response = await api.get(`/email/attachments/${attachmentId}/download`, {
      responseType: 'blob'
    })
    return new Blob([response.data])
  }

  async function sendEmail(email: ComposeEmail) {
    loading.value = true
    try {
      if (!currentAccount.value) {
        throw new Error('No account selected')
      }

      const hasAttachments = email.attachments && email.attachments.length > 0

      // Build FormData if attachments present, otherwise use JSON
      function buildFormData(): FormData {
        const fd = new FormData()
        fd.append('account_id', currentAccount.value!.id)
        fd.append('subject', email.subject)
        fd.append('body', email.body)
        fd.append('is_html', String(email.is_html || false))
        if (email.reply_to) fd.append('reply_to', email.reply_to)
        for (const addr of email.to) fd.append('to', addr)
        if (email.cc) for (const addr of email.cc) fd.append('cc', addr)
        if (email.bcc) for (const addr of email.bcc) fd.append('bcc', addr)
        if (email.attachments) {
          for (const file of email.attachments) {
            fd.append('attachments', file, file.name)
          }
        }
        return fd
      }

      function buildJSON(): object {
        return { account_id: currentAccount.value!.id, ...email, attachments: undefined }
      }
      
      // If account has send_delay, use queue endpoint for undo-send
      if (currentAccount.value.send_delay > 0) {
        const response = hasAttachments
          ? await api.post('/email/send/queue', buildFormData(), { headers: { 'Content-Type': 'multipart/form-data' } })
          : await api.post('/email/send/queue', buildJSON())
        
        if (response.data.send_id) {
          // Add to undo-send list
          const item: UndoSendItem = {
            sendId: response.data.send_id,
            subject: email.subject || '(No Subject)',
            delay: response.data.delay,
            sentAt: Date.now(),
            timer: null
          }
          // Auto-remove after delay expires
          item.timer = window.setTimeout(() => {
            undoSendItems.value = undoSendItems.value.filter(i => i.sendId !== item.sendId)
          }, (response.data.delay + 1) * 1000)
          undoSendItems.value.push(item)
        }
        
        return response.data
      }
      
      if (hasAttachments) {
        await api.post('/email/send', buildFormData(), { headers: { 'Content-Type': 'multipart/form-data' } })
      } else {
        await api.post('/email/send', buildJSON())
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to send email'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function cancelSend(sendId: string) {
    try {
      await api.post(`/email/send/${sendId}/cancel`)
      const item = undoSendItems.value.find(i => i.sendId === sendId)
      if (item?.timer) {
        clearTimeout(item.timer)
      }
      undoSendItems.value = undoSendItems.value.filter(i => i.sendId !== sendId)
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to cancel send'
      throw e
    }
  }

  // ============ Batch Operations ============

  function toggleEmailSelection(emailId: string) {
    if (selectedEmailIds.value.has(emailId)) {
      selectedEmailIds.value.delete(emailId)
    } else {
      selectedEmailIds.value.add(emailId)
    }
    // Force reactivity
    selectedEmailIds.value = new Set(selectedEmailIds.value)
  }

  function selectAllEmails() {
    if (selectedEmailIds.value.size === emails.value.length) {
      // Deselect all
      selectedEmailIds.value = new Set()
      selectAllMode.value = false
    } else {
      // Select all
      selectedEmailIds.value = new Set(emails.value.map(e => e.id))
      selectAllMode.value = true
    }
  }

  function clearSelection() {
    selectedEmailIds.value = new Set()
    selectAllMode.value = false
  }

  async function batchMarkAsRead(isRead: boolean) {
    const ids = Array.from(selectedEmailIds.value)
    if (ids.length === 0) return
    try {
      await api.post('/email/batch/read', { email_ids: ids, is_read: isRead })
      // Update local state
      emails.value = emails.value.map(e => 
        selectedEmailIds.value.has(e.id) ? { ...e, is_read: isRead } : e
      )
      clearSelection()
      // Refresh folder counts
      if (currentAccount.value) {
        await fetchFolders(currentAccount.value.id)
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update emails'
    }
  }

  async function batchMarkAsStarred(isStarred: boolean) {
    const ids = Array.from(selectedEmailIds.value)
    if (ids.length === 0) return
    try {
      await api.post('/email/batch/star', { email_ids: ids, is_starred: isStarred })
      emails.value = emails.value.map(e => 
        selectedEmailIds.value.has(e.id) ? { ...e, is_starred: isStarred } : e
      )
      clearSelection()
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update emails'
    }
  }

  async function batchMoveEmails(targetFolderId: string) {
    const ids = Array.from(selectedEmailIds.value)
    if (ids.length === 0) return
    try {
      await api.post('/email/batch/move', { email_ids: ids, folder_id: targetFolderId })
      emails.value = emails.value.filter(e => !selectedEmailIds.value.has(e.id))
      clearSelection()
      if (currentAccount.value) {
        await fetchFolders(currentAccount.value.id)
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to move emails'
    }
  }

  async function batchDeleteEmails() {
    const ids = Array.from(selectedEmailIds.value)
    if (ids.length === 0) return
    try {
      await api.post('/email/batch/delete', { email_ids: ids })
      emails.value = emails.value.filter(e => !selectedEmailIds.value.has(e.id))
      clearSelection()
      if (currentAccount.value) {
        await fetchFolders(currentAccount.value.id)
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to delete emails'
    }
  }

  async function batchAssignLabel(labelId: string) {
    const ids = Array.from(selectedEmailIds.value)
    if (ids.length === 0) return
    try {
      await api.post('/email/batch/label', { email_ids: ids, label_id: labelId })
      clearSelection()
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to assign label'
    }
  }

  // ============ Compose Drafts ============

  async function saveDraft(draft: Partial<EmailDraft>) {
    if (!currentAccount.value) return null
    try {
      const response = await api.post('/email/drafts', {
        ...draft,
        account_id: currentAccount.value.id
      })
      currentDraft.value = response.data
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to save draft'
      return null
    }
  }

  async function fetchComposeDrafts() {
    if (!currentAccount.value) return []
    try {
      const response = await api.get(`/email/accounts/${currentAccount.value.id}/compose-drafts`)
      return response.data || []
    } catch (e: any) {
      return []
    }
  }

  async function deleteDraft(draftId: string) {
    try {
      await api.delete(`/email/drafts/${draftId}`)
      if (currentDraft.value?.id === draftId) {
        currentDraft.value = null
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to delete draft'
    }
  }

  function startDraftAutoSave(getDraftData: () => Partial<EmailDraft>) {
    stopDraftAutoSave()
    draftAutoSaveTimer = window.setInterval(async () => {
      const data = getDraftData()
      // Only save if there's meaningful content
      if (data.subject || data.body || (data.to && data.to.length > 0)) {
        await saveDraft({ ...data, id: currentDraft.value?.id || '' })
      }
    }, DRAFT_AUTO_SAVE_INTERVAL)
  }

  function stopDraftAutoSave() {
    if (draftAutoSaveTimer) {
      clearInterval(draftAutoSaveTimer)
      draftAutoSaveTimer = null
    }
  }

  // ============ Account Settings ============

  async function updateSignature(signature: string) {
    if (!currentAccount.value) return
    try {
      await api.put(`/email/accounts/${currentAccount.value.id}/signature`, { signature })
      currentAccount.value.signature = signature
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update signature'
      throw e
    }
  }

  async function updateSendDelay(sendDelay: number) {
    if (!currentAccount.value) return
    try {
      await api.put(`/email/accounts/${currentAccount.value.id}/send-delay`, { send_delay: sendDelay })
      currentAccount.value.send_delay = sendDelay
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update send delay'
      throw e
    }
  }

  // ============ Labels ============

  async function fetchLabels(accountId: string) {
    try {
      const response = await api.get(`/email/accounts/${accountId}/labels`)
      labels.value = response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch labels'
    }
  }

  async function createLabel(name: string, color: string = '#6B7280') {
    if (!currentAccount.value) return
    try {
      const response = await api.post(`/email/accounts/${currentAccount.value.id}/labels`, { name, color })
      labels.value.push(response.data)
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to create label'
      throw e
    }
  }

  async function updateLabel(labelId: string, name: string, color: string) {
    try {
      const response = await api.put(`/email/labels/${labelId}`, { name, color })
      const index = labels.value.findIndex(l => l.id === labelId)
      if (index !== -1) {
        labels.value[index] = response.data
      }
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update label'
      throw e
    }
  }

  async function deleteLabel(labelId: string) {
    try {
      await api.delete(`/email/labels/${labelId}`)
      labels.value = labels.value.filter(l => l.id !== labelId)
      if (currentLabel.value?.id === labelId) {
        currentLabel.value = null
        // Switch to inbox
        if (inboxFolder.value) {
          await selectFolder(inboxFolder.value)
        }
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to delete label'
      throw e
    }
  }

  async function assignLabelToEmail(emailId: string, labelId: string) {
    try {
      await api.post(`/email/emails/${emailId}/labels/${labelId}`)
      // Update current email labels if viewing this email
      if (currentEmail.value?.id === emailId) {
        const label = labels.value.find(l => l.id === labelId)
        if (label && currentEmail.value.labels) {
          currentEmail.value.labels.push(label)
        }
      }
      // Update label count
      const label = labels.value.find(l => l.id === labelId)
      if (label) {
        label.email_count = (label.email_count || 0) + 1
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to assign label'
    }
  }

  async function removeLabelFromEmail(emailId: string, labelId: string) {
    try {
      await api.delete(`/email/emails/${emailId}/labels/${labelId}`)
      // Update current email labels if viewing this email
      if (currentEmail.value?.id === emailId && currentEmail.value.labels) {
        currentEmail.value.labels = currentEmail.value.labels.filter(l => l.id !== labelId)
      }
      // Update label count
      const label = labels.value.find(l => l.id === labelId)
      if (label && label.email_count) {
        label.email_count--
      }
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to remove label'
    }
  }

  // ============ Rules ============

  async function fetchRules(accountId: string) {
    try {
      const response = await api.get(`/email/accounts/${accountId}/rules`)
      rules.value = response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to fetch rules'
    }
  }

  async function createRule(rule: Omit<EmailRule, 'id' | 'account_id' | 'created_at' | 'updated_at'>) {
    if (!currentAccount.value) return
    try {
      const response = await api.post(`/email/accounts/${currentAccount.value.id}/rules`, rule)
      rules.value.push(response.data)
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to create rule'
      throw e
    }
  }

  async function updateRule(ruleId: string, updates: Partial<EmailRule>) {
    try {
      const response = await api.put(`/email/rules/${ruleId}`, updates)
      const index = rules.value.findIndex(r => r.id === ruleId)
      if (index !== -1) {
        rules.value[index] = response.data
      }
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to update rule'
      throw e
    }
  }

  async function deleteRule(ruleId: string) {
    try {
      await api.delete(`/email/rules/${ruleId}`)
      rules.value = rules.value.filter(r => r.id !== ruleId)
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to delete rule'
      throw e
    }
  }

  async function runRuleNow(ruleId: string): Promise<{ affected: number; message: string }> {
    try {
      const response = await api.post(`/email/rules/${ruleId}/run`)
      // Refresh emails after running the rule
      if (currentFolder.value) {
        await fetchEmails(currentFolder.value.id)
      }
      return response.data
    } catch (e: any) {
      error.value = e.response?.data?.error || 'Failed to run rule'
      throw e
    }
  }

  function clearError() {
    error.value = null
  }

  // Auto-refresh functions
  async function refreshCurrentFolder() {
    if (!currentFolder.value || syncing.value) return
    
    try {
      // Fetch folder tree to get updated counts
      if (currentAccount.value) {
        const foldersResponse = await api.get(`/email/accounts/${currentAccount.value.id}/folders/tree`)
        folders.value = foldersResponse.data
        
        // Update current folder reference with new data
        const updatedFolder = allFoldersFlat.value.find((f: EmailFolder) => f.id === currentFolder.value?.id)
        if (updatedFolder) {
          currentFolder.value = updatedFolder
        }
      }
      
      // Refresh emails in current folder
      if (currentFolder.value) {
        const response = await api.get(`/email/folders/${currentFolder.value.id}/emails`, {
          params: { page: 1, limit: 50 }
        })
        emails.value = response.data
      }
    } catch (e) {
      // Silently fail for background refresh
      console.error('Auto-refresh failed:', e)
    }
  }

  function startAutoRefresh() {
    if (autoRefreshInterval) return
    
    autoRefreshInterval = window.setInterval(() => {
      if (autoRefreshEnabled.value && currentAccount.value) {
        refreshCurrentFolder()
      }
    }, AUTO_REFRESH_INTERVAL)
    
    console.log('[Email] Auto-refresh started (every 30s)')
  }

  function stopAutoRefresh() {
    if (autoRefreshInterval) {
      clearInterval(autoRefreshInterval)
      autoRefreshInterval = null
      console.log('[Email] Auto-refresh stopped')
    }
  }

  function setAutoRefresh(enabled: boolean) {
    autoRefreshEnabled.value = enabled
    if (enabled) {
      startAutoRefresh()
    } else {
      stopAutoRefresh()
    }
  }

  return {
    // State
    accounts,
    currentAccount,
    folders,
    currentFolder,
    currentVirtualFolder,
    emails,
    currentEmail,
    labels,
    currentLabel,
    rules,
    loading,
    loadingEmail,
    syncing,
    syncProgress,
    error,
    starredCount,
    draftCount,
    // Threading state
    threads,
    threadViewEnabled,
    totalThreads,
    threadConversation,
    currentThreadId,
    loadingConversation,
    
    // Computed
    inboxFolder,
    sentFolder,
    draftsFolder,
    trashFolder,
    spamFolder,
    customLabels,
    unreadCount,
    
    // Actions
    fetchAccounts,
    createAccount,
    deleteAccount,
    selectAccount,
    syncAccount,
    fetchFolders,
    fetchCounts,
    selectFolder,
    selectVirtualFolder,
    selectLabel,
    fetchEmails,
    loadMoreEmails,
    loadingMore,
    hasMoreEmails,
    fetchEmail,
    markAsRead,
    markFolderAsRead,
    markAsStarred,
    moveEmail,
    deleteEmail,
    searchEmails,
    sendEmail,
    cancelSend,
    // Batch operations
    selectedEmailIds,
    selectAllMode,
    toggleEmailSelection,
    selectAllEmails,
    clearSelection,
    batchMarkAsRead,
    batchMarkAsStarred,
    batchMoveEmails,
    batchDeleteEmails,
    batchAssignLabel,
    // Undo send
    undoSendItems,
    // Compose drafts
    currentDraft,
    saveDraft,
    fetchComposeDrafts,
    deleteDraft,
    startDraftAutoSave,
    stopDraftAutoSave,
    // Account settings
    updateSignature,
    updateSendDelay,
    // Thread actions
    fetchThreads,
    fetchThreadEmails,
    expandThread,
    collapseThread,
    toggleThread,
    reindexThreads,
    setThreadViewEnabled,
    openThread,
    closeThread,
    // Folder management
    createFolder,
    updateFolder,
    deleteFolder,
    moveFolder,
    reorderFolders,
    // Labels
    fetchLabels,
    createLabel,
    updateLabel,
    deleteLabel,
    assignLabelToEmail,
    removeLabelFromEmail,
    // Rules
    fetchRules,
    createRule,
    updateRule,
    deleteRule,
    runRuleNow,
    // Attachments
    downloadAttachment,
    getAttachmentBlob,
    // Prefetch
    prefetchEmail,
    // Helpers
    allFoldersFlat,
    clearError,
    // Auto-refresh
    autoRefreshEnabled,
    startAutoRefresh,
    stopAutoRefresh,
    setAutoRefresh,
    refreshCurrentFolder
  }
})
