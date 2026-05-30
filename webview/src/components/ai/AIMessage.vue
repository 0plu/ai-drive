<template>
  <div class="ai-message" :class="role === 'user' ? 'ai-message--user' : 'ai-message--ai'">
    <!-- 头像 -->
    <div class="ai-message__avatar">
      <el-avatar
        v-if="role === 'assistant'"
        :size="30"
        class="ai-message__avatar-img ai-message__avatar--bot"
      >
        <el-icon :size="16"><MagicStick /></el-icon>
      </el-avatar>
      <el-avatar
        v-else
        :size="30"
        class="ai-message__avatar-img ai-message__avatar--human"
      >
        {{ avatarChar }}
      </el-avatar>
    </div>

    <!-- 气泡 -->
    <div class="ai-message__body">
      <div class="ai-message__bubble" :class="{ 'ai-message__bubble--streaming': isStreaming }">
        <div v-if="role === 'assistant'" class="ai-message__content ai-message__content--markdown">
          <div class="ai-message__markdown" v-html="renderedMarkdown"></div>
          <span v-if="isStreaming" class="ai-message__cursor">|</span>
        </div>
        <div v-else class="ai-message__content ai-message__content--text">
          {{ content }}<span v-if="isStreaming" class="ai-message__cursor">|</span>
        </div>
      </div>
      <div class="ai-message__meta">
        <span class="ai-message__time">{{ time }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import css from 'highlight.js/lib/languages/css'
import go from 'highlight.js/lib/languages/go'
import javascript from 'highlight.js/lib/languages/javascript'
import json from 'highlight.js/lib/languages/json'
import typescript from 'highlight.js/lib/languages/typescript'
import vue from 'highlight.js/lib/languages/xml'
import 'highlight.js/styles/github.css'

hljs.registerLanguage('bash', bash)
hljs.registerLanguage('sh', bash)
hljs.registerLanguage('shell', bash)
hljs.registerLanguage('css', css)
hljs.registerLanguage('go', go)
hljs.registerLanguage('golang', go)
hljs.registerLanguage('html', vue)
hljs.registerLanguage('xml', vue)
hljs.registerLanguage('vue', vue)
hljs.registerLanguage('js', javascript)
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('json', json)
hljs.registerLanguage('ts', typescript)
hljs.registerLanguage('typescript', typescript)

const escapeHtml = (value: string) =>
  value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')

const props = defineProps<{
  role: 'user' | 'assistant'
  content: string
  time: string
  avatarChar?: string
  isStreaming?: boolean
}>()

const markdown = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  breaks: true,
  highlight(code: string, lang: string) {
    const language = lang.trim().toLowerCase()

    if (language && hljs.getLanguage(language)) {
      return hljs.highlight(code, { language, ignoreIllegals: true }).value
    }

    return escapeHtml(code)
  }
})

const renderedMarkdown = computed(() => markdown.render(props.content || ''))
</script>

<style scoped>
.ai-message {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  padding: 4px 0;
}

.ai-message--user {
  flex-direction: row-reverse;
}

/* 头像 */
.ai-message__avatar {
  flex-shrink: 0;
  padding-top: 2px;
}

.ai-message__avatar--bot {
  background: linear-gradient(135deg, #6366f1, #8b5cf6) !important;
  color: #fff;
}

.ai-message__avatar--human {
  background: var(--el-color-primary) !important;
  color: #fff;
  font-weight: 600;
  font-size: 13px;
}

/* 气泡体 */
.ai-message__body {
  display: flex;
  flex-direction: column;
  max-width: 78%;
  position: relative;
}

.ai-message--user .ai-message__body {
  align-items: flex-end;
}

.ai-message--ai .ai-message__body {
  align-items: flex-start;
}

.ai-message__bubble {
  padding: 10px 14px;
  border-radius: 16px;
  font-size: 13px;
  line-height: 1.6;
  word-break: break-word;
  max-width: 100%;
  overflow: hidden;
}

.ai-message--ai .ai-message__bubble {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  border-top-left-radius: 4px;
}

.ai-message--user .ai-message__bubble {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff;
  border-top-right-radius: 4px;
}

.ai-message__content {
  min-width: 0;
  max-width: 100%;
}

.ai-message__content--text {
  white-space: pre-line;
}

.ai-message__content--markdown {
  white-space: normal;
}

.ai-message__markdown {
  max-width: 100%;
  overflow-wrap: anywhere;
}

.ai-message__markdown :deep(*) {
  max-width: 100%;
}

.ai-message__markdown :deep(p),
.ai-message__markdown :deep(ul),
.ai-message__markdown :deep(ol),
.ai-message__markdown :deep(blockquote),
.ai-message__markdown :deep(pre),
.ai-message__markdown :deep(table) {
  margin: 0 0 10px;
}

.ai-message__markdown :deep(*:last-child) {
  margin-bottom: 0;
}

.ai-message__markdown :deep(h1),
.ai-message__markdown :deep(h2),
.ai-message__markdown :deep(h3),
.ai-message__markdown :deep(h4),
.ai-message__markdown :deep(h5),
.ai-message__markdown :deep(h6) {
  margin: 12px 0 8px;
  font-weight: 700;
  line-height: 1.35;
  color: var(--el-text-color-primary);
}

.ai-message__markdown :deep(h1) {
  font-size: 18px;
}

.ai-message__markdown :deep(h2) {
  font-size: 16px;
}

.ai-message__markdown :deep(h3),
.ai-message__markdown :deep(h4),
.ai-message__markdown :deep(h5),
.ai-message__markdown :deep(h6) {
  font-size: 14px;
}

.ai-message__markdown :deep(ul),
.ai-message__markdown :deep(ol) {
  padding-left: 20px;
}

.ai-message__markdown :deep(li + li) {
  margin-top: 3px;
}

.ai-message__markdown :deep(blockquote) {
  padding: 6px 10px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
  border-left: 3px solid var(--el-color-primary-light-5);
  border-radius: 4px;
}

.ai-message__markdown :deep(:not(pre) > code) {
  padding: 2px 5px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  color: #b42318;
  background: rgba(99, 102, 241, 0.1);
  border-radius: 4px;
}

.ai-message__markdown :deep(pre) {
  padding: 12px;
  overflow-x: auto;
  background: #f6f8fa;
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
}

.ai-message__markdown :deep(pre code) {
  display: block;
  min-width: max-content;
  padding: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  line-height: 1.55;
  white-space: pre;
  word-break: normal;
  overflow-wrap: normal;
  background: transparent;
  border-radius: 0;
}

.ai-message__markdown :deep(table) {
  display: block;
  width: max-content;
  max-width: 100%;
  overflow-x: auto;
  border-collapse: collapse;
}

.ai-message__markdown :deep(th),
.ai-message__markdown :deep(td) {
  padding: 6px 8px;
  border: 1px solid var(--el-border-color);
}

.ai-message__markdown :deep(th) {
  font-weight: 600;
  background: var(--el-fill-color);
}

.ai-message__markdown :deep(a) {
  color: var(--el-color-primary);
  text-decoration: none;
}

.ai-message__markdown :deep(a:hover) {
  text-decoration: underline;
}

/* 元信息 */
.ai-message__meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 3px;
  padding: 0 4px;
}

.ai-message__time {
  font-size: 10px;
  color: var(--el-text-color-placeholder);
  user-select: none;
}

.ai-message--user .ai-message__time {
  color: var(--el-text-color-disabled);
}

/* 流式光标 */
.ai-message__cursor {
  display: inline-block;
  font-weight: 300;
  color: var(--el-color-primary);
  animation: cursor-blink 0.8s step-end infinite;
}

@keyframes cursor-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.ai-message__bubble--streaming {
  min-height: 24px;
}
</style>
