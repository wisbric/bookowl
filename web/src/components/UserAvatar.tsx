import type { ProfileResponse } from '@/api/client'

const AVATAR_COLORS = [
  'bg-emerald-600',
  'bg-teal-600',
  'bg-cyan-600',
  'bg-sky-600',
  'bg-blue-600',
  'bg-indigo-600',
  'bg-violet-600',
  'bg-purple-600',
  'bg-fuchsia-600',
  'bg-pink-600',
]

function getInitials(name: string): string {
  const parts = name.trim().split(/\s+/)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
  }
  return name.slice(0, 2).toUpperCase()
}

function getColor(id: string): string {
  let hash = 0
  for (let i = 0; i < id.length; i++) {
    hash = ((hash << 5) - hash + id.charCodeAt(i)) | 0
  }
  return AVATAR_COLORS[Math.abs(hash) % AVATAR_COLORS.length]
}

interface UserAvatarProps {
  profile: ProfileResponse
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

export function UserAvatar({ profile, size = 'md', className = '' }: UserAvatarProps) {
  const initials = getInitials(profile.display_name || profile.email)
  const color = getColor(profile.id)

  const sizeClasses = {
    sm: 'h-7 w-7 text-xs',
    md: 'h-8 w-8 text-sm',
    lg: 'h-14 w-14 text-lg',
  }

  return (
    <div
      className={`flex shrink-0 items-center justify-center rounded-full font-medium text-white ${color} ${sizeClasses[size]} ${className}`}
    >
      {initials}
    </div>
  )
}
