import { useEffect, useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/lib/auth'
import { ApiError } from '@/lib/api'

interface SignUpModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export default function SignUpModal({ open, onOpenChange }: SignUpModalProps) {
  const { register } = useAuth()
  const [phase, setPhase] = useState<'form' | 'verify'>('form')
  const [handle, setHandle] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open) {
      setPhase('form')
      setHandle('')
      setDisplayName('')
      setEmail('')
      setPassword('')
      setError(null)
      setLoading(false)
    }
  }, [open])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setLoading(true)
    try {
      await register(handle, email, password, displayName || undefined)
      setPhase('verify')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'Something went wrong.')
    } finally {
      setLoading(false)
    }
  }

  function handleVerifyDismiss() {
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        {phase === 'form'
          ? (
              <>
                <DialogHeader>
                  <DialogTitle>Sign up</DialogTitle>
                </DialogHeader>
                <form onSubmit={handleSubmit} className="space-y-3">
                  <Input
                    placeholder="Username"
                    autoComplete="username"
                    required
                    value={handle}
                    onChange={(e) => { setHandle(e.target.value); setError(null) }}
                  />
                  <Input
                    placeholder="Display name (optional)"
                    autoComplete="name"
                    value={displayName}
                    onChange={(e) => { setDisplayName(e.target.value) }}
                  />
                  <Input
                    type="email"
                    placeholder="Email"
                    autoComplete="email"
                    required
                    value={email}
                    onChange={(e) => { setEmail(e.target.value); setError(null) }}
                  />
                  <Input
                    type="password"
                    placeholder="Password"
                    autoComplete="new-password"
                    required
                    value={password}
                    onChange={(e) => { setPassword(e.target.value); setError(null) }}
                  />
                  {error && <p className="text-sm text-destructive">{error}</p>}
                  <Button type="submit" className="w-full" disabled={loading}>
                    {loading ? 'Creating account…' : 'Sign up'}
                  </Button>
                </form>
              </>
            )
          : (
              <>
                <DialogHeader>
                  <DialogTitle>Verify your email</DialogTitle>
                </DialogHeader>
                <p className="text-sm text-muted-foreground">
                  We sent a verification link to
                  {' '}
                  <span className="font-medium text-foreground">{email}</span>
                  . Please open it to confirm your address. You can sign in after your email is verified.
                </p>
                <Button type="button" className="w-full" onClick={handleVerifyDismiss}>
                  OK
                </Button>
              </>
            )}
      </DialogContent>
    </Dialog>
  )
}
