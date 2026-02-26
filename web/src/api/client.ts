const API_BASE = '/api/v1'

type TokenProvider = () => Promise<string | null>

let _tokenProvider: TokenProvider | null = null
let _sessionMode = false

export function setTokenProvider(provider: TokenProvider) {
  _tokenProvider = provider
}

export function setSessionMode(enabled: boolean) {
  _sessionMode = enabled
}

class ApiError extends Error {
  constructor(
    public status: number,
    public body: { error: string; message?: string },
  ) {
    super(body.message || body.error)
    this.name = 'ApiError'
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    'Accept': 'application/json',
    ...(init?.headers as Record<string, string>),
  }

  // Attach auth: Bearer token (OIDC), session cookie (automatic), or dev fallback.
  if (_sessionMode) {
    // Cookie-based auth — browser sends wisbric_session cookie automatically.
  } else if (_tokenProvider) {
    const token = await _tokenProvider()
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
  } else {
    headers['X-Tenant-Slug'] = 'acme'
  }

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers, credentials: 'same-origin' })

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new ApiError(res.status, body)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}

function get<T>(path: string) {
  return request<T>(path)
}

function post<T>(path: string, body: unknown) {
  return request<T>(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

function put<T>(path: string, body: unknown) {
  return request<T>(path, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

function del(path: string) {
  return request<void>(path, { method: 'DELETE' })
}

async function upload<T>(path: string, file: File): Promise<T> {
  const formData = new FormData()
  formData.append('file', file)

  const headers: Record<string, string> = {
    'Accept': 'application/json',
  }

  // Attach auth: Bearer token (OIDC), session cookie (automatic), or dev fallback.
  if (_sessionMode) {
    // Cookie-based auth — browser sends wisbric_session cookie automatically.
  } else if (_tokenProvider) {
    const token = await _tokenProvider()
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
  } else {
    headers['X-Tenant-Slug'] = 'acme'
  }

  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers,
    body: formData,
    credentials: 'same-origin',
  })

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new ApiError(res.status, body)
  }

  return res.json()
}

export const api = { get, post, put, del, upload, ApiError }

// --- Types ---

export interface Space {
  id: string
  name: string
  slug: string
  description?: string
  icon?: string
  is_private: boolean
  created_at: string
  updated_at: string
}

export interface Collection {
  id: string
  space_id: string
  parent_id?: string
  name: string
  slug: string
  icon?: string
  position: number
}

export interface Document {
  id: string
  space_id: string
  collection_id?: string
  title: string
  slug: string
  content: Record<string, unknown>
  doc_type: string
  status: string
  tags: string[]
  icon?: string
  position: number
  word_count: number
  version: number
  created_at: string
  updated_at: string
}

export interface ListResponse<T> {
  items: T[]
  total: number
  limit: number
  offset: number
}

export interface SearchResult {
  id: string
  title: string
  doc_type: string
  slug: string
  space_id: string
  space_name: string
  space_slug: string
  rank: number
  title_highlight: string
  content_highlight: string
}

export interface ImageUploadResponse {
  id: string
  url: string
  filename: string
  size: number
  content_type: string
}

export interface DocumentVersion {
  id: string
  document_id: string
  version: number
  title: string
  content: Record<string, unknown>
  change_summary: string
  created_by: string
  created_at: string
}

export interface AdminConfig {
  nightowl_api_url: string
  nightowl_api_key: string
}

export interface OIDCConfig {
  issuer_url: string
  client_id: string
  client_secret: string
  allowed_groups: string[]
  admin_groups: string[]
  editor_groups: string[]
  enabled: boolean
  last_tested_at?: string
  last_test_result?: string
}

export interface OIDCTestDetails {
  discovery_ok: boolean
  jwks_uri?: string
  jwks_ok: boolean
  client_credentials_ok: boolean
  supported_scopes?: string[]
  has_groups_scope: boolean
}

export interface OIDCTestResponse {
  ok: boolean
  latency_ms: number
  issuer?: string
  error?: string
  details: OIDCTestDetails
}

export interface HealthStatus {
  status: string
  checks: Record<string, string>
}

export interface SpaceTree {
  collections: TreeCollection[]
  documents: TreeDoc[]
}

export interface TreeCollection {
  collection_id: string
  collection_name: string
  collection_slug: string
  parent_id?: string
  position: number
  icon?: string
  documents: TreeDoc[]
}

export interface TreeDoc {
  id: string
  title: string
  slug: string
  doc_type: string
  status: string
  position: number
  icon?: string
}

export interface ProfileResponse {
  id: string
  email: string
  display_name: string
  timezone: string
  role: string
  auth_method: string
  created_at?: string
}

export interface PersonalToken {
  id: string
  name: string
  prefix: string
  created_at: string
  last_used_at?: string
  expires_at?: string
}

export interface PersonalTokenCreated extends PersonalToken {
  plaintext: string
}

export interface Template {
  id: string
  name: string
  description?: string
  doc_type: string
  content: Record<string, unknown>
  icon?: string
  is_system: boolean
  is_global: boolean
  space_id?: string
  created_by?: string
  sort_order: number
  created_at: string
  updated_at: string
}

export interface CommentAuthor {
  id: string
  display_name: string
  initials: string
  avatar_url?: string
}

export interface Comment {
  id: string
  document_id: string
  parent_id: string | null
  author: CommentAuthor
  body: string
  body_rendered: string
  is_resolved: boolean
  is_deleted: boolean
  edited_at: string | null
  created_at: string
  replies: Comment[]
}

export interface CommentCount {
  count: number
}

export interface Notification {
  id: string
  type: string
  document_id: string | null
  comment_id: string | null
  actor: CommentAuthor | null
  is_read: boolean
  created_at: string
}

export interface NotificationCount {
  count: number
}
