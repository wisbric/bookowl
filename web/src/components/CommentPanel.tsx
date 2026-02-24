import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { X, MessageSquare, Check, ChevronDown, ChevronRight, Reply, Pencil, Trash2 } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { api } from '@/api/client'
import type { Comment, CommentAuthor } from '@/api/client'
import { useAuth } from '@/auth/auth-provider'

interface CommentPanelProps {
  documentId: string
  onClose: () => void
}

export function CommentPanel({ documentId, onClose }: CommentPanelProps) {
  const queryClient = useQueryClient()
  const { profile } = useAuth()
  const [newComment, setNewComment] = useState('')
  const [showResolved, setShowResolved] = useState(false)

  const { data: comments, isLoading } = useQuery({
    queryKey: ['comments', documentId, showResolved],
    queryFn: () =>
      api.get<Comment[]>(
        `/documents/${documentId}/comments?include_resolved=${showResolved}`,
      ),
  })

  const createMutation = useMutation({
    mutationFn: (body: string) =>
      api.post<Comment>(`/documents/${documentId}/comments`, { body }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', documentId] })
      queryClient.invalidateQueries({ queryKey: ['comment-count', documentId] })
      setNewComment('')
    },
  })

  const handleSubmit = () => {
    const body = newComment.trim()
    if (body) createMutation.mutate(body)
  }

  return (
    <div className="flex h-full flex-col border-l border-border bg-sidebar">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h3 className="flex items-center gap-2 text-sm font-semibold text-sidebar-foreground">
          <MessageSquare className="h-4 w-4" />
          Comments {comments && comments.length > 0 && `(${comments.length})`}
        </h3>
        <button
          onClick={onClose}
          className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Toggle resolved */}
      <div className="border-b border-border px-4 py-2">
        <label className="flex items-center gap-2 text-xs text-muted-foreground">
          <input
            type="checkbox"
            checked={showResolved}
            onChange={(e) => setShowResolved(e.target.checked)}
            className="accent-accent"
          />
          Show resolved threads
        </label>
      </div>

      {/* Comment list */}
      <div className="flex-1 overflow-y-auto">
        {isLoading && (
          <div className="px-4 py-6 text-center text-sm text-muted-foreground">
            Loading comments...
          </div>
        )}

        {comments?.map((comment) => (
          <CommentThread
            key={comment.id}
            comment={comment}
            documentId={documentId}
            currentUserId={profile?.id}
          />
        ))}

        {comments?.length === 0 && !isLoading && (
          <div className="px-4 py-6 text-center text-sm text-muted-foreground">
            No comments yet. Start a discussion.
          </div>
        )}
      </div>

      {/* New comment input */}
      <div className="border-t border-border p-4">
        <textarea
          value={newComment}
          onChange={(e) => setNewComment(e.target.value)}
          placeholder="Add a comment..."
          rows={3}
          className="w-full resize-none rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
              handleSubmit()
            }
          }}
        />
        <div className="mt-2 flex items-center justify-between">
          <span className="text-xs text-muted-foreground">
            Markdown supported · @mention users
          </span>
          <button
            onClick={handleSubmit}
            disabled={!newComment.trim() || createMutation.isPending}
            className="rounded-md bg-accent px-3 py-1.5 text-xs font-medium text-accent-foreground hover:bg-accent/90 disabled:opacity-50"
          >
            {createMutation.isPending ? 'Posting...' : 'Post comment'}
          </button>
        </div>
      </div>
    </div>
  )
}

function CommentThread({
  comment,
  documentId,
  currentUserId,
}: {
  comment: Comment
  documentId: string
  currentUserId?: string
}) {
  const queryClient = useQueryClient()
  const [expanded, setExpanded] = useState(!comment.is_resolved)
  const [showReply, setShowReply] = useState(false)
  const [replyText, setReplyText] = useState('')
  const [editing, setEditing] = useState(false)
  const [editText, setEditText] = useState(comment.body)

  const replyMutation = useMutation({
    mutationFn: (body: string) =>
      api.post<Comment>(
        `/documents/${documentId}/comments/${comment.id}/replies`,
        { body },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', documentId] })
      queryClient.invalidateQueries({ queryKey: ['comment-count', documentId] })
      setReplyText('')
      setShowReply(false)
    },
  })

  const editMutation = useMutation({
    mutationFn: (body: string) =>
      api.put<Comment>(
        `/documents/${documentId}/comments/${comment.id}`,
        { body },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', documentId] })
      setEditing(false)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () =>
      api.del(`/documents/${documentId}/comments/${comment.id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', documentId] })
      queryClient.invalidateQueries({ queryKey: ['comment-count', documentId] })
    },
  })

  const resolveMutation = useMutation({
    mutationFn: () =>
      api.post(`/documents/${documentId}/comments/${comment.id}/resolve`, {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', documentId] })
    },
  })

  const unresolveMutation = useMutation({
    mutationFn: () =>
      api.post(`/documents/${documentId}/comments/${comment.id}/unresolve`, {}),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', documentId] })
    },
  })

  const isAuthor = currentUserId === comment.author.id
  const isWithinEditWindow =
    new Date().getTime() - new Date(comment.created_at).getTime() < 15 * 60 * 1000

  // Resolved collapsed view.
  if (comment.is_resolved && !expanded) {
    return (
      <div className="border-b border-border px-4 py-3">
        <button
          onClick={() => setExpanded(true)}
          className="flex w-full items-center gap-2 text-sm text-muted-foreground hover:text-foreground"
        >
          <Check className="h-3.5 w-3.5 text-accent" />
          <span>Resolved</span>
          <span className="text-xs">
            · {comment.replies?.length || 0} {comment.replies?.length === 1 ? 'reply' : 'replies'}
          </span>
          <ChevronRight className="ml-auto h-3.5 w-3.5" />
        </button>
      </div>
    )
  }

  return (
    <div className={`border-b border-border ${comment.is_resolved ? 'opacity-60' : ''}`}>
      {/* Collapse toggle for resolved */}
      {comment.is_resolved && (
        <button
          onClick={() => setExpanded(false)}
          className="flex w-full items-center gap-1 px-4 pt-2 text-xs text-accent"
        >
          <Check className="h-3 w-3" />
          Resolved
          <ChevronDown className="ml-1 h-3 w-3" />
        </button>
      )}

      {/* Main comment */}
      <div className="px-4 py-3">
        <div className="flex items-start gap-2">
          <AuthorAvatar author={comment.author} />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-foreground">
                {comment.author.display_name}
              </span>
              <span className="text-xs text-muted-foreground">
                {formatDistanceToNow(new Date(comment.created_at))} ago
                {comment.edited_at && ' (edited)'}
              </span>
            </div>

            {editing ? (
              <div className="mt-2">
                <textarea
                  value={editText}
                  onChange={(e) => setEditText(e.target.value)}
                  rows={2}
                  className="w-full resize-none rounded-md border border-border bg-background px-2 py-1.5 text-sm focus:border-accent focus:outline-none"
                />
                <div className="mt-1 flex gap-2">
                  <button
                    onClick={() => editMutation.mutate(editText)}
                    disabled={!editText.trim() || editMutation.isPending}
                    className="rounded bg-accent px-2 py-1 text-xs font-medium text-accent-foreground disabled:opacity-50"
                  >
                    Save
                  </button>
                  <button
                    onClick={() => { setEditing(false); setEditText(comment.body) }}
                    className="rounded px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
                  >
                    Cancel
                  </button>
                </div>
              </div>
            ) : (
              <div
                className="mt-1 text-sm text-foreground [&_a]:text-accent [&_a]:underline [&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_code]:text-xs [&_pre]:my-1 [&_pre]:rounded [&_pre]:bg-muted [&_pre]:p-2 [&_pre]:text-xs"
                dangerouslySetInnerHTML={{ __html: comment.body_rendered }}
              />
            )}

            {/* Actions */}
            {!editing && !comment.is_deleted && (
              <div className="mt-2 flex items-center gap-3">
                <button
                  onClick={() => setShowReply(!showReply)}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                >
                  <Reply className="h-3 w-3" />
                  Reply
                </button>
                {isAuthor && isWithinEditWindow && (
                  <button
                    onClick={() => { setEditing(true); setEditText(comment.body) }}
                    className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                  >
                    <Pencil className="h-3 w-3" />
                    Edit
                  </button>
                )}
                {isAuthor && (
                  <button
                    onClick={() => deleteMutation.mutate()}
                    className="flex items-center gap-1 text-xs text-muted-foreground hover:text-red-400"
                  >
                    <Trash2 className="h-3 w-3" />
                    Delete
                  </button>
                )}
                {!comment.is_resolved ? (
                  <button
                    onClick={() => resolveMutation.mutate()}
                    className="flex items-center gap-1 text-xs text-muted-foreground hover:text-accent"
                  >
                    <Check className="h-3 w-3" />
                    Resolve
                  </button>
                ) : (
                  <button
                    onClick={() => unresolveMutation.mutate()}
                    className="text-xs text-muted-foreground hover:text-foreground"
                  >
                    Unresolve
                  </button>
                )}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Replies */}
      {comment.replies?.map((reply) => (
        <ReplyComment
          key={reply.id}
          reply={reply}
          documentId={documentId}
          currentUserId={currentUserId}
        />
      ))}

      {/* Reply input */}
      {showReply && (
        <div className="ml-10 border-t border-border/50 px-4 py-2">
          <textarea
            value={replyText}
            onChange={(e) => setReplyText(e.target.value)}
            placeholder="Write a reply..."
            rows={2}
            autoFocus
            className="w-full resize-none rounded-md border border-border bg-background px-2 py-1.5 text-sm focus:border-accent focus:outline-none"
            onKeyDown={(e) => {
              if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                const body = replyText.trim()
                if (body) replyMutation.mutate(body)
              }
            }}
          />
          <div className="mt-1 flex gap-2">
            <button
              onClick={() => {
                const body = replyText.trim()
                if (body) replyMutation.mutate(body)
              }}
              disabled={!replyText.trim() || replyMutation.isPending}
              className="rounded bg-accent px-2 py-1 text-xs font-medium text-accent-foreground disabled:opacity-50"
            >
              {replyMutation.isPending ? 'Posting...' : 'Reply'}
            </button>
            <button
              onClick={() => { setShowReply(false); setReplyText('') }}
              className="rounded px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function ReplyComment({
  reply,
  documentId,
  currentUserId,
}: {
  reply: Comment
  documentId: string
  currentUserId?: string
}) {
  const queryClient = useQueryClient()
  const [editing, setEditing] = useState(false)
  const [editText, setEditText] = useState(reply.body)

  const isAuthor = currentUserId === reply.author.id
  const isWithinEditWindow =
    new Date().getTime() - new Date(reply.created_at).getTime() < 15 * 60 * 1000

  const editMutation = useMutation({
    mutationFn: (body: string) =>
      api.put<Comment>(
        `/documents/${documentId}/comments/${reply.id}`,
        { body },
      ),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', documentId] })
      setEditing(false)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () =>
      api.del(`/documents/${documentId}/comments/${reply.id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', documentId] })
      queryClient.invalidateQueries({ queryKey: ['comment-count', documentId] })
    },
  })

  return (
    <div className="ml-10 border-t border-border/50 px-4 py-2">
      <div className="flex items-start gap-2">
        <AuthorAvatar author={reply.author} size="sm" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-foreground">
              {reply.author.display_name}
            </span>
            <span className="text-xs text-muted-foreground">
              {formatDistanceToNow(new Date(reply.created_at))} ago
              {reply.edited_at && ' (edited)'}
            </span>
          </div>

          {editing ? (
            <div className="mt-1">
              <textarea
                value={editText}
                onChange={(e) => setEditText(e.target.value)}
                rows={2}
                className="w-full resize-none rounded-md border border-border bg-background px-2 py-1.5 text-sm focus:border-accent focus:outline-none"
              />
              <div className="mt-1 flex gap-2">
                <button
                  onClick={() => editMutation.mutate(editText)}
                  disabled={!editText.trim() || editMutation.isPending}
                  className="rounded bg-accent px-2 py-1 text-xs font-medium text-accent-foreground disabled:opacity-50"
                >
                  Save
                </button>
                <button
                  onClick={() => { setEditing(false); setEditText(reply.body) }}
                  className="rounded px-2 py-1 text-xs text-muted-foreground hover:text-foreground"
                >
                  Cancel
                </button>
              </div>
            </div>
          ) : (
            <div
              className="mt-0.5 text-sm text-foreground [&_a]:text-accent [&_a]:underline [&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_code]:text-xs"
              dangerouslySetInnerHTML={{ __html: reply.body_rendered }}
            />
          )}

          {!editing && !reply.is_deleted && (
            <div className="mt-1 flex items-center gap-3">
              {isAuthor && isWithinEditWindow && (
                <button
                  onClick={() => { setEditing(true); setEditText(reply.body) }}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
                >
                  <Pencil className="h-3 w-3" />
                  Edit
                </button>
              )}
              {isAuthor && (
                <button
                  onClick={() => deleteMutation.mutate()}
                  className="flex items-center gap-1 text-xs text-muted-foreground hover:text-red-400"
                >
                  <Trash2 className="h-3 w-3" />
                  Delete
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

const AVATAR_COLORS = [
  'bg-emerald-600', 'bg-teal-600', 'bg-cyan-600', 'bg-sky-600',
  'bg-blue-600', 'bg-indigo-600', 'bg-violet-600', 'bg-purple-600',
]

function AuthorAvatar({ author, size = 'md' }: { author: CommentAuthor; size?: 'sm' | 'md' }) {
  const hash = author.id.split('').reduce((a, c) => ((a << 5) - a + c.charCodeAt(0)) | 0, 0)
  const color = AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length]
  const sizeClass = size === 'sm' ? 'h-6 w-6 text-[10px]' : 'h-7 w-7 text-xs'

  return (
    <div className={`flex shrink-0 items-center justify-center rounded-full font-medium text-white ${color} ${sizeClass}`}>
      {author.initials}
    </div>
  )
}
