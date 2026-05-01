import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import type { Difficulty } from '@/lib/problems'

const config: Record<Difficulty, { label: string; badgeClass: string; dotClass: string }> = {
  easy: {
    label: 'Easy',
    badgeClass: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300',
    dotClass: 'bg-emerald-500',
  },
  medium: {
    label: 'Medium',
    badgeClass: 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
    dotClass: 'bg-amber-500',
  },
  hard: {
    label: 'Hard',
    badgeClass: 'bg-rose-100 text-rose-700 dark:bg-rose-500/15 dark:text-rose-300',
    dotClass: 'bg-rose-500',
  },
}

interface DifficultyBadgeProps {
  difficulty: Difficulty
  className?: string
}

export function DifficultyBadge({ difficulty, className }: DifficultyBadgeProps) {
  const { label, badgeClass, dotClass } = config[difficulty]
  return (
    <Badge
      variant="secondary"
      className={cn('gap-1.5 text-xs font-medium', badgeClass, className)}
    >
      <span className={cn('inline-block h-1.5 w-1.5 rounded-full', dotClass)} />
      {label}
    </Badge>
  )
}
