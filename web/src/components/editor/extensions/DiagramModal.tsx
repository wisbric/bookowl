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

// Use local draw.io in production (Docker), hosted embed in dev.
const DRAWIO_PARAMS = 'embed=1&proto=json&spin=1&libraries=1&lang=en'
const DRAWIO_LOCAL = `/drawio/index.html?${DRAWIO_PARAMS}`
const DRAWIO_HOSTED = `https://embed.diagrams.net?${DRAWIO_PARAMS}`

function getDrawioUrl(): string {
  // In production (served by nginx), /drawio/ is available.
  // In dev (Vite), use the hosted embed.
  if (import.meta.env.DEV) return DRAWIO_HOSTED
  return DRAWIO_LOCAL
}

function getDrawioOrigin(url: string): string {
  try {
    return new URL(url).origin
  } catch {
    return window.location.origin
  }
}

export function DiagramModal({ xml, onSave, onClose }: DiagramModalProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null)
  const pendingExport = useRef<string | null>(null)
  const drawioUrl = getDrawioUrl()
  const drawioOrigin = getDrawioOrigin(drawioUrl)

  const handleMessage = useCallback(
    (event: MessageEvent) => {
      if (event.origin !== drawioOrigin) return

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
          drawioOrigin,
        )
      }

      if (msg.event === 'save' && msg.xml) {
        // Save received — now request SVG export
        pendingExport.current = sanitizeDiagramXml(msg.xml)
        iframeRef.current?.contentWindow?.postMessage(
          JSON.stringify({ action: 'export', format: 'svg' }),
          drawioOrigin,
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
    [xml, onSave, onClose, drawioOrigin],
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
          src={drawioUrl}
          sandbox="allow-scripts allow-same-origin allow-forms allow-popups"
          className="h-full w-full border-0"
          title="draw.io diagram editor"
        />
      </div>
    </div>
  )
}
