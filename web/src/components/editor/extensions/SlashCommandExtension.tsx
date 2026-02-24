import { Extension } from '@tiptap/core'
import { ReactRenderer } from '@tiptap/react'
import Suggestion from '@tiptap/suggestion'
import type { SuggestionProps, SuggestionKeyDownProps } from '@tiptap/suggestion'
import tippy from 'tippy.js'
import type { Instance as TippyInstance } from 'tippy.js'
import {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useState,
  useCallback,
  useRef,
} from 'react'
import { slashCommands, groupedCommands } from '../SlashCommandMenu'
import type { SlashCommand } from '../SlashCommandMenu'

interface CommandListProps {
  items: SlashCommand[]
  command: (item: SlashCommand) => void
}

interface CommandListRef {
  onKeyDown: (props: SuggestionKeyDownProps) => boolean
}

const CommandList = forwardRef<CommandListRef, CommandListProps>(
  ({ items, command }, ref) => {
    const [selectedIndex, setSelectedIndex] = useState(0)
    const containerRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
      setSelectedIndex(0)
    }, [items])

    // Scroll selected item into view.
    useEffect(() => {
      const el = containerRef.current?.querySelector(`[data-index="${selectedIndex}"]`)
      el?.scrollIntoView({ block: 'nearest' })
    }, [selectedIndex])

    const selectItem = useCallback(
      (index: number) => {
        const item = items[index]
        if (item) command(item)
      },
      [items, command],
    )

    useImperativeHandle(ref, () => ({
      onKeyDown: ({ event }: SuggestionKeyDownProps) => {
        if (event.key === 'ArrowUp') {
          setSelectedIndex((prev) => (prev + items.length - 1) % items.length)
          return true
        }
        if (event.key === 'ArrowDown') {
          setSelectedIndex((prev) => (prev + 1) % items.length)
          return true
        }
        if (event.key === 'Enter') {
          selectItem(selectedIndex)
          return true
        }
        return false
      },
    }))

    if (!items.length) {
      return (
        <div className="rounded-lg border border-border bg-card p-3 text-sm text-muted-foreground shadow-lg">
          No matching commands
        </div>
      )
    }

    // Group items by category for display.
    const groups = new Map<string, { item: SlashCommand; flatIndex: number }[]>()
    items.forEach((item, flatIndex) => {
      const group = groups.get(item.category) || []
      group.push({ item, flatIndex })
      groups.set(item.category, group)
    })

    return (
      <div
        ref={containerRef}
        className="max-h-80 w-64 overflow-y-auto rounded-lg border border-border bg-card shadow-lg"
      >
        {Array.from(groups.entries()).map(([category, entries]) => (
          <div key={category}>
            <div className="px-3 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
              {category}
            </div>
            {entries.map(({ item, flatIndex }) => (
              <button
                key={item.title}
                data-index={flatIndex}
                onClick={() => selectItem(flatIndex)}
                className={`flex w-full items-center gap-3 px-3 py-1.5 text-left text-sm transition-colors ${
                  flatIndex === selectedIndex
                    ? 'bg-accent/20 text-accent'
                    : 'text-foreground hover:bg-muted'
                }`}
              >
                <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-border bg-muted text-muted-foreground">
                  {item.icon}
                </span>
                <div className="min-w-0">
                  <div className="font-medium">{item.title}</div>
                  <div className="truncate text-xs text-muted-foreground">
                    {item.description}
                  </div>
                </div>
              </button>
            ))}
          </div>
        ))}
      </div>
    )
  },
)
CommandList.displayName = 'CommandList'

export const SlashCommandExtension = Extension.create({
  name: 'slashCommand',

  addOptions() {
    return {
      suggestion: {
        char: '/',
        command: ({
          editor,
          range,
          props,
        }: {
          editor: SuggestionProps['editor']
          range: SuggestionProps['range']
          props: SlashCommand
        }) => {
          // Delete the slash trigger text then run the command.
          editor.chain().focus().deleteRange(range).run()
          props.action(editor)
        },
        items: ({ query }: { query: string }) => {
          if (!query) return slashCommands
          const groups = groupedCommands(query)
          const result: SlashCommand[] = []
          for (const items of groups.values()) {
            result.push(...items)
          }
          return result
        },
        render: () => {
          let component: ReactRenderer<CommandListRef> | null = null
          let popup: TippyInstance[] | null = null

          return {
            onStart: (props: SuggestionProps) => {
              component = new ReactRenderer(CommandList, {
                props: { items: props.items, command: props.command },
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
                placement: 'bottom-start',
              })
            },

            onUpdate(props: SuggestionProps) {
              component?.updateProps({
                items: props.items,
                command: props.command,
              })

              if (popup?.[0] && props.clientRect) {
                popup[0].setProps({
                  getReferenceClientRect: props.clientRect as () => DOMRect,
                })
              }
            },

            onKeyDown(props: SuggestionKeyDownProps) {
              if (props.event.key === 'Escape') {
                popup?.[0]?.hide()
                return true
              }
              return component?.ref?.onKeyDown(props) ?? false
            },

            onExit() {
              popup?.[0]?.destroy()
              component?.destroy()
            },
          }
        },
      } satisfies Partial<typeof Suggestion>,
    }
  },

  addProseMirrorPlugins() {
    return [
      Suggestion({
        editor: this.editor,
        ...this.options.suggestion,
      }),
    ]
  },
})
