import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export interface ChecklistItem {
  id: string
  title: string
  completed: boolean
  order: number
}

export interface Task {
  id: string
  title: string
  description: string
  status: 'todo' | 'in-progress' | 'done'
  priority: 'low' | 'medium' | 'high'
  dueDate: string | null
  groupId: string | null
  groupName?: string
  recurrence: RecurrenceRule | null
  checklist: ChecklistItem[]
  tags: string[]
  linkedEmailId: string | null
  linkedEmailSubject: string | null
  createdAt: string
  updatedAt: string
  completedAt: string | null
  order: number
}

export interface RecurrenceRule {
  type: 'daily' | 'weekly' | 'monthly' | 'yearly' | 'custom'
  interval: number  // Every N days/weeks/months/years
  daysOfWeek?: number[]  // 0-6 for weekly (0=Sunday, 6=Saturday)
  dayOfMonth?: number  // 1-31 for monthly
  endDate?: string
  occurrences?: number  // Max number of occurrences
  occurrencesCompleted?: number  // How many have been completed
}

export interface TaskGroup {
  id: string
  name: string
  color: string
  recurrence: RecurrenceRule | null
  createdAt: string
}

export interface TaskColumn {
  id: 'todo' | 'in-progress' | 'done'
  title: string
  tasks: Task[]
}

export const useTasksStore = defineStore('tasks', () => {
  const tasks = ref<Task[]>([])
  const groups = ref<TaskGroup[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const columns = computed<TaskColumn[]>(() => [
    {
      id: 'todo',
      title: 'To Do',
      tasks: tasks.value
        .filter(t => t.status === 'todo')
        .sort((a, b) => a.order - b.order)
    },
    {
      id: 'in-progress',
      title: 'In Progress',
      tasks: tasks.value
        .filter(t => t.status === 'in-progress')
        .sort((a, b) => a.order - b.order)
    },
    {
      id: 'done',
      title: 'Done',
      tasks: tasks.value
        .filter(t => t.status === 'done')
        .sort((a, b) => a.order - b.order)
    }
  ])

  const tasksByGroup = computed(() => {
    const grouped: Record<string, Task[]> = { ungrouped: [] }
    
    groups.value.forEach(g => {
      grouped[g.id] = []
    })
    
    tasks.value.forEach(task => {
      if (task.groupId && grouped[task.groupId]) {
        grouped[task.groupId].push(task)
      } else {
        grouped.ungrouped.push(task)
      }
    })
    
    return grouped
  })

  async function fetchTasks() {
    loading.value = true
    error.value = null

    try {
      const response = await api.get('/tasks')
      tasks.value = response.data.tasks || []
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to fetch tasks'
    } finally {
      loading.value = false
    }
  }

  async function fetchGroups() {
    try {
      const response = await api.get('/tasks/groups')
      groups.value = response.data.groups || []
    } catch (err: any) {
      // Silently fail
    }
  }

  async function createTask(task: Partial<Task>) {
    loading.value = true
    error.value = null

    try {
      const response = await api.post('/tasks', task)
      tasks.value.push(response.data)
      return response.data
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to create task'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateTask(id: string, updates: Partial<Task>) {
    loading.value = true
    error.value = null

    try {
      const response = await api.put(`/tasks/${id}`, updates)
      const index = tasks.value.findIndex(t => t.id === id)
      if (index !== -1) {
        tasks.value[index] = response.data
      }
      return response.data
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to update task'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteTask(id: string) {
    loading.value = true
    error.value = null

    try {
      await api.delete(`/tasks/${id}`)
      tasks.value = tasks.value.filter(t => t.id !== id)
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to delete task'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function moveTask(taskId: string, newStatus: Task['status'], newOrder: number) {
    const task = tasks.value.find(t => t.id === taskId)
    if (!task) return

    const oldStatus = task.status
    const oldOrder = task.order

    // Optimistic update
    task.status = newStatus
    task.order = newOrder

    // Handle recurrence when completing a task
    if (newStatus === 'done' && oldStatus !== 'done' && task.recurrence) {
      task.completedAt = new Date().toISOString()
      // Server will create new instance
    }

    try {
      await api.put(`/api/tasks/${taskId}/move`, {
        status: newStatus,
        order: newOrder
      })
    } catch (err: any) {
      // Rollback on failure
      task.status = oldStatus
      task.order = oldOrder
      error.value = err.response?.data?.error || 'Failed to move task'
    }
  }

  async function reorderTasks(columnId: Task['status'], taskIds: string[]) {
    // Optimistic update
    taskIds.forEach((id, index) => {
      const task = tasks.value.find(t => t.id === id)
      if (task) {
        task.order = index
        task.status = columnId
      }
    })

    try {
      await api.put('/tasks/reorder', {
        status: columnId,
        taskIds
      })
    } catch (err: any) {
      // Refetch on failure
      await fetchTasks()
      error.value = err.response?.data?.error || 'Failed to reorder tasks'
    }
  }

  async function createGroup(group: Partial<TaskGroup>) {
    try {
      const response = await api.post('/tasks/groups', group)
      groups.value.push(response.data)
      return response.data
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to create group'
      throw err
    }
  }

  async function updateGroup(id: string, updates: Partial<TaskGroup>) {
    try {
      const response = await api.put(`/api/tasks/groups/${id}`, updates)
      const index = groups.value.findIndex(g => g.id === id)
      if (index !== -1) {
        groups.value[index] = response.data
      }
      return response.data
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to update group'
      throw err
    }
  }

  async function deleteGroup(id: string) {
    try {
      await api.delete(`/api/tasks/groups/${id}`)
      groups.value = groups.value.filter(g => g.id !== id)
      // Ungroup tasks that were in this group
      tasks.value.forEach(task => {
        if (task.groupId === id) {
          task.groupId = null
        }
      })
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to delete group'
      throw err
    }
  }

  function getNextRecurrenceDate(task: Task): Date | null {
    if (!task.recurrence || !task.dueDate) return null

    const currentDue = new Date(task.dueDate)
    const rule = task.recurrence

    switch (rule.type) {
      case 'daily':
        currentDue.setDate(currentDue.getDate() + rule.interval)
        break
      case 'weekly':
        currentDue.setDate(currentDue.getDate() + (rule.interval * 7))
        break
      case 'monthly':
        currentDue.setMonth(currentDue.getMonth() + rule.interval)
        break
      case 'yearly':
        currentDue.setFullYear(currentDue.getFullYear() + rule.interval)
        break
    }

    if (rule.endDate && currentDue > new Date(rule.endDate)) {
      return null
    }

    return currentDue
  }

  return {
    tasks,
    groups,
    columns,
    tasksByGroup,
    loading,
    error,
    fetchTasks,
    fetchGroups,
    createTask,
    updateTask,
    deleteTask,
    moveTask,
    reorderTasks,
    createGroup,
    updateGroup,
    deleteGroup,
    getNextRecurrenceDate
  }
})
