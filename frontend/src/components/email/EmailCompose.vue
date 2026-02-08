<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useEmailStore, type Email } from '@/stores/email'
import api from '@/api'

const props = defineProps<{
  mode?: 'reply' | 'replyAll' | 'forward' | null
  originalEmail?: Email | null
  prefillTo?: string
}>()

const emit = defineEmits<{
  close: []
  sent: []
}>()

const emailStore = useEmailStore()

// Contact type
interface Contact {
  id: string
  firstName: string
  lastName: string
  email: string
  phone?: string
  company?: string
  favorite?: boolean
}

const form = reactive({
  to: '',
  cc: '',
  bcc: '',
  subject: '',
  body: ''
})

const loading = ref(false)
const error = ref<string | null>(null)
const showCc = ref(false)
const showBcc = ref(false)
const isMinimized = ref(false)
const draftSaved = ref(false)
const attachmentFiles = ref<File[]>([])
const fileInput = ref<HTMLInputElement | null>(null)

// Contact autocomplete state
const contacts = ref<Contact[]>([])
const activeField = ref<'to' | 'cc' | 'bcc' | null>(null)
const showSuggestions = ref(false)
const suggestionIndex = ref(-1)

// Get the current input value after the last comma
function getCurrentInput(field: 'to' | 'cc' | 'bcc'): string {
  const value = form[field]
  const parts = value.split(',')
  return parts[parts.length - 1].trim().toLowerCase()
}

// Extract email from "Name <email>" format or plain email
function extractEmailFromEntry(entry: string): string {
  const match = entry.match(/<([^>]+)>/)
  return match ? match[1].toLowerCase() : entry.toLowerCase()
}

// Filtered contacts based on current input
const filteredContacts = computed(() => {
  if (!activeField.value) return []
  
  const query = getCurrentInput(activeField.value)
  if (query.length < 1) return []
  
  // Get already entered emails to exclude from suggestions
  const currentValue = form[activeField.value]
  const enteredEmails = currentValue.split(',')
    .map(e => extractEmailFromEntry(e.trim()))
    .filter(Boolean)
  
  return contacts.value
    .filter(contact => {
      if (!contact.email) return false
      // Don't suggest already entered emails
      if (enteredEmails.includes(contact.email.toLowerCase())) return false
      
      const fullName = `${contact.firstName || ''} ${contact.lastName || ''}`.toLowerCase()
      const email = contact.email.toLowerCase()
      return fullName.includes(query) || email.includes(query)
    })
    .slice(0, 5) // Limit to 5 suggestions
})

// Select a contact from suggestions
function selectContact(contact: Contact) {
  if (!activeField.value) return
  
  const value = form[activeField.value]
  const parts = value.split(',')
  // Replace the last part with the selected contact
  parts[parts.length - 1] = ` ${contact.firstName} ${contact.lastName} <${contact.email}>`
  form[activeField.value] = parts.join(',').replace(/^,?\s*/, '')
  
  showSuggestions.value = false
  suggestionIndex.value = -1
}

// Handle keyboard navigation in suggestions
function handleKeydown(e: KeyboardEvent, field: 'to' | 'cc' | 'bcc') {
  if (!showSuggestions.value || filteredContacts.value.length === 0) return
  
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    suggestionIndex.value = Math.min(suggestionIndex.value + 1, filteredContacts.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    suggestionIndex.value = Math.max(suggestionIndex.value - 1, 0)
  } else if (e.key === 'Enter' && suggestionIndex.value >= 0) {
    e.preventDefault()
    selectContact(filteredContacts.value[suggestionIndex.value])
  } else if (e.key === 'Escape') {
    showSuggestions.value = false
  }
}

// Show suggestions when typing
function handleInput(field: 'to' | 'cc' | 'bcc') {
  activeField.value = field
  showSuggestions.value = true
  suggestionIndex.value = -1
}

// Hide suggestions on blur (with delay to allow click)
function handleBlur() {
  setTimeout(() => {
    showSuggestions.value = false
  }, 200)
}

const title = computed(() => {
  switch (props.mode) {
    case 'reply': return 'Reply'
    case 'replyAll': return 'Reply All'
    case 'forward': return 'Forward'
    default: return 'New Message'
  }
})

onMounted(async () => {
  // Fetch contacts for autocomplete
  try {
    const response = await api.get('/contacts')
    contacts.value = response.data || []
  } catch (e) {
    console.error('Failed to fetch contacts:', e)
  }
  
  // Pre-fill To field if provided (e.g., from Contacts page)
  if (props.prefillTo) {
    form.to = props.prefillTo
  }
  
  // Set up reply/forward if applicable
  if (props.originalEmail && props.mode) {
    // Set up reply/forward
    const orig = props.originalEmail

    if (props.mode === 'reply' || props.mode === 'replyAll') {
      form.to = orig.reply_to || orig.from_address
      form.subject = orig.subject.startsWith('Re:') ? orig.subject : `Re: ${orig.subject}`
      
      if (props.mode === 'replyAll') {
        // Add other To recipients and original CC recipients except ourselves
        const currentEmail = emailStore.currentAccount?.email_address?.toLowerCase()
        const otherToRecipients = orig.to
          .filter(t => t.address.toLowerCase() !== currentEmail)
          .map(t => t.address)
        const originalCcRecipients = (orig.cc || [])
          .filter(c => c.address.toLowerCase() !== currentEmail && c.address.toLowerCase() !== orig.from_address.toLowerCase())
          .map(c => c.address)
        const allCcRecipients = [...otherToRecipients, ...originalCcRecipients]
        if (allCcRecipients.length > 0) {
          form.cc = allCcRecipients.join(', ')
          showCc.value = true
        }
      }
    } else if (props.mode === 'forward') {
      form.subject = orig.subject.startsWith('Fwd:') ? orig.subject : `Fwd: ${orig.subject}`
    }

    // Build quoted message
    const quotedHeader = `\n\n---------- Forwarded message ----------\nFrom: ${orig.from_name || orig.from_address} <${orig.from_address}>\nDate: ${new Date(orig.date).toLocaleString()}\nSubject: ${orig.subject}\nTo: ${orig.to.map(t => t.address).join(', ')}\n\n`
    
    if (props.mode === 'forward') {
      form.body = quotedHeader + (orig.text_body || '')
    } else {
      const replyHeader = `\n\nOn ${new Date(orig.date).toLocaleString()}, ${orig.from_name || orig.from_address} wrote:\n`
      const quotedBody = (orig.text_body || '').split('\n').map(line => `> ${line}`).join('\n')
      form.body = replyHeader + quotedBody
    }
  }

  // Append signature if account has one
  const sig = emailStore.currentAccount?.signature
  if (sig) {
    form.body = form.body + '\n\n-- \n' + sig
  }

  // Start draft auto-save
  emailStore.startDraftAutoSave(() => ({
    to: form.to ? form.to.split(',').map(a => ({ address: a.trim() })) : [],
    cc: form.cc ? form.cc.split(',').map(a => ({ address: a.trim() })) : [],
    bcc: form.bcc ? form.bcc.split(',').map(a => ({ address: a.trim() })) : [],
    subject: form.subject,
    body: form.body,
    is_html: false,
    reply_to_id: props.originalEmail?.message_id || ''
  }))
})

onUnmounted(() => {
  emailStore.stopDraftAutoSave()
})

// Parse email address from various formats:
// - "email@example.com"
// - "Name <email@example.com>"
// - "<email@example.com>"
function parseEmailAddress(input: string): string {
  const trimmed = input.trim()
  // Check for "Name <email>" or "<email>" format
  const match = trimmed.match(/<([^>]+)>/)
  if (match) {
    return match[1].trim()
  }
  // Otherwise return as-is (plain email)
  return trimmed
}

function parseEmailAddresses(input: string): string[] {
  return input.split(',')
    .map(s => parseEmailAddress(s))
    .filter(Boolean)
}

function handleAttachmentSelect() {
  fileInput.value?.click()
}

function onFilesSelected(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files) {
    attachmentFiles.value = [...attachmentFiles.value, ...Array.from(target.files)]
  }
}

function removeAttachment(index: number) {
  attachmentFiles.value.splice(index, 1)
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

async function handleSend() {
  error.value = null

  if (!form.to.trim()) {
    error.value = 'Please enter at least one recipient'
    return
  }

  loading.value = true
  try {
    emailStore.stopDraftAutoSave()
    
    await emailStore.sendEmail({
      to: parseEmailAddresses(form.to),
      cc: form.cc ? parseEmailAddresses(form.cc) : undefined,
      bcc: form.bcc ? parseEmailAddresses(form.bcc) : undefined,
      subject: form.subject,
      body: form.body,
      is_html: false,
      reply_to: props.originalEmail?.message_id,
      attachments: attachmentFiles.value.length > 0 ? attachmentFiles.value : undefined
    })
    
    // Delete draft if it was saved
    if (emailStore.currentDraft?.id) {
      await emailStore.deleteDraft(emailStore.currentDraft.id)
    }
    
    emit('sent')
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to send email'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div
    :class="[
      'fixed z-50 bg-white dark:bg-gray-800 rounded-t-xl shadow-2xl border dark:border-gray-700 flex flex-col',
      isMinimized
        ? 'bottom-0 right-4 w-80 h-10'
        : 'bottom-0 right-4 w-[560px] h-[480px]'
    ]"
  >
    <!-- Header -->
    <div
      class="flex items-center justify-between px-4 py-2 bg-gray-100 dark:bg-gray-700 rounded-t-xl cursor-pointer"
      @click="isMinimized = !isMinimized"
    >
      <span class="text-sm font-medium text-gray-900 dark:text-white">{{ title }}</span>
      <div class="flex items-center gap-1">
        <button
          @click.stop="isMinimized = !isMinimized"
          class="p-1 hover:bg-gray-200 dark:hover:bg-gray-600 rounded"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 12H4" />
          </svg>
        </button>
        <button
          @click.stop="emit('close')"
          class="p-1 hover:bg-gray-200 dark:hover:bg-gray-600 rounded"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Content (hidden when minimized) -->
    <template v-if="!isMinimized">
      <!-- Error message -->
      <div v-if="error" class="px-4 py-2 bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-300 text-sm">
        {{ error }}
      </div>

      <!-- To field -->
      <div class="relative flex items-center border-b dark:border-gray-700 px-4">
        <span class="text-sm text-gray-500 dark:text-gray-400 w-12">To</span>
        <div class="flex-1 relative">
          <input
            v-model="form.to"
            type="text"
            placeholder="Recipients"
            class="w-full py-2 bg-transparent border-0 focus:ring-0 text-sm"
            @input="handleInput('to')"
            @focus="handleInput('to')"
            @blur="handleBlur"
            @keydown="handleKeydown($event, 'to')"
          />
          <!-- Contact suggestions dropdown -->
          <div 
            v-if="showSuggestions && activeField === 'to' && filteredContacts.length > 0"
            class="absolute left-0 right-0 top-full mt-1 bg-white dark:bg-gray-800 border dark:border-gray-700 rounded-lg shadow-lg z-50 overflow-hidden"
          >
            <div
              v-for="(contact, index) in filteredContacts"
              :key="contact.id"
              @mousedown.prevent="selectContact(contact)"
              :class="[
                'px-3 py-2 cursor-pointer flex items-center gap-3',
                index === suggestionIndex 
                  ? 'bg-blue-50 dark:bg-blue-900/30' 
                  : 'hover:bg-gray-50 dark:hover:bg-gray-700'
              ]"
            >
              <div class="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center text-blue-600 dark:text-blue-400 text-xs font-medium">
                {{ contact.firstName?.charAt(0) || '' }}{{ contact.lastName?.charAt(0) || '' }}
              </div>
              <div class="flex-1 min-w-0">
                <div class="text-sm font-medium text-gray-900 dark:text-white truncate">
                  {{ contact.firstName }} {{ contact.lastName }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400 truncate">
                  {{ contact.email }}
                </div>
              </div>
              <span v-if="contact.favorite" class="text-yellow-400">★</span>
            </div>
          </div>
        </div>
        <div class="flex gap-2 text-sm">
          <button
            v-if="!showCc"
            @click="showCc = true"
            class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
          >
            Cc
          </button>
          <button
            v-if="!showBcc"
            @click="showBcc = true"
            class="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
          >
            Bcc
          </button>
        </div>
      </div>

      <!-- Cc field -->
      <div v-if="showCc" class="relative flex items-center border-b dark:border-gray-700 px-4">
        <span class="text-sm text-gray-500 dark:text-gray-400 w-12">Cc</span>
        <div class="flex-1 relative">
          <input
            v-model="form.cc"
            type="text"
            placeholder="Cc recipients"
            class="w-full py-2 bg-transparent border-0 focus:ring-0 text-sm"
            @input="handleInput('cc')"
            @focus="handleInput('cc')"
            @blur="handleBlur"
            @keydown="handleKeydown($event, 'cc')"
          />
          <!-- Contact suggestions dropdown for Cc -->
          <div 
            v-if="showSuggestions && activeField === 'cc' && filteredContacts.length > 0"
            class="absolute left-0 right-0 top-full mt-1 bg-white dark:bg-gray-800 border dark:border-gray-700 rounded-lg shadow-lg z-50 overflow-hidden"
          >
            <div
              v-for="(contact, index) in filteredContacts"
              :key="contact.id"
              @mousedown.prevent="selectContact(contact)"
              :class="[
                'px-3 py-2 cursor-pointer flex items-center gap-3',
                index === suggestionIndex 
                  ? 'bg-blue-50 dark:bg-blue-900/30' 
                  : 'hover:bg-gray-50 dark:hover:bg-gray-700'
              ]"
            >
              <div class="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center text-blue-600 dark:text-blue-400 text-xs font-medium">
                {{ contact.firstName?.charAt(0) || '' }}{{ contact.lastName?.charAt(0) || '' }}
              </div>
              <div class="flex-1 min-w-0">
                <div class="text-sm font-medium text-gray-900 dark:text-white truncate">
                  {{ contact.firstName }} {{ contact.lastName }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400 truncate">
                  {{ contact.email }}
                </div>
              </div>
              <span v-if="contact.favorite" class="text-yellow-400">★</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Bcc field -->
      <div v-if="showBcc" class="relative flex items-center border-b dark:border-gray-700 px-4">
        <span class="text-sm text-gray-500 dark:text-gray-400 w-12">Bcc</span>
        <div class="flex-1 relative">
          <input
            v-model="form.bcc"
            type="text"
            placeholder="Bcc recipients"
            class="w-full py-2 bg-transparent border-0 focus:ring-0 text-sm"
            @input="handleInput('bcc')"
            @focus="handleInput('bcc')"
            @blur="handleBlur"
            @keydown="handleKeydown($event, 'bcc')"
          />
          <!-- Contact suggestions dropdown for Bcc -->
          <div 
            v-if="showSuggestions && activeField === 'bcc' && filteredContacts.length > 0"
            class="absolute left-0 right-0 top-full mt-1 bg-white dark:bg-gray-800 border dark:border-gray-700 rounded-lg shadow-lg z-50 overflow-hidden"
          >
            <div
              v-for="(contact, index) in filteredContacts"
              :key="contact.id"
              @mousedown.prevent="selectContact(contact)"
              :class="[
                'px-3 py-2 cursor-pointer flex items-center gap-3',
                index === suggestionIndex 
                  ? 'bg-blue-50 dark:bg-blue-900/30' 
                  : 'hover:bg-gray-50 dark:hover:bg-gray-700'
              ]"
            >
              <div class="w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900 flex items-center justify-center text-blue-600 dark:text-blue-400 text-xs font-medium">
                {{ contact.firstName?.charAt(0) || '' }}{{ contact.lastName?.charAt(0) || '' }}
              </div>
              <div class="flex-1 min-w-0">
                <div class="text-sm font-medium text-gray-900 dark:text-white truncate">
                  {{ contact.firstName }} {{ contact.lastName }}
                </div>
                <div class="text-xs text-gray-500 dark:text-gray-400 truncate">
                  {{ contact.email }}
                </div>
              </div>
              <span v-if="contact.favorite" class="text-yellow-400">★</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Subject field -->
      <div class="flex items-center border-b dark:border-gray-700 px-4">
        <span class="text-sm text-gray-500 dark:text-gray-400 w-12">Subject</span>
        <input
          v-model="form.subject"
          type="text"
          placeholder="Subject"
          class="flex-1 py-2 bg-transparent border-0 focus:ring-0 text-sm"
        />
      </div>

      <!-- Body -->
      <div class="flex-1 overflow-hidden">
        <textarea
          v-model="form.body"
          placeholder="Write your message..."
          class="w-full h-full p-4 bg-transparent border-0 focus:ring-0 resize-none text-sm"
        ></textarea>
      </div>

      <!-- Attachments preview -->
      <div v-if="attachmentFiles.length > 0" class="px-4 py-2 border-t dark:border-gray-700 flex flex-wrap gap-2">
        <div
          v-for="(file, index) in attachmentFiles"
          :key="index"
          class="flex items-center gap-2 bg-gray-100 dark:bg-gray-700 rounded-lg px-3 py-1 text-sm"
        >
          <svg class="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
          </svg>
          <span class="truncate max-w-[120px]">{{ file.name }}</span>
          <span class="text-gray-400 text-xs">({{ formatFileSize(file.size) }})</span>
          <button @click="removeAttachment(index)" class="text-gray-400 hover:text-red-500">
            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Hidden file input -->
      <input
        ref="fileInput"
        type="file"
        multiple
        class="hidden"
        @change="onFilesSelected"
      />

      <!-- Footer -->
      <div class="flex items-center justify-between px-4 py-2 border-t dark:border-gray-700">
        <button
          @click="handleSend"
          :disabled="loading"
          class="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50 flex items-center gap-2"
        >
          <svg v-if="loading" class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          {{ loading ? 'Sending...' : 'Send' }}
        </button>

        <div class="flex items-center gap-2">
          <!-- Attach file button -->
          <button @click="handleAttachmentSelect" class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded" title="Attach files">
            <svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.172 7l-6.586 6.586a2 2 0 102.828 2.828l6.414-6.586a4 4 0 00-5.656-5.656l-6.415 6.585a6 6 0 108.486 8.486L20.5 13" />
            </svg>
          </button>
          <!-- Draft saved indicator -->
          <span v-if="emailStore.currentDraft" class="text-xs text-gray-400">Draft saved</span>
          <button
            @click="emit('close')"
            class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded text-red-500"
            title="Discard"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>
    </template>
  </div>
</template>
