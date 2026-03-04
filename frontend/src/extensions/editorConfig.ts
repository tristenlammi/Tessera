import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Highlight from '@tiptap/extension-highlight'
import Underline from '@tiptap/extension-underline'
import TextAlign from '@tiptap/extension-text-align'
import Image from '@tiptap/extension-image'
import Link from '@tiptap/extension-link'
import { Table } from '@tiptap/extension-table'
import { TableRow } from '@tiptap/extension-table-row'
import { TableCell } from '@tiptap/extension-table-cell'
import { TableHeader } from '@tiptap/extension-table-header'
import { TextStyle } from '@tiptap/extension-text-style'
import Color from '@tiptap/extension-color'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import { Markdown } from '@tiptap/markdown'
import Youtube from '@tiptap/extension-youtube'
import FontFamily from '@tiptap/extension-font-family'
import TableOfContents from '@tiptap/extension-table-of-contents'
import Emoji from '@tiptap/extension-emoji'
import { AudioNode } from './AudioNode'
import { createSlashCommandsExtension } from './SlashCommands'
import { slashCommandsRender } from './slashCommandsRender'
import type { AnyExtension } from '@tiptap/core'

export interface EditorConfigOptions {
  placeholder?: string
  enableSlashCommands?: boolean
  enabledToolbarIds?: string[]
}

function has(ids: string[] | undefined, id: string): boolean {
  return !ids || ids.includes(id)
}

export function createEditorExtensions(options: EditorConfigOptions = {}): AnyExtension[] {
  const ids = options.enabledToolbarIds

  const extensions: AnyExtension[] = [
    StarterKit.configure({
      heading: { levels: [1, 2, 3] },
      bulletList: { keepMarks: true, keepAttributes: false },
      orderedList: { keepMarks: true, keepAttributes: false },
    }),
    Placeholder.configure({
      placeholder: options.placeholder ?? "Type '/' for commands...",
    }),
    Markdown.configure({
      markedOptions: { gfm: true, breaks: false },
    }),
    TextStyle,
  ]

  if (has(ids, 'highlight')) {
    extensions.push(Highlight.configure({ multicolor: true }))
  }

  if (has(ids, 'underline')) {
    extensions.push(Underline)
  }

  if (has(ids, 'textAlign')) {
    extensions.push(TextAlign.configure({ types: ['heading', 'paragraph'] }))
  }

  if (has(ids, 'image')) {
    extensions.push(Image.configure({ inline: true, allowBase64: true }))
  }

  if (has(ids, 'link')) {
    extensions.push(Link.configure({ openOnClick: false }))
  }

  if (has(ids, 'table')) {
    extensions.push(
      Table.configure({ resizable: true }),
      TableRow,
      TableCell,
      TableHeader,
    )
  }

  if (has(ids, 'color')) {
    extensions.push(Color)
  }

  if (has(ids, 'taskList')) {
    extensions.push(TaskList, TaskItem.configure({ nested: true }))
  }

  if (has(ids, 'youtube')) {
    extensions.push(
      Youtube.configure({
        controls: true,
        nocookie: true,
      }),
    )
  }

  if (has(ids, 'fontFamily')) {
    extensions.push(FontFamily)
  }

  if (has(ids, 'tableOfContents')) {
    extensions.push(TableOfContents)
  }

  if (has(ids, 'emoji')) {
    extensions.push(Emoji)
  }

  if (has(ids, 'audio')) {
    extensions.push(AudioNode)
  }

  if (options.enableSlashCommands !== false) {
    extensions.push(
      createSlashCommandsExtension({
        render: slashCommandsRender,
      }),
    )
  }

  return extensions
}
