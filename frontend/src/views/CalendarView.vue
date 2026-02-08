<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '@/api'

interface CalendarEvent {
  id: string
  title: string
  description: string
  startDate: string
  endDate: string
  allDay: boolean
  color: string
  recurrence: RecurrenceRule | null
  reminders: Reminder[]
  createdAt: string
  updatedAt: string
}

interface RecurrenceRule {
  type: 'daily' | 'weekly' | 'monthly' | 'yearly'
  interval: number
  daysOfWeek?: number[]
  dayOfMonth?: number
  endDate?: string
  occurrences?: number
}

interface Reminder {
  id: string
  minutes: number  // minutes before event
}

const currentDate = ref(new Date())
const viewMode = ref<'month' | 'week' | 'day'>('month')
const events = ref<CalendarEvent[]>([])
const loading = ref(false)
const showEventModal = ref(false)
const editingEvent = ref<CalendarEvent | null>(null)

// Event form
const eventForm = ref({
  title: '',
  description: '',
  startDate: '',
  startTime: '',
  endDate: '',
  endTime: '',
  allDay: false,
  color: '#3b82f6'
})

const eventColors = [
  '#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6',
  '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#6366f1'
]

// Calendar grid computation
const currentYear = computed(() => currentDate.value.getFullYear())
const currentMonth = computed(() => currentDate.value.getMonth())
const currentMonthName = computed(() => {
  return currentDate.value.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })
})

const daysOfWeek = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

const calendarDays = computed(() => {
  const year = currentYear.value
  const month = currentMonth.value
  
  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)
  const startingDay = firstDay.getDay()
  const totalDays = lastDay.getDate()
  
  const days: { date: Date; isCurrentMonth: boolean; isToday: boolean }[] = []
  
  // Previous month days
  const prevMonthLastDay = new Date(year, month, 0).getDate()
  for (let i = startingDay - 1; i >= 0; i--) {
    days.push({
      date: new Date(year, month - 1, prevMonthLastDay - i),
      isCurrentMonth: false,
      isToday: false
    })
  }
  
  // Current month days
  const today = new Date()
  for (let day = 1; day <= totalDays; day++) {
    const date = new Date(year, month, day)
    days.push({
      date,
      isCurrentMonth: true,
      isToday: date.toDateString() === today.toDateString()
    })
  }
  
  // Next month days to fill grid (6 rows * 7 days = 42)
  const remainingDays = 42 - days.length
  for (let day = 1; day <= remainingDays; day++) {
    days.push({
      date: new Date(year, month + 1, day),
      isCurrentMonth: false,
      isToday: false
    })
  }
  
  return days
})

// Get events for a specific day
function getEventsForDay(date: Date): CalendarEvent[] {
  const dateStr = date.toISOString().split('T')[0]
  return events.value.filter(event => {
    const eventStart = event.startDate.split('T')[0]
    const eventEnd = event.endDate.split('T')[0]
    return dateStr >= eventStart && dateStr <= eventEnd
  })
}

// Navigation
function previousMonth() {
  currentDate.value = new Date(currentYear.value, currentMonth.value - 1, 1)
}

function nextMonth() {
  currentDate.value = new Date(currentYear.value, currentMonth.value + 1, 1)
}

function goToToday() {
  currentDate.value = new Date()
}

// Event management
function openNewEventModal(date?: Date) {
  editingEvent.value = null
  const targetDate = date || new Date()
  const dateStr = targetDate.toISOString().split('T')[0]
  
  eventForm.value = {
    title: '',
    description: '',
    startDate: dateStr,
    startTime: '09:00',
    endDate: dateStr,
    endTime: '10:00',
    allDay: false,
    color: '#3b82f6'
  }
  showEventModal.value = true
}

function openEditEventModal(event: CalendarEvent) {
  editingEvent.value = event
  const startDate = new Date(event.startDate)
  const endDate = new Date(event.endDate)
  
  eventForm.value = {
    title: event.title,
    description: event.description,
    startDate: startDate.toISOString().split('T')[0],
    startTime: event.allDay ? '09:00' : startDate.toTimeString().slice(0, 5),
    endDate: endDate.toISOString().split('T')[0],
    endTime: event.allDay ? '10:00' : endDate.toTimeString().slice(0, 5),
    allDay: event.allDay,
    color: event.color
  }
  showEventModal.value = true
}

async function saveEvent() {
  const startDateTime = eventForm.value.allDay
    ? `${eventForm.value.startDate}T00:00:00`
    : `${eventForm.value.startDate}T${eventForm.value.startTime}:00`
  
  const endDateTime = eventForm.value.allDay
    ? `${eventForm.value.endDate}T23:59:59`
    : `${eventForm.value.endDate}T${eventForm.value.endTime}:00`
  
  const data = {
    title: eventForm.value.title,
    description: eventForm.value.description,
    startDate: startDateTime,
    endDate: endDateTime,
    allDay: eventForm.value.allDay,
    color: eventForm.value.color
  }
  
  try {
    if (editingEvent.value) {
      const response = await api.put(`/calendar/events/${editingEvent.value.id}`, data)
      const index = events.value.findIndex(e => e.id === editingEvent.value!.id)
      if (index !== -1) {
        events.value[index] = response.data
      }
    } else {
      const response = await api.post('/calendar/events', data)
      events.value.push(response.data)
    }
    showEventModal.value = false
  } catch (err) {
    console.error('Failed to save event:', err)
  }
}

async function deleteEvent(event: CalendarEvent) {
  if (!confirm('Are you sure you want to delete this event?')) return
  
  try {
    await api.delete(`/calendar/events/${event.id}`)
    events.value = events.value.filter(e => e.id !== event.id)
    showEventModal.value = false
  } catch (err) {
    console.error('Failed to delete event:', err)
  }
}

async function fetchEvents() {
  loading.value = true
  try {
    // Fetch events for the visible month range
    const startOfMonth = new Date(currentYear.value, currentMonth.value, 1)
    const endOfMonth = new Date(currentYear.value, currentMonth.value + 1, 0)
    
    // Add padding for previous/next month days shown
    startOfMonth.setDate(startOfMonth.getDate() - 7)
    endOfMonth.setDate(endOfMonth.getDate() + 7)
    
    const response = await api.get('/calendar/events', {
      params: {
        start: startOfMonth.toISOString(),
        end: endOfMonth.toISOString()
      }
    })
    events.value = response.data || []
  } catch (err) {
    console.error('Failed to fetch events:', err)
    events.value = []
  } finally {
    loading.value = false
  }
}

function formatEventTime(event: CalendarEvent): string {
  if (event.allDay) return 'All day'
  const start = new Date(event.startDate)
  return start.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' })
}

onMounted(() => {
  fetchEvents()
})
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50 dark:bg-gray-900">
    <!-- Header -->
    <div class="bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 px-6 py-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-4">
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Calendar</h1>
          <div class="flex items-center gap-2">
            <button
              @click="previousMonth"
              class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
              </svg>
            </button>
            <button
              @click="goToToday"
              class="px-3 py-1 text-sm font-medium text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              Today
            </button>
            <button
              @click="nextMonth"
              class="p-2 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
            >
              <svg class="w-5 h-5 text-gray-600 dark:text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
              </svg>
            </button>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white ml-2">
              {{ currentMonthName }}
            </h2>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <!-- View mode toggle -->
          <div class="flex items-center bg-gray-100 dark:bg-gray-700 rounded-lg p-1">
            <button
              v-for="mode in ['month', 'week', 'day'] as const"
              :key="mode"
              @click="viewMode = mode"
              :class="[
                'px-3 py-1 text-sm font-medium rounded-md capitalize',
                viewMode === mode
                  ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm'
                  : 'text-gray-600 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white'
              ]"
            >
              {{ mode }}
            </button>
          </div>
          <button
            @click="openNewEventModal()"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
            </svg>
            New Event
          </button>
        </div>
      </div>
    </div>

    <!-- Calendar Grid (Month View) -->
    <div class="flex-1 p-6 overflow-hidden">
      <div v-if="viewMode === 'month'" class="h-full bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 overflow-hidden">
        <!-- Days of week header -->
        <div class="grid grid-cols-7 border-b border-gray-200 dark:border-gray-700">
          <div
            v-for="day in daysOfWeek"
            :key="day"
            class="py-3 text-center text-sm font-semibold text-gray-600 dark:text-gray-400"
          >
            {{ day }}
          </div>
        </div>

        <!-- Calendar days grid -->
        <div class="grid grid-cols-7 grid-rows-6 h-[calc(100%-48px)]">
          <div
            v-for="(day, index) in calendarDays"
            :key="index"
            @click="openNewEventModal(day.date)"
            :class="[
              'border-b border-r border-gray-100 dark:border-gray-700 p-1 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors overflow-hidden',
              !day.isCurrentMonth && 'bg-gray-50 dark:bg-gray-900/50'
            ]"
          >
            <!-- Date number -->
            <div class="flex items-center justify-between mb-1">
              <span
                :class="[
                  'w-7 h-7 flex items-center justify-center text-sm rounded-full',
                  day.isToday
                    ? 'bg-blue-600 text-white font-semibold'
                    : day.isCurrentMonth
                      ? 'text-gray-900 dark:text-white'
                      : 'text-gray-400 dark:text-gray-500'
                ]"
              >
                {{ day.date.getDate() }}
              </span>
            </div>

            <!-- Events for this day -->
            <div class="space-y-0.5 overflow-y-auto max-h-20">
              <div
                v-for="event in getEventsForDay(day.date).slice(0, 3)"
                :key="event.id"
                @click.stop="openEditEventModal(event)"
                class="px-1.5 py-0.5 text-xs rounded truncate cursor-pointer hover:opacity-80"
                :style="{ backgroundColor: event.color + '20', color: event.color }"
              >
                <span class="font-medium">{{ formatEventTime(event) }}</span>
                {{ event.title }}
              </div>
              <div
                v-if="getEventsForDay(day.date).length > 3"
                class="text-xs text-gray-500 dark:text-gray-400 px-1.5"
              >
                +{{ getEventsForDay(day.date).length - 3 }} more
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Week/Day view placeholder -->
      <div v-else class="h-full bg-white dark:bg-gray-800 rounded-xl shadow-sm border border-gray-200 dark:border-gray-700 flex items-center justify-center">
        <div class="text-center text-gray-500 dark:text-gray-400">
          <svg class="w-16 h-16 mx-auto mb-4 opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <p class="text-lg font-medium">{{ viewMode.charAt(0).toUpperCase() + viewMode.slice(1) }} view</p>
          <p class="text-sm">Coming soon</p>
        </div>
      </div>
    </div>

    <!-- Event Modal -->
    <Teleport to="body">
      <div
        v-if="showEventModal"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
        @click.self="showEventModal = false"
      >
        <div class="bg-white dark:bg-gray-800 rounded-xl shadow-xl w-full max-w-lg">
          <div class="p-6">
            <div class="flex items-center justify-between mb-4">
              <h2 class="text-xl font-bold text-gray-900 dark:text-white">
                {{ editingEvent ? 'Edit Event' : 'New Event' }}
              </h2>
              <button
                v-if="editingEvent"
                @click="deleteEvent(editingEvent)"
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
                v-model="eventForm.title"
                type="text"
                class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                placeholder="Event title"
              />
            </div>

            <!-- All Day Toggle -->
            <div class="mb-4">
              <label class="flex items-center gap-2 cursor-pointer">
                <input
                  v-model="eventForm.allDay"
                  type="checkbox"
                  class="w-4 h-4 rounded"
                />
                <span class="text-sm font-medium text-gray-700 dark:text-gray-300">All day event</span>
              </label>
            </div>

            <!-- Date & Time -->
            <div class="grid grid-cols-2 gap-4 mb-4">
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Start</label>
                <input
                  v-model="eventForm.startDate"
                  type="date"
                  class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white mb-2"
                />
                <input
                  v-if="!eventForm.allDay"
                  v-model="eventForm.startTime"
                  type="time"
                  class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">End</label>
                <input
                  v-model="eventForm.endDate"
                  type="date"
                  class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white mb-2"
                />
                <input
                  v-if="!eventForm.allDay"
                  v-model="eventForm.endTime"
                  type="time"
                  class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white"
                />
              </div>
            </div>

            <!-- Description -->
            <div class="mb-4">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Description</label>
              <textarea
                v-model="eventForm.description"
                rows="3"
                class="w-full px-4 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 dark:text-white resize-none"
                placeholder="Event description"
              ></textarea>
            </div>

            <!-- Color -->
            <div class="mb-4">
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Color</label>
              <div class="flex items-center gap-2 flex-wrap">
                <button
                  v-for="color in eventColors"
                  :key="color"
                  @click="eventForm.color = color"
                  :class="[
                    'w-8 h-8 rounded-full border-2 transition-transform',
                    eventForm.color === color ? 'border-gray-800 dark:border-white scale-110' : 'border-transparent hover:scale-105'
                  ]"
                  :style="{ backgroundColor: color }"
                ></button>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex justify-end gap-2 pt-4 border-t dark:border-gray-700">
              <button
                @click="showEventModal = false"
                class="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg"
              >
                Cancel
              </button>
              <button
                @click="saveEvent"
                :disabled="!eventForm.title"
                class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {{ editingEvent ? 'Save' : 'Create' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
