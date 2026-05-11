import React, { useEffect, useRef, useState } from 'react'
import { Loader2, Send } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { ScrollArea } from '@/components/ui/scroll-area'
import { ApiError } from '@/lib/api'
import { getHintHistory, sendHint, type ChatMessage } from '@/lib/hints'

interface HintChatProps {
  problemId: string
  code: string
  languageKey: string
}

export function HintChat({ problemId, code, languageKey }: HintChatProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [initialLoading, setInitialLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let cancelled = false
    setInitialLoading(true)
    getHintHistory(problemId)
      .then((data) => {
        if (!cancelled) setMessages(data.messages ?? [])
      })
      .catch(() => {
        // non-fatal: just start with empty history
      })
      .finally(() => {
        if (!cancelled) setInitialLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [problemId])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, loading])

  async function handleSend() {
    const text = input.trim()
    if (!text || loading) return

    const tempId = `temp-${Date.now()}`
    const optimistic: ChatMessage = {
      id: tempId,
      role: 'user',
      content: text,
      created_at: new Date().toISOString(),
    }

    setMessages((prev) => [...prev, optimistic])
    setInput('')
    setLoading(true)
    setError(null)

    try {
      const resp = await sendHint(problemId, {
        language_key: languageKey,
        code,
        message: text,
      })
      setMessages((prev) => [
        ...prev.filter((m) => m.id !== tempId),
        resp.user_message,
        resp.assistant_message,
      ])
    } catch (err) {
      setMessages((prev) => prev.filter((m) => m.id !== tempId))
      setError(err instanceof ApiError ? err.message : 'Failed to get hint.')
    } finally {
      setLoading(false)
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void handleSend()
    }
  }

  return (
    <div className="flex h-full flex-col">
      <ScrollArea className="min-h-0 flex-1 px-4 py-3">
        {initialLoading ? (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Loader2 className="size-3 animate-spin" />
            Loading history…
          </div>
        ) : messages.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            Ask for a hint. The assistant knows the problem and your current
            code.
          </p>
        ) : null}

        <div className="space-y-3">
          {messages.map((msg) => (
            <div
              key={msg.id}
              className={
                msg.role === 'user' ? 'flex justify-end' : 'flex justify-start'
              }
            >
              <div
                className={`max-w-[85%] rounded-lg px-3 py-2 text-xs leading-relaxed whitespace-pre-wrap ${
                  msg.role === 'user'
                    ? 'bg-primary text-primary-foreground'
                    : 'bg-muted text-foreground'
                }`}
              >
                {msg.content}
              </div>
            </div>
          ))}

          {loading && (
            <div className="flex justify-start">
              <div className="rounded-lg bg-muted px-3 py-2">
                <Loader2 className="size-3.5 animate-spin text-muted-foreground" />
              </div>
            </div>
          )}

          {error && <p className="text-xs text-destructive">{error}</p>}

          <div ref={bottomRef} />
        </div>
      </ScrollArea>

      <div className="shrink-0 border-t border-border p-3">
        <div className="flex gap-2">
          <Textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask for a hint… (Enter to send)"
            className="min-h-0 resize-none text-xs"
            rows={2}
            disabled={loading || initialLoading}
          />
          <Button
            size="icon"
            className="h-auto w-9 shrink-0 self-end"
            onClick={() => void handleSend()}
            disabled={loading || initialLoading || !input.trim()}
          >
            <Send className="size-3.5" />
          </Button>
        </div>
      </div>
    </div>
  )
}
