import { parseSSEStream } from '@/utils/stream'

const AI_API_URL = import.meta.env.VITE_AI_API_URL || ''
const AI_API_KEY = import.meta.env.VITE_AI_API_KEY || ''
const AI_MODEL = import.meta.env.VITE_AI_MODEL || 'gpt-3.5-turbo'
const AI_SYSTEM_PROMPT =
  import.meta.env.VITE_AI_SYSTEM_PROMPT ||
  '你是 MyObj 网盘的智能助手，可以帮助用户管理文件、分析存储、搜索内容。请用简洁专业的中文回复。'

export interface ChatMessage {
  role: 'system' | 'user' | 'assistant'
  content: string
}

export async function* streamChat(messages: ChatMessage[]): AsyncGenerator<string, void, undefined> {
  const response = await fetch(`${AI_API_URL}/v1/chat/completions`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${AI_API_KEY}`
    },
    body: JSON.stringify({
      model: AI_MODEL,
      messages,
      stream: true,
      max_tokens: 1024
    })
  })

  if (!response.ok) {
    const err = await response.text().catch(() => 'Unknown error')
    throw new Error(`AI API error ${response.status}: ${err}`)
  }

  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error('Stream not supported')
  }

  yield* parseSSEStream(reader)
}

export function buildChatMessages(history: { role: string; content: string }[]): ChatMessage[] {
  return [
    { role: 'system', content: AI_SYSTEM_PROMPT },
    ...history.map(m => ({
      role: m.role === 'ai' ? ('assistant' as const) : (m.role as 'user'),
      content: m.content
    }))
  ]
}
