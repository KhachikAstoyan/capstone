import { apiGet, apiPost } from './api'

export interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  created_at: string
}

export function getHintHistory(
  problemId: string,
): Promise<{ messages: ChatMessage[] }> {
  return apiGet(`/problems/${encodeURIComponent(problemId)}/hint/history`)
}

export function sendHint(
  problemId: string,
  body: { language_key: string; code: string; message: string },
): Promise<{ user_message: ChatMessage; assistant_message: ChatMessage }> {
  return apiPost(
    `/problems/${encodeURIComponent(problemId)}/hint`,
    body,
  )
}
