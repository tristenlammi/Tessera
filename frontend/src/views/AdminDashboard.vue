<template>
  <div class="min-h-screen bg-stone-50 dark:bg-neutral-900">
    <!-- Admin Header -->
    <header class="bg-white dark:bg-neutral-800 shadow-sm border-b border-stone-200 dark:border-neutral-700 pt-[env(safe-area-inset-top)]">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="p-2 bg-stone-100 dark:bg-neutral-800 rounded-lg">
              <ShieldCheckIcon class="w-6 h-6 text-stone-800 dark:text-stone-200 dark:text-stone-400" />
            </div>
            <div>
              <h1 class="text-xl font-bold text-stone-900 dark:text-stone-100">Admin Dashboard</h1>
              <p class="text-sm text-stone-500 dark:text-stone-400">System management and monitoring</p>
            </div>
          </div>
          <router-link
            to="/files"
            class="min-h-[44px] min-w-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center md:justify-start text-sm text-stone-800 dark:text-stone-200 dark:text-stone-400 hover:underline gap-1"
          >
            <ArrowLeftIcon class="w-4 h-4" />
            Back to Files
          </router-link>
        </div>
      </div>
    </header>

    <!-- Navigation Tabs -->
    <nav class="bg-white dark:bg-neutral-800 border-b border-stone-200 dark:border-neutral-700">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 overflow-x-auto scrollbar-hide">
        <div class="flex space-x-4 md:space-x-8 min-w-max pb-px -mb-px">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            @click="activeTab = tab.id"
            :class="[
              'py-4 px-2 md:px-1 border-b-2 font-medium text-sm transition-colors flex-shrink-0 min-h-[44px] flex items-center gap-2',
              activeTab === tab.id
                ? 'border-neutral-700 dark:border-neutral-300 text-stone-800 dark:text-stone-200 dark:text-stone-400'
                : 'border-transparent text-stone-500 hover:text-stone-700 dark:text-stone-400 dark:hover:text-stone-300'
            ]"
          >
            <component :is="tab.icon" class="w-5 h-5 flex-shrink-0" />
            {{ tab.name }}
          </button>
        </div>
      </div>
    </nav>

    <!-- Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Overview Tab -->
      <div v-if="activeTab === 'overview'" class="space-y-6">
        <!-- Stats Cards -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <div
            v-for="stat in statsCards"
            :key="stat.label"
            class="bg-white dark:bg-neutral-800 rounded-xl shadow-sm p-6 border border-stone-200 dark:border-neutral-700"
          >
            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm text-stone-500 dark:text-stone-400">{{ stat.label }}</p>
                <p class="text-2xl font-bold text-stone-900 dark:text-stone-100 mt-1">{{ stat.value }}</p>
              </div>
              <div :class="['p-3 rounded-lg', stat.bgColor]">
                <component :is="stat.icon" :class="['w-6 h-6', stat.iconColor]" />
              </div>
            </div>
            <div v-if="stat.change" class="mt-3 flex items-center text-sm">
              <span :class="stat.change > 0 ? 'text-green-600' : 'text-red-600'">
                {{ stat.change > 0 ? '+' : '' }}{{ stat.change }}%
              </span>
              <span class="text-stone-500 dark:text-stone-400 ml-2">from last week</span>
            </div>
          </div>
        </div>

        <!-- Charts Row -->
        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <!-- Storage Distribution -->
          <div class="bg-white dark:bg-neutral-800 rounded-xl shadow-sm p-6 border border-stone-200 dark:border-neutral-700">
            <h3 class="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-4">Storage Distribution</h3>
            <div class="space-y-4">
              <div>
                <div class="flex justify-between text-sm mb-1">
                  <span class="text-stone-600 dark:text-stone-400">Used Storage</span>
                  <span class="text-stone-900 dark:text-stone-100 font-medium">{{ formatBytes(systemStats?.usedStorage || 0) }}</span>
                </div>
                <div class="w-full bg-stone-200 dark:bg-neutral-700 rounded-full h-3">
                  <div
                    class="bg-neutral-800 dark:bg-neutral-200 h-3 rounded-full transition-all"
                    :style="{ width: storagePercentage + '%' }"
                  ></div>
                </div>
                <p class="text-xs text-stone-500 dark:text-stone-400 mt-1">
                  {{ storagePercentage.toFixed(1) }}% of {{ formatBytes(systemStats?.totalStorage || 0) }} used
                </p>
              </div>
            </div>
          </div>

          <!-- Activity Summary -->
          <div class="bg-white dark:bg-neutral-800 rounded-xl shadow-sm p-6 border border-stone-200 dark:border-neutral-700">
            <h3 class="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-4">Today's Activity</h3>
            <div class="grid grid-cols-2 gap-4">
              <div class="p-4 bg-green-50 dark:bg-green-900/20 rounded-lg">
                <ArrowUpTrayIcon class="w-8 h-8 text-green-600 dark:text-green-400 mb-2" />
                <p class="text-2xl font-bold text-stone-900 dark:text-stone-100">{{ systemStats?.uploadsToday || 0 }}</p>
                <p class="text-sm text-stone-500 dark:text-stone-400">Uploads</p>
              </div>
              <div class="p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                <ArrowDownTrayIcon class="w-8 h-8 text-purple-600 dark:text-purple-400 mb-2" />
                <p class="text-2xl font-bold text-stone-900 dark:text-stone-100">{{ systemStats?.downloadsToday || 0 }}</p>
                <p class="text-sm text-stone-500 dark:text-stone-400">Downloads</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Quick Actions -->
        <div class="bg-white dark:bg-neutral-800 rounded-xl shadow-sm p-6 border border-stone-200 dark:border-neutral-700">
          <h3 class="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-4">Quick Actions</h3>
          <div class="flex flex-wrap gap-3">
            <button
              @click="handleClearCache"
              :disabled="loading"
              class="min-h-[44px] px-4 py-2 flex items-center bg-stone-100 dark:bg-neutral-700 text-stone-700 dark:text-stone-300 rounded-lg hover:bg-stone-200 dark:hover:bg-neutral-600 transition-colors disabled:opacity-50"
            >
              <TrashIcon class="w-4 h-4 inline-block mr-2" />
              Clear Cache
            </button>
            <button
              @click="handleRunCleanup"
              :disabled="loading"
              class="min-h-[44px] px-4 py-2 flex items-center bg-stone-100 dark:bg-neutral-700 text-stone-700 dark:text-stone-300 rounded-lg hover:bg-stone-200 dark:hover:bg-neutral-600 transition-colors disabled:opacity-50"
            >
              <ArrowPathIcon class="w-4 h-4 inline-block mr-2" />
              Run Cleanup
            </button>
            <button
              @click="toggleMaintenance"
              :disabled="loading"
              :class="[
                'min-h-[44px] px-4 py-2 flex items-center rounded-lg transition-colors disabled:opacity-50',
                systemSettings?.maintenanceMode
                  ? 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300 hover:bg-green-200'
                  : 'bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300 hover:bg-yellow-200'
              ]"
            >
              <WrenchIcon class="w-4 h-4 inline-block mr-2" />
              {{ systemSettings?.maintenanceMode ? 'Disable' : 'Enable' }} Maintenance Mode
            </button>
          </div>
        </div>
      </div>

      <!-- Users Tab -->
      <div v-else-if="activeTab === 'users'" class="space-y-6">
        <!-- Search and Filters -->
        <div class="flex flex-col sm:flex-row gap-4">
          <div class="relative flex-1">
            <MagnifyingGlassIcon class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-stone-400" />
            <input
              v-model="userSearch"
              @input="debouncedSearchUsers"
              type="text"
              placeholder="Search users by name or email..."
              class="w-full min-h-[44px] pl-10 pr-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100 focus:ring-2 focus:ring-stone-400 focus:border-transparent"
            />
          </div>
          <button
            @click="showCreateUserModal = true"
            class="min-h-[44px] px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 transition-colors flex items-center justify-center gap-2"
          >
            <PlusIcon class="w-5 h-5" />
            Add User
          </button>
        </div>

        <!-- Users Table -->
        <div class="bg-white dark:bg-neutral-800 rounded-xl shadow-sm border border-stone-200 dark:border-neutral-700 overflow-hidden">
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-stone-200 dark:divide-neutral-700">
              <thead class="bg-stone-50 dark:bg-neutral-900">
                <tr>
                  <th class="px-4 md:px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">User</th>
                  <th class="px-4 md:px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">Role</th>
                  <th class="px-4 md:px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">Storage</th>
                  <th class="px-4 md:px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">Status</th>
                  <th class="px-4 md:px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">Last Login</th>
                  <th class="px-4 md:px-6 py-3 text-right text-xs font-medium text-stone-500 dark:text-stone-400 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody class="bg-white dark:bg-neutral-800 divide-y divide-stone-200 dark:divide-neutral-700">
                <tr v-for="user in users" :key="user.id" class="hover:bg-stone-50 dark:hover:bg-neutral-700/50">
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap min-h-[44px]">
                    <div class="flex items-center">
                      <div class="w-10 h-10 rounded-full bg-stone-100 dark:bg-neutral-800 flex items-center justify-center">
                        <span class="text-stone-800 dark:text-stone-200 dark:text-stone-400 font-medium">{{ user.name.charAt(0).toUpperCase() }}</span>
                      </div>
                      <div class="ml-4">
                        <div class="text-sm font-medium text-stone-900 dark:text-stone-100">{{ user.name }}</div>
                        <div class="text-sm text-stone-500 dark:text-stone-400">{{ user.email }}</div>
                      </div>
                    </div>
                  </td>
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap min-h-[44px]">
                    <span :class="[
                      'px-2 py-1 text-xs font-medium rounded-full',
                      user.role === 'admin'
                        ? 'bg-purple-100 dark:bg-purple-900 text-purple-800 dark:text-purple-200'
                        : 'bg-stone-100 dark:bg-neutral-700 text-stone-800 dark:text-stone-200'
                    ]">
                      {{ user.role }}
                    </span>
                  </td>
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap min-h-[44px]">
                    <div class="text-sm text-stone-900 dark:text-stone-100">{{ formatBytes(user.storageUsed) }}</div>
                    <div class="text-xs text-stone-500 dark:text-stone-400">of {{ formatBytes(user.storageQuota) }}</div>
                    <div class="w-24 bg-stone-200 dark:bg-neutral-700 rounded-full h-1.5 mt-1">
                      <div
                        class="bg-neutral-800 dark:bg-neutral-200 h-1.5 rounded-full"
                        :style="{ width: Math.min((user.storageUsed / user.storageQuota) * 100, 100) + '%' }"
                      ></div>
                    </div>
                  </td>
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap min-h-[44px]">
                    <span :class="[
                      'px-2 py-1 text-xs font-medium rounded-full',
                      user.isActive
                        ? 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200'
                        : 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200'
                    ]">
                      {{ user.isActive ? 'Active' : 'Disabled' }}
                    </span>
                  </td>
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap text-sm text-stone-500 dark:text-stone-400">
                    {{ user.lastLoginAt ? formatDate(user.lastLoginAt) : 'Never' }}
                  </td>
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap text-right text-sm font-medium">
                    <div class="flex items-center justify-end gap-1">
                      <button
                        @click="editUser(user)"
                        class="min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center p-2 rounded text-stone-800 dark:text-stone-400 hover:text-stone-900 dark:hover:text-stone-300 hover:bg-stone-100 dark:hover:bg-neutral-700"
                        title="Edit"
                      >
                        <PencilIcon class="w-5 h-5" />
                      </button>
                      <button
                        @click="generateResetLink(user)"
                        class="min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center p-2 rounded text-amber-600 dark:text-amber-400 hover:text-amber-900 dark:hover:text-amber-300 hover:bg-amber-50 dark:hover:bg-amber-900/20"
                        title="Generate Password Reset Link"
                      >
                        <KeyIcon class="w-5 h-5" />
                      </button>
                      <button
                        @click="confirmDeleteUser(user)"
                        class="min-w-[44px] min-h-[44px] md:min-w-0 md:min-h-0 flex items-center justify-center p-2 rounded text-red-600 dark:text-red-400 hover:text-red-900 dark:hover:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/20"
                        title="Delete"
                      >
                        <TrashIcon class="w-5 h-5" />
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Pagination -->
          <div class="px-4 md:px-6 py-4 border-t border-stone-200 dark:border-neutral-700 flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-2">
            <p class="text-sm text-stone-500 dark:text-stone-400">
              Showing {{ (usersPage - 1) * 20 + 1 }} to {{ Math.min(usersPage * 20, usersTotal) }} of {{ usersTotal }} users
            </p>
            <div class="flex gap-2">
              <button
                @click="loadUsers(usersPage - 1)"
                :disabled="usersPage <= 1"
                class="min-h-[44px] px-3 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-stone-50 dark:hover:bg-neutral-700"
              >
                Previous
              </button>
              <button
                @click="loadUsers(usersPage + 1)"
                :disabled="usersPage >= totalPages"
                class="min-h-[44px] px-3 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-stone-50 dark:hover:bg-neutral-700"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Settings Tab -->
      <div v-else-if="activeTab === 'settings'" class="space-y-6">
        <div class="bg-white dark:bg-neutral-800 rounded-xl shadow-sm border border-stone-200 dark:border-neutral-700 p-6">
          <h3 class="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-6">General Settings</h3>
          
          <form @submit.prevent="saveSettings" class="space-y-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">Site Name</label>
                <input
                  v-model="settingsForm.siteName"
                  type="text"
                  class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100 focus:ring-2 focus:ring-stone-400"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">Site URL</label>
                <input
                  v-model="settingsForm.siteUrl"
                  type="url"
                  class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100 focus:ring-2 focus:ring-stone-400"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">Default Storage Quota (GB)</label>
                <input
                  v-model.number="settingsForm.defaultQuotaGB"
                  type="number"
                  min="1"
                  class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100 focus:ring-2 focus:ring-stone-400"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">Max Upload Size (MB)</label>
                <input
                  v-model.number="settingsForm.maxUploadSizeMB"
                  type="number"
                  min="1"
                  class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100 focus:ring-2 focus:ring-stone-400"
                />
              </div>
            </div>

            <div class="space-y-4">
              <label class="flex items-center gap-3 min-h-[44px] cursor-pointer">
                <input
                  v-model="settingsForm.allowRegistration"
                  type="checkbox"
                  class="w-5 h-5 text-stone-800 dark:text-stone-200 rounded border-stone-300 focus:ring-stone-400"
                />
                <span class="text-sm text-stone-700 dark:text-stone-300">Allow public registration</span>
              </label>
              <label class="flex items-center gap-3 min-h-[44px] cursor-pointer">
                <input
                  v-model="settingsForm.requireEmailVerification"
                  type="checkbox"
                  class="w-5 h-5 text-stone-800 dark:text-stone-200 rounded border-stone-300 focus:ring-stone-400"
                />
                <span class="text-sm text-stone-700 dark:text-stone-300">Require email verification</span>
              </label>
            </div>

            <div class="pt-4 border-t border-stone-200 dark:border-neutral-700">
              <button
                type="submit"
                :disabled="loading"
                class="min-h-[44px] px-6 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 transition-colors disabled:opacity-50"
              >
                Save Settings
              </button>
            </div>
          </form>
        </div>

        <!-- SMTP Settings -->
        <div class="bg-white dark:bg-neutral-800 rounded-xl shadow-sm border border-stone-200 dark:border-neutral-700 p-6">
          <h3 class="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-6">Email Settings (SMTP)</h3>
          
          <form @submit.prevent="saveSmtpSettings" class="space-y-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">SMTP Host</label>
                <input
                  v-model="smtpForm.smtpHost"
                  type="text"
                  placeholder="smtp.example.com"
                  class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100 focus:ring-2 focus:ring-stone-400"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">SMTP Port</label>
                <input
                  v-model.number="smtpForm.smtpPort"
                  type="number"
                  placeholder="587"
                  class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100 focus:ring-2 focus:ring-stone-400"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">SMTP Username</label>
                <input
                  v-model="smtpForm.smtpUser"
                  type="text"
                  class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100 focus:ring-2 focus:ring-stone-400"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-2">From Address</label>
                <input
                  v-model="smtpForm.smtpFrom"
                  type="email"
                  placeholder="noreply@example.com"
                  class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100 focus:ring-2 focus:ring-stone-400"
                />
              </div>
            </div>

            <div class="pt-4 border-t border-stone-200 dark:border-neutral-700 flex flex-wrap gap-3">
              <button
                type="submit"
                :disabled="loading"
                class="min-h-[44px] px-6 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 transition-colors disabled:opacity-50"
              >
                Save SMTP Settings
              </button>
              <button
                type="button"
                @click="testSmtp"
                :disabled="loading"
                class="min-h-[44px] px-6 py-2 border border-stone-300 dark:border-neutral-700 text-stone-700 dark:text-stone-300 rounded-lg hover:bg-stone-50 dark:hover:bg-neutral-700 transition-colors disabled:opacity-50"
              >
                Send Test Email
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- Modules Tab -->
      <div v-else-if="activeTab === 'modules'" class="space-y-6">
        <div class="bg-white dark:bg-neutral-800 rounded-xl shadow-sm border border-stone-200 dark:border-neutral-700">
          <div class="p-6 border-b border-stone-200 dark:border-neutral-700">
            <h2 class="text-lg font-semibold text-stone-900 dark:text-stone-100">Optional Modules</h2>
            <p class="text-sm text-stone-500 dark:text-stone-400 mt-1">
              Enable or disable optional features. Disabled modules will be hidden from the navigation sidebar.
            </p>
          </div>
          <div class="divide-y divide-stone-200 dark:divide-neutral-700">
            <div
              v-for="module in modules"
              :key="module.id"
              class="p-6 flex items-center justify-between min-h-[44px]"
            >
              <div class="flex items-center gap-4">
                <div :class="[
                  'p-3 rounded-lg',
                  module.enabled
                    ? 'bg-stone-100 dark:bg-neutral-700/30'
                    : 'bg-stone-100 dark:bg-neutral-700'
                ]">
                  <component
                    :is="moduleIcons[module.id] || PuzzlePieceIcon"
                    :class="[
                      'w-6 h-6',
                      module.enabled
                        ? 'text-stone-800 dark:text-stone-200'
                        : 'text-stone-400 dark:text-stone-500'
                    ]"
                  />
                </div>
                <div>
                  <h3 class="font-medium text-stone-900 dark:text-stone-100">{{ module.name }}</h3>
                  <p class="text-sm text-stone-500 dark:text-stone-400 mt-0.5">
                    {{ moduleDescriptions[module.id] || module.description }}
                  </p>
                </div>
              </div>
              <button
                type="button"
                role="switch"
                :aria-checked="module.enabled"
                :aria-label="`${module.enabled ? 'Disable' : 'Enable'} ${module.name}`"
                @click="toggleModule(module.id)"
                :class="[
                  'flex flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-stone-400 focus:ring-offset-2 dark:focus:ring-offset-neutral-800 items-center',
                  'h-6 w-11 min-h-[44px] min-w-[44px] p-0.5',
                  module.enabled ? 'bg-neutral-800 dark:bg-neutral-200 justify-end' : 'bg-stone-300 dark:bg-neutral-600 justify-start'
                ]"
              >
                <span
                  class="pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow ring-0 transition-colors duration-200"
                />
              </button>
            </div>
          </div>
        </div>

        <div class="bg-amber-50 dark:bg-amber-900/20 rounded-xl p-4 border border-amber-200 dark:border-amber-800">
          <p class="text-sm text-amber-800 dark:text-amber-200">
            <strong>Note:</strong> Some modules (Calendar, Contacts, Email) require additional server configuration.
            Enabling them here will show their navigation items, but full functionality requires backend setup.
          </p>
        </div>
      </div>

      <!-- Logs Tab -->
      <div v-else-if="activeTab === 'logs'" class="space-y-6">
        <!-- Filters -->
        <div class="flex flex-col sm:flex-row gap-4">
          <select
            v-model="logFilters.action"
            class="px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100"
          >
            <option value="">All Actions</option>
            <option value="login">Login</option>
            <option value="logout">Logout</option>
            <option value="upload">Upload</option>
            <option value="download">Download</option>
            <option value="delete">Delete</option>
            <option value="share">Share</option>
          </select>
          <input
            v-model="logFilters.userId"
            type="text"
            placeholder="Filter by user ID..."
            class="flex-1 px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100"
          />
          <button
            @click="loadLogs(1)"
            class="px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 transition-colors"
          >
            Filter
          </button>
        </div>

        <!-- Logs Table -->
        <div class="bg-white dark:bg-neutral-800 rounded-xl shadow-sm border border-stone-200 dark:border-neutral-700 overflow-hidden">
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-stone-200 dark:divide-neutral-700">
              <thead class="bg-stone-50 dark:bg-neutral-900">
                <tr>
                  <th class="px-4 md:px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase">Time</th>
                  <th class="px-4 md:px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase">User</th>
                  <th class="px-4 md:px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase">Action</th>
                  <th class="px-4 md:px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase">Resource</th>
                  <th class="px-4 md:px-6 py-3 text-left text-xs font-medium text-stone-500 dark:text-stone-400 uppercase">IP Address</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-stone-200 dark:divide-neutral-700">
                <tr v-for="log in activityLogs" :key="log.id" class="hover:bg-stone-50 dark:hover:bg-neutral-700/50">
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap text-sm text-stone-500 dark:text-stone-400">
                    {{ formatDateTime(log.createdAt) }}
                  </td>
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap text-sm text-stone-900 dark:text-stone-100">
                    {{ log.userEmail }}
                  </td>
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap min-h-[44px]">
                    <span :class="[
                      'px-2 py-1 text-xs font-medium rounded-full',
                      getActionColor(log.action)
                    ]">
                      {{ log.action }}
                    </span>
                  </td>
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap text-sm text-stone-500 dark:text-stone-400">
                    {{ log.resourceType }}: {{ log.resourceId?.slice(0, 8) }}...
                  </td>
                  <td class="px-4 md:px-6 py-3 md:py-4 whitespace-nowrap text-sm text-stone-500 dark:text-stone-400">
                    {{ log.ipAddress }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Pagination -->
          <div class="px-4 md:px-6 py-4 border-t border-stone-200 dark:border-neutral-700 flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-2">
            <p class="text-sm text-stone-500 dark:text-stone-400">
              Page {{ logsPage }} of {{ logsTotalPages }}
            </p>
            <div class="flex gap-2">
              <button
                @click="loadLogs(logsPage - 1)"
                :disabled="logsPage <= 1"
                class="min-h-[44px] px-3 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg text-sm disabled:opacity-50 hover:bg-stone-50 dark:hover:bg-neutral-700"
              >
                Previous
              </button>
              <button
                @click="loadLogs(logsPage + 1)"
                :disabled="logsPage >= logsTotalPages"
                class="min-h-[44px] px-3 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg text-sm disabled:opacity-50 hover:bg-stone-50 dark:hover:bg-neutral-700"
              >
                Next
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Edit User Modal -->
    <div v-if="editingUser" class="fixed inset-0 z-50 flex items-center justify-center p-4 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)] bg-black/50 overflow-y-auto">
      <div class="modal-panel bg-white dark:bg-neutral-800 shadow-xl flex-shrink-0 my-auto">
        <div class="p-4 md:p-6 overflow-y-auto flex-1">
          <h3 class="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-4">Edit User</h3>
          <form @submit.prevent="saveUser" class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">Name</label>
              <input
                v-model="editingUser.name"
                type="text"
                required
                class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">Role</label>
              <select
                v-model="editingUser.role"
                class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100"
              >
                <option value="user">User</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">Storage Quota (GB)</label>
              <input
                v-model.number="editUserQuotaGB"
                type="number"
                min="1"
                required
                class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100"
              />
            </div>
            <div>
              <label class="flex items-center gap-3 min-h-[44px] cursor-pointer">
                <input v-model="editingUser.isActive" type="checkbox" class="w-5 h-5 text-stone-800 dark:text-stone-200 rounded" />
                <span class="text-sm text-stone-700 dark:text-stone-300">Active</span>
              </label>
            </div>
            <div class="flex gap-3 pt-4">
              <button
                type="button"
                @click="editingUser = null"
                class="flex-1 min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 text-stone-700 dark:text-stone-300 rounded-lg hover:bg-stone-50 dark:hover:bg-neutral-700"
              >
                Cancel
              </button>
              <button
                type="submit"
                :disabled="loading"
                class="flex-1 min-h-[44px] px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 disabled:opacity-50"
              >
                Save
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>

    <!-- Create User Modal -->
    <div v-if="showCreateUserModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 pt-[env(safe-area-inset-top)] pb-[env(safe-area-inset-bottom)] bg-black/50 overflow-y-auto">
      <div class="modal-panel bg-white dark:bg-neutral-800 shadow-xl flex-shrink-0 my-auto">
        <div class="p-4 md:p-6 overflow-y-auto flex-1">
          <h3 class="text-lg font-semibold text-stone-900 dark:text-stone-100 mb-4">Create New User</h3>
          <form @submit.prevent="handleCreateUser" class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">Email *</label>
              <input
                v-model="newUserForm.email"
                type="email"
                required
                placeholder="user@example.com"
                class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">Name *</label>
              <input
                v-model="newUserForm.name"
                type="text"
                required
                placeholder="Full Name"
                class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">Password *</label>
              <input
                v-model="newUserForm.password"
                type="password"
                required
                minlength="8"
                placeholder="Minimum 8 characters"
                class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100"
              />
            </div>
            <div>
              <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">Role</label>
              <select
                v-model="newUserForm.role"
                class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100"
              >
                <option value="user">User</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <div>
              <label class="block text-sm font-medium text-stone-700 dark:text-stone-300 mb-1">Storage Quota (GB)</label>
              <input
                v-model.number="newUserForm.storageQuotaGB"
                type="number"
                min="1"
                required
                class="w-full min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 rounded-lg bg-white dark:bg-neutral-800 text-stone-900 dark:text-stone-100"
              />
            </div>
            <p v-if="createUserError" class="text-sm text-red-600 dark:text-red-400">{{ createUserError }}</p>
            <div class="flex gap-3 pt-4">
              <button
                type="button"
                @click="closeCreateUserModal"
                class="flex-1 min-h-[44px] px-4 py-2 border border-stone-300 dark:border-neutral-700 text-stone-700 dark:text-stone-300 rounded-lg hover:bg-stone-50 dark:hover:bg-neutral-700"
              >
                Cancel
              </button>
              <button
                type="submit"
                :disabled="loading"
                class="flex-1 min-h-[44px] px-4 py-2 bg-neutral-800 dark:bg-neutral-200 text-white dark:text-neutral-800 rounded-lg hover:bg-neutral-700 dark:hover:bg-neutral-300 disabled:opacity-50"
              >
                Create User
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, reactive, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useAdminStore, type User } from '@/stores/admin'
import { useModulesStore } from '@/stores/modules'
import api from '@/api'
import {
  ShieldCheckIcon,
  ArrowLeftIcon,
  ChartBarIcon,
  UsersIcon,
  Cog6ToothIcon,
  ClipboardDocumentListIcon,
  UserGroupIcon,
  ServerIcon,
  FolderIcon,
  ShareIcon,
  ArrowUpTrayIcon,
  ArrowDownTrayIcon,
  TrashIcon,
  ArrowPathIcon,
  WrenchIcon,
  MagnifyingGlassIcon,
  PlusIcon,
  PencilIcon,
  KeyIcon,
  PuzzlePieceIcon,
  DocumentTextIcon,
  DocumentIcon,
  ClipboardDocumentCheckIcon,
  CalendarIcon,
  UserIcon,
  EnvelopeIcon,
} from '@heroicons/vue/24/outline'

const adminStore = useAdminStore()
const modulesStore = useModulesStore()
const { modules } = storeToRefs(modulesStore)
const {
  users,
  systemStats,
  systemSettings,
  activityLogs,
  loading,
  usersPage,
  usersTotal,
  totalPages,
  logsPage,
  logsTotalPages,
} = storeToRefs(adminStore)

const activeTab = ref('overview')
const userSearch = ref('')
const editingUser = ref<User | null>(null)
const editUserQuotaGB = ref(10)
const showCreateUserModal = ref(false)
const createUserError = ref('')

// New user form
const newUserForm = reactive({
  email: '',
  name: '',
  password: '',
  role: 'user' as 'admin' | 'user',
  storageQuotaGB: 10,
})

function resetNewUserForm() {
  newUserForm.email = ''
  newUserForm.name = ''
  newUserForm.password = ''
  newUserForm.role = 'user'
  newUserForm.storageQuotaGB = 10
  createUserError.value = ''
}

function closeCreateUserModal() {
  showCreateUserModal.value = false
  resetNewUserForm()
}

async function handleCreateUser() {
  createUserError.value = ''
  try {
    await adminStore.createUser({
      email: newUserForm.email,
      password: newUserForm.password,
      name: newUserForm.name,
      role: newUserForm.role,
      storageQuota: newUserForm.storageQuotaGB * 1024 * 1024 * 1024, // Convert GB to bytes
    })
    closeCreateUserModal()
  } catch (err: any) {
    createUserError.value = err.response?.data?.error || err.message || 'Failed to create user'
  }
}

const tabs = [
  { id: 'overview', name: 'Overview', icon: ChartBarIcon },
  { id: 'users', name: 'Users', icon: UsersIcon },
  { id: 'modules', name: 'Modules', icon: PuzzlePieceIcon },
  { id: 'settings', name: 'Settings', icon: Cog6ToothIcon },
  { id: 'logs', name: 'Activity Logs', icon: ClipboardDocumentListIcon },
]

const moduleIcons: Record<string, any> = {
  documents: DocumentTextIcon,
  pdf: DocumentIcon,
  tasks: ClipboardDocumentCheckIcon,
  calendar: CalendarIcon,
  contacts: UserIcon,
  email: EnvelopeIcon,
}

const moduleDescriptions: Record<string, string> = {
  documents: 'Rich text editor for creating and collaborating on documents (Google Docs alternative)',
  pdf: 'PDF viewer with annotation tools including highlight, underline, and comments',
  tasks: 'Kanban-style task management with drag-and-drop, groups, and recurrence',
  calendar: 'Calendar with CalDAV integration for events and reminders',
  contacts: 'Contact management with CardDAV sync',
  email: 'Email client connecting to your IMAP/SMTP servers',
}

async function toggleModule(moduleId: string) {
  const module = modules.value.find(m => m.id === moduleId)
  if (module) {
    await modulesStore.toggleModule(moduleId, !module.enabled)
  }
}

const settingsForm = reactive({
  siteName: '',
  siteUrl: '',
  defaultQuotaGB: 10,
  maxUploadSizeMB: 100,
  allowRegistration: true,
  requireEmailVerification: true,
})

const smtpForm = reactive({
  smtpHost: '',
  smtpPort: 587,
  smtpUser: '',
  smtpFrom: '',
})

const logFilters = reactive({
  action: '',
  userId: '',
})

const statsCards = computed(() => [
  {
    label: 'Total Users',
    value: systemStats.value?.totalUsers || 0,
    icon: UserGroupIcon,
    bgColor: 'bg-stone-100 dark:bg-neutral-800',
    iconColor: 'text-stone-800 dark:text-stone-200 dark:text-stone-400',
    change: 12,
  },
  {
    label: 'Active Users',
    value: systemStats.value?.activeUsers || 0,
    icon: UsersIcon,
    bgColor: 'bg-green-100 dark:bg-green-900',
    iconColor: 'text-green-600 dark:text-green-400',
    change: 5,
  },
  {
    label: 'Total Files',
    value: systemStats.value?.totalFiles?.toLocaleString() || 0,
    icon: FolderIcon,
    bgColor: 'bg-purple-100 dark:bg-purple-900',
    iconColor: 'text-purple-600 dark:text-purple-400',
    change: 8,
  },
  {
    label: 'Active Shares',
    value: systemStats.value?.totalShares || 0,
    icon: ShareIcon,
    bgColor: 'bg-orange-100 dark:bg-orange-900',
    iconColor: 'text-orange-600 dark:text-orange-400',
    change: -3,
  },
])

const storagePercentage = computed(() => {
  if (!systemStats.value?.totalStorage) return 0
  return (systemStats.value.usedStorage / systemStats.value.totalStorage) * 100
})

// Utility functions
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString()
}

function formatDateTime(date: string): string {
  return new Date(date).toLocaleString()
}

function getActionColor(action: string): string {
  const colors: Record<string, string> = {
    login: 'bg-green-100 dark:bg-green-900 text-green-800 dark:text-green-200',
    logout: 'bg-stone-100 dark:bg-neutral-700 text-stone-800 dark:text-stone-200',
    upload: 'bg-stone-100 dark:bg-neutral-800 text-stone-800 dark:text-stone-200',
    download: 'bg-purple-100 dark:bg-purple-900 text-purple-800 dark:text-purple-200',
    delete: 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200',
    share: 'bg-orange-100 dark:bg-orange-900 text-orange-800 dark:text-orange-200',
  }
  return colors[action] || 'bg-stone-100 dark:bg-neutral-700 text-stone-800 dark:text-stone-200'
}

// Debounced search
let searchTimeout: ReturnType<typeof setTimeout>
function debouncedSearchUsers() {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => loadUsers(1), 300)
}

// Actions
async function loadUsers(page: number) {
  await adminStore.fetchUsers(page, userSearch.value)
}

async function loadLogs(page: number) {
  const filters: Record<string, string> = {}
  if (logFilters.action) filters.action = logFilters.action
  if (logFilters.userId) filters.userId = logFilters.userId
  await adminStore.fetchActivityLogs(page, filters)
}

function editUser(user: User) {
  editingUser.value = { ...user }
  editUserQuotaGB.value = Math.round(user.storageQuota / (1024 * 1024 * 1024))
}

async function saveUser() {
  if (!editingUser.value) return
  await adminStore.updateUser(editingUser.value.id, {
    name: editingUser.value.name,
    role: editingUser.value.role,
    storageQuota: editUserQuotaGB.value * 1024 * 1024 * 1024,
    isActive: editingUser.value.isActive,
  })
  editingUser.value = null
}

async function generateResetLink(user: User) {
  try {
    const response = await api.post(`/admin/users/${user.id}/reset-link`)
    const resetUrl = response.data.reset_url
    
    // Copy to clipboard
    await navigator.clipboard.writeText(resetUrl)
    alert(`Password reset link copied to clipboard!\n\nLink: ${resetUrl}\n\nThis link expires in 24 hours.`)
  } catch (err: any) {
    alert(err.response?.data?.error || 'Failed to generate reset link')
  }
}

async function confirmDeleteUser(user: User) {
  if (confirm(`Are you sure you want to delete ${user.name}? This action cannot be undone.`)) {
    await adminStore.deleteUser(user.id)
  }
}

async function saveSettings() {
  await adminStore.updateSystemSettings({
    siteName: settingsForm.siteName,
    siteUrl: settingsForm.siteUrl,
    defaultQuota: settingsForm.defaultQuotaGB * 1024 * 1024 * 1024,
    maxUploadSize: settingsForm.maxUploadSizeMB * 1024 * 1024,
    allowRegistration: settingsForm.allowRegistration,
    requireEmailVerification: settingsForm.requireEmailVerification,
  })
}

async function saveSmtpSettings() {
  await adminStore.updateSystemSettings({
    smtpHost: smtpForm.smtpHost,
    smtpPort: smtpForm.smtpPort,
    smtpUser: smtpForm.smtpUser,
    smtpFrom: smtpForm.smtpFrom,
  })
}

async function testSmtp() {
  // TODO: Implement test email endpoint
  alert('Test email sent!')
}

async function handleClearCache() {
  if (confirm('Are you sure you want to clear the cache?')) {
    await adminStore.clearCache()
    alert('Cache cleared successfully')
  }
}

async function handleRunCleanup() {
  if (confirm('This will remove orphaned files and expired shares. Continue?')) {
    const result = await adminStore.runCleanup()
    alert(`Cleanup complete: ${result.filesRemoved} files removed, ${result.sharesExpired} shares expired`)
  }
}

async function toggleMaintenance() {
  const enabled = !systemSettings.value?.maintenanceMode
  const message = enabled
    ? 'Enable maintenance mode? Users will not be able to access the system.'
    : 'Disable maintenance mode?'
  
  if (confirm(message)) {
    await adminStore.toggleMaintenanceMode(enabled)
  }
}

// Watch for settings changes
watch(systemSettings, (settings) => {
  if (settings) {
    settingsForm.siteName = settings.siteName
    settingsForm.siteUrl = settings.siteUrl
    settingsForm.defaultQuotaGB = Math.round(settings.defaultQuota / (1024 * 1024 * 1024))
    settingsForm.maxUploadSizeMB = Math.round(settings.maxUploadSize / (1024 * 1024))
    settingsForm.allowRegistration = settings.allowRegistration
    settingsForm.requireEmailVerification = settings.requireEmailVerification
    
    smtpForm.smtpHost = settings.smtpHost
    smtpForm.smtpPort = settings.smtpPort
    smtpForm.smtpUser = settings.smtpUser
    smtpForm.smtpFrom = settings.smtpFrom
  }
}, { immediate: true })

// Load initial data
onMounted(async () => {
  await Promise.all([
    adminStore.fetchSystemStats(),
    adminStore.fetchSystemSettings(),
    adminStore.fetchUsers(1),
    modulesStore.fetchModuleSettings(),
  ])
})
</script>
