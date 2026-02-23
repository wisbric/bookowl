import type { Editor } from '@tiptap/core'
import {
  Bold,
  Italic,
  Strikethrough,
  Code,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  ListChecks,
  Quote,
  Minus,
  Undo,
  Redo,
  Table,
  Image,
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface EditorToolbarProps {
  editor: Editor
}

export function EditorToolbar({ editor }: EditorToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-0.5 border-b border-border px-3 py-2">
      {/* Text formatting */}
      <ToolbarGroup>
        <ToolbarButton
          active={editor.isActive('bold')}
          onClick={() => editor.chain().focus().toggleBold().run()}
          title="Bold"
        >
          <Bold className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          active={editor.isActive('italic')}
          onClick={() => editor.chain().focus().toggleItalic().run()}
          title="Italic"
        >
          <Italic className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          active={editor.isActive('strike')}
          onClick={() => editor.chain().focus().toggleStrike().run()}
          title="Strikethrough"
        >
          <Strikethrough className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          active={editor.isActive('code')}
          onClick={() => editor.chain().focus().toggleCode().run()}
          title="Inline code"
        >
          <Code className="h-4 w-4" />
        </ToolbarButton>
      </ToolbarGroup>

      <ToolbarDivider />

      {/* Headings */}
      <ToolbarGroup>
        <ToolbarButton
          active={editor.isActive('heading', { level: 1 })}
          onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
          title="Heading 1"
        >
          <Heading1 className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          active={editor.isActive('heading', { level: 2 })}
          onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
          title="Heading 2"
        >
          <Heading2 className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          active={editor.isActive('heading', { level: 3 })}
          onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
          title="Heading 3"
        >
          <Heading3 className="h-4 w-4" />
        </ToolbarButton>
      </ToolbarGroup>

      <ToolbarDivider />

      {/* Lists */}
      <ToolbarGroup>
        <ToolbarButton
          active={editor.isActive('bulletList')}
          onClick={() => editor.chain().focus().toggleBulletList().run()}
          title="Bullet list"
        >
          <List className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          active={editor.isActive('orderedList')}
          onClick={() => editor.chain().focus().toggleOrderedList().run()}
          title="Ordered list"
        >
          <ListOrdered className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          active={editor.isActive('taskList')}
          onClick={() => editor.chain().focus().toggleTaskList().run()}
          title="Task list"
        >
          <ListChecks className="h-4 w-4" />
        </ToolbarButton>
      </ToolbarGroup>

      <ToolbarDivider />

      {/* Blocks */}
      <ToolbarGroup>
        <ToolbarButton
          active={editor.isActive('blockquote')}
          onClick={() => editor.chain().focus().toggleBlockquote().run()}
          title="Blockquote"
        >
          <Quote className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          active={editor.isActive('codeBlock')}
          onClick={() => editor.chain().focus().toggleCodeBlock().run()}
          title="Code block"
        >
          <Code className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          onClick={() => editor.chain().focus().setHorizontalRule().run()}
          title="Divider"
        >
          <Minus className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          onClick={() =>
            editor
              .chain()
              .focus()
              .insertTable({ rows: 3, cols: 3, withHeaderRow: true })
              .run()
          }
          title="Table"
        >
          <Table className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          onClick={() => {
            const url = window.prompt('Image URL:')
            if (url) editor.chain().focus().setImage({ src: url }).run()
          }}
          title="Image"
        >
          <Image className="h-4 w-4" />
        </ToolbarButton>
      </ToolbarGroup>

      <ToolbarDivider />

      {/* Callouts */}
      <ToolbarGroup>
        <ToolbarButton
          onClick={() => editor.chain().focus().setCallout({ type: 'info' }).run()}
          title="Info callout"
        >
          <span className="text-xs">ℹ️</span>
        </ToolbarButton>
        <ToolbarButton
          onClick={() => editor.chain().focus().setCallout({ type: 'warning' }).run()}
          title="Warning callout"
        >
          <span className="text-xs">⚠️</span>
        </ToolbarButton>
        <ToolbarButton
          onClick={() => editor.chain().focus().setCallout({ type: 'danger' }).run()}
          title="Danger callout"
        >
          <span className="text-xs">🚨</span>
        </ToolbarButton>
      </ToolbarGroup>

      <div className="ml-auto flex items-center gap-0.5">
        <ToolbarButton
          onClick={() => editor.chain().focus().undo().run()}
          disabled={!editor.can().undo()}
          title="Undo"
        >
          <Undo className="h-4 w-4" />
        </ToolbarButton>
        <ToolbarButton
          onClick={() => editor.chain().focus().redo().run()}
          disabled={!editor.can().redo()}
          title="Redo"
        >
          <Redo className="h-4 w-4" />
        </ToolbarButton>
      </div>
    </div>
  )
}

function ToolbarButton({
  children,
  active,
  disabled,
  onClick,
  title,
}: {
  children: React.ReactNode
  active?: boolean
  disabled?: boolean
  onClick: () => void
  title: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      title={title}
      className={cn(
        'rounded p-1.5 transition-colors',
        active
          ? 'bg-accent/20 text-accent'
          : 'text-muted-foreground hover:bg-muted hover:text-foreground',
        disabled && 'cursor-not-allowed opacity-40',
      )}
    >
      {children}
    </button>
  )
}

function ToolbarGroup({ children }: { children: React.ReactNode }) {
  return <div className="flex items-center gap-0.5">{children}</div>
}

function ToolbarDivider() {
  return <div className="mx-1 h-5 w-px bg-border" />
}
