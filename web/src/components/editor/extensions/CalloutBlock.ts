import { Node, mergeAttributes } from '@tiptap/core'

export type CalloutType = 'info' | 'warning' | 'danger'

declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    calloutBlock: {
      setCallout: (attrs: { type: CalloutType }) => ReturnType
    }
  }
}

export const CalloutBlock = Node.create({
  name: 'calloutBlock',
  group: 'block',
  content: 'block+',
  defining: true,

  addAttributes() {
    return {
      type: {
        default: 'info' as CalloutType,
        parseHTML: (element) =>
          (element.getAttribute('data-callout-type') as CalloutType) || 'info',
        renderHTML: (attributes) => ({
          'data-callout-type': attributes.type,
        }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-callout-type]' }]
  },

  renderHTML({ HTMLAttributes }) {
    const type = HTMLAttributes['data-callout-type'] || 'info'
    return [
      'div',
      mergeAttributes(HTMLAttributes, {
        class: `callout callout-${type}`,
        'data-callout-type': type,
      }),
      0,
    ]
  },

  addCommands() {
    return {
      setCallout:
        (attrs) =>
        ({ commands }) => {
          return commands.wrapIn(this.name, attrs)
        },
    }
  },
})
