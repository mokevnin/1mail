import { RichTextEditor } from '@mantine/tiptap'
import { useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { type ComponentProps, useEffect } from 'react'

interface RichTextFieldProps {
  value: string
  onChange: (html: string) => void
}

// RichTextField is the broadcast HTML body editor, built on the official Mantine
// TipTap wrapper (no custom CSS — Mantine's stylesheet is imported in main.tsx).
// It is controlled: external value changes (e.g. an edited broadcast loaded from
// the API) are synced into the editor without re-emitting an update.
export function RichTextField({ value, onChange }: RichTextFieldProps) {
  // StarterKit v3 already bundles the Link extension, so the RichTextEditor.Link
  // control works without registering a separate one (a separate @tiptap/extension-link
  // import pulls a second @tiptap/core instance and breaks the types).
  const editor = useEditor({
    extensions: [StarterKit],
    content: value,
    // The param is typed structurally on purpose: tsgo flags TipTap's Editor as
    // two unrelated identities (a private `commandManager` field) under
    // exactOptionalPropertyTypes; we only need getHTML() here.
    onUpdate: ({ editor }: { editor: { getHTML: () => string } }) => onChange(editor.getHTML()),
  })

  useEffect(() => {
    if (editor && value !== editor.getHTML()) {
      editor.commands.setContent(value, { emitUpdate: false })
    }
  }, [value, editor])

  // tsgo treats @tiptap/react's Editor (from useEditor) and the one in
  // RichTextEditor's prop as two unrelated identities (private `commandManager`
  // field under exactOptionalPropertyTypes); cast to the prop's exact type.
  const editorProp = editor as unknown as ComponentProps<typeof RichTextEditor>['editor']

  return (
    <RichTextEditor editor={editorProp}>
      <RichTextEditor.Toolbar sticky>
        <RichTextEditor.ControlsGroup>
          <RichTextEditor.Bold />
          <RichTextEditor.Italic />
        </RichTextEditor.ControlsGroup>
        <RichTextEditor.ControlsGroup>
          <RichTextEditor.H1 />
          <RichTextEditor.H2 />
          <RichTextEditor.BulletList />
          <RichTextEditor.OrderedList />
        </RichTextEditor.ControlsGroup>
        <RichTextEditor.ControlsGroup>
          <RichTextEditor.Link />
          <RichTextEditor.Unlink />
        </RichTextEditor.ControlsGroup>
      </RichTextEditor.Toolbar>
      <RichTextEditor.Content />
    </RichTextEditor>
  )
}
