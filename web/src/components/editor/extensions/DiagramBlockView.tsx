import { NodeViewWrapper } from '@tiptap/react'
import type { NodeViewProps } from '@tiptap/react'
import { useState, useCallback } from 'react'
import { Pencil, Download, Maximize2 } from 'lucide-react'
import { DiagramModal } from './DiagramModal'

export function DiagramBlockView({ node, updateAttributes, selected, editor }: NodeViewProps) {
  const { xml, svg } = node.attrs as { xml: string; svg: string; width: number | null; height: number | null }
  const [modalOpen, setModalOpen] = useState(false)
  const isEmpty = !xml

  const handleSave = useCallback(
    (newXml: string, newSvg: string) => {
      updateAttributes({ xml: newXml, svg: newSvg })
      setModalOpen(false)
    },
    [updateAttributes],
  )

  const handleClick = () => {
    if (editor.isEditable) setModalOpen(true)
  }

  const downloadSvg = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (!svg) return
    const blob = new Blob([svg], { type: 'image/svg+xml' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'diagram.svg'
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <NodeViewWrapper>
      <div
        className={`diagram-block group relative my-4 rounded-lg border ${
          selected ? 'border-accent' : 'border-border'
        }`}
      >
        {isEmpty ? (
          <div
            className="flex h-32 cursor-pointer items-center justify-center rounded-lg border-2 border-dashed border-border text-sm text-muted-foreground hover:border-accent hover:text-accent"
            onClick={handleClick}
          >
            Click to create diagram · draw.io
          </div>
        ) : svg ? (
          <div
            className={editor.isEditable ? 'cursor-pointer' : ''}
            onClick={handleClick}
            dangerouslySetInnerHTML={{ __html: svg }}
          />
        ) : (
          <div className="flex h-24 items-center justify-center gap-2 text-sm text-destructive">
            Diagram could not be rendered
            {editor.isEditable && (
              <button className="underline" onClick={handleClick}>
                Edit
              </button>
            )}
          </div>
        )}

        {/* Hover toolbar */}
        {!isEmpty && svg && (
          <div className="absolute right-2 top-2 flex gap-1 rounded-md border border-border bg-card p-1 opacity-0 shadow-md transition-opacity group-hover:opacity-100">
            {editor.isEditable && (
              <button
                onClick={handleClick}
                className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
                title="Edit diagram"
              >
                <Pencil className="h-3.5 w-3.5" />
              </button>
            )}
            <button
              onClick={downloadSvg}
              className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
              title="Download SVG"
            >
              <Download className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation()
                const win = window.open('', '_blank')
                if (win) {
                  win.document.write(svg)
                  win.document.close()
                }
              }}
              className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
              title="Fullscreen"
            >
              <Maximize2 className="h-3.5 w-3.5" />
            </button>
          </div>
        )}
      </div>

      {modalOpen && (
        <DiagramModal xml={xml} onSave={handleSave} onClose={() => setModalOpen(false)} />
      )}
    </NodeViewWrapper>
  )
}
