import { Node, mergeAttributes } from '@tiptap/core'
import { ReactNodeViewRenderer } from '@tiptap/react'
import { DiagramBlockView } from './DiagramBlockView'

export const DiagramBlock = Node.create({
  name: 'diagramBlock',
  group: 'block',
  atom: true,
  draggable: true,
  selectable: true,

  addAttributes() {
    return {
      xml: {
        default: '',
        parseHTML: (el) => el.getAttribute('data-xml') ?? '',
        renderHTML: (attrs) => ({ 'data-xml': attrs.xml }),
      },
      svg: {
        default: '',
        parseHTML: (el) => el.getAttribute('data-svg') ?? '',
        renderHTML: (attrs) => attrs.svg ? { 'data-svg': attrs.svg } : {},
      },
      width: {
        default: null,
        parseHTML: (el) => el.getAttribute('data-width'),
        renderHTML: (attrs) => attrs.width ? { 'data-width': attrs.width } : {},
      },
      height: {
        default: null,
        parseHTML: (el) => el.getAttribute('data-height'),
        renderHTML: (attrs) => attrs.height ? { 'data-height': attrs.height } : {},
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-type="diagram-block"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { 'data-type': 'diagram-block' })]
  },

  addNodeView() {
    return ReactNodeViewRenderer(DiagramBlockView)
  },
})
