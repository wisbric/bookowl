import { Node, mergeAttributes } from '@tiptap/core'
import { ReactNodeViewRenderer, NodeViewWrapper } from '@tiptap/react'
import type { NodeViewProps } from '@tiptap/react'
import { OnCallBlock } from '../live-context/OnCallBlock'
import { ServiceStatusBlock } from '../live-context/ServiceStatusBlock'
import { ActiveAlertsBlock } from '../live-context/ActiveAlertsBlock'

export type LiveContextSubtype = 'oncall' | 'alerts' | 'service-status' | 'incident-link'

export const LiveContextBlock = Node.create({
  name: 'liveContextBlock',
  group: 'block',
  atom: true, // Not editable inline — renders as a single unit.

  addAttributes() {
    return {
      subtype: { default: 'oncall' as LiveContextSubtype },
      rosterId: { default: null },
      serviceName: { default: null },
      alertId: { default: null },
      label: { default: null },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-live-context]' }]
  },

  renderHTML({ HTMLAttributes }) {
    return [
      'div',
      mergeAttributes(HTMLAttributes, {
        'data-live-context': HTMLAttributes.subtype,
        class: 'live-context-block',
      }),
    ]
  },

  addNodeView() {
    return ReactNodeViewRenderer(LiveContextNodeView)
  },
})

function LiveContextNodeView({ node, editor }: NodeViewProps) {
  const { subtype, rosterId, serviceName, alertId, label } = node.attrs
  const isEditable = editor.isEditable

  return (
    <NodeViewWrapper className="live-context-block" data-live-context={subtype}>
      {subtype === 'oncall' && (
        <OnCallBlock rosterId={rosterId} label={label} editable={isEditable} />
      )}
      {subtype === 'service-status' && (
        <ServiceStatusBlock serviceName={serviceName} label={label} editable={isEditable} />
      )}
      {subtype === 'alerts' && (
        <ActiveAlertsBlock serviceName={serviceName} label={label} editable={isEditable} />
      )}
      {subtype === 'incident-link' && (
        <div className="text-sm text-muted-foreground">
          Incident: {alertId || 'Not configured'}
        </div>
      )}
    </NodeViewWrapper>
  )
}
