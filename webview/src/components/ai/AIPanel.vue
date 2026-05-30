<template>
  <el-drawer
    v-model="visible"
    :with-header="false"
    size="420px"
    direction="rtl"
    :modal="true"
    :close-on-click-modal="true"
    :append-to-body="true"
    class="ai-panel-drawer"
    @close="handleClose"
  >
    <div class="ai-panel">
      <div class="ai-panel-header">
        <div class="ai-header-left">
          <el-icon class="ai-icon" :size="22"><MagicStick /></el-icon>
          <span class="ai-title">AI 助手</span>
          <span v-if="aiStore.mode === 'file' && aiStore.currentConversation" class="ai-subtitle">
            {{ aiStore.currentConversation.fileName }}
          </span>
          <span v-else class="ai-subtitle">全局</span>
        </div>
        <el-button class="ai-close-btn" icon="Close" circle text @click="handleClose" />
      </div>

      <div class="ai-panel-body">
        <div
          v-if="aiStore.mode === 'file' && aiStore.currentConversation"
          class="ai-analysis-result"
          :class="{ 'is-collapsed': aiStore.isCurrentSummaryCollapsed }"
        >
          <div class="analysis-header">
            <el-icon :size="16"><DataAnalysis /></el-icon>
            <span>AI 分析</span>
            <el-button
              class="analysis-action"
              size="small"
              text
              :disabled="aiStore.isCurrentSummarizing"
              :loading="aiStore.isCurrentSummarizing"
              @click="aiStore.regenerateSummary()"
            >
              重新分析
            </el-button>
            <el-button class="analysis-toggle" size="small" text @click="toggleSummary">
              {{ aiStore.isCurrentSummaryCollapsed ? '展开' : '收回' }}
            </el-button>
          </div>

          <div v-show="!aiStore.isCurrentSummaryCollapsed" class="analysis-body">
            <div
              v-if="aiStore.isCurrentSummarizing && !aiStore.currentConversation.summary"
              class="analysis-loading"
            >
              <span class="analysis-dot"></span>
              <span class="analysis-dot"></span>
              <span class="analysis-dot"></span>
            </div>
            <div
              v-else
              class="analysis-text"
              v-html="renderMarkdown(aiStore.currentConversation.summary || '')"
            />
          </div>
        </div>

        <AIChat />
      </div>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { useAiStore } from '@/stores'
import AIChat from './AIChat.vue'

const visible = defineModel<boolean>({ required: true })
const aiStore = useAiStore()

function handleClose() {
  visible.value = false
  aiStore.closePanel()
}

function toggleSummary() {
  if (aiStore.isCurrentSummaryCollapsed) {
    aiStore.expandCurrentSummary()
  } else {
    aiStore.collapseCurrentSummary()
  }
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
}

function renderMarkdown(text: string): string {
  return escapeHtml(text)
    .replace(/^### (.*$)/gim, '<h4 class="md-h4">$1</h4>')
    .replace(/^## (.*$)/gim, '<h3 class="md-h3">$1</h3>')
    .replace(/^# (.*$)/gim, '<h2 class="md-h2">$1</h2>')
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code class="md-code">$1</code>')
    .replace(/^- (.*$)/gim, '<li class="md-li">$1</li>')
    .replace(/(<li class="md-li">.*<\/li>)/gs, (match: string) => `<ul class="md-ul">${match}</ul>`)
    .replace(/\n/g, '<br/>')
}
</script>

<style scoped>
.ai-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.ai-panel-header {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px 0 18px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  flex-shrink: 0;
}

.ai-header-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.ai-icon {
  color: #6366f1;
}

.ai-title {
  font-size: 15px;
  font-weight: 800;
  color: var(--el-text-color-primary);
}

.ai-subtitle {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-close-btn {
  color: var(--el-text-color-regular);
}

.ai-panel-body {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.ai-analysis-result {
  position: relative;
  z-index: 1;
  flex: 0 0 auto;
  margin: 16px 20px 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.08), rgba(139, 92, 246, 0.1));
  border: 1px solid rgba(99, 102, 241, 0.2);
  border-radius: 14px;
  box-shadow: none;
}

.ai-analysis-result:not(.is-collapsed) {
  position: absolute;
  z-index: 5;
  top: 16px;
  left: 20px;
  right: 20px;
  bottom: 88px;
  margin: 0;
  background:
    linear-gradient(135deg, rgba(99, 102, 241, 0.06), rgba(139, 92, 246, 0.08)),
    var(--el-bg-color);
  box-shadow: 0 12px 30px rgba(99, 102, 241, 0.08);
}

.analysis-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px 10px;
  font-size: 14px;
  font-weight: 700;
  color: #6366f1;
  border-bottom: 1px solid rgba(99, 102, 241, 0.1);
  background: var(--el-bg-color);
  flex-shrink: 0;
}

.analysis-action {
  margin-left: auto;
  color: var(--el-text-color-secondary);
}

.analysis-toggle {
  color: #6366f1;
  font-weight: 700;
}

.analysis-body {
  flex: 1;
  min-height: 0;
  padding: 14px 16px 18px;
  overflow-y: auto;
}

.analysis-text {
  font-size: 13px;
  line-height: 1.65;
  color: var(--el-text-color-regular);
}

.analysis-loading {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 42px;
}

.analysis-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #6366f1;
  animation: analysis-dot-bounce 1.2s ease-in-out infinite;
}

.analysis-dot:nth-child(2) {
  animation-delay: 0.16s;
}

.analysis-dot:nth-child(3) {
  animation-delay: 0.32s;
}

@keyframes analysis-dot-bounce {
  0%,
  70%,
  100% {
    transform: translateY(0);
    opacity: 0.35;
  }
  35% {
    transform: translateY(-6px);
    opacity: 1;
  }
}

.analysis-text :deep(.md-h2) {
  font-size: 15px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  margin: 12px 0 6px;
}

.analysis-text :deep(.md-h3) {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin: 10px 0 4px;
}

.analysis-text :deep(.md-h4) {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-regular);
  margin: 8px 0 4px;
}

.analysis-text :deep(.md-ul) {
  padding-left: 18px;
  margin: 4px 0;
}

.analysis-text :deep(.md-li) {
  margin-bottom: 2px;
}

.analysis-text :deep(strong) {
  color: #6366f1;
}

.analysis-text :deep(.md-code) {
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(99, 102, 241, 0.08);
  color: var(--el-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 12px;
}

html.dark .analysis-header {
  background: var(--el-bg-color);
}

.ai-panel-drawer :deep(.el-drawer__body) {
  padding: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  height: 100%;
}
</style>
