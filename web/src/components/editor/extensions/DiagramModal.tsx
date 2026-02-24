import { useEffect, useRef, useCallback } from 'react'

interface DiagramModalProps {
  xml: string
  onSave: (xml: string, svg: string) => void
  onClose: () => void
}

function sanitizeDiagramXml(xml: string): string {
  const parser = new DOMParser()
  const doc = parser.parseFromString(xml, 'text/xml')
  doc.querySelectorAll('script').forEach((el) => el.remove())
  doc.querySelectorAll('[href^="javascript:"]').forEach((el) => el.removeAttribute('href'))
  doc.querySelectorAll('[xlink\\:href^="javascript:"]').forEach((el) =>
    el.removeAttribute('xlink:href'),
  )
  return new XMLSerializer().serializeToString(doc)
}

function sanitizeSvg(svg: string): string {
  const parser = new DOMParser()
  const doc = parser.parseFromString(svg, 'image/svg+xml')
  doc.querySelectorAll('script').forEach((el) => el.remove())
  doc.querySelectorAll('[href^="javascript:"]').forEach((el) => el.removeAttribute('href'))
  doc.querySelectorAll('[xlink\\:href^="javascript:"]').forEach((el) =>
    el.removeAttribute('xlink:href'),
  )
  // Remove event handlers
  const allElements = doc.querySelectorAll('*')
  allElements.forEach((el) => {
    for (const attr of Array.from(el.attributes)) {
      if (attr.name.startsWith('on')) {
        el.removeAttribute(attr.name)
      }
    }
  })
  return new XMLSerializer().serializeToString(doc)
}

export function DiagramModal({ xml, onSave, onClose }: DiagramModalProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const pendingExport = useRef<string | null>(null)

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return

      let msg: { event?: string; xml?: string; data?: string }
      try {
        msg = typeof event.data === 'string' ? JSON.parse(event.data) : event.data
      } catch {
        return
      }

      if (msg.event === 'init') {
        // draw.io is ready — load the diagram
        iframeRef.current?.contentWindow?.postMessage(
          JSON.stringify({ action: 'load', xml: xml || '' }),
          window.location.origin,
        )
      }

      if (msg.event === 'save' && msg.xml) {
        // Save received — now request SVG export
        pendingExport.current = sanitizeDiagramXml(msg.xml)
        iframeRef.current?.contentWindow?.postMessage(
          JSON.stringify({ action: 'export', format: 'svg' }),
          window.location.origin,
        )
      }

      if (msg.event === 'export' && msg.data && pendingExport.current) {
        // SVG export received — save both xml and svg
        const cleanXml = pendingExport.current
        pendingExport.current = null
        // data is an SVG string (or data URI); extract raw SVG
        let svgStr = msg.data
        if (svgStr.startsWith('data:image/svg+xml;base64,')) {
          svgStr = atob(svgStr.replace('data:image/svg+xml;base64,', ''))
        }
        onSave(cleanXml, sanitizeSvg(svgStr))
      }

      if (msg.event === 'exit') {
        onClose()
      }
    },
    [xml, onSave, onClose],
  )

  useEffect(() => {
    window.addEventListener('message', handleMessage)
    return () => window.removeEventListener('message', handleMessage)
  }, [handleMessage])

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80">
      <div className="relative h-[90vh] w-[95vw] overflow-hidden rounded-xl border border-border bg-background shadow-2xl">
        <iframe
          ref={iframeRef}
          src="/drawio/index.html?embed=1&proto=json&spin=1&libraries=1&lang=en"
          sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
          className="h-full w-full border-0"
          title="draw.io diagram editor"
        />
      </div>
    </div>
  )
}
