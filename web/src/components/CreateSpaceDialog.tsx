import { useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { X } from 'lucide-react'
import { api } from '@/api/client'
import type { Space } from '@/api/client'

const SPACE_ICONS = [
  '📁', '📂', '📚', '📖', '📝', '📋', '🔖', '🗂️', '📌', '💡',
  '🔧', '🛠️', '⚙️', '🚀', '🎯', '🔒', '🌐', '📊', '📈', '🧪',
  '🐛', '🔥', '⚡', '🏗️', '📦', '🎨', '💻', '🤖', '🦉', '🌙',
]

interface CreateSpaceDialogProps {
  open: boolean
  onClose: () => void
  space?: Space
}

export function CreateSpaceDialog({ open, onClose, space }: CreateSpaceDialogProps) {
  const queryClient = useQueryClient()
  const isEdit = !!space
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [icon, setIcon] = useState('')

  useEffect(() => {
    if (open && space) {
      setName(space.name)
      setDescription(space.description ?? '')
      setIcon(space.icon ?? '')
    } else if (open && !space) {
      setName('')
      setDescription('')
      setIcon('')
    }
  }, [open, space])

  const mutation = useMutation({
    mutationFn: () => {
      const slug = name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '')
      const body = {
        name,
        slug,
        description: description || undefined,
        icon: icon || undefined,
      }
      if (isEdit) {
        return api.put<Space>(`/spaces/${space.id}`, body)
      }
      return api.post<Space>('/spaces', body)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['spaces'] })
      if (isEdit) {
        queryClient.invalidateQueries({ queryKey: ['space', space.id] })
      }
      setName('')
      setDescription('')
      setIcon('')
      onClose()
    },
  })

  if (!open) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="fixed inset-0 bg-black/60" onClick={onClose} />
      <div className="relative w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-2xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold">{isEdit ? 'Edit Space' : 'Create Space'}</h2>
          <button
            onClick={onClose}
            className="rounded p-1 text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (name.trim()) mutation.mutate()
          }}
          className="space-y-4"
        >
          <div>
            <label className="mb-1 block text-sm font-medium">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Engineering Runbooks"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
              autoFocus
            />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium">Description</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Optional description"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            />
          </div>

          <div>
            <label className="mb-1 block text-sm font-medium">
              Icon {icon && <span className="ml-1 text-muted-foreground">— {icon}</span>}
            </label>
            <div className="grid grid-cols-10 gap-1">
              {SPACE_ICONS.map((emoji) => (
                <button
                  key={emoji}
                  type="button"
                  onClick={() => setIcon(icon === emoji ? '' : emoji)}
                  className={`flex h-8 w-8 items-center justify-center rounded-md text-base transition-colors ${
                    icon === emoji
                      ? 'bg-accent/20 ring-2 ring-accent'
                      : 'hover:bg-muted'
                  }`}
                >
                  {emoji}
                </button>
              ))}
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-md border border-border px-4 py-2 text-sm hover:bg-muted"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!name.trim() || mutation.isPending}
              className="rounded-md bg-accent px-4 py-2 text-sm font-medium text-accent-foreground hover:bg-accent/90 disabled:opacity-50"
            >
              {mutation.isPending ? (isEdit ? 'Saving...' : 'Creating...') : (isEdit ? 'Save' : 'Create')}
            </button>
          </div>

          {mutation.isError && (
            <p className="text-sm text-destructive">
              Failed to {isEdit ? 'update' : 'create'} space. Please try again.
            </p>
          )}
        </form>
      </div>
    </div>
  )
}
