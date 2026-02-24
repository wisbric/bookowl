import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import Image from '@tiptap/extension-image'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'
import Placeholder from '@tiptap/extension-placeholder'
import { common, createLowlight } from 'lowlight'
import { useEffect, useRef, useCallback } from 'react'
import { CalloutBlock } from './extensions/CalloutBlock'
import { LiveContextBlock } from './extensions/LiveContextBlock'
import { EditorToolbar } from './EditorToolbar'
import { api } from '@/api/client'
import type { ImageUploadResponse } from '@/api/client'

const lowlight = createLowlight(common)

interface BookOwlEditorProps {
  content: Record<string, unknown>
  editable: boolean
  onSave?: (content: Record<string, unknown>) => void
}

async function uploadImageFile(file: File): Promise<string | null> {
  try {
    const result = await api.upload<ImageUploadResponse>('/images', file)
    return result.url
  } catch {
    return null
  }
}

export function BookOwlEditor({ content, editable, onSave }: BookOwlEditorProps) {
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined)

  const debouncedSave = useCallback(
    (json: Record<string, unknown>) => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      debounceRef.current = setTimeout(() => {
        onSave?.(json)
      }, 1000)
    },
    [onSave],
  )

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        codeBlock: false, // Replaced by CodeBlockLowlight
      }),
      TaskList,
      TaskItem.configure({ nested: false }),
      CodeBlockLowlight.configure({ lowlight }),
      Image,
      Table.configure({ resizable: true }),
      TableRow,
      TableCell,
      TableHeader,
      Placeholder.configure({
        placeholder: "Type '/' for commands\u2026",
      }),
      CalloutBlock,
      LiveContextBlock,
    ],
    content,
    editable,
    onUpdate: ({ editor }) => {
      if (editable) {
        debouncedSave(editor.getJSON())
      }
    },
    editorProps: {
      attributes: {
        class: 'tiptap prose prose-invert max-w-none focus:outline-none min-h-[300px]',
      },
      handleDrop: (view, event, _slice, moved) => {
        if (moved || !event.dataTransfer?.files.length) return false

        const images = Array.from(event.dataTransfer.files).filter((f) =>
          f.type.startsWith('image/'),
        )
        if (!images.length) return false

        event.preventDefault()
        const coords = view.posAtCoords({ left: event.clientX, top: event.clientY })

        for (const file of images) {
          uploadImageFile(file).then((url) => {
            if (url) {
              const node = view.state.schema.nodes.image.create({ src: url })
              const tr = view.state.tr.insert(coords?.pos ?? view.state.selection.anchor, node)
              view.dispatch(tr)
            }
          })
        }
        return true
      },
      handlePaste: (view, event) => {
        const items = event.clipboardData?.items
        if (!items) return false

        const imageItems = Array.from(items).filter((item) =>
          item.type.startsWith('image/'),
        )
        if (!imageItems.length) return false

        event.preventDefault()
        for (const item of imageItems) {
          const file = item.getAsFile()
          if (!file) continue

          uploadImageFile(file).then((url) => {
            if (url) {
              const node = view.state.schema.nodes.image.create({ src: url })
              const tr = view.state.tr.replaceSelectionWith(node)
              view.dispatch(tr)
            }
          })
        }
        return true
      },
    },
  })

  // Sync editable state when it changes.
  useEffect(() => {
    if (editor) {
      editor.setEditable(editable)
    }
  }, [editor, editable])

  // Cleanup debounce on unmount.
  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
    }
  }, [])

  if (!editor) return null

  return (
    <div className="rounded-lg border border-border bg-card">
      {editable && <EditorToolbar editor={editor} />}
      <div className="px-6 py-4">
        <EditorContent editor={editor} />
      </div>
    </div>
  )
}
