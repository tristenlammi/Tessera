/**
 * Returns editorProps.handlePaste to insert pasted images as base64 into the editor.
 * Use with useEditor({ editorProps: { handlePaste: useEditorPasteImage() } }).
 */
export function useEditorPasteImage() {
  return (view: any, event: ClipboardEvent) => {
    const items = event.clipboardData?.items
    if (!items) return false

    for (const item of items) {
      if (!item.type.startsWith('image/')) continue

      event.preventDefault()
      const blob = item.getAsFile()
      if (!blob) continue

      const reader = new FileReader()
      reader.onload = () => {
        const dataUrl = reader.result as string
        const schema = view.state.schema
        const imageNode = schema.nodes.image?.create?.({ src: dataUrl })
        if (imageNode) {
          const tr = view.state.tr.replaceSelectionWith(imageNode)
          view.dispatch(tr)
        }
      }
      reader.readAsDataURL(blob)
      return true
    }
    return false
  }
}
