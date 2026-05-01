import { useEffect, useRef, useState } from 'react'
import { useRouter } from '@tanstack/react-router'
import { Search, X } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import type { Tag } from '@/lib/problems'
import type { HomeSearch } from '@/routes/index'

interface ProblemFiltersProps {
  tags: Tag[]
  currentSearch: HomeSearch
}

export function ProblemFilters({ tags, currentSearch }: ProblemFiltersProps) {
  const router = useRouter()
  const { q: currentQ, difficulty: currentDifficulty, tags: currentTags = [], sort: currentSort } = currentSearch
  const [searchValue, setSearchValue] = useState(currentQ ?? '')
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Sync local search input if URL changes externally (e.g. browser back/forward).
  useEffect(() => {
    setSearchValue(currentQ ?? '')
  }, [currentQ])

  function navigate(patch: Partial<HomeSearch>) {
    const nextTags = 'tags' in patch ? patch.tags : currentTags
    router.navigate({
      to: '/',
      search: {
        q: 'q' in patch ? patch.q : currentQ,
        difficulty: 'difficulty' in patch ? patch.difficulty : currentDifficulty,
        tags: nextTags && nextTags.length > 0 ? nextTags : undefined,
        page: patch.page ?? 1,
        sort: 'sort' in patch ? patch.sort : currentSort,
      },
    })
  }

  function handleSearchChange(e: React.ChangeEvent<HTMLInputElement>) {
    const value = e.target.value
    setSearchValue(value)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      navigate({ q: value || undefined })
    }, 300)
  }

  function handleDifficultyChange(value: string) {
    navigate({ difficulty: value === 'all' ? undefined : value })
  }

  function handleSortChange(value: string) {
    navigate({ sort: value === 'default' ? undefined : value })
  }

  function toggleTag(tagName: string) {
    const next = currentTags.includes(tagName)
      ? currentTags.filter((t) => t !== tagName)
      : [...currentTags, tagName]
    navigate({ tags: next.length > 0 ? next : undefined })
  }

  function clearAll() {
    setSearchValue('')
    navigate({ q: undefined, difficulty: undefined, tags: undefined, sort: undefined })
  }

  const hasFilters =
    (currentQ && currentQ.length > 0) ||
    currentDifficulty ||
    currentTags.length > 0

  return (
    <div className="flex flex-col gap-3">
      {/* Top row: search + difficulty + sort */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-48 flex-1 sm:max-w-72">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search problems…"
            value={searchValue}
            onChange={handleSearchChange}
            className="h-9 pl-8"
          />
        </div>

        <Select
          value={currentDifficulty ?? 'all'}
          onValueChange={handleDifficultyChange}
        >
          <SelectTrigger className="h-9 w-36">
            <SelectValue placeholder="Difficulty" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All difficulties</SelectItem>
            <SelectItem value="easy">Easy</SelectItem>
            <SelectItem value="medium">Medium</SelectItem>
            <SelectItem value="hard">Hard</SelectItem>
          </SelectContent>
        </Select>

        <Select
          value={currentSort ?? 'default'}
          onValueChange={handleSortChange}
        >
          <SelectTrigger className="h-9 w-40">
            <SelectValue placeholder="Sort" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="default">Default</SelectItem>
            <SelectItem value="easiest">Easiest first</SelectItem>
            <SelectItem value="hardest">Hardest first</SelectItem>
            <SelectItem value="acceptance-high">Highest acceptance</SelectItem>
            <SelectItem value="acceptance-low">Lowest acceptance</SelectItem>
            <SelectItem value="a-z">A-Z</SelectItem>
          </SelectContent>
        </Select>

        {hasFilters && (
          <Button
            variant="ghost"
            size="sm"
            onClick={clearAll}
            className="h-9 gap-1 text-muted-foreground"
          >
            <X className="h-3.5 w-3.5" />
            Clear
          </Button>
        )}
      </div>

      {/* Tag chips */}
      {tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {tags.map((tag) => {
            const active = currentTags.includes(tag.name)
            return (
              <button
                key={tag.id}
                type="button"
                onClick={() => toggleTag(tag.name)}
                className={cn(
                  'rounded-full border px-2.5 py-0.5 text-xs font-medium transition-colors',
                  active
                    ? 'border-primary/60 bg-primary/10 text-primary font-semibold'
                    : 'border-border bg-background text-muted-foreground hover:border-primary/50 hover:text-foreground',
                )}
              >
                {tag.name}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}
