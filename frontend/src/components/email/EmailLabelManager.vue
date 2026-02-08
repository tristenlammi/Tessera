<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useEmailStore, type EmailLabel } from '@/stores/email'

const emit = defineEmits<{
  close: []
}>()

const emailStore = useEmailStore()

const newLabelName = ref('')
const newLabelColor = ref('#6B7280')
const editingLabel = ref<EmailLabel | null>(null)
const editName = ref('')
const editColor = ref('')
const loading = ref(false)
const error = ref('')
const showDeleteModal = ref(false)
const labelToDelete = ref<EmailLabel | null>(null)

const labels = computed(() => emailStore.customLabels)

const presetColors = [
  '#EF4444', // Red
  '#F97316', // Orange
  '#F59E0B', // Amber
  '#EAB308', // Yellow
  '#84CC16', // Lime
  '#22C55E', // Green
  '#10B981', // Emerald
  '#14B8A6', // Teal
  '#06B6D4', // Cyan
  '#0EA5E9', // Sky
  '#3B82F6', // Blue
  '#6366F1', // Indigo
  '#8B5CF6', // Violet
  '#A855F7', // Purple
  '#D946EF', // Fuchsia
  '#EC4899', // Pink
  '#6B7280', // Gray
]

async function createLabel() {
  if (!newLabelName.value.trim()) {
    error.value = 'Label name is required'
    return
  }

  loading.value = true
  error.value = ''
  try {
    await emailStore.createLabel(newLabelName.value.trim(), newLabelColor.value)
    newLabelName.value = ''
    newLabelColor.value = '#6B7280'
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to create label'
  } finally {
    loading.value = false
  }
}

function startEdit(label: EmailLabel) {
  editingLabel.value = label
  editName.value = label.name
  editColor.value = label.color
}

function cancelEdit() {
  editingLabel.value = null
  editName.value = ''
  editColor.value = ''
}

async function saveEdit() {
  if (!editingLabel.value || !editName.value.trim()) return

  loading.value = true
  error.value = ''
  try {
    await emailStore.updateLabel(editingLabel.value.id, editName.value.trim(), editColor.value)
    cancelEdit()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to update label'
  } finally {
    loading.value = false
  }
}

function confirmDeleteLabel(label: EmailLabel) {
  labelToDelete.value = label
  showDeleteModal.value = true
}

function cancelDelete() {
  labelToDelete.value = null
  showDeleteModal.value = false
}

async function deleteLabel() {
  if (!labelToDelete.value) return

  loading.value = true
  error.value = ''
  try {
    await emailStore.deleteLabel(labelToDelete.value.id)
    cancelDelete()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to delete label'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-md w-full max-h-[80vh] flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between p-4 border-b dark:border-gray-700">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">Manage Labels</h2>
        <button
          @click="emit('close')"
          class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
        >
          <svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Create new label -->
      <div class="p-4 border-b dark:border-gray-700">
        <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">Create new label</h3>
        <div class="flex gap-2">
          <div class="flex-1 relative">
            <input
              v-model="newLabelName"
              type="text"
              placeholder="Label name"
              class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              @keyup.enter="createLabel"
            />
          </div>
          <div class="relative">
            <input
              v-model="newLabelColor"
              type="color"
              class="w-10 h-10 rounded-lg border dark:border-gray-600 cursor-pointer"
              title="Choose color"
            />
          </div>
          <button
            @click="createLabel"
            :disabled="loading || !newLabelName.trim()"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors text-sm font-medium"
          >
            Add
          </button>
        </div>
        <!-- Preset colors -->
        <div class="flex flex-wrap gap-1 mt-2">
          <button
            v-for="color in presetColors"
            :key="color"
            @click="newLabelColor = color"
            :class="[
              'w-5 h-5 rounded-full transition-transform',
              newLabelColor === color ? 'ring-2 ring-offset-2 ring-blue-500 scale-110' : 'hover:scale-110'
            ]"
            :style="{ backgroundColor: color }"
          ></button>
        </div>
      </div>

      <!-- Error message -->
      <div v-if="error" class="px-4 py-2 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm">
        {{ error }}
      </div>

      <!-- Label list -->
      <div class="flex-1 overflow-y-auto p-4">
        <div v-if="labels.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
          <svg class="w-12 h-12 mx-auto mb-3 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
          </svg>
          <p>No labels yet</p>
          <p class="text-sm">Create a label to organize your emails</p>
        </div>

        <div v-else class="space-y-2">
          <div
            v-for="label in labels"
            :key="label.id"
            class="flex items-center gap-3 p-3 rounded-lg border dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-700/50"
          >
            <!-- Edit mode -->
            <template v-if="editingLabel?.id === label.id">
              <input
                v-model="editColor"
                type="color"
                class="w-8 h-8 rounded cursor-pointer flex-shrink-0"
              />
              <input
                v-model="editName"
                type="text"
                class="flex-1 px-2 py-1 border dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
                @keyup.enter="saveEdit"
                @keyup.escape="cancelEdit"
              />
              <button
                @click="saveEdit"
                class="p-1.5 text-green-600 hover:bg-green-100 dark:hover:bg-green-900/30 rounded"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                </svg>
              </button>
              <button
                @click="cancelEdit"
                class="p-1.5 text-gray-500 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </template>

            <!-- View mode -->
            <template v-else>
              <span
                class="w-4 h-4 rounded-full flex-shrink-0"
                :style="{ backgroundColor: label.color }"
              ></span>
              <span class="flex-1 text-gray-900 dark:text-white">{{ label.name }}</span>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ label.email_count || 0 }} emails
              </span>
              <button
                @click="startEdit(label)"
                class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded"
                title="Edit label"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                </svg>
              </button>
              <button
                @click="confirmDeleteLabel(label)"
                class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/30 rounded"
                title="Delete label"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </template>
          </div>
        </div>
      </div>
    </div>

    <!-- Delete confirmation modal -->
    <div v-if="showDeleteModal" class="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-black/50">
      <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-sm w-full p-6">
        <div class="flex items-center gap-3 mb-4">
          <div class="w-10 h-10 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
            <svg class="w-5 h-5 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Delete Label</h3>
            <p class="text-sm text-gray-500 dark:text-gray-400">This action cannot be undone</p>
          </div>
        </div>
        
        <p class="text-gray-700 dark:text-gray-300 mb-6">
          Are you sure you want to delete the label "<strong>{{ labelToDelete?.name }}</strong>"? 
          Emails with this label will not be deleted.
        </p>
        
        <div class="flex justify-end gap-3">
          <button
            @click="cancelDelete"
            class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            Cancel
          </button>
          <button
            @click="deleteLabel"
            :disabled="loading"
            class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50 transition-colors"
          >
            {{ loading ? 'Deleting...' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
