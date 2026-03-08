<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import api from '@/api'

export interface DocFile {
  id: string
  parent_id: string | null
  name: string
  is_folder: boolean
  mime_type: string
  updated_at: string
}

const props = defineProps<{
  documentsFolderId: string | null
  currentFileId?: string | null
}>()

const emit = defineEmits<{
  openFile: [fileId: string]
  close: []
}>()

const router = useRouter()
const route = useRoute()

// --- Resizable sidebar ---
const STORAGE_KEY = 'tessera:docs-sidebar-width'
const DEFAULT_WIDTH = 260
const MIN_WIDTH = 180
const MAX_WIDTH = 480

const sidebarWidth = ref(parseInt(localStorage.getItem(STORAGE_KEY) || String(DEFAULT_WIDTH), 10))
let dragging = false
let dragStartX = 0
let dragStartWidth = 0

function onDragStart(e: MouseEvent | TouchEvent) {
  dragging = true
  dragStartX = 'touches' in e ? e.touches[0].clientX : e.clientX
  dragStartWidth = sidebarWidth.value
  document.addEventListener('mousemove', onDragMove)
  document.addEventListener('mouseup', onDragEnd)
  document.addEventListener('touchmove', onDragMove)
  document.addEventListener('touchend', onDragEnd)
}

function onDragMove(e: MouseEvent | TouchEvent) {
  if (!dragging) return
  const x = 'touches' in e ? e.touches[0].clientX : e.clientX
  const delta = x - dragStartX
  const newWidth = Math.min(MAX_WIDTH, Math.max(MIN_WIDTH, dragStartWidth + delta))
  sidebarWidth.value = newWidth
}

function onDragEnd() {
  dragging = false
  localStorage.setItem(STORAGE_KEY, String(sidebarWidth.value))
  document.removeEventListener('mousemove', onDragMove)
  document.removeEventListener('mouseup', onDragEnd)
  document.removeEventListener('touchmove', onDragMove)
  document.removeEventListener('touchend', onDragEnd)
}

// --- File tree ---
// Map from folderId -> children list
const childrenMap = ref<Record<string, DocFile[]>>({})
const expandedFolders = ref<Set<string>>(new Set())
const createTargetId = ref<string | null>(null)
const showNewFolderInput = ref(false)
const showNewDocInput = ref(false)
const newItemName = ref('')
const error = ref<string | null>(null)

async function loadChildren(folderId: string) {
  try {
    const response = await api.get('/files', { params: { parent_id: folderId } })
    const files: DocFile[] = response.data.files || []
    childrenMap.value[folderId] = files
  } catch (err: any) {
    error.value = err.response?.data?.error ?? 'Failed to load files'
  }
}

async function toggleFolder(folderId: string) {
  if (expandedFolders.value.has(folderId)) {
    expandedFolders.value.delete(folderId)
  } else {
    expandedFolders.value.add(folderId)
    if (!childrenMap.value[folderId]) {
      await loadChildren(folderId)
    }
  }
}

function openFile(file: DocFile) {
  emit('openFile', file.id)
  router.push({ name: 'documents-doc', params: { fileId: file.id } })
}

const currentFileId = computed(() => props.currentFileId ?? (route.params.fileId as string | undefined))

// --- Folder navigation ---
const currentBrowseFolderId = ref<string | null>(props.documentsFolderId)

watch(() => props.documentsFolderId, (id) => {
  if (id) {
    currentBrowseFolderId.value = id
    expandedFolders.value.add(id)
    loadChildren(id)
  }
}, { immediate: true })

// --- Create new document ---
function startNewDoc(parentFolderId: string) {
  createTargetId.value = parentFolderId
  showNewDocInput.value = true
  showNewFolderInput.value = false
  newItemName.value = ''
}

function startNewFolder(parentFolderId: string) {
  createTargetId.value = parentFolderId
  showNewFolderInput.value = true
  showNewDocInput.value = false
  newItemName.value = ''
}

async function confirmNewDoc() {
  if (!newItemName.value.trim() || !createTargetId.value) return
  const name = newItemName.value.trim()
  try {
    const response = await api.post('/documents/create-file', {
      name,
      title: name,
      content: '',
      format: 'markdown',
      parentId: createTargetId.value,
    })
    const file = response.data
    showNewDocInput.value = false
    newItemName.value = ''
    await loadChildren(createTargetId.value)
    if (file?.id) {
      router.push({ name: 'documents-doc', params: { fileId: file.id } })
    }
  } catch (err: any) {
    error.value = err.response?.data?.error ?? 'Failed to create document'
  }
}

async function confirmNewFolder() {
  if (!newItemName.value.trim() || !createTargetId.value) return
  const name = newItemName.value.trim()
  try {
    await api.post('/files/folder', { name, parent_id: createTargetId.value })
    showNewFolderInput.value = false
    newItemName.value = ''
    await loadChildren(createTargetId.value)
    if (!expandedFolders.value.has(createTargetId.value)) {
      expandedFolders.value.add(createTargetId.value)
    }
  } catch (err: any) {
    error.value = err.response?.data?.error ?? 'Failed to create folder'
  }
}

function cancelNew() {
  showNewDocInput.value = false
  showNewFolderInput.value = false
  newItemName.value = ''
  createTargetId.value = null
}

function handleNewInputKey(e: KeyboardEvent, isFolder: boolean) {
  if (e.key === 'Enter') isFolder ? confirmNewFolder() : confirmNewDoc()
  if (e.key === 'Escape') cancelNew()
}

// --- Refresh on route change ---
watch(() => route.params.fileId, () => {
  if (props.documentsFolderId && !childrenMap.value[props.documentsFolderId]) {
    loadChildren(props.documentsFolderId)
  }
})

onMounted(async () => {
  if (props.documentsFolderId) {
    expandedFolders.value.add(props.documentsFolderId)
    await loadChildren(props.documentsFolderId)
  }
})
</script>

<template>
  <div
    class="relative flex-shrink-0 h-full flex flex-col bg-neutral-50 dark:bg-neutral-900 border-r border-stone-200 dark:border-neutral-700 overflow-hidden"
    :style="{ width: sidebarWidth + 'px' }"
  >
    <!-- Header -->
    <div class="flex items-center justify-between px-3 py-2 border-b border-stone-200 dark:border-neutral-700 flex-shrink-0">
      <span class="text-xs font-semibold text-stone-500 dark:text-stone-400 uppercase tracking-wide">Documents</span>
      <div class="flex items-center gap-1">
        <!-- New folder button -->
        <button
          v-if="documentsFolderId"
          @click="startNewFolder(documentsFolderId!)"
          class="p-1 rounded hover:bg-stone-200 dark:hover:bg-neutral-700 text-stone-500 dark:text-stone-400"
          title="New Folder"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
          </svg>
        </button>
        <!-- New document button -->
        <button
          v-if="documentsFolderId"
          @click="startNewDoc(documentsFolderId!)"
          class="p-1 rounded hover:bg-stone-200 dark:hover:bg-neutral-700 text-stone-500 dark:text-stone-400"
          title="New Document"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
          </svg>
        </button>
        <!-- Close (mobile only) -->
        <button
          @click="emit('close')"
          class="p-1 rounded hover:bg-stone-200 dark:hover:bg-neutral-700 text-stone-500 dark:text-stone-400 md:hidden"
          title="Close sidebar"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Error -->
    <div v-if="error" class="px-3 py-2 text-xs text-red-500">{{ error }}</div>

    <!-- Tree -->
    <div class="flex-1 overflow-y-auto py-1">
      <div v-if="!documentsFolderId" class="px-3 py-4 text-xs text-stone-400">Loading...</div>
      <template v-else>
        <!-- Root new item inputs -->
        <div v-if="showNewFolderInput && createTargetId === documentsFolderId" class="flex items-center gap-1 px-2 py-0.5">
          <svg class="w-4 h-4 text-stone-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
          <input
            v-model="newItemName"
            autofocus
            placeholder="Folder name"
            class="flex-1 text-xs bg-white dark:bg-neutral-800 border border-stone-300 dark:border-neutral-600 rounded px-1 py-0.5 outline-none focus:ring-1 focus:ring-stone-400"
            @keydown="handleNewInputKey($event, true)"
            @blur="cancelNew"
          />
        </div>
        <div v-if="showNewDocInput && createTargetId === documentsFolderId" class="flex items-center gap-1 px-2 py-0.5">
          <svg class="w-4 h-4 text-stone-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <input
            v-model="newItemName"
            autofocus
            placeholder="Document name"
            class="flex-1 text-xs bg-white dark:bg-neutral-800 border border-stone-300 dark:border-neutral-600 rounded px-1 py-0.5 outline-none focus:ring-1 focus:ring-stone-400"
            @keydown="handleNewInputKey($event, false)"
            @blur="cancelNew"
          />
        </div>

        <!-- File tree nodes -->
        <DocTreeNode
          v-for="item in (childrenMap[documentsFolderId] || [])"
          :key="item.id"
          :file="item"
          :depth="0"
          :expanded-folders="expandedFolders"
          :children-map="childrenMap"
          :current-file-id="currentFileId"
          :create-target-id="createTargetId"
          :show-new-folder-input="showNewFolderInput"
          :show-new-doc-input="showNewDocInput"
          :new-item-name="newItemName"
          @toggle="toggleFolder"
          @open="openFile"
          @new-doc="startNewDoc"
          @new-folder="startNewFolder"
          @load-children="loadChildren"
          @cancel-new="cancelNew"
          @update:new-item-name="newItemName = $event"
          @confirm-new-folder="confirmNewFolder"
          @confirm-new-doc="confirmNewDoc"
        />
      </template>
    </div>

    <!-- Drag handle -->
    <div
      class="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-stone-300 dark:hover:bg-neutral-600 transition-colors z-10"
      @mousedown="onDragStart"
      @touchstart.prevent="onDragStart"
    />
  </div>
</template>

<!-- Recursive tree node as a separate component in same file -->
<script lang="ts">
import { defineComponent, h, resolveComponent } from 'vue'

export const DocTreeNode = defineComponent({
  name: 'DocTreeNode',
  props: {
    file: { type: Object as () => DocFile, required: true },
    depth: { type: Number, default: 0 },
    expandedFolders: { type: Object as () => Set<string>, required: true },
    childrenMap: { type: Object as () => Record<string, DocFile[]>, required: true },
    currentFileId: { type: String as () => string | undefined, default: undefined },
    createTargetId: { type: String as () => string | null, default: null },
    showNewFolderInput: { type: Boolean, default: false },
    showNewDocInput: { type: Boolean, default: false },
    newItemName: { type: String, default: '' },
  },
  emits: ['toggle', 'open', 'new-doc', 'new-folder', 'load-children', 'cancel-new', 'update:new-item-name', 'confirm-new-folder', 'confirm-new-doc'],
  setup(props, { emit }) {
    return () => {
      const file = props.file
      const isFolder = file.is_folder
      const isDoc = !isFolder && (file.name.endsWith('.tdoc') || file.name.endsWith('.md'))
      if (!isFolder && !isDoc) return null

      const indent = props.depth * 12 + 8
      const isExpanded = props.expandedFolders.has(file.id)
      const isActive = props.currentFileId === file.id
      const displayName = file.name.replace(/\.tdoc$/, '').replace(/\.md$/, '')

      const children = props.childrenMap[file.id] || []

      const nodeElements: any[] = []

      // Row
      nodeElements.push(
        h('div', {
          class: [
            'flex items-center gap-1 py-0.5 pr-2 rounded cursor-pointer group select-none',
            isActive
              ? 'bg-stone-200 dark:bg-neutral-700 text-stone-900 dark:text-stone-100'
              : 'hover:bg-stone-100 dark:hover:bg-neutral-800 text-stone-700 dark:text-stone-300',
          ].join(' '),
          style: { paddingLeft: indent + 'px' },
          onClick: (e: MouseEvent) => {
            e.stopPropagation()
            if (isFolder) emit('toggle', file.id)
            else emit('open', file)
          },
        }, [
          // Folder chevron or spacer
          isFolder
            ? h('svg', { class: 'w-3 h-3 flex-shrink-0 transition-transform ' + (isExpanded ? 'rotate-90' : ''), fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' },
                [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M9 5l7 7-7 7' })]
              )
            : h('span', { class: 'w-3 flex-shrink-0' }),
          // Icon
          isFolder
            ? h('svg', { class: 'w-4 h-4 flex-shrink-0 text-stone-400', fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' },
                [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z' })]
              )
            : h('svg', { class: 'w-4 h-4 flex-shrink-0 text-stone-400', fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' },
                [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z' })]
              ),
          // Name
          h('span', { class: 'text-xs truncate flex-1 min-w-0', title: displayName }, displayName),
          // Folder action buttons (show on hover)
          isFolder
            ? h('div', { class: 'hidden group-hover:flex items-center gap-0.5 flex-shrink-0', onClick: (e: MouseEvent) => e.stopPropagation() }, [
                h('button', {
                  class: 'p-0.5 rounded hover:bg-stone-200 dark:hover:bg-neutral-700 text-stone-400',
                  title: 'New Document',
                  onClick: () => emit('new-doc', file.id),
                },
                  [h('svg', { class: 'w-3 h-3', fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' },
                    [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M12 4v16m8-8H4' })]
                  )]
                ),
                h('button', {
                  class: 'p-0.5 rounded hover:bg-stone-200 dark:hover:bg-neutral-700 text-stone-400',
                  title: 'New Subfolder',
                  onClick: () => emit('new-folder', file.id),
                },
                  [h('svg', { class: 'w-3 h-3', fill: 'none', stroke: 'currentColor', viewBox: '0 0 24 24' },
                    [h('path', { 'stroke-linecap': 'round', 'stroke-linejoin': 'round', 'stroke-width': '2', d: 'M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z' })]
                  )]
                ),
              ])
            : null,
        ])
      )

      // New item inputs inside this folder
      if (isFolder && isExpanded) {
        if (props.showNewFolderInput && props.createTargetId === file.id) {
          nodeElements.push(
            h('div', { class: 'flex items-center gap-1 py-0.5 pr-2', style: { paddingLeft: (indent + 12 + 8) + 'px' } }, [
              h('input', {
                value: props.newItemName,
                autofocus: true,
                placeholder: 'Folder name',
                class: 'flex-1 text-xs bg-white dark:bg-neutral-800 border border-stone-300 dark:border-neutral-600 rounded px-1 py-0.5 outline-none focus:ring-1 focus:ring-stone-400',
                onInput: (e: Event) => emit('update:new-item-name', (e.target as HTMLInputElement).value),
                onKeydown: (e: KeyboardEvent) => {
                  if (e.key === 'Enter') emit('confirm-new-folder')
                  if (e.key === 'Escape') emit('cancel-new')
                },
                onBlur: () => emit('cancel-new'),
              })
            ])
          )
        }
        if (props.showNewDocInput && props.createTargetId === file.id) {
          nodeElements.push(
            h('div', { class: 'flex items-center gap-1 py-0.5 pr-2', style: { paddingLeft: (indent + 12 + 8) + 'px' } }, [
              h('input', {
                value: props.newItemName,
                autofocus: true,
                placeholder: 'Document name',
                class: 'flex-1 text-xs bg-white dark:bg-neutral-800 border border-stone-300 dark:border-neutral-600 rounded px-1 py-0.5 outline-none focus:ring-1 focus:ring-stone-400',
                onInput: (e: Event) => emit('update:new-item-name', (e.target as HTMLInputElement).value),
                onKeydown: (e: KeyboardEvent) => {
                  if (e.key === 'Enter') emit('confirm-new-doc')
                  if (e.key === 'Escape') emit('cancel-new')
                },
                onBlur: () => emit('cancel-new'),
              })
            ])
          )
        }

        // Children
        const DocTreeNodeComp = resolveComponent('DocTreeNode')
        for (const child of children) {
          nodeElements.push(
            h(DocTreeNodeComp, {
              key: child.id,
              file: child,
              depth: props.depth + 1,
              expandedFolders: props.expandedFolders,
              childrenMap: props.childrenMap,
              currentFileId: props.currentFileId,
              createTargetId: props.createTargetId,
              showNewFolderInput: props.showNewFolderInput,
              showNewDocInput: props.showNewDocInput,
              newItemName: props.newItemName,
              onToggle: (id: string) => emit('toggle', id),
              onOpen: (f: DocFile) => emit('open', f),
              'onNew-doc': (id: string) => emit('new-doc', id),
              'onNew-folder': (id: string) => emit('new-folder', id),
              'onLoad-children': (id: string) => emit('load-children', id),
              'onCancel-new': () => emit('cancel-new'),
              'onUpdate:new-item-name': (v: string) => emit('update:new-item-name', v),
              'onConfirm-new-folder': () => emit('confirm-new-folder'),
              'onConfirm-new-doc': () => emit('confirm-new-doc'),
            })
          )
        }
      }

      return h('div', nodeElements)
    }
  }
})
</script>
