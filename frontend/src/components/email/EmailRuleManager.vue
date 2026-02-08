<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useEmailStore, type EmailRule, type RuleCondition, type RuleAction, type EmailLabel, type EmailFolder } from '@/stores/email'

const emit = defineEmits<{
  close: []
}>()

const emailStore = useEmailStore()

const showCreateForm = ref(false)
const editingRule = ref<EmailRule | null>(null)
const loading = ref(false)
const runningRuleId = ref<string | null>(null)
const error = ref('')

// Form state
const ruleName = ref('')
const matchType = ref<'any' | 'all'>('any')
const conditions = ref<RuleCondition[]>([])
const actions = ref<RuleAction[]>([])
const stopProcessing = ref(false)
const isEnabled = ref(true)

const rules = computed(() => emailStore.rules)
const labels = computed(() => emailStore.labels)
const folders = computed(() => emailStore.folders)

const fieldOptions = [
  { value: 'from', label: 'From' },
  { value: 'to', label: 'To' },
  { value: 'subject', label: 'Subject' },
  { value: 'body', label: 'Body' },
]

const operatorOptions = [
  { value: 'contains', label: 'Contains' },
  { value: 'equals', label: 'Equals' },
  { value: 'startswith', label: 'Starts with' },
  { value: 'endswith', label: 'Ends with' },
  { value: 'regex', label: 'Matches regex' },
]

const actionOptions = [
  { value: 'label', label: 'Apply label' },
  { value: 'move', label: 'Move to folder' },
  { value: 'star', label: 'Star email' },
  { value: 'mark_read', label: 'Mark as read' },
  { value: 'delete', label: 'Delete' },
]

onMounted(async () => {
  if (emailStore.currentAccount) {
    await emailStore.fetchRules(emailStore.currentAccount.id)
  }
})

function resetForm() {
  ruleName.value = ''
  matchType.value = 'any'
  conditions.value = [{ field: 'from', operator: 'contains', value: '' }]
  actions.value = [{ type: 'label', value: '' }]
  stopProcessing.value = false
  isEnabled.value = true
  editingRule.value = null
  error.value = ''
}

function openCreateForm() {
  resetForm()
  showCreateForm.value = true
}

function cancelForm() {
  showCreateForm.value = false
  editingRule.value = null
  resetForm()
}

function editRule(rule: EmailRule) {
  editingRule.value = rule
  ruleName.value = rule.name
  matchType.value = rule.match_type
  conditions.value = [...rule.conditions]
  actions.value = [...rule.actions]
  stopProcessing.value = rule.stop_processing
  isEnabled.value = rule.is_enabled
  showCreateForm.value = true
}

function addCondition() {
  conditions.value.push({ field: 'from', operator: 'contains', value: '' })
}

function removeCondition(index: number) {
  conditions.value.splice(index, 1)
}

function addAction() {
  actions.value.push({ type: 'label', value: '' })
}

function removeAction(index: number) {
  actions.value.splice(index, 1)
}

async function saveRule() {
  if (!ruleName.value.trim()) {
    error.value = 'Rule name is required'
    return
  }
  if (conditions.value.length === 0 || !conditions.value.every(c => c.value.trim())) {
    error.value = 'At least one condition with a value is required'
    return
  }
  if (actions.value.length === 0) {
    error.value = 'At least one action is required'
    return
  }

  // Validate action values for label/move
  for (const action of actions.value) {
    if ((action.type === 'label' || action.type === 'move') && !action.value) {
      error.value = `Please select a ${action.type === 'label' ? 'label' : 'folder'} for the action`
      return
    }
  }

  loading.value = true
  error.value = ''
  try {
    const ruleData = {
      name: ruleName.value.trim(),
      is_enabled: isEnabled.value,
      priority: 0,
      match_type: matchType.value,
      conditions: conditions.value,
      actions: actions.value,
      stop_processing: stopProcessing.value,
    }

    if (editingRule.value) {
      await emailStore.updateRule(editingRule.value.id, ruleData)
    } else {
      await emailStore.createRule(ruleData)
    }
    cancelForm()
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to save rule'
  } finally {
    loading.value = false
  }
}

async function deleteRule(rule: EmailRule) {
  if (!confirm(`Delete rule "${rule.name}"?`)) return

  loading.value = true
  error.value = ''
  try {
    await emailStore.deleteRule(rule.id)
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to delete rule'
  } finally {
    loading.value = false
  }
}

async function toggleRule(rule: EmailRule) {
  try {
    await emailStore.updateRule(rule.id, { is_enabled: !rule.is_enabled })
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to update rule'
  }
}

async function runRule(rule: EmailRule) {
  runningRuleId.value = rule.id
  error.value = ''
  try {
    const result = await emailStore.runRuleNow(rule.id)
    // Show success message briefly
    error.value = `✓ ${result.message}`
    setTimeout(() => {
      if (error.value.startsWith('✓')) error.value = ''
    }, 3000)
  } catch (e: any) {
    error.value = e.response?.data?.error || 'Failed to run rule'
  } finally {
    runningRuleId.value = null
  }
}

function getLabelName(id: string): string {
  const label = labels.value.find(l => l.id === id)
  return label?.name || 'Unknown label'
}

function getFolderName(id: string): string {
  const folder = folders.value.find(f => f.id === id)
  return folder?.name || 'Unknown folder'
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
    <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl max-w-2xl w-full max-h-[85vh] flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between p-4 border-b dark:border-gray-700">
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ showCreateForm ? (editingRule ? 'Edit Rule' : 'Create Rule') : 'Email Rules' }}
        </h2>
        <button
          @click="emit('close')"
          class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
        >
          <svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Error message -->
      <div v-if="error" class="px-4 py-2 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm">
        {{ error }}
      </div>

      <!-- Rule list view -->
      <template v-if="!showCreateForm">
        <div class="p-4 border-b dark:border-gray-700">
          <button
            @click="openCreateForm"
            class="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors text-sm font-medium"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            Create Rule
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-4">
          <div v-if="rules.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
            <svg class="w-12 h-12 mx-auto mb-3 text-gray-300 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
            </svg>
            <p>No rules yet</p>
            <p class="text-sm">Create a rule to automatically organize your emails</p>
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="rule in rules"
              :key="rule.id"
              :class="[
                'p-4 rounded-lg border transition-colors',
                rule.is_enabled
                  ? 'border-gray-200 dark:border-gray-700'
                  : 'border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800/50 opacity-60'
              ]"
            >
              <div class="flex items-start justify-between">
                <div class="flex-1">
                  <div class="flex items-center gap-2">
                    <h3 class="font-medium text-gray-900 dark:text-white">{{ rule.name }}</h3>
                    <span
                      v-if="!rule.is_enabled"
                      class="px-2 py-0.5 text-xs bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded"
                    >
                      Disabled
                    </span>
                  </div>
                  <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                    When {{ rule.match_type === 'any' ? 'any' : 'all' }} of {{ rule.conditions.length }} condition(s) match,
                    perform {{ rule.actions.length }} action(s)
                  </p>
                  <div class="flex flex-wrap gap-2 mt-2">
                    <span
                      v-for="(action, i) in rule.actions"
                      :key="i"
                      class="inline-flex items-center gap-1 px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded text-xs"
                    >
                      <template v-if="action.type === 'label'">Apply: {{ getLabelName(action.value) }}</template>
                      <template v-else-if="action.type === 'move'">Move to: {{ getFolderName(action.value) }}</template>
                      <template v-else-if="action.type === 'star'">⭐ Star</template>
                      <template v-else-if="action.type === 'mark_read'">Mark read</template>
                      <template v-else-if="action.type === 'delete'">🗑️ Delete</template>
                    </span>
                  </div>
                </div>
                <div class="flex items-center gap-1 ml-4">
                  <button
                    @click="toggleRule(rule)"
                    :class="[
                      'p-1.5 rounded transition-colors',
                      rule.is_enabled
                        ? 'text-green-600 hover:bg-green-50 dark:hover:bg-green-900/30'
                        : 'text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700'
                    ]"
                    :title="rule.is_enabled ? 'Disable rule' : 'Enable rule'"
                  >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path v-if="rule.is_enabled" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                      <path v-else stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
                    </svg>
                  </button>
                  <button
                    @click="runRule(rule)"
                    :disabled="runningRuleId === rule.id"
                    class="p-1.5 text-gray-400 hover:text-purple-600 hover:bg-purple-50 dark:hover:bg-purple-900/30 rounded disabled:opacity-50"
                    title="Run rule now on all emails"
                  >
                    <svg v-if="runningRuleId === rule.id" class="w-5 h-5 animate-spin" fill="none" viewBox="0 0 24 24">
                      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                      <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  </button>
                  <button
                    @click="editRule(rule)"
                    class="p-1.5 text-gray-400 hover:text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/30 rounded"
                    title="Edit rule"
                  >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                    </svg>
                  </button>
                  <button
                    @click="deleteRule(rule)"
                    class="p-1.5 text-gray-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/30 rounded"
                    title="Delete rule"
                  >
                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- Create/Edit form -->
      <template v-else>
        <div class="flex-1 overflow-y-auto p-4 space-y-4">
          <!-- Rule name -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Rule name</label>
            <input
              v-model="ruleName"
              type="text"
              placeholder="e.g., Move newsletters to Promotions"
              class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          <!-- Match type -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Match</label>
            <select
              v-model="matchType"
              class="px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500"
            >
              <option value="any">Any of the following conditions</option>
              <option value="all">All of the following conditions</option>
            </select>
          </div>

          <!-- Conditions -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Conditions</label>
              <button
                @click="addCondition"
                class="text-sm text-blue-600 hover:text-blue-700 font-medium"
              >
                + Add condition
              </button>
            </div>
            <div class="space-y-2">
              <div
                v-for="(condition, index) in conditions"
                :key="index"
                class="flex items-center gap-2"
              >
                <select
                  v-model="condition.field"
                  class="px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500"
                >
                  <option v-for="opt in fieldOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </option>
                </select>
                <select
                  v-model="condition.operator"
                  class="px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500"
                >
                  <option v-for="opt in operatorOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </option>
                </select>
                <input
                  v-model="condition.value"
                  type="text"
                  placeholder="Value"
                  class="flex-1 px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
                <button
                  v-if="conditions.length > 1"
                  @click="removeCondition(index)"
                  class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/30 rounded"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          </div>

          <!-- Actions -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">Actions</label>
              <button
                @click="addAction"
                class="text-sm text-blue-600 hover:text-blue-700 font-medium"
              >
                + Add action
              </button>
            </div>
            <div class="space-y-2">
              <div
                v-for="(action, index) in actions"
                :key="index"
                class="flex items-center gap-2"
              >
                <select
                  v-model="action.type"
                  @change="action.value = ''"
                  class="px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500"
                >
                  <option v-for="opt in actionOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </option>
                </select>
                <select
                  v-if="action.type === 'label'"
                  v-model="action.value"
                  class="flex-1 px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500"
                >
                  <option value="">Select label...</option>
                  <option v-for="label in labels" :key="label.id" :value="label.id">
                    {{ label.name }}
                  </option>
                </select>
                <select
                  v-else-if="action.type === 'move'"
                  v-model="action.value"
                  class="flex-1 px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm focus:ring-2 focus:ring-blue-500"
                >
                  <option value="">Select folder...</option>
                  <option v-for="folder in folders" :key="folder.id" :value="folder.id">
                    {{ folder.name }}
                  </option>
                </select>
                <span
                  v-else
                  class="flex-1 text-sm text-gray-500 dark:text-gray-400 italic"
                >
                  {{ action.type === 'star' ? 'Star the email' : action.type === 'mark_read' ? 'Mark as read' : 'Delete the email' }}
                </span>
                <button
                  v-if="actions.length > 1"
                  @click="removeAction(index)"
                  class="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 dark:hover:bg-red-900/30 rounded"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          </div>

          <!-- Options -->
          <div class="space-y-3">
            <label class="flex items-center gap-2">
              <input
                v-model="isEnabled"
                type="checkbox"
                class="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">Enable rule</span>
            </label>
            <label class="flex items-center gap-2">
              <input
                v-model="stopProcessing"
                type="checkbox"
                class="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500"
              />
              <span class="text-sm text-gray-700 dark:text-gray-300">Stop processing other rules when this matches</span>
            </label>
          </div>
        </div>

        <!-- Form actions -->
        <div class="flex items-center justify-end gap-3 p-4 border-t dark:border-gray-700">
          <button
            @click="cancelForm"
            class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
          >
            Cancel
          </button>
          <button
            @click="saveRule"
            :disabled="loading"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors font-medium"
          >
            {{ editingRule ? 'Save Changes' : 'Create Rule' }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>
