import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { StoreId } from '@/enums/StoreId'
import {
  getSupportedTypes,
  isSupportedFileType,
  resolveFileType,
  streamAISummary,
  streamChatFile
} from '@/api/ai'

export type AiMode = 'global' | 'file'

export interface AiMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  time: string
  timestamp: number
  isStreaming?: boolean
}

export interface AnalyzeFileRef {
  id: string
  name: string
  type: string
}

export interface FileConversation {
  fileId: string
  fileName: string
  fileType: string
  summary: string
  messages: AiMessage[]
  updatedAt: number
}

let idCounter = 0
function genId(): string {
  return `msg_${Date.now()}_${++idCounter}`
}

function formatTime(date: Date): string {
  const h = date.getHours().toString().padStart(2, '0')
  const m = date.getMinutes().toString().padStart(2, '0')
  return `${h}:${m}`
}

function createMessage(role: 'user' | 'assistant', content: string): AiMessage {
  const now = new Date()
  return {
    id: genId(),
    role,
    content,
    time: formatTime(now),
    timestamp: now.getTime(),
    isStreaming: false
  }
}

function getErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) return error.message
  return 'AI 服务暂不可用，请稍后重试。'
}

function pickRecentHistory(messages: AiMessage[], maxRounds: number): Array<{ role: 'user' | 'assistant'; content: string }> {
  const maxMessages = Math.max(0, maxRounds * 2)
  if (maxMessages === 0) return []

  const cleaned = messages
    .filter(m => (m.role === 'user' || m.role === 'assistant') && m.content.trim().length > 0)
    .map(m => ({ role: m.role, content: m.content }))

  return cleaned.length <= maxMessages ? cleaned : cleaned.slice(cleaned.length - maxMessages)
}

export const useAiStore = defineStore(StoreId.Ai, () => {
  const visible = ref(false)
  const mode = ref<AiMode>('global')
  const currentFileId = ref<string | null>(null)

  const conversationsByFileId = ref<Record<string, FileConversation>>({})
  const globalMessages = ref<AiMessage[]>([
    createMessage('assistant', '你好，我是 MyObj AI 助手。可以帮你分析文件、回答问题、整理要点。')
  ])

  const summarizingByFileId = ref<Record<string, boolean>>({})
  const summaryCollapsedByFileId = ref<Record<string, boolean>>({})
  const isChatLoading = ref(false)
  const chatAbortController = ref<AbortController | null>(null)

  const currentConversation = computed<FileConversation | null>(() => {
    if (mode.value !== 'file') return null
    if (!currentFileId.value) return null
    return conversationsByFileId.value[currentFileId.value] || null
  })

  const activeMessages = computed<AiMessage[]>(() => {
    return mode.value === 'file' ? (currentConversation.value?.messages || []) : globalMessages.value
  })

  const isCurrentSummarizing = computed(() => {
    if (mode.value !== 'file' || !currentFileId.value) return false
    return !!summarizingByFileId.value[currentFileId.value]
  })

  const isCurrentSummaryCollapsed = computed(() => {
    if (mode.value !== 'file' || !currentFileId.value) return true
    return !!summaryCollapsedByFileId.value[currentFileId.value]
  })

  function ensureFileConversation(file: AnalyzeFileRef): FileConversation {
    const existing = conversationsByFileId.value[file.id]
    if (existing) {
      existing.fileName = file.name
      existing.fileType = resolveFileType(file.name, file.type)
      return existing
    }

    const conv: FileConversation = {
      fileId: file.id,
      fileName: file.name,
      fileType: resolveFileType(file.name, file.type),
      summary: '',
      messages: [],
      updatedAt: Date.now()
    }
    conversationsByFileId.value[file.id] = conv
    summaryCollapsedByFileId.value[file.id] = false
    return conv
  }

  function openGlobalPanel() {
    visible.value = true
    mode.value = 'global'
    currentFileId.value = null
  }

  function openFilePanel(file: AnalyzeFileRef) {
    visible.value = true
    mode.value = 'file'
    currentFileId.value = file.id

    const conv = ensureFileConversation(file)
    summaryCollapsedByFileId.value[file.id] = false
    if (!conv.summary) {
      Promise.resolve().then(() => summarizeFileStream(file))
    }
  }

  function closePanel() {
    visible.value = false
  }

  async function summarizeFileStream(file: AnalyzeFileRef) {
    if (!file.id) return

    const conv = ensureFileConversation(file)
    conv.summary = ''
    conv.updatedAt = Date.now()
    summaryCollapsedByFileId.value[file.id] = false

    if (!isSupportedFileType(file.name, file.type)) {
      conv.summary = `暂不支持该文件类型的 AI 分析（${resolveFileType(file.name, file.type) || 'unknown'}），当前仅支持：${getSupportedTypes()}`
      return
    }

    if (summarizingByFileId.value[file.id]) return
    summarizingByFileId.value[file.id] = true

    try {
      await streamAISummary(
        {
          fileId: file.id,
          fileName: file.name,
          fileType: file.type
        },
        content => {
          const c = conversationsByFileId.value[file.id]
          if (!c) return
          c.summary += content
          c.updatedAt = Date.now()
        }
      )
    } catch (error) {
      conv.summary = 'AI error: ' + getErrorMessage(error)
    } finally {
      summarizingByFileId.value[file.id] = false
    }
  }

  async function sendMessage(text: string) {
    const trimmed = text.trim()
    if (!trimmed) return
    if (isChatLoading.value) return
    if (isCurrentSummarizing.value) return

    const targetMessages = mode.value === 'file' ? currentConversation.value?.messages : globalMessages.value
    if (!targetMessages) return

    if (mode.value === 'file' && currentFileId.value) {
      summaryCollapsedByFileId.value[currentFileId.value] = true
    }

    targetMessages.push(createMessage('user', trimmed))
    const assistantMsg = createMessage('assistant', '')
    assistantMsg.isStreaming = true
    targetMessages.push(assistantMsg)
    isChatLoading.value = true
    chatAbortController.value = new AbortController()

    const history = pickRecentHistory(targetMessages.slice(0, Math.max(0, targetMessages.length - 2)), 5)

    try {
      if (mode.value === 'file' && currentConversation.value) {
        const c = currentConversation.value
        await streamChatFile(
          {
            fileId: c.fileId,
            fileName: c.fileName,
            fileType: c.fileType,
            message: trimmed,
            history
          },
          chunk => {
            const idx = targetMessages.findIndex(m => m.id === assistantMsg.id)
            if (idx !== -1) targetMessages[idx].content += chunk
          },
          { signal: chatAbortController.value.signal }
        )
        c.updatedAt = Date.now()
      } else {
        await streamChatFile(
          {
            message: trimmed,
            history
          },
          chunk => {
            const idx = targetMessages.findIndex(m => m.id === assistantMsg.id)
            if (idx !== -1) targetMessages[idx].content += chunk
          },
          { signal: chatAbortController.value.signal }
        )
      }
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        const idx = targetMessages.findIndex(m => m.id === assistantMsg.id)
        if (idx !== -1 && !targetMessages[idx].content) {
          targetMessages[idx].content = '已停止回答。'
        }
        return
      }
      const idx = targetMessages.findIndex(m => m.id === assistantMsg.id)
      if (idx !== -1) targetMessages[idx].content = 'AI error: ' + getErrorMessage(error)
    } finally {
      const idx = targetMessages.findIndex(m => m.id === assistantMsg.id)
      if (idx !== -1) targetMessages[idx].isStreaming = false
      chatAbortController.value = null
      isChatLoading.value = false
    }
  }

  function stopChatStream() {
    chatAbortController.value?.abort()
  }

  function clearCurrentConversation() {
    if (mode.value === 'global') {
      globalMessages.value = [createMessage('assistant', '你好，我是 MyObj AI 助手。')]
      return
    }

    const conv = currentConversation.value
    if (!conv) return
    conv.messages = []
    conv.updatedAt = Date.now()
  }

  function regenerateSummary() {
    if (mode.value !== 'file' || !currentConversation.value) return
    const c = currentConversation.value
    summaryCollapsedByFileId.value[c.fileId] = false
    summarizeFileStream({ id: c.fileId, name: c.fileName, type: c.fileType })
  }

  function collapseCurrentSummary() {
    if (mode.value !== 'file' || !currentFileId.value) return
    summaryCollapsedByFileId.value[currentFileId.value] = true
  }

  function expandCurrentSummary() {
    if (mode.value !== 'file' || !currentFileId.value) return
    summaryCollapsedByFileId.value[currentFileId.value] = false
  }

  function switchToFileConversation(fileId: string) {
    if (!conversationsByFileId.value[fileId]) return
    visible.value = true
    mode.value = 'file'
    currentFileId.value = fileId
  }

  function removeFileConversation(fileId: string) {
    delete conversationsByFileId.value[fileId]
    delete summarizingByFileId.value[fileId]
    delete summaryCollapsedByFileId.value[fileId]
    if (currentFileId.value === fileId) {
      currentFileId.value = null
      mode.value = 'global'
    }
  }

  return {
    visible,
    mode,
    currentFileId,
    conversationsByFileId,
    globalMessages,
    summaryCollapsedByFileId,

    currentConversation,
    activeMessages,
    isChatLoading,
    isCurrentSummarizing,
    isCurrentSummaryCollapsed,

    openGlobalPanel,
    openFilePanel,
    closePanel,

    summarizeFileStream,
    sendMessage,
    stopChatStream,

    clearCurrentConversation,
    regenerateSummary,
    collapseCurrentSummary,
    expandCurrentSummary,
    switchToFileConversation,
    removeFileConversation
  }
})
