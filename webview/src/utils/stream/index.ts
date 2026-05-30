/**
 * 流式输出工具
 *
 * 支持两种模式：
 * 1. simulateStream  — mock 模拟逐字输出（开发/演示用）
 * 2. parseSSEStream  — 真实 SSE 流解析（生产用）
 */

// ---------- mock 模拟 ----------

export interface StreamOptions {
  charsPerTick?: number
  interval?: number
}

/** 模拟流式输出（setInterval 逐字回调） */
export async function simulateStream(
  fullText: string,
  onUpdate: (currentText: string) => void,
  options: StreamOptions = {}
): Promise<void> {
  const { charsPerTick = 1, interval = 30 } = options
  let position = 0

  return new Promise<void>(resolve => {
    const timer = setInterval(() => {
      position += charsPerTick
      if (position >= fullText.length) {
        onUpdate(fullText)
        clearInterval(timer)
        resolve()
      } else {
        onUpdate(fullText.slice(0, position))
      }
    }, interval)
  })
}

// ---------- 真实 SSE 流解析 ----------

/** OpenAI 兼容 SSE 流中的 delta 数据结构 */
interface SSEDelta {
  choices?: Array<{
    delta?: { content?: string }
    finish_reason?: string | null
  }>
}

/**
 * 解析 SSE 流式响应（async generator）
 *
 * 用法：
 *   const reader = response.body!.getReader()
 *   for await (const chunk of parseSSEStream(reader)) {
 *     fullText += chunk   // 逐 token 累积
 *   }
 */
export async function* parseSSEStream(
  reader: ReadableStreamDefaultReader<Uint8Array>
): AsyncGenerator<string, void, undefined> {
  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    // 最后一段可能是不完整的行，放回 buffer
    buffer = lines.pop() || ''

    for (const line of lines) {
      const trimmed = line.trim()
      if (!trimmed || !trimmed.startsWith('data:')) continue

      const data = trimmed.slice(5).trim()
      if (data === '[DONE]') return

      try {
        const parsed: SSEDelta = JSON.parse(data)
        const content = parsed.choices?.[0]?.delta?.content
        if (content) yield content
      } catch {
        // 跳过无法解析的行
      }
    }
  }
}
