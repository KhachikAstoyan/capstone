import { apiPost, ApiError } from './api'

export type VerifyEmailStatus = 'verified' | 'resent'

export interface VerifyEmailResponse {
  status: VerifyEmailStatus
  message: string
}

const MAX_ATTEMPTS = 3

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/** POST /auth/verify-email with transient-failure retries (network / 5xx). No retry on 4xx. */
export async function verifyEmailWithRetries(
  token: string,
): Promise<VerifyEmailResponse> {
  let lastErr: unknown
  for (let attempt = 0; attempt < MAX_ATTEMPTS; attempt++) {
    try {
      return await apiPost<VerifyEmailResponse>('/auth/verify-email', {
        token,
      })
    } catch (e) {
      lastErr = e
      const retryable =
        !(e instanceof ApiError) ||
        e.status >= 500 ||
        e.status === 408 ||
        e.status === 429
      if (!retryable || attempt === MAX_ATTEMPTS - 1) {
        throw e
      }
      await delay(300 * (attempt + 1))
    }
  }
  throw lastErr
}
