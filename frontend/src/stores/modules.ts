import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/api'

export interface ModuleConfig {
  id: string
  name: string
  description: string
  icon: string
  enabled: boolean
  route?: string
}

export const useModulesStore = defineStore('modules', () => {
  const modules = ref<ModuleConfig[]>([
    {
      id: 'documents',
      name: 'Documents',
      description: 'Rich text editor with real-time collaboration',
      icon: 'document-text',
      enabled: false,
      route: '/documents'
    },
    {
      id: 'pdf',
      name: 'PDF Suite',
      description: 'PDF viewer with annotations and signatures',
      icon: 'document',
      enabled: false
    },
    {
      id: 'tasks',
      name: 'Tasks',
      description: 'Kanban board with recurring tasks',
      icon: 'clipboard-list',
      enabled: false,
      route: '/tasks'
    },
    {
      id: 'calendar',
      name: 'Calendar',
      description: 'CalDAV calendar with events and reminders',
      icon: 'calendar',
      enabled: false,
      route: '/calendar'
    },
    {
      id: 'contacts',
      name: 'Contacts',
      description: 'CardDAV contact management',
      icon: 'user-group',
      enabled: false,
      route: '/contacts'
    },
    {
      id: 'email',
      name: 'Email',
      description: 'IMAP/SMTP email client',
      icon: 'envelope',
      enabled: false,
      route: '/email'
    }
  ])

  const loading = ref(false)
  const loaded = ref(false)
  const error = ref<string | null>(null)

  const enabledModules = computed(() => 
    modules.value.filter(m => m.enabled)
  )

  const isModuleEnabled = (moduleId: string) => {
    const module = modules.value.find(m => m.id === moduleId)
    return module?.enabled ?? false
  }

  async function fetchModuleSettings() {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.get('/admin/modules')
      const serverModules = response.data.modules as Record<string, boolean>
      
      // Update local state with server settings
      modules.value.forEach(module => {
        if (serverModules[module.id] !== undefined) {
          module.enabled = serverModules[module.id]
        }
      })
      loaded.value = true
    } catch (err: any) {
      // If not admin or endpoint doesn't exist, try user endpoint
      try {
        const response = await api.get('/modules')
        const serverModules = response.data.modules as Record<string, boolean>
        
        modules.value.forEach(module => {
          if (serverModules[module.id] !== undefined) {
            module.enabled = serverModules[module.id]
          }
        })
        loaded.value = true
      } catch {
        // Silently fail - modules stay disabled by default
      }
    } finally {
      loading.value = false
    }
  }

  async function toggleModule(moduleId: string, enabled: boolean) {
    loading.value = true
    error.value = null

    try {
      await api.put(`/admin/modules/${moduleId}`, { enabled })
      
      const module = modules.value.find(m => m.id === moduleId)
      if (module) {
        module.enabled = enabled
      }
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to update module'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function saveAllModules() {
    loading.value = true
    error.value = null

    try {
      const settings: Record<string, boolean> = {}
      modules.value.forEach(m => {
        settings[m.id] = m.enabled
      })

      await api.put('/admin/modules', { modules: settings })
    } catch (err: any) {
      error.value = err.response?.data?.error || 'Failed to save module settings'
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    modules,
    loading,
    loaded,
    error,
    enabledModules,
    isModuleEnabled,
    fetchModuleSettings,
    toggleModule,
    saveAllModules
  }
})
