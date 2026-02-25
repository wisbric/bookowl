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
import Collaboration from '@tiptap/extension-collaboration'
import CollaborationCursor from '@tiptap/extension-collaboration-cursor'
import { common, createLowlight } from 'lowlight'
import { useEffect, useRef, useCallback, useMemo } from 'react'
import { CalloutBlock } from './extensions/CalloutBlock'
import { LiveContextBlock } from './extensions/LiveContextBlock'
import { DiagramBlock } from './extensions/DiagramBlock'
import { SlashCommandExtension } from './extensions/SlashCommandExtension'
import { EditorToolbar } from './EditorToolbar'
import { api } from '@/api/client'
import type { ImageUploadResponse } from '@/api/client'
import type { HocuspocusProvider } from '@hocuspocus/provider'
import type { Doc as YDoc } from 'yjs'

const lowlight = createLowlight(common)

interface BookOwlEditorProps {
  content: Record<string, unknown>
  editable: boolean
  onSave?: (content: Record<string, unknown>) => void
  // Collab props — when provided, collab mode is active.
  collabProvider?: HocuspocusProvider | null
  ydoc?: YDoc
  userName?: string
  userColor?: string
}

async function uploadImageFile(file: File): Promise<string | null> {
  try {
    const result = await api.upload<ImageUploadResponse>('/images', file)
    return result.url
  } catch {
    return null
  }
}

export function BookOwlEditor({
  content,
  editable,
  onSave,
  collabProvider,
  ydoc,
  userName,
  userColor,
}: BookOwlEditorProps) {
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const isCollab = !!collabProvider && !!ydoc

  const debouncedSave = useCallback(
    (json: Record<string, unknown>) => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      debounceRef.current = setTimeout(() => {
        onSave?.(json)
      }, 1000)
    },
    [onSave],
  )

  // Build extensions list based on collab mode.
  const extensions = useMemo(() => {
    const exts = [
      StarterKit.configure({
        codeBlock: false,
        // Disable history when collab is active — Yjs handles undo/redo.
        history: isCollab ? false : undefined,
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
      DiagramBlock,
      SlashCommandExtension,
    ]

    if (isCollab) {
      exts.push(
        Collaboration.configure({ document: ydoc }),
        CollaborationCursor.configure({
          provider: collabProvider,
          user: {
            name: userName || 'Anonymous',
            color: userColor || '#6B7280',
          },
        }),
      )
    }

    return exts
  }, [isCollab, ydoc, collabProvider, userName, userColor])

  const editor = useEditor({
    extensions,
    // When collab is active, Yjs owns the content — don't set initial content.
    content: isCollab ? undefined : content,
    editable,
    immediatelyRender: true,
    // Only auto-save when NOT in collab mode — Hocuspocus handles persistence.
    onUpdate: isCollab
      ? undefined
      : ({ editor }) => {
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
