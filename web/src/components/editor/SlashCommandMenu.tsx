import type { ReactNode } from 'react'
import type { Editor } from '@tiptap/core'
import {
  Pilcrow,
  Heading1,
  Heading2,
  Heading3,
  List,
  ListOrdered,
  ListChecks,
  Code,
  Table,
  Image,
  Quote,
  Minus,
  Info,
  AlertTriangle,
  OctagonAlert,
  Radio,
  Activity,
  Bell,
} from 'lucide-react'

export interface SlashCommand {
  title: string
  description: string
  icon: ReactNode
  category: string
  action: (editor: Editor) => void
}

export const slashCommands: SlashCommand[] = [
  // Basic
  {
    title: 'Paragraph',
    description: 'Plain text block',
    icon: <Pilcrow className="h-4 w-4" />,
    category: 'Basic',
    action: (editor) => editor.chain().focus().setParagraph().run(),
  },
  {
    title: 'Heading 1',
    description: 'Large heading',
    icon: <Heading1 className="h-4 w-4" />,
    category: 'Basic',
    action: (editor) => editor.chain().focus().toggleHeading({ level: 1 }).run(),
  },
  {
    title: 'Heading 2',
    description: 'Medium heading',
    icon: <Heading2 className="h-4 w-4" />,
    category: 'Basic',
    action: (editor) => editor.chain().focus().toggleHeading({ level: 2 }).run(),
  },
  {
    title: 'Heading 3',
    description: 'Small heading',
    icon: <Heading3 className="h-4 w-4" />,
    category: 'Basic',
    action: (editor) => editor.chain().focus().toggleHeading({ level: 3 }).run(),
  },
  {
    title: 'Bullet List',
    description: 'Unordered list',
    icon: <List className="h-4 w-4" />,
    category: 'Basic',
    action: (editor) => editor.chain().focus().toggleBulletList().run(),
  },
  {
    title: 'Numbered List',
    description: 'Ordered list',
    icon: <ListOrdered className="h-4 w-4" />,
    category: 'Basic',
    action: (editor) => editor.chain().focus().toggleOrderedList().run(),
  },
  {
    title: 'Task List',
    description: 'Checklist with checkboxes',
    icon: <ListChecks className="h-4 w-4" />,
    category: 'Basic',
    action: (editor) => editor.chain().focus().toggleTaskList().run(),
  },
  {
    title: 'Divider',
    description: 'Horizontal rule',
    icon: <Minus className="h-4 w-4" />,
    category: 'Basic',
    action: (editor) => editor.chain().focus().setHorizontalRule().run(),
  },

  // Media & Content
  {
    title: 'Code Block',
    description: 'Syntax-highlighted code',
    icon: <Code className="h-4 w-4" />,
    category: 'Media & Content',
    action: (editor) => editor.chain().focus().toggleCodeBlock().run(),
  },
  {
    title: 'Table',
    description: 'Insert a table',
    icon: <Table className="h-4 w-4" />,
    category: 'Media & Content',
    action: (editor) =>
      editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run(),
  },
  {
    title: 'Image',
    description: 'Upload an image',
    icon: <Image className="h-4 w-4" />,
    category: 'Media & Content',
    action: (editor) => {
      const input = document.createElement('input')
      input.type = 'file'
      input.accept = 'image/*'
      input.onchange = async () => {
        const file = input.files?.[0]
        if (!file) return
        try {
          const { api } = await import('@/api/client')
          type UploadResult = { url: string }
          const result = await api.upload<UploadResult>('/images', file)
          editor.chain().focus().setImage({ src: result.url }).run()
        } catch {
          // silently fail
        }
      }
      input.click()
    },
  },
  {
    title: 'Quote',
    description: 'Blockquote',
    icon: <Quote className="h-4 w-4" />,
    category: 'Media & Content',
    action: (editor) => editor.chain().focus().toggleBlockquote().run(),
  },

  // Callouts
  {
    title: 'Info',
    description: 'Info callout box',
    icon: <Info className="h-4 w-4" />,
    category: 'Callouts',
    action: (editor) => editor.chain().focus().setCallout({ type: 'info' }).run(),
  },
  {
    title: 'Warning',
    description: 'Warning callout box',
    icon: <AlertTriangle className="h-4 w-4" />,
    category: 'Callouts',
    action: (editor) => editor.chain().focus().setCallout({ type: 'warning' }).run(),
  },
  {
    title: 'Danger',
    description: 'Danger callout box',
    icon: <OctagonAlert className="h-4 w-4" />,
    category: 'Callouts',
    action: (editor) => editor.chain().focus().setCallout({ type: 'danger' }).run(),
  },

  // NightOwl (Live)
  {
    title: 'On-Call',
    description: 'Live on-call roster block',
    icon: <Radio className="h-4 w-4" />,
    category: 'NightOwl (Live)',
    action: (editor) =>
      editor
        .chain()
        .focus()
        .insertContent({
          type: 'liveContextBlock',
          attrs: { subtype: 'oncall' },
        })
        .run(),
  },
  {
    title: 'Service Status',
    description: 'Live service status block',
    icon: <Activity className="h-4 w-4" />,
    category: 'NightOwl (Live)',
    action: (editor) =>
      editor
        .chain()
        .focus()
        .insertContent({
          type: 'liveContextBlock',
          attrs: { subtype: 'service-status' },
        })
        .run(),
  },
  {
    title: 'Active Alerts',
    description: 'Live alerts block',
    icon: <Bell className="h-4 w-4" />,
    category: 'NightOwl (Live)',
    action: (editor) =>
      editor
        .chain()
        .focus()
        .insertContent({
          type: 'liveContextBlock',
          attrs: { subtype: 'alerts' },
        })
        .run(),
  },
]

// Group commands by category for rendering.
export function groupedCommands(filter: string) {
  const filtered = filter
    ? slashCommands.filter(
        (cmd) =>
          cmd.title.toLowerCase().includes(filter.toLowerCase()) ||
          cmd.description.toLowerCase().includes(filter.toLowerCase()),
      )
    : slashCommands

  const groups = new Map<string, SlashCommand[]>()
  for (const cmd of filtered) {
    const group = groups.get(cmd.category) || []
    group.push(cmd)
    groups.set(cmd.category, group)
  }
  return groups
}
