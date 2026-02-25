import { Server } from '@hocuspocus/server'
import { Database } from '@hocuspocus/extension-database'
import { Redis } from '@hocuspocus/extension-redis'
import { Logger } from '@hocuspocus/extension-logger'
import * as Y from 'yjs'
import pg from 'pg'
import jwt from 'jsonwebtoken'
import { Schema } from 'prosemirror-model'

const { Pool } = pg

const pool = new Pool({ connectionString: process.env.DATABASE_URL })
const SECRET = process.env.BOOKOWL_SECRET_KEY || ''
const PORT = parseInt(process.env.PORT || '1234', 10)

// Minimal ProseMirror schema matching the BookOwl Tiptap editor extensions.
// Used only for converting existing Tiptap JSON to Yjs when bootstrapping.
const pmSchema = new Schema({
  nodes: {
    doc: { content: 'block+' },
    paragraph: { content: 'inline*', group: 'block' },
    heading: { content: 'inline*', group: 'block', attrs: { level: { default: 1 } } },
    bulletList: { content: 'listItem+', group: 'block' },
    orderedList: { content: 'listItem+', group: 'block', attrs: { start: { default: 1 } } },
    listItem: { content: 'block+' },
    taskList: { content: 'taskItem+', group: 'block' },
    taskItem: { content: 'block+', attrs: { checked: { default: false } } },
    codeBlock: { content: 'text*', group: 'block', marks: '', attrs: { language: { default: null } } },
    blockquote: { content: 'block+', group: 'block' },
    horizontalRule: { group: 'block' },
    image: { group: 'block', attrs: { src: { default: null }, alt: { default: null }, title: { default: null } } },
    table: { content: 'tableRow+', group: 'block' },
    tableRow: { content: '(tableCell | tableHeader)+' },
    tableCell: { content: 'block+', attrs: { colspan: { default: 1 }, rowspan: { default: 1 } } },
    tableHeader: { content: 'block+', attrs: { colspan: { default: 1 }, rowspan: { default: 1 } } },
    calloutBlock: { content: 'block+', group: 'block', attrs: { type: { default: 'info' } } },
    liveContextBlock: { group: 'block', atom: true, attrs: { subtype: { default: 'oncall' }, rosterId: { default: null }, serviceName: { default: null }, alertId: { default: null }, label: { default: null } } },
    diagramBlock: { group: 'block', atom: true, attrs: { xml: { default: '' }, svg: { default: '' }, width: { default: null }, height: { default: null } } },
    hardBreak: { inline: true, group: 'inline' },
    text: { group: 'inline' },
  },
  marks: {
    bold: {},
    italic: {},
    strike: {},
    code: {},
    underline: {},
    link: { attrs: { href: { default: null }, target: { default: null }, rel: { default: null } } },
  },
})

// Deterministic color from user ID for cursor display.
const CURSOR_COLORS = ['#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#EC4899', '#06B6D4', '#84CC16']

function getCursorColor(userId: string): string {
  let hash = 0
  for (let i = 0; i < userId.length; i++) {
    hash = hash + userId.charCodeAt(i)
  }
  return CURSOR_COLORS[hash % CURSOR_COLORS.length]!
}

interface SessionClaims {
  sub: string
  tenant: string
  role: string
  auth_method: string
  user_id?: string
  name?: string
}

const server = Server.configure({
  port: PORT,
  address: '0.0.0.0',

  // Auth: validate bw_session JWT passed as token param.
  async onAuthenticate({ token, documentName }) {
    if (!token) throw new Error('unauthenticated')

    // In dev mode (no secret configured), accept a dev token.
    if (!SECRET && token === 'dev-token') {
      return {
        user: {
          sub: 'dev-user',
          tenant: documentName.split('/')[0],
          role: 'admin',
          auth_method: 'dev',
        } satisfies SessionClaims,
      }
    }

    if (!SECRET) throw new Error('no secret configured')

    try {
      const claims = jwt.verify(token, SECRET) as SessionClaims
      const [tenantSlug] = documentName.split('/')
      if (claims.tenant !== tenantSlug) throw new Error('wrong tenant')
      return { user: claims }
    } catch {
      throw new Error('invalid token')
    }
  },

  extensions: [
    new Logger(),

    // Redis: share document state across replicas.
    ...(process.env.REDIS_HOST
      ? [
          new Redis({
            host: process.env.REDIS_HOST,
            port: parseInt(process.env.REDIS_PORT || '6379', 10),
            options: {
              password: process.env.REDIS_PASSWORD || undefined,
            },
          }),
        ]
      : []),

    // Database: persist to PostgreSQL.
    new Database({
      // Load: try Yjs snapshot first, fall back to converting existing content JSON.
      async fetch({ documentName }) {
        const [tenantSlug, documentId] = documentName.split('/')
        if (!tenantSlug || !documentId) return null

        const schema = `tenant_${tenantSlug}`

        // Try Yjs snapshot.
        const snap = await pool.query(
          `SELECT yjs_state FROM "${schema}".document_yjs_state WHERE document_id = $1`,
          [documentId],
        )
        if (snap.rows[0]?.yjs_state) {
          return new Uint8Array(snap.rows[0].yjs_state)
        }

        // Bootstrap from existing Tiptap JSON content.
        const doc = await pool.query(
          `SELECT content FROM "${schema}".documents WHERE id = $1`,
          [documentId],
        )
        if (!doc.rows[0]) return null

        // Convert Tiptap JSON → Yjs doc using the ProseMirror schema.
        const content = doc.rows[0].content
        if (content && typeof content === 'object') {
          try {
            const { prosemirrorJSONToYDoc } = await import('y-prosemirror')
            const ydoc = prosemirrorJSONToYDoc(pmSchema, content, 'default')
            return Y.encodeStateAsUpdate(ydoc)
          } catch (err) {
            console.warn(`Failed to convert existing content for ${documentName}, starting fresh:`, err)
          }
        }
        return null
      },

      // Store: persist Yjs state, update documents.content, and create version snapshot.
      async store({ documentName, state }) {
        const [tenantSlug, documentId] = documentName.split('/')
        if (!tenantSlug || !documentId) return

        const schema = `tenant_${tenantSlug}`

        // Save Yjs binary snapshot.
        await pool.query(
          `INSERT INTO "${schema}".document_yjs_state (document_id, yjs_state, updated_at)
           VALUES ($1, $2, now())
           ON CONFLICT (document_id) DO UPDATE SET yjs_state = $2, updated_at = now()`,
          [documentId, Buffer.from(state)],
        )

        // Convert Yjs → Tiptap JSON and update documents.content.
        let content: unknown = null
        try {
          const ydoc = new Y.Doc()
          Y.applyUpdate(ydoc, state)
          const { yDocToProsemirrorJSON } = await import('y-prosemirror')
          content = yDocToProsemirrorJSON(ydoc, 'default')
        } catch (err) {
          console.error(`Failed to convert Yjs to ProseMirror JSON for ${documentName}:`, err)
          return
        }

        // Update document content and bump version.
        const result = await pool.query(
          `UPDATE "${schema}".documents
           SET content = $1, version = version + 1, updated_at = now()
           WHERE id = $2
           RETURNING version, title, doc_type, tags`,
          [JSON.stringify(content), documentId],
        )

        if (!result.rows[0]) return
        const doc = result.rows[0]

        // Extract plain text from content for content_text column.
        let contentText = ''
        try {
          contentText = extractText(content)
        } catch {
          // Non-critical — version still gets created.
        }

        // Create version snapshot directly in document_versions table.
        await pool.query(
          `INSERT INTO "${schema}".document_versions
           (document_id, version, title, content, content_text, doc_type, tags, change_summary)
           VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
          [
            documentId,
            doc.version,
            doc.title,
            JSON.stringify(content),
            contentText,
            doc.doc_type,
            doc.tags,
            'Auto-saved via collaborative editing',
          ],
        )
      },
    }),
  ],
})

// Extract plain text from Tiptap JSON for full-text search.
function extractText(node: unknown): string {
  if (!node || typeof node !== 'object') return ''
  const n = node as Record<string, unknown>
  let text = ''
  if (n.text && typeof n.text === 'string') {
    text += n.text
  }
  if (Array.isArray(n.content)) {
    for (const child of n.content) {
      text += ' ' + extractText(child)
    }
  }
  return text.trim()
}

server.listen()
console.log(`Hocuspocus collab server listening on 0.0.0.0:${PORT}`)

export { getCursorColor, CURSOR_COLORS }
