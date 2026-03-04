export type ToolbarGroup =
  | 'formatting'
  | 'headings'
  | 'lists'
  | 'alignment'
  | 'blocks'
  | 'media'
  | 'advanced'

export interface ToolbarExtensionMeta {
  id: string
  name: string
  description: string
  group: ToolbarGroup
  icon: string
  defaultEnabled: boolean
  alwaysLoaded?: boolean
}

export const TOOLBAR_EXTENSIONS: ToolbarExtensionMeta[] = [
  // Formatting
  {
    id: 'bold',
    name: 'Bold',
    description: 'Bold text formatting (Ctrl+B)',
    group: 'formatting',
    icon: 'B',
    defaultEnabled: true,
    alwaysLoaded: true,
  },
  {
    id: 'italic',
    name: 'Italic',
    description: 'Italic text formatting (Ctrl+I)',
    group: 'formatting',
    icon: 'I',
    defaultEnabled: true,
    alwaysLoaded: true,
  },
  {
    id: 'underline',
    name: 'Underline',
    description: 'Underline text formatting (Ctrl+U)',
    group: 'formatting',
    icon: 'U',
    defaultEnabled: true,
  },
  {
    id: 'strike',
    name: 'Strikethrough',
    description: 'Strikethrough text formatting',
    group: 'formatting',
    icon: 'S',
    defaultEnabled: true,
    alwaysLoaded: true,
  },
  {
    id: 'highlight',
    name: 'Highlight',
    description: 'Highlight text with color',
    group: 'formatting',
    icon: 'HL',
    defaultEnabled: true,
  },
  {
    id: 'color',
    name: 'Text Color',
    description: 'Change text color',
    group: 'formatting',
    icon: 'A',
    defaultEnabled: false,
  },
  {
    id: 'fontFamily',
    name: 'Font Family',
    description: 'Change font family',
    group: 'formatting',
    icon: 'Ff',
    defaultEnabled: false,
  },

  // Headings
  {
    id: 'heading',
    name: 'Headings',
    description: 'Heading levels 1, 2, and 3',
    group: 'headings',
    icon: 'H',
    defaultEnabled: true,
    alwaysLoaded: true,
  },

  // Lists
  {
    id: 'bulletList',
    name: 'Bullet List',
    description: 'Unordered bullet list',
    group: 'lists',
    icon: '•',
    defaultEnabled: true,
    alwaysLoaded: true,
  },
  {
    id: 'orderedList',
    name: 'Ordered List',
    description: 'Numbered ordered list',
    group: 'lists',
    icon: '1.',
    defaultEnabled: true,
    alwaysLoaded: true,
  },
  {
    id: 'taskList',
    name: 'Task List',
    description: 'Checklist with checkboxes',
    group: 'lists',
    icon: '☑',
    defaultEnabled: true,
  },

  // Alignment
  {
    id: 'textAlign',
    name: 'Text Alignment',
    description: 'Left, center, and right alignment',
    group: 'alignment',
    icon: '≡',
    defaultEnabled: true,
  },

  // Blocks
  {
    id: 'blockquote',
    name: 'Blockquote',
    description: 'Quoted text block',
    group: 'blocks',
    icon: '"',
    defaultEnabled: true,
    alwaysLoaded: true,
  },
  {
    id: 'codeBlock',
    name: 'Code Block',
    description: 'Fenced code block',
    group: 'blocks',
    icon: '<>',
    defaultEnabled: true,
    alwaysLoaded: true,
  },
  {
    id: 'horizontalRule',
    name: 'Horizontal Rule',
    description: 'Visual divider line',
    group: 'blocks',
    icon: '—',
    defaultEnabled: true,
    alwaysLoaded: true,
  },

  // Media
  {
    id: 'image',
    name: 'Image',
    description: 'Insert images from URL',
    group: 'media',
    icon: '🖼',
    defaultEnabled: true,
  },
  {
    id: 'link',
    name: 'Link',
    description: 'Insert hyperlinks',
    group: 'media',
    icon: '🔗',
    defaultEnabled: true,
  },
  {
    id: 'youtube',
    name: 'YouTube',
    description: 'Embed YouTube videos',
    group: 'media',
    icon: '▶',
    defaultEnabled: true,
  },
  {
    id: 'audio',
    name: 'Audio',
    description: 'Embed audio files',
    group: 'media',
    icon: '♪',
    defaultEnabled: false,
  },

  // Advanced
  {
    id: 'table',
    name: 'Table',
    description: 'Insert and edit tables',
    group: 'advanced',
    icon: '⊞',
    defaultEnabled: true,
  },
  {
    id: 'tableOfContents',
    name: 'Table of Contents',
    description: 'Auto-generated document outline',
    group: 'advanced',
    icon: '≡',
    defaultEnabled: false,
  },
  {
    id: 'emoji',
    name: 'Emoji',
    description: 'Insert emoji characters',
    group: 'advanced',
    icon: '😀',
    defaultEnabled: false,
  },
]

export const GROUP_LABELS: Record<ToolbarGroup, string> = {
  formatting: 'Text Formatting',
  headings: 'Headings',
  lists: 'Lists',
  alignment: 'Alignment',
  blocks: 'Blocks',
  media: 'Media & Embeds',
  advanced: 'Advanced',
}

export function getDefaultEnabledIds(): string[] {
  return TOOLBAR_EXTENSIONS.filter((e) => e.defaultEnabled).map((e) => e.id)
}
