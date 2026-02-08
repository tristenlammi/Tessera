<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '@/api'
import EmailCompose from '@/components/email/EmailCompose.vue'

interface Contact {
  id: string
  firstName: string
  lastName: string
  email: string
  phone: string
  company: string
  jobTitle: string
  birthday: string | null
  notes: string
  avatar: string | null
  favorite: boolean
  createdAt: string
  updatedAt: string
}

const contacts = ref<Contact[]>([])
const loading = ref(false)
const searchQuery = ref('')
const showContactModal = ref(false)
const editingContact = ref<Contact | null>(null)
const showDeleteConfirm = ref(false)
const contactToDelete = ref<Contact | null>(null)

// Email compose state
const showEmailCompose = ref(false)
const emailComposeContact = ref<Contact | null>(null)

// Contact form
const contactForm = ref({
  firstName: '',
  lastName: '',
  email: '',
  phone: '',
  company: '',
  jobTitle: '',
  birthday: '',
  notes: ''
})

// Filtered and sorted contacts
const filteredContacts = computed(() => {
  let result = [...contacts.value]
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(c => 
      c.firstName.toLowerCase().includes(query) ||
      c.lastName.toLowerCase().includes(query) ||
      c.email.toLowerCase().includes(query) ||
      c.company.toLowerCase().includes(query)
    )
  }
  
  // Sort: favorites first, then alphabetically
  return result.sort((a, b) => {
    if (a.favorite !== b.favorite) return b.favorite ? 1 : -1
    const nameA = `${a.firstName} ${a.lastName}`.toLowerCase()
    const nameB = `${b.firstName} ${b.lastName}`.toLowerCase()
    return nameA.localeCompare(nameB)
  })
})

// Group contacts by first letter
const groupedContacts = computed(() => {
  const groups: Record<string, Contact[]> = {}
  
  // First add favorites
  const favorites = filteredContacts.value.filter(c => c.favorite)
  if (favorites.length > 0) {
    groups['★'] = favorites
  }
  
  // Then group by letter
  filteredContacts.value.filter(c => !c.favorite).forEach(contact => {
    const letter = (contact.firstName || contact.lastName || '#').charAt(0).toUpperCase()
    if (!groups[letter]) {
      groups[letter] = []
    }
    groups[letter].push(contact)
  })
  
  return groups
})

function getInitials(contact: Contact): string {
  const first = contact.firstName?.charAt(0) || ''
  const last = contact.lastName?.charAt(0) || ''
  return (first + last).toUpperCase() || '?'
}

function getAvatarColor(contact: Contact): string {
  const colors = [
    'bg-blue-500', 'bg-green-500', 'bg-yellow-500', 'bg-red-500',
    'bg-purple-500', 'bg-pink-500', 'bg-indigo-500', 'bg-teal-500'
  ]
  const hash = (contact.firstName + contact.lastName).split('').reduce((acc, char) => acc + char.charCodeAt(0), 0)
  return colors[hash % colors.length]
}

function openNewContactModal() {
  editingContact.value = null
  contactForm.value = {
    firstName: '',
    lastName: '',
    email: '',
    phone: '',
    company: '',
    jobTitle: '',
    birthday: '',
    notes: ''
  }
  showContactModal.value = true
}

function openEditContactModal(contact: Contact) {
  editingContact.value = contact
  contactForm.value = {
    firstName: contact.firstName,
    lastName: contact.lastName,
    email: contact.email,
    phone: contact.phone,
    company: contact.company,
    jobTitle: contact.jobTitle,
    birthday: contact.birthday?.split('T')[0] || '',
    notes: contact.notes
  }
  showContactModal.value = true
}

async function saveContact() {
  const data = {
    firstName: contactForm.value.firstName,
    lastName: contactForm.value.lastName,
    email: contactForm.value.email,
    phone: contactForm.value.phone,
    company: contactForm.value.company,
    jobTitle: contactForm.value.jobTitle,
    birthday: contactForm.value.birthday ? new Date(contactForm.value.birthday + 'T12:00:00').toISOString() : null,
    notes: contactForm.value.notes
  }
  
  try {
    if (editingContact.value) {
      const response = await api.put(`/contacts/${editingContact.value.id}`, data)
      const index = contacts.value.findIndex(c => c.id === editingContact.value!.id)
      if (index !== -1) {
        contacts.value[index] = response.data
      }
    } else {
      const response = await api.post('/contacts', data)
      contacts.value.push(response.data)
    }
    showContactModal.value = false
  } catch (err) {
    console.error('Failed to save contact:', err)
  }
}

function confirmDelete(contact: Contact) {
  contactToDelete.value = contact
  showDeleteConfirm.value = true
}

async function deleteContact() {
  if (!contactToDelete.value) return
  
  try {
    await api.delete(`/contacts/${contactToDelete.value.id}`)
    contacts.value = contacts.value.filter(c => c.id !== contactToDelete.value!.id)
    showDeleteConfirm.value = false
    showContactModal.value = false
    contactToDelete.value = null
  } catch (err) {
    console.error('Failed to delete contact:', err)
  }
}

async function toggleFavorite(contact: Contact) {
  try {
    const response = await api.patch(`/contacts/${contact.id}/favorite`, {
      favorite: !contact.favorite
    })
    const index = contacts.value.findIndex(c => c.id === contact.id)
    if (index !== -1) {
      contacts.value[index] = response.data
    }
  } catch (err) {
    console.error('Failed to toggle favorite:', err)
  }
}

function openEmailCompose(contact: Contact, event: Event) {
  event.stopPropagation()
  if (!contact.email) return
  emailComposeContact.value = contact
  showEmailCompose.value = true
}

function closeEmailCompose() {
  showEmailCompose.value = false
  emailComposeContact.value = null
}

async function fetchContacts() {
  loading.value = true
  try {
    const response = await api.get('/contacts')
    contacts.value = response.data || []
  } catch (err) {
    console.error('Failed to fetch contacts:', err)
    contacts.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchContacts()
})
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50 dark:bg-gray-900">
    <!-- Header -->
    <div class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4">
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Contacts</h1>
        <button
          @click="openNewContactModal"
          class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
          New Contact
        </button>
      </div>
      
      <!-- Search -->
      <div class="mt-4">
        <div class="relative">
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
          <input
            v-model="searchQuery"
            type="text"
            placeholder="Search contacts..."
            class="w-full pl-10 pr-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
          />
        </div>
      </div>
    </div>

    <!-- Contacts List -->
    <div class="flex-1 overflow-y-auto p-6">
      <div v-if="loading" class="flex items-center justify-center h-64">
        <svg class="w-8 h-8 text-blue-600 animate-spin" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>

      <div v-else-if="filteredContacts.length === 0" class="flex flex-col items-center justify-center h-64 text-gray-500 dark:text-gray-400">
        <svg class="w-16 h-16 mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
        </svg>
        <p class="text-lg font-medium">No contacts yet</p>
        <p class="text-sm">Add your first contact to get started</p>
      </div>

      <div v-else class="space-y-6">
        <div v-for="(groupContacts, letter) in groupedContacts" :key="letter">
          <h3 class="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-2 sticky top-0 bg-gray-50 dark:bg-gray-900 py-1">
            {{ letter }}
          </h3>
          <div class="bg-white dark:bg-gray-800 rounded-lg shadow-sm divide-y divide-gray-100 dark:divide-gray-700">
            <div
              v-for="contact in groupContacts"
              :key="contact.id"
              @click="openEditContactModal(contact)"
              class="flex items-center gap-4 p-4 hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer transition-colors"
            >
              <!-- Avatar -->
              <div
                :class="[getAvatarColor(contact), 'w-12 h-12 rounded-full flex items-center justify-center text-white font-semibold text-lg']"
              >
                {{ getInitials(contact) }}
              </div>

              <!-- Info -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <h4 class="font-medium text-gray-900 dark:text-white truncate">
                    {{ contact.firstName }} {{ contact.lastName }}
                  </h4>
                  <button
                    @click.stop="toggleFavorite(contact)"
                    :class="contact.favorite ? 'text-yellow-500' : 'text-gray-300 dark:text-gray-600 hover:text-yellow-500'"
                  >
                    <svg class="w-5 h-5" :fill="contact.favorite ? 'currentColor' : 'none'" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z" />
                    </svg>
                  </button>
                </div>
                <!-- Email with mail icon -->
                <div v-if="contact.email" class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
                  <button
                    @click="openEmailCompose(contact, $event)"
                    class="flex items-center gap-1.5 hover:text-blue-600 dark:hover:text-blue-400 transition-colors"
                    title="Send email"
                  >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                    <span class="truncate">{{ contact.email }}</span>
                  </button>
                </div>
                <p v-if="contact.company" class="text-sm text-gray-400 dark:text-gray-500 truncate">
                  {{ contact.company }}{{ contact.jobTitle ? ` · ${contact.jobTitle}` : '' }}
                </p>
              </div>

              <!-- Phone -->
              <div v-if="contact.phone" class="text-sm text-gray-500 dark:text-gray-400">
                {{ contact.phone }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Contact Modal -->
    <Teleport to="body">
      <div
        v-if="showContactModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
        @click.self="showContactModal = false"
      >
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
          <div class="p-6">
            <div class="flex items-center justify-between mb-4">
              <h2 class="text-xl font-bold text-gray-900 dark:text-white">
                {{ editingContact ? 'Edit Contact' : 'New Contact' }}
              </h2>
              <button
                v-if="editingContact"
                @click="confirmDelete(editingContact)"
                class="text-red-500 hover:text-red-700"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>

            <!-- Name -->
            <div class="grid grid-cols-2 gap-4 mb-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">First Name</label>
                <input
                  v-model="contactForm.firstName"
                  type="text"
                  class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  placeholder="John"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Last Name</label>
                <input
                  v-model="contactForm.lastName"
                  type="text"
                  class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  placeholder="Doe"
                />
              </div>
            </div>

            <!-- Email & Phone -->
            <div class="grid grid-cols-2 gap-4 mb-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Email</label>
                <input
                  v-model="contactForm.email"
                  type="email"
                  class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  placeholder="john@example.com"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Phone</label>
                <input
                  v-model="contactForm.phone"
                  type="tel"
                  class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  placeholder="+1 234 567 8900"
                />
              </div>
            </div>

            <!-- Company & Job Title -->
            <div class="grid grid-cols-2 gap-4 mb-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Company</label>
                <input
                  v-model="contactForm.company"
                  type="text"
                  class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  placeholder="Acme Inc."
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Job Title</label>
                <input
                  v-model="contactForm.jobTitle"
                  type="text"
                  class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                  placeholder="Software Engineer"
                />
              </div>
            </div>

            <!-- Birthday -->
            <div class="mb-4">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Birthday</label>
              <input
                v-model="contactForm.birthday"
                type="date"
                class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
              />
            </div>

            <!-- Notes -->
            <div class="mb-4">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Notes</label>
              <textarea
                v-model="contactForm.notes"
                rows="3"
                class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white resize-none"
                placeholder="Additional notes..."
              ></textarea>
            </div>

            <!-- Actions -->
            <div class="flex justify-end gap-2 pt-4 border-t dark:border-gray-700">
              <button
                @click="showContactModal = false"
                class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
              >
                Cancel
              </button>
              <button
                @click="saveContact"
                :disabled="!contactForm.firstName && !contactForm.lastName"
                class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {{ editingContact ? 'Save' : 'Create' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Delete Confirmation Modal -->
    <Teleport to="body">
      <div
        v-if="showDeleteConfirm"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        @click.self="showDeleteConfirm = false"
      >
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md p-6">
          <div class="flex items-center gap-3 mb-4">
            <div class="flex-shrink-0 w-10 h-10 flex items-center justify-center bg-red-100 dark:bg-red-900/30 rounded-full">
              <svg class="w-6 h-6 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </div>
            <div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Delete Contact</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">This action cannot be undone.</p>
            </div>
          </div>
          
          <p class="text-gray-700 dark:text-gray-300 mb-6">
            Are you sure you want to delete <span class="font-medium">{{ contactToDelete?.firstName }} {{ contactToDelete?.lastName }}</span>?
          </p>
          
          <div class="flex justify-end gap-3">
            <button
              @click="showDeleteConfirm = false"
              class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              Cancel
            </button>
            <button
              @click="deleteContact"
              class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
            >
              Delete
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Email Compose Modal -->
    <Teleport to="body">
      <EmailCompose
        v-if="showEmailCompose && emailComposeContact"
        :prefill-to="`${emailComposeContact.firstName} ${emailComposeContact.lastName} <${emailComposeContact.email}>`"
        @close="closeEmailCompose"
        @sent="closeEmailCompose"
      />
    </Teleport>
  </div>
</template>
