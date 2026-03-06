import { VueRenderer } from '@tiptap/vue-3'
import tippy, { type Instance as TippyInstance } from 'tippy.js'
import SlashCommandMenu from '@/components/SlashCommandMenu.vue'
import type { SuggestionProps, SuggestionKeyDownProps } from '@tiptap/suggestion'
import type { SlashCommandItem } from './SlashCommands'

export const slashCommandsRender = () => {
  let component: VueRenderer | null = null
  let popup: TippyInstance[] | null = null

  return {
    onStart: (props: SuggestionProps<SlashCommandItem>) => {
      component = new VueRenderer(SlashCommandMenu, {
        props,
        editor: props.editor,
      })

      if (!props.clientRect) return

      popup = tippy('body', {
        getReferenceClientRect: props.clientRect as () => DOMRect,
        appendTo: () => document.body,
        content: component.element,
        showOnCreate: true,
        interactive: true,
        trigger: 'manual',
        placement: 'top-start',
        maxWidth: 'min(90vw, 256px)',
        strategy: 'fixed',
        popperOptions: {
          modifiers: [
            { name: 'preventOverflow', options: { padding: 8, boundary: 'viewport' } },
            { name: 'flip', options: { fallbackPlacements: ['bottom-start', 'top-end', 'bottom-end'] } },
          ],
        },
      })
    },

    onUpdate: (props: SuggestionProps<SlashCommandItem>) => {
      component?.updateProps(props)

      if (!props.clientRect) return

      popup?.[0]?.setProps({
        getReferenceClientRect: props.clientRect as () => DOMRect,
      })
    },

    onKeyDown: (props: SuggestionKeyDownProps) => {
      if (props.event.key === 'Escape') {
        popup?.[0]?.hide()
        return true
      }

      return (component?.ref as any)?.onKeyDown(props.event) ?? false
    },

    onExit: () => {
      popup?.[0]?.destroy()
      component?.destroy()
    },
  }
}
