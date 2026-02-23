const API_BASE = '/api/v1'

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

  // Dev mode: include tenant slug header.
  headers['X-Tenant-Slug'] = 'acme'

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers })

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

export const api = { get, post, put, del, ApiError }

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

export interface TreeNode {
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
