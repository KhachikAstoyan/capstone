import { useEffect, useRef } from 'react'
import { toast } from 'sonner'
import { verifyEmailWithRetries } from '@/lib/verify-email'
import { ApiError } from '@/lib/api'
import { useAuth } from '@/lib/auth'

/**
 * On first load, if the URL contains ?token= (email verification link),
 * calls the API, strips the param, shows toasts, and refreshes session when verified.
 */
export function EmailVerificationOnLoad() {
  const { refreshSession } = useAuth()
  const ran = useRef(false)

  useEffect(() => {
    if (ran.current) return
    if (typeof window === 'undefined') return

    const url = new URL(window.location.href)
    const token = url.searchParams.get('token')
    if (!token) return

    ran.current = true

    url.searchParams.delete('token')
    const qs = url.searchParams.toString()
    const path = url.pathname + (qs ? `?${qs}` : '') + url.hash
    window.history.replaceState({}, '', path)

    let cancelled = false

    void (async () => {
      try {
        const res = await verifyEmailWithRetries(token)
        if (cancelled) return
        if (res.status === 'verified') {
          toast.success(res.message)
          await refreshSession()
        } else {
          toast.info(res.message)
        }
      } catch (e) {
        if (cancelled) return
        const msg =
          e instanceof ApiError
            ? e.message
            : 'Could not verify email. Please try again later.'
        toast.error(msg)
      }
    })()

    return () => {
      cancelled = true
    }
  }, [refreshSession])

  return null
}
