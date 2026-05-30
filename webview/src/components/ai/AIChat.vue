<template>
  <div class="ai-chat">
    <!-- 消息列表 -->
    <div ref="scrollRef" class="ai-chat__messages">
      <AIMessage
        v-for="msg in visibleMessages"
        :key="msg.id"
        :role="msg.role"
        :content="msg.content"
        :time="msg.time"
        :avatar-char="avatarChar"
        :is-streaming="msg.isStreaming"
      />

      <!-- loading 状态：仅思考阶段显示，流式输出时由消息自带光标替代 -->
      <div v-if="showThinking" class="ai-chat__loading">
        <div class="ai-chat__loading-avatar">
          <el-avatar :size="30" class="ai-chat__loading-avatar-img">
            <el-icon :size="16"><MagicStick /></el-icon>
          </el-avatar>
        </div>
        <div class="ai-chat__loading-bubble">
          <span class="dot"></span>
          <span class="dot"></span>
          <span class="dot"></span>
        </div>
      </div>
    </div>

    <!-- 输入框 -->
    <div class="ai-chat__footer">
      <div class="ai-chat__input-wrap">
        <el-input
          ref="inputRef"
          v-model="inputText"
          placeholder="输入消息，Enter 发送..."
          :autosize="{ minRows: 1, maxRows: 3 }"
          type="textarea"
          resize="none"
          :disabled="store.isCurrentSummarizing"
          @keydown.enter.exact.prevent="handleSend"
        />
        <el-button
          class="ai-chat__send-btn"
          :class="{ 'ai-chat__send-btn--stop': store.isChatLoading }"
          circle
          :disabled="(!store.isChatLoading && !inputText.trim()) || store.isCurrentSummarizing"
          @click="handleSendAction"
        >
          <span v-if="store.isChatLoading" class="ai-chat__stop-icon"></span>
          <el-icon v-else :size="16"><Promotion /></el-icon>
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAiStore, useUserStore } from '@/stores'
import AIMessage from './AIMessage.vue'

const store = useAiStore()
const userStore = useUserStore()
const scrollRef = ref<HTMLElement>()
const inputRef = ref()
const inputText = ref('')

const avatarChar = computed(() => {
  const name = userStore.nickname || userStore.username || 'U'
  return name.charAt(0).toUpperCase()
})

/** 过滤掉 loading 阶段 content 为空的 AI 消息，避免空消息气泡与三点同时显示 */
const visibleMessages = computed(() => {
  return store.activeMessages.filter(m => m.content.length > 0 || m.role === 'user')
})

/** loading 点仅在下述条件满足时显示：AI 消息已创建但内容尚未开始输出 */
const showThinking = computed(() => {
  if (!store.isChatLoading) return false
  const msgs = store.activeMessages
  if (msgs.length === 0) return true
  const last = msgs[msgs.length - 1]
  return last.role === 'assistant' && last.content.length === 0
})

function scrollToBottom() {
  nextTick(() => {
    if (scrollRef.value) {
      scrollRef.value.scrollTop = scrollRef.value.scrollHeight
    }
  })
}

function focusInput() {
  nextTick(() => {
    inputRef.value?.focus?.()
  })
}

function handleSend() {
  const text = inputText.value.trim()
  if (!text || store.isChatLoading || store.isCurrentSummarizing) {
    focusInput()
    return
  }
  inputText.value = ''
  store.sendMessage(text)
  focusInput()
}

function handleSendAction() {
  if (store.isChatLoading) {
    store.stopChatStream()
    focusInput()
    return
  }

  handleSend()
}

// 监听消息数量变化 → 滚动
watch(
  () => store.activeMessages.length,
  () => scrollToBottom()
)

// 监听最后一条消息内容变化（流式输出时内容持续增长）→ 滚动
watch(
  () => {
    const msgs = store.activeMessages
    return msgs.length > 0 ? msgs[msgs.length - 1].content : ''
  },
  () => scrollToBottom()
)

// loading 开始时滚动
watch(
  () => store.isChatLoading,
  (val) => {
    if (val) {
      scrollToBottom()
    } else {
      focusInput()
    }
  }
)

onMounted(() => {
  scrollToBottom()
  focusInput()
})
</script>

<style scoped>
.ai-chat {
  display: flex;
  flex-direction: column;
  flex: 1 1 auto;
  height: auto;
  min-height: 0;
}

/* 消息滚动区 */
.ai-chat__messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* loading 提示 */
.ai-chat__loading {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}

.ai-chat__loading-avatar {
  flex-shrink: 0;
}

.ai-chat__loading-avatar-img {
  background: linear-gradient(135deg, #6366f1, #8b5cf6) !important;
  color: #fff;
}

.ai-chat__loading-bubble {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 12px 16px;
  border-radius: 16px;
  border-top-left-radius: 4px;
  background: var(--el-fill-color-light);
  min-width: 48px;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--el-text-color-secondary);
  animation: dot-bounce 1.4s ease-in-out infinite;
}

.dot:nth-child(2) { animation-delay: 0.2s; }
.dot:nth-child(3) { animation-delay: 0.4s; }

@keyframes dot-bounce {
  0%, 60%, 100% {
    transform: translateY(0);
    opacity: 0.4;
  }
  30% {
    transform: translateY(-6px);
    opacity: 1;
  }
}

/* 底部输入区 */
.ai-chat__footer {
  flex-shrink: 0;
  padding: 12px 20px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.ai-chat__input-wrap {
  position: relative;
}

.ai-chat__input-wrap :deep(.el-textarea__inner) {
  border-radius: 12px;
  padding: 10px 48px 10px 14px;
  background: var(--el-fill-color-light);
  border-color: var(--el-border-color-lighter);
  font-size: 13px;
  line-height: 1.5;
  box-shadow: none;
  transition: all 0.2s ease;
}

.ai-chat__input-wrap :deep(.el-textarea__inner):focus {
  border-color: #6366f1;
  background: var(--el-bg-color);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.1);
}

.ai-chat__input-wrap :deep(.el-textarea__inner):hover {
  border-color: var(--el-border-color);
}

.ai-chat__send-btn {
  position: absolute;
  right: 6px;
  bottom: 6px;
  width: 30px;
  height: 30px;
  padding: 0;
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  border: none;
  color: #fff;
}

.ai-chat__send-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
}

.ai-chat__send-btn--stop,
.ai-chat__send-btn--stop:hover:not(:disabled) {
  background: var(--el-color-danger);
}

.ai-chat__stop-icon {
  width: 11px;
  height: 11px;
  border-radius: 2px;
  background: #fff;
}

.ai-chat__send-btn:disabled {
  background: var(--el-fill-color);
  color: var(--el-text-color-placeholder);
}
</style>
