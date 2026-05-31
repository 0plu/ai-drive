import { post } from '@/utils/network/request'
import { API_BASE_URL, API_ENDPOINTS } from '@/config/api'
import cache from '@/plugins/cache'

export interface SummarizeContext {
  fileId: string
  fileName: string
  fileType: string
}

export interface AISummaryResponse {
  summary: string
}

export interface ChatContext {
  fileId?: string
  fileName?: string
  fileType?: string
  message: string
  history?: Array<{
    role: 'user' | 'assistant'
    content: string
  }>
}

export interface ChatResponseData {
  reply: string
}

type StreamTokenHandler = (content: string) => void
interface StreamOptions {
  signal?: AbortSignal
}

interface JsonResponse<T> {
  code: number
  message: string
  data: T
}

const SUPPORTED_EXTENSIONS = ['txt', 'md', 'pdf', 'docx']
const MIME_TO_EXT: Record<string, string> = {
  'text/plain': 'txt',
  'text/markdown': 'md',
  'application/pdf': 'pdf',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document': 'docx'
}

function getFileExtension(fileName: string): string {
  const parts = fileName.split('.')
  return parts.length > 1 ? parts.pop()!.toLowerCase() : ''
}

function mimeToExt(mimeType: string): string {
  const lower = mimeType.toLowerCase()
  if (MIME_TO_EXT[lower]) return MIME_TO_EXT[lower]
  if (lower.startsWith('text/')) return 'txt'
  return ''
}

export function resolveFileType(fileName: string, mimeType?: string): string {
  const ext = getFileExtension(fileName)
  if (ext && SUPPORTED_EXTENSIONS.includes(ext)) return ext
  if (mimeType) {
    const resolved = mimeToExt(mimeType)
    if (resolved) return resolved
  }
  return ext || 'unknown'
}

export function isSupportedFileType(fileName: string, mimeType?: string): boolean {
  return SUPPORTED_EXTENSIONS.includes(resolveFileType(fileName, mimeType))
}

export function getSupportedTypes(): string {
  return SUPPORTED_EXTENSIONS.join(', ')
}

async function postSSE(
  url: string,
  payload: unknown,
  onToken: StreamTokenHandler,
  options: StreamOptions = {}
): Promise<void> {
  const token = cache.local.get('token')
  const response = await fetch(API_BASE_URL + url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    body: JSON.stringify(payload),
    signal: options.signal
  })

  if (!response.ok) {
    throw new Error(await response.text().catch(() => 'stream request failed'))
  }
  if (!response.body) {
    throw new Error('stream response body is empty')
  }

  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { value, done } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const events = buffer.split('\n\n')
    buffer = events.pop() || ''

    for (const rawEvent of events) {
      const lines = rawEvent.split('\n').map(line => line.trim()).filter(Boolean)
      const eventType = lines.find(line => line.startsWith('event:'))?.slice(6).trim()
      const dataLines = lines.filter(line => line.startsWith('data:')).map(line => line.slice(5).trim())
      if (dataLines.length === 0) continue

      const data = dataLines.join('\n')
      if (data === '[DONE]') return

      const parsed = JSON.parse(data) as { content?: string; error?: string }
      if (eventType === 'error' || parsed.error) {
        throw new Error(parsed.error || 'AI stream failed')
      }
      if (parsed.content) {
        onToken(parsed.content)
      }
    }
  }
}

/** 调用后端 AI 文件总结接口 */
export async function getAISummary(ctx: SummarizeContext): Promise<AISummaryResponse> {
  if (!ctx.fileId) {
    throw new Error('fileId is required for summarization')
  }

  const res = await post<JsonResponse<AISummaryResponse | string>>(API_ENDPOINTS.AI.SUMMARIZE, {
    fileId: ctx.fileId,
    fileName: ctx.fileName,
    fileType: resolveFileType(ctx.fileName, ctx.fileType)
  })

  if (res.code !== 200) {
    throw new Error(typeof res.data === 'string' ? res.data : res.message)
  }

  return res.data as AISummaryResponse
}

export async function streamAISummary(ctx: SummarizeContext, onToken: StreamTokenHandler): Promise<void> {
  if (!ctx.fileId) {
    throw new Error('fileId is required for summarization')
  }

  await postSSE(API_ENDPOINTS.AI.SUMMARIZE_STREAM, {
    fileId: ctx.fileId,
    fileName: ctx.fileName,
    fileType: resolveFileType(ctx.fileName, ctx.fileType)
  }, onToken)
}

/** 调用后端 AI 文件问答接口 */
export async function chatFile(ctx: ChatContext): Promise<ChatResponseData> {
  const payload: Record<string, unknown> = { message: ctx.message }

  if (ctx.fileId) {
    payload.fileId = ctx.fileId
    payload.fileName = ctx.fileName || ''
    payload.fileType = resolveFileType(ctx.fileName || '', ctx.fileType)
  }
  if (ctx.history?.length) {
    payload.history = ctx.history
  }

  const res = await post<JsonResponse<ChatResponseData | string>>(API_ENDPOINTS.AI.CHAT, payload)

  if (res.code !== 200) {
    throw new Error(typeof res.data === 'string' ? res.data : res.message)
  }

  return res.data as ChatResponseData
}

export async function streamChatFile(
  ctx: ChatContext,
  onToken: StreamTokenHandler,
  options: StreamOptions = {}
): Promise<void> {
  const payload: Record<string, unknown> = { message: ctx.message }

  if (ctx.fileId) {
    payload.fileId = ctx.fileId
    payload.fileName = ctx.fileName || ''
    payload.fileType = resolveFileType(ctx.fileName || '', ctx.fileType)
  }
  if (ctx.history?.length) {
    payload.history = ctx.history
  }

  await postSSE(API_ENDPOINTS.AI.CHAT_STREAM, payload, onToken, options)
}
