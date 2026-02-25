/**
 * Simple read-only renderer for Tiptap/ProseMirror JSON content.
 * Used as the default view for documents — Tiptap editor only loads when editing.
 * This avoids all Tiptap initialization issues while rendering content reliably.
 */

import type { ReactNode } from 'react'

interface TiptapNode {
  type: string
  content?: TiptapNode[]
  text?: string
  attrs?: Record<string, unknown>
  marks?: { type: string; attrs?: Record<string, unknown> }[]
}

interface ContentRendererProps {
  content: Record<string, unknown>
}

export function ContentRenderer({ content }: ContentRendererProps) {
  const doc = content as unknown as TiptapNode
  if (!doc || !doc.content) {
    return (
      <div className="text-muted-foreground text-sm">No content</div>
    )
  }

  return (
    <div className="tiptap prose prose-invert max-w-none">
      {doc.content.map((node, i) => (
        <RenderNode key={i} node={node} />
      ))}
    </div>
  )
}

function RenderNode({ node }: { node: TiptapNode }): ReactNode {
  switch (node.type) {
    case 'heading': {
      const level = (node.attrs?.level as number) || 1
      const Tag = `h${level}` as 'h1' | 'h2' | 'h3' | 'h4' | 'h5' | 'h6'
      return <Tag><RenderChildren nodes={node.content} /></Tag>
    }

    case 'paragraph':
      return <p><RenderChildren nodes={node.content} /></p>

    case 'bulletList':
      return (
        <ul>
          {node.content?.map((child, i) => (
            <RenderNode key={i} node={child} />
          ))}
        </ul>
      )

    case 'orderedList':
      return (
        <ol>
          {node.content?.map((child, i) => (
            <RenderNode key={i} node={child} />
          ))}
        </ol>
      )

    case 'listItem':
      return (
        <li>
          {node.content?.map((child, i) => (
            <RenderNode key={i} node={child} />
          ))}
        </li>
      )

    case 'taskList':
      return (
        <ul className="list-none pl-0" data-type="taskList">
          {node.content?.map((child, i) => (
            <RenderNode key={i} node={child} />
          ))}
        </ul>
      )

    case 'taskItem': {
      const checked = node.attrs?.checked as boolean
      return (
        <li className="flex items-start gap-2">
          <input type="checkbox" checked={checked} readOnly className="mt-1.5" />
          <div>
            {node.content?.map((child, i) => (
              <RenderNode key={i} node={child} />
            ))}
          </div>
        </li>
      )
    }

    case 'codeBlock': {
      const language = (node.attrs?.language as string) || ''
      return (
        <pre>
          <code className={language ? `language-${language}` : undefined}>
            <RenderChildren nodes={node.content} />
          </code>
        </pre>
      )
    }

    case 'blockquote':
      return (
        <blockquote>
          {node.content?.map((child, i) => (
            <RenderNode key={i} node={child} />
          ))}
        </blockquote>
      )

    case 'calloutBlock': {
      const calloutType = (node.attrs?.type as string) || 'info'
      return (
        <div className={`callout callout-${calloutType}`}>
          {node.content?.map((child, i) => (
            <RenderNode key={i} node={child} />
          ))}
        </div>
      )
    }

    case 'horizontalRule':
      return <hr />

    case 'table':
      return (
        <table>
          <tbody>
            {node.content?.map((child, i) => (
              <RenderNode key={i} node={child} />
            ))}
          </tbody>
        </table>
      )

    case 'tableRow':
      return (
        <tr>
          {node.content?.map((child, i) => (
            <RenderNode key={i} node={child} />
          ))}
        </tr>
      )

    case 'tableCell':
      return (
        <td>
          {node.content?.map((child, i) => (
            <RenderNode key={i} node={child} />
          ))}
        </td>
      )

    case 'tableHeader':
      return (
        <th>
          {node.content?.map((child, i) => (
            <RenderNode key={i} node={child} />
          ))}
        </th>
      )

    case 'image': {
      const src = node.attrs?.src as string
      const alt = (node.attrs?.alt as string) || ''
      return <img src={src} alt={alt} />
    }

    case 'liveContextBlock': {
      const subtype = (node.attrs?.subtype as string) || 'unknown'
      const label = node.attrs?.label as string
      return (
        <div className="live-context-block">
          <span className="text-sm text-muted-foreground">
            Live context: {label || subtype}
          </span>
        </div>
      )
    }

    case 'diagramBlock': {
      const svg = node.attrs?.svg as string
      if (svg) {
        return (
          <div className="diagram-block" dangerouslySetInnerHTML={{ __html: svg }} />
        )
      }
      return (
        <div className="text-sm text-muted-foreground border border-border rounded p-2">
          [Diagram]
        </div>
      )
    }

    case 'text':
      return <RenderText node={node} />

    default:
      // Unknown node type — render children if any
      if (node.content) {
        return <>{node.content.map((child, i) => <RenderNode key={i} node={child} />)}</>
      }
      return null
  }
}

function RenderChildren({ nodes }: { nodes?: TiptapNode[] }) {
  if (!nodes) return null
  return (
    <>
      {nodes.map((node, i) => (
        <RenderNode key={i} node={node} />
      ))}
    </>
  )
}

function RenderText({ node }: { node: TiptapNode }): ReactNode {
  if (!node.text) return null

  let el: ReactNode = node.text

  if (node.marks) {
    for (const mark of node.marks) {
      switch (mark.type) {
        case 'bold':
          el = <strong>{el}</strong>
          break
        case 'italic':
          el = <em>{el}</em>
          break
        case 'strike':
          el = <s>{el}</s>
          break
        case 'code':
          el = <code>{el}</code>
          break
        case 'link':
          el = (
            <a href={mark.attrs?.href as string} target="_blank" rel="noopener noreferrer">
              {el}
            </a>
          )
          break
        case 'underline':
          el = <u>{el}</u>
          break
      }
    }
  }

  return el
}
