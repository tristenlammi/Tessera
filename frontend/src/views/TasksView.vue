<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import draggable from 'vuedraggable'
import DOMPurify from 'dompurify'
import { useTasksStore, type Task, type RecurrenceRule, type TaskGroup, type ChecklistItem } from '@/stores/tasks'
import { useEmailStore } from '@/stores/email'
import api from '@/api'

const tasksStore = useTasksStore()
const emailStore = useEmailStore()

const showTaskModal = ref(false)
const showGroupModal = ref(false)
const showDeleteConfirm = ref(false)
const taskToDelete = ref<Task | null>(null)
const editingTask = ref<Task | null>(null)
const editingGroup = ref<TaskGroup | null>(null)

// Task form
const taskForm = ref({
  title: '',
  description: '',
  priority: 'medium' as Task['priority'],
  dueDate: '',
  groupId: null as string | null,
  tags: [] as string[],
  recurrence: null as RecurrenceRule | null,
  checklist: [] as ChecklistItem[]
})

// Group form
const groupForm = ref({
  name: '',
  color: '#3b82f6'
})

// Recurrence form
const recurrenceEnabled = ref(false)
const recurrenceForm = ref({
  type: 'daily' as RecurrenceRule['type'],
  interval: 1,
  endDate: '',
  daysOfWeek: [] as number[],
  dayOfMonth: null as number | null,
  occurrences: null as number | null
})

// Tag input
const tagInput = ref('')

// Add to calendar option
const addToCalendar = ref(false)

// Checklist input
const checklistInput = ref('')

// Email preview modal
const showEmailPreview = ref(false)
const previewEmail = ref<any>(null)
const loadingEmail = ref(false)

// Days of week for selection
const weekDays = [
  { value: 0, label: 'Sun' },
  { value: 1, label: 'Mon' },
  { value: 2, label: 'Tue' },
  { value: 3, label: 'Wed' },
  { value: 4, label: 'Thu' },
  { value: 5, label: 'Fri' },
  { value: 6, label: 'Sat' }
]

// Recurrence preview - computed from form settings
const recurrencePreview = computed(() => {
  if (!recurrenceEnabled.value || !taskForm.value.dueDate) return []
  
  const previews: string[] = []
  let current = new Date(taskForm.value.dueDate)
  
  for (let i = 0; i < 5; i++) {
    if (i === 0) {
      previews.push(formatPreviewDate(current))
    } else {
      current = getNextOccurrence(current)
      if (!current) break
      
      // Check end date
      if (recurrenceForm.value.endDate && current > new Date(recurrenceForm.value.endDate)) break
      
      // Check max occurrences
      if (recurrenceForm.value.occurrences && i >= recurrenceForm.value.occurrences) break
      
      previews.push(formatPreviewDate(current))
    }
  }
  
  return previews
})

function getNextOccurrence(current: Date): Date | null {
  const interval = recurrenceForm.value.interval || 1
  let next = new Date(current)
  
  switch (recurrenceForm.value.type) {
    case 'daily':
      next.setDate(next.getDate() + interval)
      break
    case 'weekly':
      if (recurrenceForm.value.daysOfWeek.length > 0) {
        // Find next day in daysOfWeek
        for (let d = 1; d <= 7 * interval; d++) {
          next.setDate(current.getDate() + d)
          if (recurrenceForm.value.daysOfWeek.includes(next.getDay())) {
            break
          }
        }
      } else {
        next.setDate(next.getDate() + interval * 7)
      }
      break
    case 'monthly':
      if (recurrenceForm.value.dayOfMonth) {
        next.setMonth(next.getMonth() + interval)
        next.setDate(Math.min(recurrenceForm.value.dayOfMonth, getDaysInMonth(next)))
      } else {
        next.setMonth(next.getMonth() + interval)
      }
      break
    case 'yearly':
      next.setFullYear(next.getFullYear() + interval)
      break
    default:
      return null
  }
  
  return next
}

function getDaysInMonth(date: Date): number {
  return new Date(date.getFullYear(), date.getMonth() + 1, 0).getDate()
}

function formatPreviewDate(date: Date): string {
  return date.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric', year: 'numeric' })
}

const priorityColors = {
  low: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
  medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300',
  high: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300'
}

const groupColors = [
  '#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6',
  '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#6366f1'
]

function openNewTaskModal() {
  editingTask.value = null
  taskForm.value = {
    title: '',
    description: '',
    priority: 'medium',
    dueDate: '',
    groupId: null,
    tags: [],
    recurrence: null,
    checklist: []
  }
  recurrenceEnabled.value = false
  recurrenceForm.value = {
    type: 'daily',
    interval: 1,
    endDate: '',
    daysOfWeek: [],
    dayOfMonth: null,
    occurrences: null
  }
  checklistInput.value = ''
  addToCalendar.value = false
  showTaskModal.value = true
}

function openEditTaskModal(task: Task) {
  editingTask.value = task
  taskForm.value = {
    title: task.title,
    description: task.description,
    priority: task.priority,
    dueDate: task.dueDate ? task.dueDate.split('T')[0] : '',
    groupId: task.groupId,
    tags: [...task.tags],
    recurrence: task.recurrence ? { ...task.recurrence } : null,
    checklist: task.checklist ? task.checklist.map(item => ({ ...item })) : []
  }
  recurrenceEnabled.value = !!task.recurrence
  if (task.recurrence) {
    recurrenceForm.value = {
      type: task.recurrence.type,
      interval: task.recurrence.interval,
      endDate: task.recurrence.endDate || '',
      daysOfWeek: task.recurrence.daysOfWeek || [],
      dayOfMonth: task.recurrence.dayOfMonth || null,
      occurrences: task.recurrence.occurrences || null
    }
  } else {
    recurrenceForm.value = {
      type: 'daily',
      interval: 1,
      endDate: '',
      daysOfWeek: [],
      dayOfMonth: null,
      occurrences: null
    }
  }
  checklistInput.value = ''
  showTaskModal.value = true
}

async function saveTask() {
  // Convert recurrence end date to ISO format if provided
  let recurrenceEndDate: string | undefined
  if (recurrenceForm.value.endDate) {
    recurrenceEndDate = new Date(recurrenceForm.value.endDate + 'T23:59:59').toISOString()
  }

  const recurrence = recurrenceEnabled.value ? {
    type: recurrenceForm.value.type,
    interval: recurrenceForm.value.interval,
    endDate: recurrenceEndDate,
    daysOfWeek: recurrenceForm.value.type === 'weekly' && recurrenceForm.value.daysOfWeek.length > 0
      ? recurrenceForm.value.daysOfWeek
      : undefined,
    dayOfMonth: recurrenceForm.value.type === 'monthly' && recurrenceForm.value.dayOfMonth
      ? recurrenceForm.value.dayOfMonth
      : undefined,
    occurrences: recurrenceForm.value.occurrences || undefined
  } : null

  // Convert date string to ISO format for backend
  let dueDate: string | null = null
  if (taskForm.value.dueDate) {
    dueDate = new Date(taskForm.value.dueDate + 'T12:00:00').toISOString()
  }

  const data: Partial<Task> = {
    title: taskForm.value.title,
    description: taskForm.value.description,
    priority: taskForm.value.priority,
    dueDate,
    groupId: taskForm.value.groupId,
    tags: taskForm.value.tags,
    recurrence,
    checklist: taskForm.value.checklist
  }

  try {
    let createdTask: any = null
    if (editingTask.value) {
      await tasksStore.updateTask(editingTask.value.id, data)
    } else {
      createdTask = await tasksStore.createTask({ ...data, status: 'todo' })
    }

    // Create calendar event if option is checked and there's a due date
    if (addToCalendar.value && dueDate) {
      try {
        const eventColor = taskForm.value.priority === 'high' ? '#ef4444' 
          : taskForm.value.priority === 'low' ? '#10b981' 
          : '#f59e0b'
        
        await api.post('/calendar/events', {
          title: taskForm.value.title,
          description: taskForm.value.description || `Task: ${taskForm.value.title}`,
          startDate: dueDate,
          endDate: dueDate,
          allDay: true,
          color: eventColor,
          linkedTaskId: createdTask?.id || editingTask.value?.id || null
        })
      } catch (calErr) {
        console.error('Failed to create calendar event:', calErr)
        // Don't fail the task save if calendar event fails
      }
    }

    showTaskModal.value = false
  } catch (err) {
    console.error('Failed to save task:', err)
  }
}

async function deleteTask(task: Task) {
  taskToDelete.value = task
  showDeleteConfirm.value = true
}

async function confirmDeleteTask() {
  if (!taskToDelete.value) return
  try {
    await tasksStore.deleteTask(taskToDelete.value.id)
    showDeleteConfirm.value = false
    showTaskModal.value = false
    taskToDelete.value = null
  } catch (err) {
    console.error('Failed to delete task:', err)
  }
}

function cancelDeleteTask() {
  showDeleteConfirm.value = false
  taskToDelete.value = null
}

function addTag() {
  const tag = tagInput.value.trim()
  if (tag && !taskForm.value.tags.includes(tag)) {
    taskForm.value.tags.push(tag)
  }
  tagInput.value = ''
}

function removeTag(tag: string) {
  taskForm.value.tags = taskForm.value.tags.filter(t => t !== tag)
}

// Checklist functions
function addChecklistItem() {
  const title = checklistInput.value.trim()
  if (title) {
    taskForm.value.checklist.push({
      id: crypto.randomUUID(),
      title,
      completed: false,
      order: taskForm.value.checklist.length
    })
    checklistInput.value = ''
  }
}

function removeChecklistItem(id: string) {
  taskForm.value.checklist = taskForm.value.checklist.filter(item => item.id !== id)
  // Re-order remaining items
  taskForm.value.checklist.forEach((item, index) => {
    item.order = index
  })
}

function toggleChecklistItem(id: string) {
  const item = taskForm.value.checklist.find(item => item.id === id)
  if (item) {
    item.completed = !item.completed
  }
}

function toggleDayOfWeek(day: number) {
  const index = recurrenceForm.value.daysOfWeek.indexOf(day)
  if (index === -1) {
    recurrenceForm.value.daysOfWeek.push(day)
    recurrenceForm.value.daysOfWeek.sort()
  } else {
    recurrenceForm.value.daysOfWeek.splice(index, 1)
  }
}

// Group functions
function openNewGroupModal() {
  editingGroup.value = null
  groupForm.value = { name: '', color: '#3b82f6' }
  showGroupModal.value = true
}

function openEditGroupModal(group: TaskGroup) {
  editingGroup.value = group
  groupForm.value = { name: group.name, color: group.color }
  showGroupModal.value = true
}

async function saveGroup() {
  try {
    if (editingGroup.value) {
      await tasksStore.updateGroup(editingGroup.value.id, groupForm.value)
    } else {
      await tasksStore.createGroup(groupForm.value)
    }
    showGroupModal.value = false
  } catch (err) {
    console.error('Failed to save group:', err)
  }
}

async function deleteGroup(group: TaskGroup) {
  if (confirm('Are you sure you want to delete this group? Tasks in this group will be ungrouped.')) {
    await tasksStore.deleteGroup(group.id)
  }
}

// Drag and drop
function onDragChange(columnId: Task['status'], event: any) {
  // Capture previous status before reorderTasks updates it
  let previousStatus: Task['status'] | null = null
  if (event.added) {
    const task = event.added.element as Task
    previousStatus = task.status  // still holds old status before reorder
  }

  if (event.added || event.moved) {
    const column = tasksStore.columns.find(c => c.id === columnId)
    if (column) {
      const taskIds = column.tasks.map(t => t.id)
      tasksStore.reorderTasks(columnId, taskIds)
    }

    // Handle calendar sync when task moves to/from done
    if (event.added) {
      const task = event.added.element as Task
      if (columnId === 'done' && task.dueDate) {
        // Task moved to done — remove from calendar
        api.delete(`/calendar/events/by-task/${task.id}`).catch(() => {})
      } else if (columnId !== 'done' && previousStatus === 'done' && task.dueDate) {
        // Task moved back from done — delete any stale event first, then re-add
        api.delete(`/calendar/events/by-task/${task.id}`)
          .catch(() => {})
          .then(() => {
            const eventColor = task.priority === 'high' ? '#ef4444'
              : task.priority === 'low' ? '#10b981'
              : '#f59e0b'
            api.post('/calendar/events', {
              title: task.title,
              description: task.description || `Task: ${task.title}`,
              startDate: task.dueDate,
              endDate: task.dueDate,
              allDay: true,
              color: eventColor,
              linkedTaskId: task.id
            }).catch(() => {})
          })
      }
    }
  }
}

function formatDate(dateStr: string | null): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const today = new Date()
  const tomorrow = new Date(today)
  tomorrow.setDate(tomorrow.getDate() + 1)

  if (date.toDateString() === today.toDateString()) {
    return 'Today'
  } else if (date.toDateString() === tomorrow.toDateString()) {
    return 'Tomorrow'
  } else {
    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
  }
}

function isOverdue(dateStr: string | null): boolean {
  if (!dateStr) return false
  return new Date(dateStr) < new Date(new Date().setHours(0, 0, 0, 0))
}

function getGroupColor(groupId: string | null): string {
  if (!groupId) return 'transparent'
  const group = tasksStore.groups.find(g => g.id === groupId)
  return group?.color || 'transparent'
}

function getGroupName(groupId: string | null): string {
  if (!groupId) return ''
  const group = tasksStore.groups.find(g => g.id === groupId)
  return group?.name || ''
}

// Linked email functions
async function openLinkedEmail(task: Task) {
  if (!task.linkedEmailId) return
  
  loadingEmail.value = true
  showEmailPreview.value = true
  
  try {
    const response = await api.get(`/email/emails/${task.linkedEmailId}`)
    previewEmail.value = response.data
  } catch (err) {
    console.error('Failed to load email:', err)
    previewEmail.value = null
  } finally {
    loadingEmail.value = false
  }
}

function closeEmailPreview() {
  showEmailPreview.value = false
  previewEmail.value = null
}

function formatEmailDate(dateStr: string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    weekday: 'short',
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit'
  })
}

function sanitizeHtml(html: string): string {
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ['p', 'br', 'b', 'i', 'u', 'strong', 'em', 'a', 'ul', 'ol', 'li', 'blockquote', 'div', 'span', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'table', 'tr', 'td', 'th', 'thead', 'tbody', 'img'],
    ALLOWED_ATTR: ['href', 'target', 'style', 'class', 'src', 'alt', 'width', 'height']
  })
}

onMounted(async () => {
  await Promise.all([
    tasksStore.fetchTasks(),
    tasksStore.fetchGroups()
  ])
})
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50 dark:bg-gray-900">
    <!-- Header -->
    <div class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4">
      <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Tasks</h1>
        <div class="flex items-center gap-3">
          <button
            @click="openNewGroupModal"
            class="px-4 py-2 text-gray-700 dark:text-gray-300 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 flex items-center gap-2"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z" />
            </svg>
            New Group
          </button>
          <button
            @click="openNewTaskModal"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            New Task
          </button>
        </div>
      </div>

      <!-- Groups -->
      <div v-if="tasksStore.groups.length > 0" class="flex items-center gap-2 mt-4">
        <span class="text-sm text-gray-500 dark:text-gray-400">Groups:</span>
        <div
          v-for="group in tasksStore.groups"
          :key="group.id"
          class="flex items-center gap-1.5 px-2 py-1 rounded-full text-sm cursor-pointer hover:opacity-80"
          :style="{ backgroundColor: group.color + '20', color: group.color }"
          @click="openEditGroupModal(group)"
        >
          <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: group.color }"></span>
          {{ group.name }}
        </div>
      </div>
    </div>

    <!-- Kanban Board -->
    <div class="flex-1 overflow-x-auto p-6">
      <div class="flex gap-6 h-full min-w-max">
        <!-- Columns -->
        <div
          v-for="column in tasksStore.columns"
          :key="column.id"
          class="w-80 flex flex-col"
        >
          <!-- Column Header -->
          <div class="flex items-center justify-between mb-4">
            <div class="flex items-center gap-2">
              <h2 class="font-semibold text-gray-900 dark:text-white">{{ column.title }}</h2>
              <span class="px-2 py-0.5 text-xs bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded-full">
                {{ column.tasks.length }}
              </span>
            </div>
            <button
              v-if="column.id === 'todo'"
              @click="openNewTaskModal"
              class="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
            >
              <svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
              </svg>
            </button>
          </div>

          <!-- Tasks Container -->
          <div class="flex-1 bg-gray-100 dark:bg-gray-800 rounded-lg p-2 overflow-y-auto">
            <draggable
              :list="column.tasks"
              group="tasks"
              item-key="id"
              class="space-y-2 min-h-[200px]"
              ghost-class="opacity-50"
              @change="(e: any) => onDragChange(column.id, e)"
            >
              <template #item="{ element: task }">
                <div
                  :class="[
                    'rounded-lg p-3 shadow-sm cursor-grab hover:shadow-md transition-shadow border-l-4',
                    task.status === 'done'
                      ? 'bg-gray-100 dark:bg-gray-700/50 opacity-60'
                      : 'bg-white dark:bg-gray-700'
                  ]"
                  :style="{ borderLeftColor: getGroupColor(task.groupId) }"
                  @click="openEditTaskModal(task)"
                >
                  <!-- Title -->
                  <h3 class="font-medium text-gray-900 dark:text-white mb-2">
                    {{ task.title }}
                  </h3>

                  <!-- Description preview -->
                  <p v-if="task.description" class="text-sm text-gray-500 dark:text-gray-400 mb-2 line-clamp-2">
                    {{ task.description }}
                  </p>

                  <!-- Meta row -->
                  <div class="flex items-center gap-2 flex-wrap">
                    <!-- Priority -->
                    <span
                      :class="priorityColors[task.priority]"
                      class="px-2 py-0.5 text-xs rounded-full capitalize"
                    >
                      {{ task.priority }}
                    </span>

                    <!-- Due date -->
                    <span
                      v-if="task.dueDate"
                      :class="[
                        'px-2 py-0.5 text-xs rounded-full',
                        isOverdue(task.dueDate) && task.status !== 'done'
                          ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300'
                          : 'bg-gray-100 text-gray-600 dark:bg-gray-600 dark:text-gray-300'
                      ]"
                    >
                      {{ formatDate(task.dueDate) }}
                    </span>

                    <!-- Recurrence indicator -->
                    <span
                      v-if="task.recurrence"
                      class="text-gray-400 dark:text-gray-500"
                      title="Recurring task"
                    >
                      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                      </svg>
                    </span>

                    <!-- Group -->
                    <span
                      v-if="task.groupId"
                      class="text-xs"
                      :style="{ color: getGroupColor(task.groupId) }"
                    >
                      {{ getGroupName(task.groupId) }}
                    </span>
                  </div>

                  <!-- Tags -->
                  <div v-if="task.tags.length > 0" class="flex items-center gap-1 mt-2 flex-wrap">
                    <span
                      v-for="tag in task.tags"
                      :key="tag"
                      class="px-2 py-0.5 text-xs bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 rounded"
                    >
                      {{ tag }}
                    </span>
                  </div>

                  <!-- Checklist Progress -->
                  <div
                    v-if="task.checklist && task.checklist.length > 0"
                    class="flex items-center gap-2 mt-2"
                  >
                    <svg class="w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
                    </svg>
                    <div class="flex-1 flex items-center gap-2">
                      <div class="flex-1 h-1.5 bg-gray-200 dark:bg-gray-600 rounded-full overflow-hidden">
                        <div
                          class="h-full bg-green-500 rounded-full transition-all"
                          :style="{ width: `${(task.checklist.filter(i => i.completed).length / task.checklist.length) * 100}%` }"
                        ></div>
                      </div>
                      <span class="text-xs text-gray-500 dark:text-gray-400">
                        {{ task.checklist.filter(i => i.completed).length }}/{{ task.checklist.length }}
                      </span>
                    </div>
                  </div>

                  <!-- Linked Email -->
                  <div
                    v-if="task.linkedEmailId"
                    @click.stop="openLinkedEmail(task)"
                    class="flex items-center gap-2 mt-2 p-2 bg-gray-50 dark:bg-gray-600 rounded hover:bg-gray-100 dark:hover:bg-gray-500 cursor-pointer transition-colors"
                  >
                    <svg class="w-4 h-4 text-blue-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                    </svg>
                    <span class="text-xs text-gray-600 dark:text-gray-300 truncate">
                      {{ task.linkedEmailSubject || 'Linked Email' }}
                    </span>
                  </div>
                </div>
              </template>
            </draggable>
          </div>
        </div>
      </div>
    </div>

    <!-- Task Modal -->
    <div v-if="showTaskModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div class="p-6">
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-xl font-bold text-gray-900 dark:text-white">
              {{ editingTask ? 'Edit Task' : 'New Task' }}
            </h2>
            <button
              v-if="editingTask"
              @click="deleteTask(editingTask)"
              class="text-red-500 hover:text-red-700"
            >
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </button>
          </div>

          <!-- Title -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Title</label>
            <input
              v-model="taskForm.title"
              type="text"
              class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
              placeholder="Task title"
            />
          </div>

          <!-- Description -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
            <textarea
              v-model="taskForm.description"
              rows="3"
              class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white resize-none"
              placeholder="Task description"
            ></textarea>
          </div>

          <!-- Priority & Due Date -->
          <div class="grid grid-cols-2 gap-4 mb-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Priority</label>
              <select
                v-model="taskForm.priority"
                class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
              >
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Due Date</label>
              <input
                v-model="taskForm.dueDate"
                type="date"
                class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
              />
            </div>
          </div>

          <!-- Add to Calendar (only show for new tasks with a due date) -->
          <div v-if="!editingTask && taskForm.dueDate" class="mb-4">
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

          <!-- Group -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Group</label>
            <select
              v-model="taskForm.groupId"
              class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
            >
              <option :value="null">No group</option>
              <option v-for="group in tasksStore.groups" :key="group.id" :value="group.id">
                {{ group.name }}
              </option>
            </select>
          </div>

          <!-- Tags -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Tags</label>
            <div class="flex items-center gap-2">
              <input
                v-model="tagInput"
                type="text"
                class="flex-1 px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                placeholder="Add tag"
                @keydown.enter.prevent="addTag"
              />
              <button
                @click="addTag"
                class="px-4 py-2 bg-gray-200 dark:bg-gray-600 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-500"
              >
                Add
              </button>
            </div>
            <div v-if="taskForm.tags.length > 0" class="flex items-center gap-2 mt-2 flex-wrap">
              <span
                v-for="tag in taskForm.tags"
                :key="tag"
                class="px-2 py-1 text-sm bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300 rounded flex items-center gap-1"
              >
                {{ tag }}
                <button @click="removeTag(tag)" class="hover:text-blue-900">×</button>
              </span>
            </div>
          </div>

          <!-- Checklist -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Checklist</label>
            <div class="flex items-center gap-2 mb-2">
              <input
                v-model="checklistInput"
                type="text"
                class="flex-1 px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                placeholder="Add checklist item"
                @keydown.enter.prevent="addChecklistItem"
              />
              <button
                @click="addChecklistItem"
                class="px-4 py-2 bg-gray-200 dark:bg-gray-600 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-500"
              >
                Add
              </button>
            </div>
            <div v-if="taskForm.checklist.length > 0" class="space-y-2 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg">
              <div
                v-for="item in taskForm.checklist"
                :key="item.id"
                class="flex items-center gap-2 group"
              >
                <input
                  type="checkbox"
                  :checked="item.completed"
                  @change="toggleChecklistItem(item.id)"
                  class="w-4 h-4 rounded"
                />
                <span
                  :class="[
                    'flex-1 text-sm',
                    item.completed ? 'line-through text-gray-400' : 'text-gray-700 dark:text-gray-300'
                  ]"
                >
                  {{ item.title }}
                </span>
                <button
                  @click="removeChecklistItem(item.id)"
                  class="opacity-0 group-hover:opacity-100 text-red-500 hover:text-red-700 transition-opacity"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          </div>

          <!-- Recurrence -->
          <div class="mb-4">
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                v-model="recurrenceEnabled"
                type="checkbox"
                class="w-4 h-4 rounded"
              />
              <span class="text-sm font-medium text-gray-700 dark:text-gray-300">Recurring task</span>
            </label>

            <div v-if="recurrenceEnabled" class="mt-3 p-3 bg-gray-50 dark:bg-gray-700 rounded-lg space-y-3">
              <div class="flex items-center gap-3">
                <span class="text-sm text-gray-600 dark:text-gray-400">Repeat every</span>
                <input
                  v-model.number="recurrenceForm.interval"
                  type="number"
                  min="1"
                  class="w-16 px-2 py-1 border rounded dark:bg-gray-600 dark:border-gray-500 dark:text-white text-center"
                />
                <select
                  v-model="recurrenceForm.type"
                  class="px-3 py-1 border rounded dark:bg-gray-600 dark:border-gray-500 dark:text-white"
                >
                  <option value="daily">day(s)</option>
                  <option value="weekly">week(s)</option>
                  <option value="monthly">month(s)</option>
                  <option value="yearly">year(s)</option>
                </select>
              </div>

              <!-- Weekly: Days of week selection -->
              <div v-if="recurrenceForm.type === 'weekly'" class="space-y-1">
                <span class="text-sm text-gray-600 dark:text-gray-400">On days:</span>
                <div class="flex flex-wrap gap-1">
                  <button
                    v-for="day in weekDays"
                    :key="day.value"
                    @click="toggleDayOfWeek(day.value)"
                    :class="[
                      'px-3 py-1 text-sm rounded-full transition-colors',
                      recurrenceForm.daysOfWeek.includes(day.value)
                        ? 'bg-blue-600 text-white'
                        : 'bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-500'
                    ]"
                  >
                    {{ day.label }}
                  </button>
                </div>
              </div>

              <!-- Monthly: Day of month selection -->
              <div v-if="recurrenceForm.type === 'monthly'" class="flex items-center gap-2">
                <span class="text-sm text-gray-600 dark:text-gray-400">On day:</span>
                <select
                  v-model.number="recurrenceForm.dayOfMonth"
                  class="px-3 py-1 border rounded dark:bg-gray-600 dark:border-gray-500 dark:text-white"
                >
                  <option :value="null">Same as due date</option>
                  <option v-for="d in 31" :key="d" :value="d">{{ d }}</option>
                </select>
              </div>

              <!-- End conditions -->
              <div class="space-y-2 pt-2 border-t dark:border-gray-600">
                <span class="text-sm text-gray-600 dark:text-gray-400">End:</span>
                <div class="flex items-center gap-4">
                  <div class="flex items-center gap-2">
                    <label class="text-sm text-gray-600 dark:text-gray-400">On date:</label>
                    <input
                      v-model="recurrenceForm.endDate"
                      type="date"
                      class="px-3 py-1 border rounded dark:bg-gray-600 dark:border-gray-500 dark:text-white"
                    />
                  </div>
                  <span class="text-sm text-gray-400">or</span>
                  <div class="flex items-center gap-2">
                    <label class="text-sm text-gray-600 dark:text-gray-400">After:</label>
                    <input
                      v-model.number="recurrenceForm.occurrences"
                      type="number"
                      min="1"
                      placeholder="∞"
                      class="w-16 px-2 py-1 border rounded dark:bg-gray-600 dark:border-gray-500 dark:text-white text-center"
                    />
                    <span class="text-sm text-gray-600 dark:text-gray-400">times</span>
                  </div>
                </div>
              </div>

              <!-- Recurrence Preview -->
              <div v-if="recurrencePreview.length > 0" class="pt-2 border-t dark:border-gray-600">
                <span class="text-sm text-gray-600 dark:text-gray-400">Next occurrences:</span>
                <div class="mt-1 flex flex-wrap gap-2">
                  <span
                    v-for="(date, index) in recurrencePreview"
                    :key="index"
                    class="text-xs px-2 py-1 bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300 rounded"
                  >
                    {{ date }}
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- Actions -->
          <div class="flex justify-end gap-2 pt-4 border-t dark:border-gray-700">
            <button
              @click="showTaskModal = false"
              class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              Cancel
            </button>
            <button
              @click="saveTask"
              class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              {{ editingTask ? 'Save' : 'Create' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Group Modal -->
    <div v-if="showGroupModal" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl p-6 w-full max-w-md">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-xl font-bold text-gray-900 dark:text-white">
            {{ editingGroup ? 'Edit Group' : 'New Group' }}
          </h2>
          <button
            v-if="editingGroup"
            @click="deleteGroup(editingGroup)"
            class="text-red-500 hover:text-red-700"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Name</label>
          <input
            v-model="groupForm.name"
            type="text"
            class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
            placeholder="Group name"
          />
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Color</label>
          <div class="flex items-center gap-2 flex-wrap">
            <button
              v-for="color in groupColors"
              :key="color"
              @click="groupForm.color = color"
              :class="[
                'w-8 h-8 rounded-full border-2',
                groupForm.color === color ? 'border-gray-800 dark:border-white' : 'border-transparent'
              ]"
              :style="{ backgroundColor: color }"
            ></button>
          </div>
        </div>

        <div class="flex justify-end gap-2">
          <button
            @click="showGroupModal = false"
            class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
          >
            Cancel
          </button>
          <button
            @click="saveGroup"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            {{ editingGroup ? 'Save' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Email Preview Modal -->
    <Teleport to="body">
      <div
        v-if="showEmailPreview"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
        @click.self="closeEmailPreview"
      >
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-3xl max-h-[90vh] flex flex-col">
          <!-- Header -->
          <div class="flex items-center justify-between px-6 py-4 border-b dark:border-gray-700">
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Linked Email</h3>
            <button
              @click="closeEmailPreview"
              class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded"
            >
              <svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <!-- Loading -->
          <div v-if="loadingEmail" class="flex-1 flex items-center justify-center py-12">
            <svg class="w-8 h-8 text-blue-600 animate-spin" fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
              <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
            </svg>
          </div>

          <!-- Email Content -->
          <div v-else-if="previewEmail" class="flex-1 overflow-y-auto">
            <!-- Email metadata -->
            <div class="px-6 py-4 border-b dark:border-gray-700 bg-gray-50 dark:bg-gray-900/50">
              <h4 class="text-lg font-semibold text-gray-900 dark:text-white mb-3">
                {{ previewEmail.subject || '(No subject)' }}
              </h4>
              <div class="space-y-1 text-sm">
                <div class="flex">
                  <span class="w-16 text-gray-500 dark:text-gray-400">From:</span>
                  <span class="text-gray-900 dark:text-white">
                    {{ previewEmail.from_name || previewEmail.from_address }}
                    <span v-if="previewEmail.from_name" class="text-gray-500 dark:text-gray-400">
                      &lt;{{ previewEmail.from_address }}&gt;
                    </span>
                  </span>
                </div>
                <div class="flex">
                  <span class="w-16 text-gray-500 dark:text-gray-400">To:</span>
                  <span class="text-gray-900 dark:text-white">{{ previewEmail.to_addresses?.join(', ') }}</span>
                </div>
                <div class="flex">
                  <span class="w-16 text-gray-500 dark:text-gray-400">Date:</span>
                  <span class="text-gray-900 dark:text-white">{{ formatEmailDate(previewEmail.date) }}</span>
                </div>
              </div>
            </div>

            <!-- Email body -->
            <div class="px-6 py-4">
              <div
                v-if="previewEmail.html_body"
                class="prose dark:prose-invert max-w-none"
                v-html="sanitizeHtml(previewEmail.html_body)"
              ></div>
              <pre
                v-else-if="previewEmail.text_body"
                class="whitespace-pre-wrap text-sm text-gray-700 dark:text-gray-300"
              >{{ previewEmail.text_body }}</pre>
              <p v-else class="text-gray-500 dark:text-gray-400 italic">No content</p>
            </div>
          </div>

          <!-- Error state -->
          <div v-else class="flex-1 flex items-center justify-center py-12">
            <p class="text-gray-500 dark:text-gray-400">Failed to load email</p>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Delete Task Confirmation Modal -->
    <Teleport to="body">
      <div
        v-if="showDeleteConfirm"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
        @click.self="cancelDeleteTask"
      >
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-md p-6">
          <div class="flex items-center gap-3 mb-4">
            <div class="flex-shrink-0 w-10 h-10 flex items-center justify-center bg-red-100 dark:bg-red-900/30 rounded-full">
              <svg class="w-6 h-6 text-red-600 dark:text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
              </svg>
            </div>
            <div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Delete Task</h3>
              <p class="text-sm text-gray-500 dark:text-gray-400">This action cannot be undone.</p>
            </div>
          </div>
          
          <p class="text-gray-700 dark:text-gray-300 mb-6">
            Are you sure you want to delete "<span class="font-medium">{{ taskToDelete?.title }}</span>"?
          </p>
          
          <div class="flex justify-end gap-3">
            <button
              @click="cancelDeleteTask"
              class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              Cancel
            </button>
            <button
              @click="confirmDeleteTask"
              class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700"
            >
              Delete
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
