<template>
  <div class="flex items-center gap-1 flex-wrap">
    <!-- Undo/Redo (always visible) -->
    <button @click="editor?.chain().focus().undo().run()" class="p-2 rounded hover:bg-stone-100 dark:hover:bg-neutral-700" title="Undo (Ctrl+Z)">
      <svg class="w-4 h-4 text-stone-600 dark:text-stone-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h10a8 8 0 018 8v2M3 10l6 6m-6-6l6-6" />
      </svg>
    </button>
    <button @click="editor?.chain().focus().redo().run()" class="p-2 rounded hover:bg-stone-100 dark:hover:bg-neutral-700" title="Redo (Ctrl+Y)">
      <svg class="w-4 h-4 text-stone-600 dark:text-stone-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 10H11a8 8 0 00-8 8v2M21 10l-6 6m6-6l-6-6" />
      </svg>
    </button>

    <ToolbarDivider />

    <!-- Bold -->
    <ToolbarButton
      v-if="show('bold')"
      :active="isActive('bold')"
      title="Bold (Ctrl+B)"
      @click="editor?.chain().focus().toggleBold().run()"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M13.5 15.5H10V12.5H13.5A1.5 1.5 0 0115 14A1.5 1.5 0 0113.5 15.5M10 6.5H13A1.5 1.5 0 0114.5 8A1.5 1.5 0 0113 9.5H10M15.6 10.79C16.57 10.11 17.25 9 17.25 8C17.25 5.74 15.5 4 13.25 4H7V18H14.04C16.14 18 17.75 16.3 17.75 14.21C17.75 12.69 16.89 11.39 15.6 10.79Z" />
      </svg>
    </ToolbarButton>

    <!-- Italic -->
    <ToolbarButton
      v-if="show('italic')"
      :active="isActive('italic')"
      title="Italic (Ctrl+I)"
      @click="editor?.chain().focus().toggleItalic().run()"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M10,4V7H12.21L8.79,15H6V18H14V15H11.79L15.21,7H18V4H10Z" />
      </svg>
    </ToolbarButton>

    <!-- Underline -->
    <ToolbarButton
      v-if="show('underline')"
      :active="isActive('underline')"
      title="Underline (Ctrl+U)"
      @click="editor?.chain().focus().toggleUnderline().run()"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M5,21H19V19H5V21M12,17A6,6 0 0,0 18,11V3H15.5V11A3.5,3.5 0 0,1 12,14.5A3.5,3.5 0 0,1 8.5,11V3H6V11A6,6 0 0,0 12,17Z" />
      </svg>
    </ToolbarButton>

    <!-- Strikethrough -->
    <ToolbarButton
      v-if="show('strike')"
      :active="isActive('strike')"
      title="Strikethrough"
      @click="editor?.chain().focus().toggleStrike().run()"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M3,14H21V12H3M5,4V7H10V10H14V7H19V4M10,19H14V16H10V19Z" />
      </svg>
    </ToolbarButton>

    <!-- Highlight -->
    <ToolbarButton
      v-if="show('highlight')"
      :active="isActive('highlight')"
      title="Highlight"
      @click="editor?.chain().focus().toggleHighlight().run()"
    >
      <svg class="w-4 h-4 text-yellow-500" viewBox="0 0 24 24" fill="currentColor">
        <path d="M15.243 3.343l5.414 5.414-1.414 1.414-5.414-5.414 1.414-1.414zm-1.414 1.414L4.1 14.486l-.707 6.364 6.364-.707 9.728-9.728-5.657-5.657zM5.686 18.313l-.465-4.187 4.652 4.652-4.187-.465z" />
      </svg>
    </ToolbarButton>

    <!-- Font Family dropdown -->
    <div v-if="show('fontFamily')" class="relative">
      <select
        @change="setFontFamily(($event.target as HTMLSelectElement).value)"
        class="p-1.5 rounded text-xs bg-transparent hover:bg-stone-100 dark:hover:bg-neutral-700 border-none cursor-pointer text-stone-600 dark:text-stone-400 focus:ring-0"
        title="Font Family"
      >
        <option value="">Default</option>
        <option value="Inter">Inter</option>
        <option value="Georgia">Georgia</option>
        <option value="monospace">Monospace</option>
        <option value="serif">Serif</option>
        <option value="Comic Sans MS, Comic Sans">Comic Sans</option>
      </select>
    </div>

    <ToolbarDivider v-if="show('bold') || show('italic') || show('underline') || show('strike') || show('highlight')" />

    <!-- Headings -->
    <template v-if="show('heading')">
      <ToolbarButton :active="isActive('heading', { level: 1 })" title="Heading 1" @click="editor?.chain().focus().toggleHeading({ level: 1 }).run()">
        <span class="text-xs font-bold">H1</span>
      </ToolbarButton>
      <ToolbarButton :active="isActive('heading', { level: 2 })" title="Heading 2" @click="editor?.chain().focus().toggleHeading({ level: 2 }).run()">
        <span class="text-xs font-bold">H2</span>
      </ToolbarButton>
      <ToolbarButton :active="isActive('heading', { level: 3 })" title="Heading 3" @click="editor?.chain().focus().toggleHeading({ level: 3 }).run()">
        <span class="text-xs font-bold">H3</span>
      </ToolbarButton>
      <ToolbarDivider />
    </template>

    <!-- Bullet List -->
    <ToolbarButton
      v-if="show('bulletList')"
      :active="isActive('bulletList')"
      title="Bullet List"
      @click="editor?.chain().focus().toggleBulletList().run()"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M7,5H21V7H7V5M7,13V11H21V13H7M4,4.5A1.5,1.5 0 0,1 5.5,6A1.5,1.5 0 0,1 4,7.5A1.5,1.5 0 0,1 2.5,6A1.5,1.5 0 0,1 4,4.5M4,10.5A1.5,1.5 0 0,1 5.5,12A1.5,1.5 0 0,1 4,13.5A1.5,1.5 0 0,1 2.5,12A1.5,1.5 0 0,1 4,10.5M7,19V17H21V19H7M4,16.5A1.5,1.5 0 0,1 5.5,18A1.5,1.5 0 0,1 4,19.5A1.5,1.5 0 0,1 2.5,18A1.5,1.5 0 0,1 4,16.5Z" />
      </svg>
    </ToolbarButton>

    <!-- Ordered List -->
    <ToolbarButton
      v-if="show('orderedList')"
      :active="isActive('orderedList')"
      title="Numbered List"
      @click="editor?.chain().focus().toggleOrderedList().run()"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M7,13V11H21V13H7M7,19V17H21V19H7M7,7V5H21V7H7M3,8V5H2V4H4V8H3M2,17V16H5V20H2V19H4V18.5H3V17.5H4V17H2M4.25,10A0.75,0.75 0 0,1 5,10.75C5,10.95 4.92,11.14 4.79,11.27L3.12,13H5V14H2V13.08L4,11H2V10H4.25Z" />
      </svg>
    </ToolbarButton>

    <!-- Task List -->
    <ToolbarButton
      v-if="show('taskList')"
      :active="isActive('taskList')"
      title="Task List"
      @click="editor?.chain().focus().toggleTaskList().run()"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M3 5h2v2H3V5zm4 0h14v2H7V5zM3 11h2v2H3v-2zm4 0h14v2H7v-2zM3 17h2v2H3v-2zm4 0h14v2H7v-2z" />
        <path d="M4 6l-1-1 .7-.7L4 4.6l1.3-1.3.7.7L4 6z" />
      </svg>
    </ToolbarButton>

    <ToolbarDivider v-if="show('bulletList') || show('orderedList') || show('taskList')" />

    <!-- Alignment -->
    <template v-if="show('textAlign')">
      <ToolbarButton title="Align Left" @click="editor?.chain().focus().setTextAlign('left').run()">
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
          <path d="M3,3H21V5H3V3M3,7H15V9H3V7M3,11H21V13H3V11M3,15H15V17H3V15M3,19H21V21H3V19Z" />
        </svg>
      </ToolbarButton>
      <ToolbarButton title="Align Center" @click="editor?.chain().focus().setTextAlign('center').run()">
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
          <path d="M3,3H21V5H3V3M7,7H17V9H7V7M3,11H21V13H3V11M7,15H17V17H7V15M3,19H21V21H3V19Z" />
        </svg>
      </ToolbarButton>
      <ToolbarButton title="Align Right" @click="editor?.chain().focus().setTextAlign('right').run()">
        <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
          <path d="M3,3H21V5H3V3M9,7H21V9H9V7M3,11H21V13H3V11M9,15H21V17H9V15M3,19H21V21H3V19Z" />
        </svg>
      </ToolbarButton>
      <ToolbarDivider />
    </template>

    <!-- Blockquote -->
    <ToolbarButton
      v-if="show('blockquote')"
      :active="isActive('blockquote')"
      title="Quote"
      @click="editor?.chain().focus().toggleBlockquote().run()"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M14,17H17L19,13V7H13V13H16M6,17H9L11,13V7H5V13H8L6,17Z" />
      </svg>
    </ToolbarButton>

    <!-- Code Block -->
    <ToolbarButton
      v-if="show('codeBlock')"
      :active="isActive('codeBlock')"
      title="Code Block"
      @click="editor?.chain().focus().toggleCodeBlock().run()"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M14.6,16.6L19.2,12L14.6,7.4L16,6L22,12L16,18L14.6,16.6M9.4,16.6L4.8,12L9.4,7.4L8,6L2,12L8,18L9.4,16.6Z" />
      </svg>
    </ToolbarButton>

    <ToolbarDivider v-if="show('blockquote') || show('codeBlock')" />

    <!-- Table -->
    <ToolbarButton
      v-if="show('table')"
      title="Insert Table"
      @click="editor?.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M5,4H19A2,2 0 0,1 21,6V18A2,2 0 0,1 19,20H5A2,2 0 0,1 3,18V6A2,2 0 0,1 5,4M5,8V12H11V8H5M13,8V12H19V8H13M5,14V18H11V14H5M13,14V18H19V14H13Z" />
      </svg>
    </ToolbarButton>

    <!-- Link -->
    <ToolbarButton
      v-if="show('link')"
      :active="isActive('link')"
      title="Add Link"
      @click="addLink"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M3.9,12C3.9,10.29 5.29,8.9 7,8.9H11V7H7A5,5 0 0,0 2,12A5,5 0 0,0 7,17H11V15.1H7C5.29,15.1 3.9,13.71 3.9,12M8,13H16V11H8V13M17,7H13V8.9H17C18.71,8.9 20.1,10.29 20.1,12C20.1,13.71 18.71,15.1 17,15.1H13V17H17A5,5 0 0,0 22,12A5,5 0 0,0 17,7Z" />
      </svg>
    </ToolbarButton>

    <!-- Image -->
    <ToolbarButton
      v-if="show('image')"
      title="Insert Image"
      @click="insertImage"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M21,3H3C2,3 1,4 1,5V19A2,2 0 0,0 3,21H21C22,21 23,20 23,19V5C23,4 22,3 21,3M5,17L8.5,12.5L11,15.5L14.5,11L19,17H5Z" />
      </svg>
    </ToolbarButton>

    <!-- YouTube -->
    <ToolbarButton
      v-if="show('youtube')"
      title="Embed YouTube Video"
      @click="insertYouTube"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M10,15L15.19,12L10,9V15M21.56,7.17C21.69,7.64 21.78,8.27 21.84,9.07C21.91,9.87 21.94,10.56 21.94,11.16L22,12C22,14.19 21.84,15.8 21.56,16.83C21.31,17.73 20.73,18.31 19.83,18.56C19.36,18.69 18.5,18.78 17.18,18.84C15.88,18.91 14.69,18.94 13.59,18.94L12,19C7.81,19 5.2,18.84 4.17,18.56C3.27,18.31 2.69,17.73 2.44,16.83C2.31,16.36 2.22,15.73 2.16,14.93C2.09,14.13 2.06,13.44 2.06,12.84L2,12C2,9.81 2.16,8.2 2.44,7.17C2.69,6.27 3.27,5.69 4.17,5.44C4.64,5.31 5.5,5.22 6.82,5.16C8.12,5.09 9.31,5.06 10.41,5.06L12,5C16.19,5 18.8,5.16 19.83,5.44C20.73,5.69 21.31,6.27 21.56,7.17Z" />
      </svg>
    </ToolbarButton>

    <!-- Audio -->
    <ToolbarButton
      v-if="show('audio')"
      title="Embed Audio"
      @click="insertAudio"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12,3V13.55C11.41,13.21 10.73,13 10,13A4,4 0 0,0 6,17A4,4 0 0,0 10,21A4,4 0 0,0 14,17V7H18V3H12Z" />
      </svg>
    </ToolbarButton>

    <!-- Emoji -->
    <ToolbarButton
      v-if="show('emoji')"
      title="Insert Emoji"
      @click="insertEmoji"
    >
      <span class="text-sm">😀</span>
    </ToolbarButton>

    <!-- Table of Contents -->
    <ToolbarButton
      v-if="show('tableOfContents')"
      title="Table of Contents"
      @click="insertTableOfContents"
    >
      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="currentColor">
        <path d="M3,4H7V8H3V4M9,5V7H21V5H9M3,10H7V14H3V10M9,11V13H21V11H9M3,16H7V20H3V16M9,17V19H21V17H9" />
      </svg>
    </ToolbarButton>
  </div>
</template>

<script setup lang="ts">
import type { Editor } from '@tiptap/core'
import { useEditorPrefsStore } from '@/stores/editorPrefs'
import ToolbarButton from './ToolbarButton.vue'
import ToolbarDivider from './ToolbarDivider.vue'

const props = defineProps<{
  editor: Editor | undefined
}>()

const prefs = useEditorPrefsStore()

function show(id: string): boolean {
  return prefs.isEnabled(id)
}

function isActive(name: string, attrs?: Record<string, any>): boolean {
  return props.editor?.isActive(name, attrs) ?? false
}

function addLink() {
  const url = window.prompt('Enter URL:')
  if (url) {
    props.editor?.chain().focus().setLink({ href: url }).run()
  }
}

function insertImage() {
  const url = window.prompt('Enter image URL:')
  if (url) {
    props.editor?.chain().focus().setImage({ src: url }).run()
  }
}

function insertYouTube() {
  const url = window.prompt('Enter YouTube URL:')
  if (url) {
    props.editor?.chain().focus().setYoutubeVideo({ src: url }).run()
  }
}

function insertAudio() {
  const url = window.prompt('Enter audio URL:')
  if (url) {
    (props.editor as any)?.chain().focus().setAudio({ src: url }).run()
  }
}

function insertEmoji() {
  const emoji = window.prompt('Enter emoji or shortcode (e.g. :smile:):')
  if (emoji) {
    props.editor?.chain().focus().insertContent(emoji).run()
  }
}

function insertTableOfContents() {
  props.editor?.chain().focus().insertContent({ type: 'tableOfContents' }).run()
}

function setFontFamily(font: string) {
  if (font) {
    props.editor?.chain().focus().setFontFamily(font).run()
  } else {
    props.editor?.chain().focus().unsetFontFamily().run()
  }
}
</script>
