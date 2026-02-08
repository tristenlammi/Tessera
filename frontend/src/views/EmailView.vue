<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import DOMPurify from 'dompurify'
import { useEmailStore, type Email, type EmailListItem, type EmailLabel, type EmailFolder, type EmailThread } from '@/stores/email'
import { useTasksStore } from '@/stores/tasks'
import api from '@/api'
import EmailAccountSetup from '@/components/email/EmailAccountSetup.vue'
import EmailCompose from '@/components/email/EmailCompose.vue'
import EmailLabelManager from '@/components/email/EmailLabelManager.vue'
import EmailRuleManager from '@/components/email/EmailRuleManager.vue'
import EmailFolderTree from '@/components/email/EmailFolderTree.vue'
import SaveToFolderPicker from '@/components/SaveToFolderPicker.vue'
import { formatDate as formatDateUtil, formatFullDate as formatFullDateUtil } from '@/utils/dateFormat'

const emailStore = useEmailStore()
const tasksStore = useTasksStore()

const showAccountSetup = ref(false)
const showCompose = ref(false)
const showLabelManager = ref(false)
const showRuleManager = ref(false)
const searchQuery = ref('')
const replyMode = ref<'reply' | 'replyAll' | 'forward' | null>(null)
const showLabelMenu = ref(false)
const labelMenuEmailId = ref<string | null>(null)

// Folder management
const showFolderModal = ref(false)
const folderModalMode = ref<'create' | 'rename'>('create')
const folderModalName = ref('')
const folderModalParentId = ref<string | null>(null)
const editingFolder = ref<EmailFolder | null>(null)
const showFolderContextMenu = ref(false)
const contextMenuFolder = ref<EmailFolder | null>(null)
const contextMenuPosition = ref({ x: 0, y: 0 })
const showDeleteConfirm = ref(false)
const rootDropZoneActive = ref(false)
const emailListContainer = ref<HTMLElement | null>(null)

// Email context menu
const showEmailContextMenu = ref(false)
const contextMenuEmail = ref<EmailListItem | null>(null)

// Create task from email modal
const showCreateTaskModal = ref(false)
const taskFromEmail = ref<{ emailId: string; emailSubject: string } | null>(null)
const newTaskForm = ref({
  title: '',
  description: '',
  priority: 'medium' as 'low' | 'medium' | 'high',
  dueDate: '',
  groupId: null as string | null
})
const addToCalendar = ref(false)

onMounted(async () => {
  await emailStore.fetchAccounts()
  // Fetch task groups for create task modal
  await tasksStore.fetchGroups()
  // Start auto-refresh for new emails
  emailStore.startAutoRefresh()
  // Add keyboard shortcut listener
  window.addEventListener('keydown', handleKeyboardShortcut)
})

onUnmounted(() => {
  // Stop auto-refresh when leaving email view
  emailStore.stopAutoRefresh()
  window.removeEventListener('keydown', handleKeyboardShortcut)
})

const currentFolder = computed(() => emailStore.currentFolder)
const currentVirtualFolder = computed(() => emailStore.currentVirtualFolder)
const currentLabel = computed(() => emailStore.currentLabel)
const currentEmail = computed(() => emailStore.currentEmail)
const emails = computed(() => emailStore.emails)
const threads = computed(() => emailStore.threads)
const threadViewEnabled = computed(() => emailStore.threadViewEnabled)

// Filter out Gmail system folders and flatten their children to the top level
const folders = computed(() => {
  const result: typeof emailStore.folders = []
  
  function processFolder(folder: typeof emailStore.folders[0]) {
    // Skip the [Gmail] parent folder but include its children at top level
    if (folder.name === '[Gmail]' || folder.remote_name === '[Gmail]') {
      // Move children up to the parent level
      if (folder.children) {
        for (const child of folder.children) {
          result.push({ ...child, parent_id: null })
        }
      }
    } else {
      result.push(folder)
    }
  }
  
  for (const folder of emailStore.folders) {
    processFolder(folder)
  }
  
  return result
})

const labels = computed(() => emailStore.labels)
const customLabels = computed(() => emailStore.customLabels)
const hasAccounts = computed(() => emailStore.accounts.length > 0)
const starredCount = computed(() => emailStore.starredCount)
const draftCount = computed(() => emailStore.draftCount)

// Batch selection
const selectedEmailIds = computed(() => emailStore.selectedEmailIds)
const hasSelection = computed(() => emailStore.selectedEmailIds.size > 0)
const undoSendItems = computed(() => emailStore.undoSendItems)

// Keyboard shortcuts
function handleKeyboardShortcut(e: KeyboardEvent) {
  // Don't handle shortcuts when typing in inputs
  const tag = (e.target as HTMLElement)?.tagName?.toLowerCase()
  if (tag === 'input' || tag === 'textarea' || tag === 'select') return
  
  const emailList = emails.value
  const currentIdx = emailList.findIndex(em => em.id === currentEmail.value?.id)

  switch (e.key) {
    case 'j': // Next email
      if (currentIdx < emailList.length - 1) {
        selectEmail(emailList[currentIdx + 1])
      }
      break
    case 'k': // Previous email
      if (currentIdx > 0) {
        selectEmail(emailList[currentIdx - 1])
      }
      break
    case 'r': // Reply
      if (currentEmail.value && !e.shiftKey) {
        e.preventDefault()
        openCompose('reply')
      }
      break
    case 'R': // Reply all (shift+r)
      if (currentEmail.value) {
        e.preventDefault()
        openCompose('replyAll')
      }
      break
    case 'f': // Forward
      if (currentEmail.value) {
        e.preventDefault()
        openCompose('forward')
      }
      break
    case 'c': // Compose new
      if (!e.ctrlKey && !e.metaKey) {
        e.preventDefault()
        openCompose('new')
      }
      break
    case 'e': // Archive (move to archive if available)
      if (currentEmail.value) {
        const archiveFolder = emailStore.allFoldersFlat.find(
          (f: EmailFolder) => f.folder_type === 'archive'
        )
        if (archiveFolder) {
          emailStore.moveEmail(currentEmail.value.id, archiveFolder.id)
        }
      }
      break
    case '#': // Delete (move to trash)
    case 'Delete':
      if (currentEmail.value) {
        const trash = emailStore.trashFolder
        if (trash) {
          emailStore.moveEmail(currentEmail.value.id, trash.id)
        } else {
          emailStore.deleteEmail(currentEmail.value.id)
        }
      }
      break
    case 'u': // Mark unread
      if (currentEmail.value) {
        emailStore.markAsRead(currentEmail.value.id, false)
      }
      break
    case 's': // Star/unstar
      if (currentEmail.value && !e.ctrlKey && !e.metaKey) {
        emailStore.markAsStarred(currentEmail.value.id, !currentEmail.value.is_starred)
      }
      break
    case 'Escape':
      if (hasSelection.value) {
        emailStore.clearSelection()
      }
      break
  }
}

function toggleEmailSelection(emailId: string, event: Event) {
  event.stopPropagation()
  emailStore.toggleEmailSelection(emailId)
}

function isEmailSelected(emailId: string): boolean {
  return emailStore.selectedEmailIds.has(emailId)
}

async function selectEmail(email: EmailListItem) {
  await emailStore.fetchEmail(email.id)
}

// Hover prefetch with 150ms debounce
let prefetchTimer: ReturnType<typeof setTimeout> | null = null
function handleEmailHover(emailId: string) {
  if (prefetchTimer) clearTimeout(prefetchTimer)
  prefetchTimer = setTimeout(() => {
    emailStore.prefetchEmail(emailId)
  }, 150)
}
function cancelPrefetch() {
  if (prefetchTimer) {
    clearTimeout(prefetchTimer)
    prefetchTimer = null
  }
}

function toggleStar(email: EmailListItem, event: Event) {
  event.stopPropagation()
  emailStore.markAsStarred(email.id, !email.is_starred)
}

// Thread functions
function openThreadConversation(thread: EmailThread, event: Event) {
  event.stopPropagation()
  // For single-email threads, just open the email directly
  if (thread.email_count <= 1 && thread.latest_email) {
    emailStore.fetchEmail(thread.latest_email.id)
    return
  }
  // For multi-email threads, open the full conversation view
  expandedConversationEmails.value = new Set()
  emailStore.openThread(thread.thread_id).then(() => {
    // Auto-expand the last (most recent) email
    const conversation = emailStore.threadConversation
    if (conversation.length > 0) {
      expandedConversationEmails.value.add(conversation[conversation.length - 1].id)
    }
  })
}

function selectThreadEmail(emailId: string, event: Event) {
  event.stopPropagation()
  emailStore.fetchEmail(emailId)
}

function toggleThreadView() {
  emailStore.setThreadViewEnabled(!emailStore.threadViewEnabled)
}

async function handleSearch() {
  if (!emailStore.currentAccount || !searchQuery.value.trim()) return
  await emailStore.searchEmails(emailStore.currentAccount.id, searchQuery.value)
}

function clearSearch() {
  searchQuery.value = ''
  if (emailStore.currentFolder) {
    emailStore.fetchEmails(emailStore.currentFolder.id)
  }
}

function handleEmailListScroll(event: Event) {
  const container = event.target as HTMLElement
  if (!container) return
  
  // Load more when scrolled to bottom (with 100px threshold)
  const { scrollTop, scrollHeight, clientHeight } = container
  if (scrollHeight - scrollTop - clientHeight < 100) {
    emailStore.loadMoreEmails()
  }
}

function openCompose(mode: 'new' | 'reply' | 'replyAll' | 'forward' = 'new') {
  if (mode !== 'new') {
    replyMode.value = mode
    // If in conversation view and no currentEmail set, use the last email in the conversation
    if (!emailStore.currentEmail && emailStore.threadConversation.length > 0) {
      emailStore.currentEmail = emailStore.threadConversation[emailStore.threadConversation.length - 1]
    }
  } else {
    replyMode.value = null
  }
  showCompose.value = true
}

function closeCompose() {
  showCompose.value = false
  replyMode.value = null
}

function selectStarred() {
  emailStore.selectVirtualFolder('starred')
}

function selectDrafts() {
  emailStore.selectVirtualFolder('drafts')
}

function selectLabel(label: EmailLabel) {
  emailStore.selectLabel(label)
}

function handleFolderSelect(folder: EmailFolder) {
  emailStore.selectFolder(folder)
}

// Folder context menu
function handleFolderContextMenu(folder: EmailFolder, event: MouseEvent) {
  contextMenuFolder.value = folder
  contextMenuPosition.value = { x: event.clientX, y: event.clientY }
  showFolderContextMenu.value = true
}

function closeFolderContextMenu() {
  showFolderContextMenu.value = false
  contextMenuFolder.value = null
}

async function handleMarkFolderAsRead() {
  if (!contextMenuFolder.value) return
  try {
    await emailStore.markFolderAsRead(contextMenuFolder.value.id)
  } catch (e) {
    console.error('Failed to mark folder as read:', e)
  }
  closeFolderContextMenu()
}

// Folder CRUD
function openCreateFolderModal(parentId: string | null = null) {
  folderModalMode.value = 'create'
  folderModalName.value = ''
  folderModalParentId.value = parentId
  editingFolder.value = null
  showFolderModal.value = true
  closeFolderContextMenu()
}

function openRenameFolderModal(folder: EmailFolder) {
  folderModalMode.value = 'rename'
  folderModalName.value = folder.name
  editingFolder.value = folder
  showFolderModal.value = true
  closeFolderContextMenu()
}

async function saveFolderModal() {
  if (!folderModalName.value.trim()) return
  
  if (folderModalMode.value === 'create') {
    await emailStore.createFolder(folderModalName.value, folderModalParentId.value)
  } else if (editingFolder.value) {
    await emailStore.updateFolder(editingFolder.value.id, { name: folderModalName.value })
  }
  
  showFolderModal.value = false
  folderModalName.value = ''
  editingFolder.value = null
}

function confirmDeleteFolder(folder: EmailFolder) {
  contextMenuFolder.value = folder
  showDeleteConfirm.value = true
  showFolderContextMenu.value = false  // Close context menu but keep contextMenuFolder
}

async function handleDeleteFolder() {
  if (!contextMenuFolder.value) return
  await emailStore.deleteFolder(contextMenuFolder.value.id)
  showDeleteConfirm.value = false
  contextMenuFolder.value = null
}

// Email context menu functions
function handleEmailContextMenu(email: EmailListItem, event: MouseEvent) {
  event.preventDefault()
  contextMenuEmail.value = email
  contextMenuPosition.value = { x: event.clientX, y: event.clientY }
  showEmailContextMenu.value = true
}

function closeEmailContextMenu() {
  showEmailContextMenu.value = false
  contextMenuEmail.value = null
}

function openCreateTaskFromEmail(email: EmailListItem) {
  closeEmailContextMenu()
  taskFromEmail.value = {
    emailId: email.id,
    emailSubject: email.subject || '(No subject)'
  }
  newTaskForm.value = {
    title: `Follow up: ${email.subject || '(No subject)'}`,
    description: `From: ${email.from_name || email.from_address}\n\nOriginal email linked to this task.`,
    priority: 'medium',
    dueDate: '',
    groupId: null
  }
  addToCalendar.value = false
  showCreateTaskModal.value = true
}

async function saveTaskFromEmail() {
  if (!taskFromEmail.value || !newTaskForm.value.title.trim()) return
  
  try {
    let dueDate: string | null = null
    if (newTaskForm.value.dueDate) {
      dueDate = new Date(newTaskForm.value.dueDate + 'T12:00:00').toISOString()
    }

    const createdTask = await tasksStore.createTask({
      title: newTaskForm.value.title,
      description: newTaskForm.value.description,
      priority: newTaskForm.value.priority,
      dueDate,
      groupId: newTaskForm.value.groupId,
      status: 'todo',
      tags: [],
      recurrence: null,
      linkedEmailId: taskFromEmail.value.emailId,
      linkedEmailSubject: taskFromEmail.value.emailSubject
    })

    // Create calendar event if option is checked and there's a due date
    if (addToCalendar.value && dueDate) {
      try {
        const eventColor = newTaskForm.value.priority === 'high' ? '#ef4444'
          : newTaskForm.value.priority === 'low' ? '#10b981'
          : '#f59e0b'

        await api.post('/calendar/events', {
          title: newTaskForm.value.title,
          description: newTaskForm.value.description || `Task: ${newTaskForm.value.title}`,
          startDate: dueDate,
          endDate: dueDate,
          allDay: true,
          color: eventColor,
          linkedTaskId: createdTask?.id || null
        })
      } catch (calErr) {
        console.error('Failed to create calendar event:', calErr)
      }
    }

    showCreateTaskModal.value = false
    taskFromEmail.value = null
  } catch (err) {
    console.error('Failed to create task:', err)
  }
}

function cancelCreateTask() {
  showCreateTaskModal.value = false
  taskFromEmail.value = null
}

async function handleFolderDrop(folderId: string, targetFolderId: string | null, position: 'before' | 'after' | 'into') {
  await emailStore.moveFolder(folderId, targetFolderId, position)
}

async function handleEmailDropToFolder(emailId: string, targetFolderId: string) {
  try {
    await emailStore.moveEmail(emailId, targetFolderId)
  } catch (e) {
    console.error('Failed to move email to folder:', e)
  }
}

function handleEmailDragStart(event: DragEvent, email: EmailListItem) {
  event.dataTransfer?.setData('application/email-id', email.id)
  event.dataTransfer?.setData('text/plain', email.subject || 'Email')
  event.dataTransfer!.effectAllowed = 'move'
}

function handleThreadDragStart(event: DragEvent, thread: EmailThread) {
  // For threads, we'll drag the latest email's ID (or thread_id as fallback)
  const emailId = thread.latest_email?.id || thread.thread_id
  event.dataTransfer?.setData('application/email-id', emailId)
  event.dataTransfer?.setData('text/plain', thread.subject || 'Email')
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
  }
}

async function handleRootDrop(event: DragEvent) {
  rootDropZoneActive.value = false
  // Ignore email drops - this zone is only for un-nesting folders
  if (event.dataTransfer?.types.includes('application/email-id')) {
    return
  }
  const folderId = event.dataTransfer?.getData('text/plain')
  if (folderId) {
    await emailStore.moveFolder(folderId, null)
  }
}

async function deleteEmailFromList(emailId: string) {
  // Move to trash if we have a trash folder, otherwise delete
  try {
    if (emailStore.trashFolder) {
      console.log('Moving to trash:', emailId, 'Trash folder:', emailStore.trashFolder.id)
      await emailStore.moveEmail(emailId, emailStore.trashFolder.id)
    } else {
      console.log('Deleting email (no trash folder):', emailId)
      await emailStore.deleteEmail(emailId)
    }
  } catch (e) {
    console.error('Delete/move failed:', e)
  }
}

function openLabelMenu(emailId: string, event: Event) {
  event.stopPropagation()
  labelMenuEmailId.value = emailId
  showLabelMenu.value = true
}

function closeLabelMenu() {
  showLabelMenu.value = false
  labelMenuEmailId.value = null
}

async function toggleEmailLabel(labelId: string) {
  const email = currentEmail.value
  if (!email) return
  
  if (email.labels?.some(l => l.id === labelId)) {
    await emailStore.removeLabelFromEmail(email.id, labelId)
  } else {
    await emailStore.assignLabelToEmail(email.id, labelId)
  }
}

async function handleDelete() {
  if (!currentEmail.value) return
  await emailStore.deleteEmail(currentEmail.value.id)
}

async function moveToTrash() {
  const email = currentEmail.value || (emailStore.threadConversation.length > 0 ? emailStore.threadConversation[emailStore.threadConversation.length - 1] : null)
  if (!email) return
  try {
    if (emailStore.trashFolder) {
      await emailStore.moveEmail(email.id, emailStore.trashFolder.id)
    } else {
      await emailStore.deleteEmail(email.id)
    }
    // Close conversation view if we were in one
    if (emailStore.threadConversation.length > 0) {
      closeConversation()
    }
  } catch (e) {
    console.error('Failed to move to trash:', e)
  }
}

function formatDate(dateStr: string): string {
  return formatDateUtil(dateStr)
}

function formatFullDate(dateStr: string): string {
  return formatFullDateUtil(dateStr)
}

// Attachment handling
import type { EmailAttachment } from '@/stores/email'
import { useFilesStore } from '@/stores/files'

const filesStore = useFilesStore()
const showAttachmentContextMenu = ref(false)
const attachmentContextMenuPosition = ref({ x: 0, y: 0 })
const contextMenuAttachment = ref<EmailAttachment | null>(null)
const showSaveToFilesModal = ref(false)
const savingAttachment = ref(false)

function getAttachmentIcon(contentType: string): string {
  if (contentType.startsWith('image/')) return '🖼️'
  if (contentType.startsWith('video/')) return '🎬'
  if (contentType.startsWith('audio/')) return '🎵'
  if (contentType === 'application/pdf') return '📄'
  if (contentType.includes('zip') || contentType.includes('archive') || contentType.includes('compressed')) return '📦'
  if (contentType.includes('spreadsheet') || contentType.includes('excel')) return '📊'
  if (contentType.includes('word') || contentType.includes('document')) return '📝'
  if (contentType.includes('presentation') || contentType.includes('powerpoint')) return '📽️'
  if (contentType.startsWith('text/')) return '📃'
  return '📎'
}

function formatFileSize(bytes: number): string {
  if (!bytes || bytes === 0) return 'Unknown size'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  while (bytes >= 1024 && i < units.length - 1) {
    bytes /= 1024
    i++
  }
  return `${bytes.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

function handleAttachmentClick(attachment: EmailAttachment) {
  handleAttachmentDownload(attachment)
}

async function handleAttachmentDownload(attachment: EmailAttachment) {
  try {
    await emailStore.downloadAttachment(attachment.id, attachment.filename)
  } catch (e) {
    console.error('Failed to download attachment:', e)
  }
}

function showAttachmentMenu(event: MouseEvent, attachment: EmailAttachment) {
  contextMenuAttachment.value = attachment
  attachmentContextMenuPosition.value = { x: event.clientX, y: event.clientY }
  showAttachmentContextMenu.value = true
}

function closeAttachmentContextMenu() {
  showAttachmentContextMenu.value = false
  contextMenuAttachment.value = null
}

async function handleSaveAttachmentToFiles() {
  if (!contextMenuAttachment.value) return
  showAttachmentContextMenu.value = false
  showSaveToFilesModal.value = true
}

async function confirmSaveToFiles(folderId: string | null) {
  if (!contextMenuAttachment.value) return
  
  savingAttachment.value = true
  try {
    // Get the attachment blob
    const blob = await emailStore.getAttachmentBlob(contextMenuAttachment.value.id)
    
    // Create a File object from the blob
    const file = new File([blob], contextMenuAttachment.value.filename, { 
      type: contextMenuAttachment.value.content_type 
    })
    
    // Upload to files using the files store
    await filesStore.uploadFile(file, folderId)
    
    showSaveToFilesModal.value = false
    contextMenuAttachment.value = null
  } catch (e) {
    console.error('Failed to save attachment to files:', e)
  } finally {
    savingAttachment.value = false
  }
}

function cancelSaveToFiles() {
  showSaveToFilesModal.value = false
  contextMenuAttachment.value = null
}

function getFolderIcon(folderType: string | null): string {
  switch (folderType) {
    case 'inbox': return 'inbox'
    case 'sent': return 'paper-airplane'
    case 'drafts': return 'pencil'
    case 'trash': return 'trash'
    case 'spam': return 'exclamation'
    case 'archive': return 'archive'
    default: return 'folder'
  }
}

// Email HTML sanitization and iframe handling
const emailIframe = ref<HTMLIFrameElement | null>(null)

// Track which emails are expanded in conversation view
const expandedConversationEmails = ref<Set<string>>(new Set())

function sanitizeHtmlBody(htmlBody: string): string {
  if (!htmlBody) return ''
  
  const cleanHtml = DOMPurify.sanitize(htmlBody, {
    FORBID_TAGS: ['script', 'object', 'embed', 'form', 'input', 'button'],
    FORBID_ATTR: ['onerror', 'onload', 'onclick', 'onmouseover', 'onfocus', 'onblur'],
    ALLOW_DATA_ATTR: false
  })
  
  const isDark = document.documentElement.classList.contains('dark')
  
  return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    * { box-sizing: border-box; }
    html, body { 
      margin: 0; 
      padding: 0;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
      font-size: 14px;
      line-height: 1.6;
      color: ${isDark ? '#e5e7eb' : '#374151'};
      background: ${isDark ? '#1f2937' : '#ffffff'};
      overflow-x: hidden;
    }
    body { padding: 16px; }
    img { max-width: 100%; height: auto; }
    a { color: #2563eb; }
    pre, code { 
      white-space: pre-wrap; 
      word-wrap: break-word;
      background: ${isDark ? '#374151' : '#f3f4f6'};
      padding: 2px 4px;
      border-radius: 4px;
    }
    table { max-width: 100%; border-collapse: collapse; }
    td, th { padding: 8px; }
    blockquote {
      margin: 8px 0;
      padding-left: 16px;
      border-left: 3px solid ${isDark ? '#4b5563' : '#d1d5db'};
      color: ${isDark ? '#9ca3af' : '#6b7280'};
    }
  </style>
</head>
<body>${cleanHtml}</body>
</html>`
}

const sanitizedEmailHtml = computed(() => {
  if (!currentEmail.value?.html_body) return ''
  return sanitizeHtmlBody(currentEmail.value.html_body)
})

// Check if a conversation email is expanded
function isConversationEmailExpanded(emailId: string): boolean {
  return expandedConversationEmails.value.has(emailId)
}

// Toggle expand/collapse of an email in conversation view
function toggleConversationEmail(emailId: string) {
  if (expandedConversationEmails.value.has(emailId)) {
    expandedConversationEmails.value.delete(emailId)
  } else {
    expandedConversationEmails.value.add(emailId)
  }
  // Force reactivity
  expandedConversationEmails.value = new Set(expandedConversationEmails.value)
}

// Get avatar color based on email address (deterministic)
function getAvatarColor(address: string): string {
  const colors = [
    'bg-blue-500', 'bg-green-500', 'bg-purple-500', 'bg-orange-500', 
    'bg-pink-500', 'bg-teal-500', 'bg-indigo-500', 'bg-red-500',
    'bg-cyan-500', 'bg-amber-500'
  ]
  let hash = 0
  for (let i = 0; i < address.length; i++) {
    hash = address.charCodeAt(i) + ((hash << 5) - hash)
  }
  return colors[Math.abs(hash) % colors.length]
}

function resizeIframe(event: Event) {
  const iframe = event.target as HTMLIFrameElement
  if (iframe && iframe.contentWindow) {
    try {
      // Set initial height then adjust based on content
      iframe.style.height = '200px'
      const contentHeight = iframe.contentWindow.document.body.scrollHeight
      // Use min-height so the iframe can still flex-grow beyond its content
      iframe.style.minHeight = Math.max(contentHeight + 32, 200) + 'px'
      iframe.style.height = ''
    } catch (e) {
      // Cross-origin issues - use a default min-height
      iframe.style.minHeight = '500px'
    }
  }
}

// Resize iframe for conversation emails (fixed height, no flex-grow)
function resizeConversationIframe(event: Event) {
  const iframe = event.target as HTMLIFrameElement
  if (iframe && iframe.contentWindow) {
    try {
      iframe.style.height = '200px'
      const contentHeight = iframe.contentWindow.document.body.scrollHeight
      iframe.style.height = Math.max(contentHeight + 32, 200) + 'px'
    } catch (e) {
      iframe.style.height = '500px'
    }
  }
}

// Close the thread conversation and go back
function closeConversation() {
  emailStore.closeThread()
  emailStore.currentEmail = null
}
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50 dark:bg-gray-900">
    <!-- No accounts - show setup -->
    <div v-if="!hasAccounts && !emailStore.loading" class="flex-1 flex items-center justify-center p-8">
      <div class="text-center max-w-md">
        <svg class="w-24 h-24 mx-auto mb-6 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
        </svg>
        <h2 class="text-2xl font-semibold text-gray-900 dark:text-white mb-2">
          Welcome to Tessera Email
        </h2>
        <p class="text-gray-600 dark:text-gray-400 mb-6">
          Connect your email account to get started. We support any email provider with IMAP/SMTP access.
        </p>
        <button
          @click="showAccountSetup = true"
          class="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          Add Email Account
        </button>
      </div>
    </div>

    <!-- Email interface -->
    <div v-else class="flex-1 flex overflow-hidden">
      <!-- Folders sidebar -->
      <div class="w-56 bg-white dark:bg-gray-800 border-r dark:border-gray-700 flex flex-col">
        <!-- Compose button -->
        <div class="p-3">
          <button
            @click="openCompose('new')"
            class="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            Compose
          </button>
        </div>

        <!-- Folder list -->
        <nav class="flex-1 overflow-y-auto px-2 pb-2">
          <!-- Virtual folders (Starred, Drafts) -->
          <div class="mb-2">
            <button
              @click="selectStarred"
              :class="[
                'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors',
                currentVirtualFolder === 'starred'
                  ? 'bg-yellow-50 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-300'
                  : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
              ]"
            >
              <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
                <path d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
              </svg>
              <span class="flex-1 text-left">Starred</span>
              <span v-if="starredCount > 0" class="px-2 py-0.5 text-xs font-medium bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300 rounded-full">
                {{ starredCount }}
              </span>
            </button>

            <button
              @click="selectDrafts"
              :class="[
                'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors',
                currentVirtualFolder === 'drafts'
                  ? 'bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white'
                  : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
              ]"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
              </svg>
              <span class="flex-1 text-left">Drafts</span>
              <span v-if="draftCount > 0" class="px-2 py-0.5 text-xs font-medium bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-300 rounded-full">
                {{ draftCount }}
              </span>
            </button>
          </div>

          <div class="border-t dark:border-gray-700 my-2"></div>

          <!-- Folders section header -->
          <div class="flex items-center justify-between px-3 py-1">
            <span class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Folders</span>
            <button
              @click="openCreateFolderModal(null)"
              class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
              title="Create folder"
            >
              <svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
            </button>
          </div>

          <!-- Root drop zone to un-nest folders -->
          <div
            class="mx-2 mb-1 py-1 px-3 text-xs text-gray-400 dark:text-gray-500 border-2 border-dashed border-transparent rounded transition-colors"
            :class="{ 'border-blue-400 bg-blue-50 dark:bg-blue-900/20': rootDropZoneActive }"
            @dragover.prevent="rootDropZoneActive = true"
            @dragleave="rootDropZoneActive = false"
            @drop.prevent="handleRootDrop"
          >
            Drop here to move to root
          </div>

          <!-- Folders tree -->
          <EmailFolderTree
            v-for="folder in folders"
            :key="folder.id"
            :folder="folder"
            :level="0"
            :current-folder-id="currentFolder?.id"
            :current-virtual-folder="currentVirtualFolder"
            :current-label-id="currentLabel?.id"
            :draggable="true"
            @select="handleFolderSelect"
            @contextmenu="handleFolderContextMenu"
            @drop="handleFolderDrop"
            @emailDrop="handleEmailDropToFolder"
          />

          <!-- Labels section -->
          <div v-if="customLabels.length > 0 || true" class="mt-4">
            <div class="flex items-center justify-between px-3 py-2">
              <span class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Labels</span>
              <button
                @click="showLabelManager = true"
                class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
                title="Manage labels"
              >
                <svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
                </svg>
              </button>
            </div>
            <button
              v-for="label in customLabels"
              :key="label.id"
              @click="selectLabel(label)"
              :class="[
                'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors',
                currentLabel?.id === label.id
                  ? 'bg-gray-100 dark:bg-gray-700'
                  : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
              ]"
            >
              <span
                class="w-3 h-3 rounded-full flex-shrink-0"
                :style="{ backgroundColor: label.color }"
              ></span>
              <span class="flex-1 text-left truncate">{{ label.name }}</span>
              <span
                v-if="label.email_count && label.email_count > 0"
                class="px-2 py-0.5 text-xs font-medium bg-gray-100 dark:bg-gray-600 text-gray-600 dark:text-gray-300 rounded-full"
              >
                {{ label.email_count }}
              </span>
            </button>
          </div>
        </nav>

        <!-- Account actions -->
        <div class="p-2 border-t dark:border-gray-700">
          <!-- Sync progress (shown when syncing is in progress) -->
          <div v-if="emailStore.syncing" class="mb-2 px-3">
            <div class="flex items-center justify-between text-xs text-gray-600 dark:text-gray-400 mb-1">
              <span class="truncate">{{ emailStore.syncProgress.message || 'Syncing...' }}</span>
              <span v-if="emailStore.syncProgress.totalFolders > 0">
                {{ emailStore.syncProgress.completedFolders }}/{{ emailStore.syncProgress.totalFolders }}
              </span>
            </div>
            <div class="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-1.5">
              <div 
                class="bg-blue-600 h-1.5 rounded-full transition-all duration-300"
                :style="{ width: emailStore.syncProgress.totalFolders > 0 ? `${(emailStore.syncProgress.completedFolders / emailStore.syncProgress.totalFolders) * 100}%` : '0%' }"
              ></div>
            </div>
          </div>
          <button
            @click="showRuleManager = true"
            class="w-full flex items-center justify-center gap-2 px-3 py-2 text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
            </svg>
            Filters & Rules
          </button>
          <button
            @click="showAccountSetup = true"
            class="w-full flex items-center justify-center gap-2 px-3 py-2 text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
            Settings
          </button>
        </div>
      </div>

      <!-- Email list -->
      <div class="w-96 bg-white dark:bg-gray-800 border-r dark:border-gray-700 flex flex-col">
        <!-- Search bar and controls -->
        <div class="p-3 border-b dark:border-gray-700">
          <div class="flex items-center gap-2">
            <div class="relative flex-1">
              <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <input
                v-model="searchQuery"
                @keyup.enter="handleSearch"
                type="text"
                placeholder="Search emails... (from: to: subject: has:attachment)"
                class="w-full pl-10 pr-10 py-2 bg-gray-100 dark:bg-gray-700 border-0 rounded-lg text-sm focus:ring-2 focus:ring-blue-500"
              />
              <button
                v-if="searchQuery"
                @click="clearSearch"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <!-- Thread view toggle -->
            <button
              @click="toggleThreadView"
              :class="[
                'p-2 rounded-lg transition-colors',
                threadViewEnabled 
                  ? 'bg-blue-100 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400' 
                  : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
              ]"
              :title="threadViewEnabled ? 'Conversation view (on)' : 'Conversation view (off)'"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
              </svg>
            </button>
          </div>

          <!-- Batch action toolbar -->
          <div v-if="hasSelection" class="flex items-center gap-2 px-4 py-2 bg-blue-50 dark:bg-blue-900/20 border-b dark:border-gray-700">
            <input
              type="checkbox"
              :checked="selectedEmailIds.size === emails.length"
              @change="emailStore.selectAllEmails()"
              class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span class="text-sm text-blue-700 dark:text-blue-300 font-medium">{{ selectedEmailIds.size }} selected</span>
            <div class="flex items-center gap-1 ml-auto">
              <button @click="emailStore.batchMarkAsRead(true)" class="px-2 py-1 text-xs bg-white dark:bg-gray-700 rounded hover:bg-gray-100 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300" title="Mark as read">
                Read
              </button>
              <button @click="emailStore.batchMarkAsRead(false)" class="px-2 py-1 text-xs bg-white dark:bg-gray-700 rounded hover:bg-gray-100 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-300" title="Mark as unread">
                Unread
              </button>
              <button @click="emailStore.batchMarkAsStarred(true)" class="px-2 py-1 text-xs bg-white dark:bg-gray-700 rounded hover:bg-gray-100 dark:hover:bg-gray-600 text-yellow-600" title="Star">
                ★
              </button>
              <button v-if="emailStore.trashFolder" @click="emailStore.batchMoveEmails(emailStore.trashFolder!.id)" class="px-2 py-1 text-xs bg-white dark:bg-gray-700 rounded hover:bg-red-50 dark:hover:bg-red-900/30 text-red-600" title="Move to trash">
                Trash
              </button>
              <button @click="emailStore.batchDeleteEmails()" class="px-2 py-1 text-xs bg-white dark:bg-gray-700 rounded hover:bg-red-50 dark:hover:bg-red-900/30 text-red-600" title="Delete permanently">
                Delete
              </button>
              <button @click="emailStore.clearSelection()" class="px-2 py-1 text-xs text-gray-500 hover:text-gray-700 dark:hover:text-gray-300" title="Clear selection">
                ✕
              </button>
            </div>
          </div>
        </div>

        <!-- Email list -->
        <div 
          ref="emailListContainer"
          class="flex-1 overflow-y-auto"
          @scroll="handleEmailListScroll"
        >
          <div v-if="emailStore.loading" class="flex items-center justify-center py-8">
            <svg class="w-8 h-8 animate-spin text-blue-600" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>

          <div v-else-if="emails.length === 0 && threads.length === 0" class="flex flex-col items-center justify-center py-12 text-gray-500 dark:text-gray-400">
            <svg class="w-16 h-16 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
            </svg>
            <p class="text-sm">No emails in this folder</p>
          </div>

          <!-- Thread view -->
          <div
            v-else-if="threadViewEnabled && threads.length > 0"
            v-for="thread in threads"
            :key="thread.thread_id"
            class="border-b dark:border-gray-700"
            draggable="true"
            @dragstart="(e) => handleThreadDragStart(e, thread)"
          >
            <!-- Thread row - click opens conversation in reading pane -->
            <div
              @click="openThreadConversation(thread, $event)"
              @mouseenter="thread.latest_email && handleEmailHover(thread.latest_email.id)"
              @mouseleave="cancelPrefetch"
              @contextmenu.prevent="thread.latest_email && handleEmailContextMenu(thread.latest_email, $event)"
              :class="[
                'group w-full text-left p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors cursor-pointer select-none',
                emailStore.currentThreadId === thread.thread_id && 'bg-blue-50 dark:bg-blue-900/20',
                currentEmail?.id === thread.latest_email?.id && 'bg-blue-50 dark:bg-blue-900/20',
                thread.unread_count > 0 && !emailStore.currentThreadId !== thread.thread_id && 'bg-gray-50 dark:bg-gray-800'
              ]"
            >
              <div class="flex items-start gap-3">
                <!-- Star button -->
                <button
                  @click.stop="emailStore.markAsStarred(thread.latest_email?.id || '', !thread.is_starred)"
                  class="mt-1 flex-shrink-0"
                >
                  <svg
                    :class="[
                      'w-5 h-5',
                      thread.is_starred
                        ? 'text-yellow-400 fill-current'
                        : 'text-gray-300 dark:text-gray-600 hover:text-yellow-400'
                    ]"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                  </svg>
                </button>

                <div class="flex-1 min-w-0">
                  <div class="flex items-center justify-between mb-1">
                    <span
                      :class="[
                        'text-sm truncate',
                        thread.unread_count > 0
                          ? 'font-semibold text-gray-900 dark:text-white'
                          : 'text-gray-700 dark:text-gray-300'
                      ]"
                    >
                      {{ thread.latest_email?.from_name || thread.latest_email?.from_address }}
                    </span>
                    <div class="flex items-center gap-2 flex-shrink-0 ml-2">
                      <!-- Thread count badge -->
                      <span
                        v-if="thread.email_count > 1"
                        class="px-1.5 py-0.5 text-xs font-medium bg-gray-200 dark:bg-gray-600 text-gray-600 dark:text-gray-300 rounded"
                      >
                        {{ thread.email_count }}
                      </span>
                      <span class="text-xs text-gray-500 dark:text-gray-400">
                        {{ formatDate(thread.latest_date) }}
                      </span>
                    </div>
                  </div>
                  <p
                    :class="[
                      'text-sm truncate mb-1',
                      thread.unread_count > 0
                        ? 'font-medium text-gray-800 dark:text-gray-200'
                        : 'text-gray-600 dark:text-gray-400'
                    ]"
                  >
                    {{ thread.subject || '(No subject)' }}
                  </p>
                  <p class="text-xs text-gray-500 dark:text-gray-400 truncate">
                    {{ thread.snippet }}
                  </p>
                </div>

                <!-- Attachment indicator & actions -->
                <div class="flex flex-col items-center gap-1 flex-shrink-0">
                  <svg
                    v-if="thread.has_attachments"
                    class="w-4 h-4 text-gray-400"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
                  </svg>
                  <!-- Delete button -->
                  <button
                    @click.stop="deleteEmailFromList(thread.latest_email?.id || '')"
                    class="p-1 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded opacity-0 group-hover:opacity-100 transition-opacity"
                    title="Delete"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Flat email list (non-thread view) -->
          <div
            v-else
            v-for="email in emails"
            :key="email.id"
            draggable="true"
            @dragstart="(e) => handleEmailDragStart(e, email)"
            @click="selectEmail(email)"
            @mouseenter="handleEmailHover(email.id)"
            @mouseleave="cancelPrefetch"
            @contextmenu.prevent="handleEmailContextMenu(email, $event)"
            :class="[
              'group w-full text-left p-4 border-b dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors cursor-grab active:cursor-grabbing select-none',
              currentEmail?.id === email.id && 'bg-blue-50 dark:bg-blue-900/20',
              !email.is_read && 'bg-gray-50 dark:bg-gray-800',
              isEmailSelected(email.id) && 'bg-blue-50 dark:bg-blue-900/30'
            ]"
          >
            <div class="flex items-start gap-3">
              <!-- Selection checkbox -->
              <input
                type="checkbox"
                :checked="isEmailSelected(email.id)"
                @click="(e) => toggleEmailSelection(email.id, e)"
                class="mt-1.5 rounded border-gray-300 text-blue-600 focus:ring-blue-500 flex-shrink-0"
              />
              <!-- Star button -->
              <button
                @click="(e) => toggleStar(email, e)"
                class="mt-1 flex-shrink-0"
              >
                <svg
                  :class="[
                    'w-5 h-5',
                    email.is_starred
                      ? 'text-yellow-400 fill-current'
                      : 'text-gray-300 dark:text-gray-600 hover:text-yellow-400'
                  ]"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                </svg>
              </button>

              <div class="flex-1 min-w-0">
                <div class="flex items-center justify-between mb-1">
                  <span
                    :class="[
                      'text-sm truncate',
                      email.is_read
                        ? 'text-gray-700 dark:text-gray-300'
                        : 'font-semibold text-gray-900 dark:text-white'
                    ]"
                  >
                    {{ email.from_name || email.from_address }}
                  </span>
                  <span class="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0 ml-2">
                    {{ formatDate(email.date) }}
                  </span>
                </div>
                <p
                  :class="[
                    'text-sm truncate mb-1',
                    email.is_read
                      ? 'text-gray-600 dark:text-gray-400'
                      : 'font-medium text-gray-800 dark:text-gray-200'
                  ]"
                >
                  {{ email.subject || '(No subject)' }}
                </p>
                <p class="text-xs text-gray-500 dark:text-gray-400 truncate">
                  {{ email.snippet }}
                </p>
              </div>

              <!-- Action buttons -->
              <div class="flex flex-col items-center gap-1 flex-shrink-0">
                <!-- Attachment indicator -->
                <svg
                  v-if="email.has_attachments"
                  class="w-4 h-4 text-gray-400"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
                </svg>
                
                <!-- Delete button -->
                <button
                  @click.stop="deleteEmailFromList(email.id)"
                  class="p-1 text-gray-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded opacity-50 group-hover:opacity-100 transition-opacity"
                  title="Delete"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              </div>
            </div>
          </div>

          <!-- Load more indicator -->
          <div v-if="emailStore.loadingMore" class="flex items-center justify-center py-4">
            <svg class="w-6 h-6 animate-spin text-blue-600" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>
          <div v-else-if="!emailStore.hasMoreEmails && emails.length > 0" class="text-center py-4 text-sm text-gray-500 dark:text-gray-400">
            No more emails
          </div>
        </div>
      </div>

      <!-- Email content -->
      <div class="flex-1 flex flex-col bg-white dark:bg-gray-800">
        <!-- Loading state -->
        <div v-if="emailStore.loadingEmail || emailStore.loadingConversation" class="flex-1 flex items-center justify-center">
          <svg class="w-8 h-8 animate-spin text-blue-600" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
        </div>

        <!-- No email selected -->
        <div v-else-if="!currentEmail && emailStore.threadConversation.length === 0" class="flex-1 flex items-center justify-center text-gray-500 dark:text-gray-400">
          <div class="text-center">
            <svg class="w-20 h-20 mx-auto mb-4 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
            <p>Select an email to read</p>
          </div>
        </div>

        <!-- ===== THREAD CONVERSATION VIEW (Gmail-style) ===== -->
        <template v-else-if="emailStore.threadConversation.length > 0">
          <!-- Thread header bar -->
          <div class="flex items-center gap-2 p-4 border-b dark:border-gray-700">
            <button
              @click="closeConversation()"
              class="lg:hidden p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
              </svg>
            </button>

            <button @click="openCompose('reply')" class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg" title="Reply">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
              </svg>
            </button>
            <button @click="openCompose('replyAll')" class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg" title="Reply All">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6M8 4h10a8 8 0 018 8v2" />
              </svg>
            </button>
            <button @click="openCompose('forward')" class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg" title="Forward">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 10h-10a8 8 0 00-8 8v2M21 10l-6 6m6-6l-6-6" />
              </svg>
            </button>
            <div class="flex-1"></div>
            <button @click="moveToTrash" class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg text-red-500" title="Delete">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>

          <!-- Thread subject -->
          <div class="px-6 pt-4 pb-2">
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">
              {{ emailStore.threadConversation[0]?.subject || '(No subject)' }}
            </h1>
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
              {{ emailStore.threadConversation.length }} messages in conversation
            </p>
          </div>

          <!-- Scrollable conversation area -->
          <div class="flex-1 overflow-y-auto px-6 pb-6">
            <div
              v-for="(email, index) in emailStore.threadConversation"
              :key="email.id"
              class="mt-3"
            >
              <!-- Collapsed email (click to expand) -->
              <div
                v-if="!isConversationEmailExpanded(email.id)"
                @click="toggleConversationEmail(email.id)"
                class="flex items-center gap-3 p-3 rounded-lg border dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700/50 cursor-pointer transition-colors"
              >
                <div :class="['w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium flex-shrink-0', getAvatarColor(email.from_address)]">
                  {{ (email.from_name || email.from_address).charAt(0).toUpperCase() }}
                </div>
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-medium text-gray-900 dark:text-white truncate">
                      {{ email.from_name || email.from_address }}
                    </span>
                    <span class="text-xs text-gray-500 dark:text-gray-400 flex-shrink-0">
                      {{ formatFullDate(email.date) }}
                    </span>
                  </div>
                  <p class="text-sm text-gray-500 dark:text-gray-400 truncate">
                    {{ email.snippet }}
                  </p>
                </div>
                <svg v-if="email.has_attachments" class="w-4 h-4 text-gray-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
                </svg>
              </div>

              <!-- Expanded email (full view) -->
              <div
                v-else
                class="rounded-lg border dark:border-gray-700 overflow-hidden"
              >
                <!-- Email header -->
                <div
                  @click="toggleConversationEmail(email.id)"
                  class="flex items-start gap-3 p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/30 transition-colors"
                >
                  <div :class="['w-10 h-10 rounded-full flex items-center justify-center text-white font-medium flex-shrink-0', getAvatarColor(email.from_address)]">
                    {{ (email.from_name || email.from_address).charAt(0).toUpperCase() }}
                  </div>
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-2 flex-wrap">
                      <span class="font-medium text-gray-900 dark:text-white">
                        {{ email.from_name || email.from_address }}
                      </span>
                      <span class="text-sm text-gray-500 dark:text-gray-400">
                        &lt;{{ email.from_address }}&gt;
                      </span>
                    </div>
                    <div class="text-sm text-gray-500 dark:text-gray-400">
                      to {{ email.to?.map((t: any) => t.name || t.address).join(', ') }}
                      <span v-if="email.cc?.length">
                        , cc: {{ email.cc?.map((c: any) => c.name || c.address).join(', ') }}
                      </span>
                    </div>
                    <div class="text-xs text-gray-400 dark:text-gray-500 mt-1">
                      {{ formatFullDate(email.date) }}
                    </div>
                  </div>
                  <div class="flex items-center gap-1 flex-shrink-0">
                    <button
                      @click.stop="emailStore.markAsStarred(email.id, !email.is_starred)"
                      class="p-1 hover:bg-gray-100 dark:hover:bg-gray-600 rounded"
                    >
                      <svg
                        :class="['w-4 h-4', email.is_starred ? 'text-yellow-400 fill-current' : 'text-gray-300 dark:text-gray-600']"
                        fill="none" stroke="currentColor" viewBox="0 0 24 24"
                      >
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                      </svg>
                    </button>
                  </div>
                </div>

                <!-- Attachments -->
                <div v-if="email.attachments?.length" class="mx-4 mb-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg border dark:border-gray-700">
                  <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
                    </svg>
                    Attachments ({{ email.attachments.length }})
                  </h4>
                  <div class="flex flex-wrap gap-2">
                    <div
                      v-for="attachment in email.attachments"
                      :key="attachment.id"
                      class="group relative flex items-center gap-2 px-3 py-2 bg-white dark:bg-gray-700 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-600 cursor-pointer transition-colors border dark:border-gray-600"
                      @click.stop="handleAttachmentClick(attachment)"
                      @contextmenu.prevent="showAttachmentMenu($event, attachment)"
                    >
                      <span class="text-lg">{{ getAttachmentIcon(attachment.content_type) }}</span>
                      <div class="flex flex-col">
                        <span class="text-sm text-gray-700 dark:text-gray-300 max-w-[200px] truncate">{{ attachment.filename }}</span>
                        <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatFileSize(attachment.size) }}</span>
                      </div>
                      <button
                        @click.stop="handleAttachmentDownload(attachment)"
                        class="ml-2 p-1 hover:bg-gray-200 dark:hover:bg-gray-500 rounded opacity-0 group-hover:opacity-100 transition-opacity"
                        title="Download"
                      >
                        <svg class="w-4 h-4 text-gray-600 dark:text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                        </svg>
                      </button>
                      <button
                        @click.stop="contextMenuAttachment = attachment; showSaveToFilesModal = true"
                        class="p-1 hover:bg-gray-200 dark:hover:bg-gray-500 rounded opacity-0 group-hover:opacity-100 transition-opacity"
                        title="Save to My Files"
                      >
                        <svg class="w-4 h-4 text-gray-600 dark:text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                        </svg>
                      </button>
                    </div>
                  </div>
                </div>

                <!-- Email body -->
                <div class="px-4 pb-4">
                  <iframe
                    v-if="email.html_body"
                    :srcdoc="sanitizeHtmlBody(email.html_body)"
                    class="conversation-iframe"
                    sandbox="allow-popups allow-popups-to-escape-sandbox"
                    frameborder="0"
                    @load="resizeConversationIframe"
                  ></iframe>
                  <pre
                    v-else
                    class="whitespace-pre-wrap font-sans text-gray-700 dark:text-gray-300 text-sm leading-relaxed"
                  >{{ email.text_body || email.snippet }}</pre>
                </div>

                <!-- Per-email reply buttons (on last email) -->
                <div v-if="index === emailStore.threadConversation.length - 1" class="px-4 pb-4 flex gap-2">
                  <button
                    @click="emailStore.currentEmail = email; openCompose('reply')"
                    class="inline-flex items-center gap-2 px-4 py-2 text-sm border dark:border-gray-600 rounded-full hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300 transition-colors"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
                    </svg>
                    Reply
                  </button>
                  <button
                    @click="emailStore.currentEmail = email; openCompose('forward')"
                    class="inline-flex items-center gap-2 px-4 py-2 text-sm border dark:border-gray-600 rounded-full hover:bg-gray-50 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300 transition-colors"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 10h-10a8 8 0 00-8 8v2M21 10l-6 6m6-6l-6-6" />
                    </svg>
                    Forward
                  </button>
                </div>
              </div>
            </div>
          </div>
        </template>

        <!-- ===== SINGLE EMAIL VIEW ===== -->
        <template v-else-if="currentEmail">
          <!-- Email header -->
          <div class="flex items-center gap-2 p-4 border-b dark:border-gray-700">
            <button
              @click="emailStore.currentEmail = null"
              class="lg:hidden p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            <button @click="openCompose('reply')" class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg" title="Reply">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
              </svg>
            </button>
            <button @click="openCompose('replyAll')" class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg" title="Reply All">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6M8 4h10a8 8 0 018 8v2" />
              </svg>
            </button>
            <button @click="openCompose('forward')" class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg" title="Forward">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 10h-10a8 8 0 00-8 8v2M21 10l-6 6m6-6l-6-6" />
              </svg>
            </button>

            <!-- Label dropdown -->
            <div class="relative">
              <button
                @click="showLabelMenu = !showLabelMenu"
                class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
                title="Add/remove labels"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
                </svg>
              </button>
              <div
                v-if="showLabelMenu && currentEmail"
                class="absolute top-full left-0 mt-1 w-56 bg-white dark:bg-gray-800 rounded-lg shadow-lg border dark:border-gray-700 z-10"
              >
                <div class="p-2">
                  <p class="text-xs font-medium text-gray-500 dark:text-gray-400 px-2 pb-2">Labels</p>
                  <button
                    v-for="label in labels"
                    :key="label.id"
                    @click="toggleEmailLabel(label.id)"
                    class="w-full flex items-center gap-2 px-2 py-1.5 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-sm"
                  >
                    <span class="w-3 h-3 rounded-full flex-shrink-0" :style="{ backgroundColor: label.color }"></span>
                    <span class="flex-1 text-left text-gray-700 dark:text-gray-300">{{ label.name }}</span>
                    <svg v-if="currentEmail.labels?.some(l => l.id === label.id)" class="w-4 h-4 text-blue-600" fill="currentColor" viewBox="0 0 20 20">
                      <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                    </svg>
                  </button>
                  <div v-if="labels.length === 0" class="px-2 py-3 text-xs text-gray-500 dark:text-gray-400 text-center">No labels yet</div>
                </div>
              </div>
            </div>

            <div class="flex-1"></div>
            <button @click="moveToTrash" class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg text-red-500" title="Delete">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>

          <!-- Email content -->
          <div class="flex-1 overflow-y-auto p-6 flex flex-col">
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white mb-2">
              {{ currentEmail.subject || '(No subject)' }}
            </h1>

            <div v-if="currentEmail.labels?.length" class="flex flex-wrap gap-1 mb-4">
              <span
                v-for="label in currentEmail.labels"
                :key="label.id"
                class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
                :style="{ backgroundColor: label.color + '20', color: label.color }"
              >
                {{ label.name }}
                <button @click="emailStore.removeLabelFromEmail(currentEmail.id, label.id)" class="hover:opacity-70">
                  <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </span>
            </div>

            <div class="flex items-start gap-4 mb-6">
              <div class="w-10 h-10 rounded-full bg-blue-500 flex items-center justify-center text-white font-medium flex-shrink-0">
                {{ (currentEmail.from_name || currentEmail.from_address).charAt(0).toUpperCase() }}
              </div>
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="font-medium text-gray-900 dark:text-white">{{ currentEmail.from_name || currentEmail.from_address }}</span>
                  <span class="text-sm text-gray-500 dark:text-gray-400">&lt;{{ currentEmail.from_address }}&gt;</span>
                </div>
                <div class="text-sm text-gray-500 dark:text-gray-400">
                  to {{ currentEmail.to.map(t => t.name || t.address).join(', ') }}
                  <span v-if="currentEmail.cc?.length">, cc: {{ currentEmail.cc.map(c => c.name || c.address).join(', ') }}</span>
                </div>
                <div class="text-xs text-gray-400 dark:text-gray-500 mt-1">{{ formatFullDate(currentEmail.date) }}</div>
              </div>
            </div>

            <div v-if="currentEmail.attachments?.length" class="mb-4 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg border dark:border-gray-700">
              <h4 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2 flex items-center gap-2">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
                </svg>
                Attachments ({{ currentEmail.attachments.length }})
              </h4>
              <div class="flex flex-wrap gap-2">
                <div
                  v-for="attachment in currentEmail.attachments"
                  :key="attachment.id"
                  class="group relative flex items-center gap-2 px-3 py-2 bg-white dark:bg-gray-700 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-600 cursor-pointer transition-colors border dark:border-gray-600"
                  @click="handleAttachmentClick(attachment)"
                  @contextmenu.prevent="showAttachmentMenu($event, attachment)"
                >
                  <span class="text-lg">{{ getAttachmentIcon(attachment.content_type) }}</span>
                  <div class="flex flex-col">
                    <span class="text-sm text-gray-700 dark:text-gray-300 max-w-[200px] truncate">{{ attachment.filename }}</span>
                    <span class="text-xs text-gray-500 dark:text-gray-400">{{ formatFileSize(attachment.size) }}</span>
                  </div>
                  <button
                    @click.stop="handleAttachmentDownload(attachment)"
                    class="ml-2 p-1 hover:bg-gray-200 dark:hover:bg-gray-500 rounded opacity-0 group-hover:opacity-100 transition-opacity"
                    title="Download"
                  >
                    <svg class="w-4 h-4 text-gray-600 dark:text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
                    </svg>
                  </button>
                  <button
                    @click.stop="contextMenuAttachment = attachment; showSaveToFilesModal = true"
                    class="p-1 hover:bg-gray-200 dark:hover:bg-gray-500 rounded opacity-0 group-hover:opacity-100 transition-opacity"
                    title="Save to My Files"
                  >
                    <svg class="w-4 h-4 text-gray-600 dark:text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>

            <div class="email-body-container">
              <iframe
                v-if="currentEmail.html_body"
                :srcdoc="sanitizedEmailHtml"
                class="email-iframe"
                sandbox="allow-popups allow-popups-to-escape-sandbox"
                frameborder="0"
                @load="resizeIframe"
                ref="emailIframe"
              ></iframe>
              <pre v-else class="whitespace-pre-wrap font-sans text-gray-700 dark:text-gray-300 p-4">{{ currentEmail.text_body }}</pre>
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- Modals -->
    <EmailAccountSetup
      v-if="showAccountSetup"
      @close="showAccountSetup = false"
      @saved="showAccountSetup = false; emailStore.fetchAccounts()"
    />

    <EmailCompose
      v-if="showCompose"
      :mode="replyMode"
      :original-email="currentEmail"
      @close="closeCompose"
      @sent="closeCompose"
    />

    <EmailLabelManager
      v-if="showLabelManager"
      @close="showLabelManager = false"
    />

    <EmailRuleManager
      v-if="showRuleManager"
      @close="showRuleManager = false"
    />

    <!-- Folder Context Menu -->
    <Teleport to="body">
      <div
        v-if="showFolderContextMenu && contextMenuFolder"
        class="fixed inset-0 z-50"
        @click="closeFolderContextMenu"
      >
        <div
          class="absolute bg-white dark:bg-gray-800 rounded-lg shadow-lg border dark:border-gray-700 py-1 min-w-[160px]"
          :style="{ left: contextMenuPosition.x + 'px', top: contextMenuPosition.y + 'px' }"
          @click.stop
        >
          <button
            @click="handleMarkFolderAsRead"
            class="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 19v-8.93a2 2 0 01.89-1.664l7-4.666a2 2 0 012.22 0l7 4.666A2 2 0 0121 10.07V19M3 19a2 2 0 002 2h14a2 2 0 002-2M3 19l6.75-4.5M21 19l-6.75-4.5M3 10l6.75 4.5M21 10l-6.75 4.5m0 0l-1.14.76a2 2 0 01-2.22 0l-1.14-.76" />
            </svg>
            Mark All as Read
          </button>
          <button
            @click="openCreateFolderModal(contextMenuFolder?.id || null)"
            class="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
            </svg>
            Create Subfolder
          </button>
          <button
            v-if="contextMenuFolder?.folder_type === 'custom'"
            @click="openRenameFolderModal(contextMenuFolder)"
            class="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
            </svg>
            Rename
          </button>
          <button
            v-if="contextMenuFolder?.folder_type === 'custom'"
            @click="confirmDeleteFolder(contextMenuFolder)"
            class="w-full px-4 py-2 text-left text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            Delete
          </button>
        </div>
      </div>
    </Teleport>

    <!-- Folder Create/Rename Modal -->
    <Teleport to="body">
      <div
        v-if="showFolderModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        @click.self="showFolderModal = false"
      >
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md p-6">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            {{ folderModalMode === 'create' ? 'Create Folder' : 'Rename Folder' }}
          </h3>
          <input
            v-model="folderModalName"
            type="text"
            placeholder="Folder name"
            class="w-full px-4 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            @keyup.enter="saveFolderModal"
            autofocus
          />
          <div v-if="folderModalMode === 'create' && folderModalParentId" class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            Will be created inside the selected folder
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <button
              @click="showFolderModal = false"
              class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              Cancel
            </button>
            <button
              @click="saveFolderModal"
              :disabled="!folderModalName.trim()"
              class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {{ folderModalMode === 'create' ? 'Create' : 'Save' }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Delete Folder Confirmation -->
    <Teleport to="body">
      <div
        v-if="showDeleteConfirm"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        @click.self="showDeleteConfirm = false"
      >
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md p-6">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">
            Delete Folder
          </h3>
          <p class="text-gray-600 dark:text-gray-400 mb-6">
            Are you sure you want to delete "{{ contextMenuFolder?.name }}"? This will also delete all emails in this folder.
          </p>
          <div class="flex justify-end gap-3">
            <button
              @click="showDeleteConfirm = false"
              class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              Cancel
            </button>
            <button
              @click="handleDeleteFolder"
              class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
            >
              Delete
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Attachment Context Menu -->
    <Teleport to="body">
      <div
        v-if="showAttachmentContextMenu && contextMenuAttachment"
        class="fixed inset-0 z-50"
        @click="closeAttachmentContextMenu"
      >
        <div
          class="absolute bg-white dark:bg-gray-800 rounded-lg shadow-lg border dark:border-gray-700 py-1 min-w-[180px]"
          :style="{ left: attachmentContextMenuPosition.x + 'px', top: attachmentContextMenuPosition.y + 'px' }"
          @click.stop
        >
          <button
            @click="handleAttachmentDownload(contextMenuAttachment!); closeAttachmentContextMenu()"
            class="w-full flex items-center gap-3 px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
            </svg>
            Download
          </button>
          <button
            @click="handleSaveAttachmentToFiles"
            class="w-full flex items-center gap-3 px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 19a2 2 0 01-2-2V7a2 2 0 012-2h4l2 2h4a2 2 0 012 2v1M5 19h14a2 2 0 002-2v-5a2 2 0 00-2-2H9a2 2 0 00-2 2v5a2 2 0 01-2 2z" />
            </svg>
            Save to My Files
          </button>
        </div>
      </div>
    </Teleport>

    <!-- Save to My Files Modal -->
    <Teleport to="body">
      <div
        v-if="showSaveToFilesModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        @click.self="cancelSaveToFiles"
      >
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md max-h-[80vh] flex flex-col">
          <div class="p-4 border-b dark:border-gray-700 flex items-center justify-between">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              Save to My Files
            </h3>
            <button
              @click="cancelSaveToFiles"
              class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          
          <div class="p-4">
            <p class="text-sm text-gray-600 dark:text-gray-400 mb-4">
              Saving: <span class="font-medium text-gray-900 dark:text-white">{{ contextMenuAttachment?.filename }}</span>
            </p>
            
            <SaveToFolderPicker
              :saving="savingAttachment"
              @select="confirmSaveToFiles"
              @cancel="cancelSaveToFiles"
            />
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Email Context Menu -->
    <Teleport to="body">
      <div
        v-if="showEmailContextMenu && contextMenuEmail"
        class="fixed inset-0 z-50"
        @click="closeEmailContextMenu"
      >
        <div
          class="absolute bg-white dark:bg-gray-800 rounded-lg shadow-lg border dark:border-gray-700 py-1 min-w-[180px]"
          :style="{ left: contextMenuPosition.x + 'px', top: contextMenuPosition.y + 'px' }"
          @click.stop
        >
          <button
            @click="openCreateTaskFromEmail(contextMenuEmail)"
            class="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
            </svg>
            Create Task
          </button>
          <button
            @click="closeEmailContextMenu(); emailStore.markAsRead(contextMenuEmail.id, !contextMenuEmail.is_read)"
            class="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 19v-8.93a2 2 0 01.89-1.664l7-4.666a2 2 0 012.22 0l7 4.666A2 2 0 0121 10.07V19M3 19a2 2 0 002 2h14a2 2 0 002-2M3 19l6.75-4.5M21 19l-6.75-4.5M3 10l6.75 4.5M21 10l-6.75 4.5m0 0l-1.14.76a2 2 0 01-2.22 0l-1.14-.76" />
            </svg>
            {{ contextMenuEmail.is_read ? 'Mark as Unread' : 'Mark as Read' }}
          </button>
          <button
            @click="closeEmailContextMenu(); toggleStar(contextMenuEmail, { stopPropagation: () => {} } as Event)"
            class="w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
            </svg>
            {{ contextMenuEmail.is_starred ? 'Remove Star' : 'Add Star' }}
          </button>
          <div class="border-t dark:border-gray-700 my-1"></div>
          <button
            @click="closeEmailContextMenu(); deleteEmailFromList(contextMenuEmail.id)"
            class="w-full px-4 py-2 text-left text-sm text-red-600 dark:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
            Delete
          </button>
        </div>
      </div>
    </Teleport>

    <!-- Create Task from Email Modal -->
    <Teleport to="body">
      <div
        v-if="showCreateTaskModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        @click.self="cancelCreateTask"
      >
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-lg p-6">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">
            Create Task from Email
          </h3>
          
          <!-- Linked Email Info -->
          <div class="mb-4 p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg flex items-center gap-2">
            <svg class="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
            <span class="text-sm text-blue-700 dark:text-blue-300 truncate">
              Linked to: {{ taskFromEmail?.emailSubject }}
            </span>
          </div>

          <!-- Title -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Title</label>
            <input
              v-model="newTaskForm.title"
              type="text"
              placeholder="Task title"
              class="w-full px-4 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <!-- Description -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
            <textarea
              v-model="newTaskForm.description"
              rows="3"
              placeholder="Task description"
              class="w-full px-4 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent resize-none"
            ></textarea>
          </div>

          <!-- Priority & Due Date -->
          <div class="grid grid-cols-2 gap-4 mb-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Priority</label>
              <select
                v-model="newTaskForm.priority"
                class="w-full px-4 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Due Date</label>
              <input
                v-model="newTaskForm.dueDate"
                type="date"
                class="w-full px-4 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </div>

          <!-- Group -->
          <div v-if="tasksStore.groups.length > 0" class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Group</label>
            <select
              v-model="newTaskForm.groupId"
              class="w-full px-4 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option :value="null">No Group</option>
              <option v-for="group in tasksStore.groups" :key="group.id" :value="group.id">
                {{ group.name }}
              </option>
            </select>
          </div>

          <!-- Add to Calendar (only show when a due date is set) -->
          <div v-if="newTaskForm.dueDate" class="mb-4">
            <label class="flex items-center gap-2 cursor-pointer p-3 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
              <input
                v-model="addToCalendar"
                type="checkbox"
                class="w-4 h-4 rounded text-blue-600"
              />
              <svg class="w-5 h-5 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
              <span class="text-sm font-medium text-blue-700 dark:text-blue-300">Add to calendar</span>
            </label>
          </div>

          <!-- Actions -->
          <div class="flex justify-end gap-3">
            <button
              @click="cancelCreateTask"
              class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              Cancel
            </button>
            <button
              @click="saveTaskFromEmail"
              :disabled="!newTaskForm.title.trim()"
              class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Create Task
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Undo Send Toast -->
    <Teleport to="body">
      <div v-if="undoSendItems.length > 0" class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex flex-col gap-2">
        <div
          v-for="item in undoSendItems"
          :key="item.sendId"
          class="flex items-center gap-4 bg-gray-900 text-white px-5 py-3 rounded-xl shadow-2xl min-w-[320px]"
        >
          <span class="text-sm flex-1">Sending "{{ item.subject }}"...</span>
          <button
            @click="emailStore.cancelSend(item.sendId)"
            class="px-3 py-1 bg-blue-500 hover:bg-blue-400 text-white text-sm font-medium rounded-lg transition-colors"
          >
            Undo
          </button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.email-body-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 200px;
}

.email-iframe {
  width: 100%;
  flex: 1;
  min-height: 200px;
  border: none;
  background: transparent;
}

.conversation-iframe {
  width: 100%;
  min-height: 100px;
  border: none;
  background: transparent;
}

/* Keep these for any remaining email-content divs */
.email-content :deep(img) {
  max-width: 100%;
  height: auto;
}

.email-content :deep(a) {
  color: #2563eb;
}

.email-content :deep(a:hover) {
  text-decoration: underline;
}
</style>
