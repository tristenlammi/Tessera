import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useModulesStore } from '@/stores/modules'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/s/:token',
      name: 'public-share',
      component: () => import('@/views/PublicShareView.vue'),
      meta: { public: true }
    },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { guest: true }
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('@/views/RegisterView.vue'),
      meta: { guest: true }
    },
    {
      path: '/reset-password',
      name: 'reset-password',
      component: () => import('@/views/ResetPasswordView.vue'),
      meta: { public: true }
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'files',
          component: () => import('@/views/FilesView.vue')
        },
        {
          path: 'folder/:id',
          name: 'folder',
          component: () => import('@/views/FilesView.vue')
        },
        {
          path: 'shared',
          name: 'shared',
          component: () => import('@/views/SharedView.vue')
        },
        {
          path: 'recent',
          name: 'recent',
          component: () => import('@/views/RecentView.vue')
        },
        {
          path: 'starred',
          name: 'starred',
          component: () => import('@/views/StarredView.vue')
        },
        {
          path: 'trash',
          name: 'trash',
          component: () => import('@/views/TrashView.vue')
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/SettingsView.vue')
        },
        {
          path: 'admin',
          name: 'admin',
          component: () => import('@/views/AdminDashboard.vue'),
          meta: { requiresAdmin: true }
        },
        // Optional modules
        {
          path: 'documents',
          name: 'documents',
          component: () => import('@/views/DocumentsView.vue'),
          meta: { module: 'documents' }
        },
        {
          path: 'documents/:id',
          name: 'document-edit',
          component: () => import('@/views/DocumentsView.vue'),
          meta: { module: 'documents' }
        },
        {
          path: 'tasks',
          name: 'tasks',
          component: () => import('@/views/TasksView.vue'),
          meta: { module: 'tasks' }
        },
        {
          path: 'email',
          name: 'email',
          component: () => import('@/views/EmailView.vue'),
          meta: { module: 'email' }
        },
        {
          path: 'calendar',
          name: 'calendar',
          component: () => import('@/views/CalendarView.vue'),
          meta: { module: 'calendar' }
        },
        {
          path: 'contacts',
          name: 'contacts',
          component: () => import('@/views/ContactsView.vue'),
          meta: { module: 'contacts' }
        }
      ]
    }
  ]
})

// Navigation guard
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  
  // Allow public routes without authentication
  if (to.meta.public) {
    next()
    return
  }

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } })
    return
  }

  if (to.meta.guest && authStore.isAuthenticated) {
    next({ name: 'files' })
    return
  }

  if (to.meta.requiresAdmin && authStore.user?.role !== 'admin') {
    next({ name: 'files' })
    return
  }

  // Module access guard: redirect to files if module is disabled
  if (to.meta.module) {
    const modulesStore = useModulesStore()
    // Fetch module settings if not yet loaded
    if (!modulesStore.loaded) {
      try {
        await modulesStore.fetchModuleSettings()
      } catch {
        // If fetch fails, deny access to module routes
      }
    }
    if (!modulesStore.isModuleEnabled(to.meta.module as string)) {
      next({ name: 'files' })
      return
    }
  }

  next()
})

export default router
