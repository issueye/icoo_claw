<script setup>
import { computed, ref, watch } from 'vue'
import { hasVisibleMarkdownContent, renderMarkdown } from '@/services/utils/markdown'
import { CheckCircle2, ChevronDown, ChevronRight, LoaderCircle, Terminal, XCircle } from 'lucide-vue-next'

const props = defineProps({
  message: {
    type: Object,
    required: true,
  },
  showTimestamps: {
    type: Boolean,
    default: true,
  },
})

const renderedContent = computed(() => renderMarkdown(props.message.content))
const toolMetadata = computed(() => props.message.metadata || {})
const hasRenderableContent = computed(() => hasVisibleMarkdownContent(props.message.content))
const toolLabel = computed(() => {
  const meta = toolMetadata.value
  if (!meta.toolCallId && !meta.toolStatus && !meta.toolKind) {
    return ''
  }
  return [meta.toolKind, meta.toolStatus].filter(Boolean).join(' · ')
})

const usageLabel = computed(() => {
  const usage = toolMetadata.value.usage
  if (!usage) {
    return ''
  }
  const parts = []
  if (Number.isFinite(usage.inputTokens)) parts.push(`in ${usage.inputTokens}`)
  if (Number.isFinite(usage.outputTokens)) parts.push(`out ${usage.outputTokens}`)
  if (Number.isFinite(usage.totalTokens)) parts.push(`total ${usage.totalTokens}`)
  return parts.join(' · ')
})

const isToolMessage = computed(() => props.message.role === 'tool' || Boolean(toolMetadata.value.toolCallId))
const showPendingAssistant = computed(() => (
  props.message.role === 'assistant' &&
  props.message.draft &&
  !isToolMessage.value &&
  !hasRenderableContent.value
))
const messageShellClass = computed(() => {
  if (props.message.role === 'user') return 'message-shell--user'
  if (isToolMessage.value) return 'message-shell--tool'
  return 'message-shell--assistant'
})
const statusLabel = computed(() => props.message.draft ? 'streaming' : '')
const toolStatus = computed(() => toolMetadata.value.toolStatus || '')
const toolIcon = computed(() => {
  if (toolStatus.value === 'completed') return CheckCircle2
  if (toolStatus.value === 'failed' || toolStatus.value === 'cancelled') return XCircle
  return LoaderCircle
})
const toolIconClass = computed(() => {
  if (toolStatus.value === 'completed') return 'tool-icon--completed'
  if (toolStatus.value === 'failed' || toolStatus.value === 'cancelled') return 'tool-icon--failed'
  return 'animate-spin text-[var(--qq-accent)]'
})
const isFinalToolStatus = computed(() => ['completed', 'failed', 'cancelled'].includes(toolStatus.value))
const toolExpanded = ref(!isFinalToolStatus.value)
const shouldShowMessageBody = computed(() => !isToolMessage.value || toolExpanded.value)
const toolToggleLabel = computed(() => toolExpanded.value ? '收起' : '展开')

watch(
  () => `${props.message.id}:${toolStatus.value}:${isToolMessage.value}`,
  () => {
    toolExpanded.value = !(isToolMessage.value && isFinalToolStatus.value)
  },
)

function roleLabel(role) {
  if (isToolMessage.value) {
    return 'Tool'
  }
  return role === 'user' ? 'You' : 'Assistant'
}

function formatTimestamp(value) {
  if (!value) {
    return ''
  }
  return new Date(value).toLocaleTimeString()
}
</script>

<template>
  <article
    class="flex px-4 py-2.5 qq-msg-fadein"
    :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
    :data-testid="`chat-message-${message.role}`"
  >
    <div class="message-shell" :class="messageShellClass">
      <header class="mb-2 flex items-center gap-2" :class="message.role === 'user' ? 'justify-end' : 'justify-start'">
        <span
          class="role-badge inline-flex rounded-[4px] px-2 py-0.5 text-[11px] uppercase tracking-[0.12em]"
          :class="
            message.role === 'user'
              ? 'role-badge--user'
              : 'role-badge--assistant'
          "
        >
          {{ roleLabel(message.role) }}
        </span>
        <span v-if="message.draft" class="text-xs text-[color:var(--qq-text-tertiary)]">streaming</span>
        <span
          v-if="toolLabel && !isToolMessage"
          class="inline-flex items-center gap-1 rounded-[4px] bg-[rgba(255,255,255,0.08)] px-2 py-0.5 text-[11px] text-[color:var(--qq-text-secondary)]"
        >
          <LoaderCircle class="h-3 w-3 animate-spin" />
          {{ toolLabel }}
        </span>
        <span v-if="usageLabel" class="text-xs text-[color:var(--qq-text-tertiary)]">{{ usageLabel }}</span>
        <span v-if="statusLabel" class="text-xs text-[color:var(--qq-text-tertiary)]">{{ statusLabel }}</span>
        <time v-if="showTimestamps" class="text-xs text-[color:var(--qq-text-tertiary)]">
          {{ formatTimestamp(message.createdAt) }}
        </time>
      </header>

      <div
        class="message-bubble border px-3 py-2.5"
        :class="[
          message.role === 'user'
            ? 'message-bubble--user'
            : 'message-bubble--assistant qq-msg-ai-bg',
          isToolMessage ? 'message-bubble--tool qq-msg-tool-bg' : '',
        ]"
        style="border-radius: 8px;"
      >
        <button
          v-if="isToolMessage"
          type="button"
          class="tool-message-toggle flex w-full items-center gap-2 rounded-[4px] text-left text-xs text-[color:var(--qq-text-secondary)]"
          :class="shouldShowMessageBody ? 'mb-2' : ''"
          :aria-expanded="toolExpanded"
          @click="toolExpanded = !toolExpanded"
        >
          <component :is="toolIcon" class="h-3.5 w-3.5 shrink-0" :class="toolIconClass" />
          <Terminal class="h-3.5 w-3.5 shrink-0 text-[color:var(--qq-text-tertiary)]" />
          <span class="min-w-0 flex-1 truncate font-medium text-[color:var(--qq-text-primary)]">
            {{ toolMetadata.toolTitle || toolMetadata.toolKind || '工具调用' }}
          </span>
          <span v-if="toolMetadata.toolStatus" class="shrink-0">· {{ toolMetadata.toolStatus }}</span>
          <span class="tool-message-toggle__label inline-flex shrink-0 items-center gap-1">
            <component :is="toolExpanded ? ChevronDown : ChevronRight" class="h-3.5 w-3.5" />
            {{ toolToggleLabel }}
          </span>
        </button>
        <div
          v-if="showPendingAssistant"
          class="flex items-center gap-2 text-sm leading-7 text-[color:var(--qq-text-primary)]"
        >
          <LoaderCircle class="h-4 w-4 animate-spin text-[var(--qq-accent)]" />
          <span>正在连接 Agent...</span>
        </div>
        <div
          v-else-if="shouldShowMessageBody"
          class="markdown-body text-sm leading-7"
          :class="message.error ? 'text-rose-100' : 'text-[color:var(--qq-text-primary)]'"
          v-html="renderedContent"
        />
        <div
          v-if="shouldShowMessageBody && (isToolMessage || usageLabel)"
          class="mt-2 border-t border-white/10 pt-2 text-xs text-[color:var(--qq-text-tertiary)]"
        >
          <span v-if="isToolMessage" class="mr-3">tool {{ toolMetadata.toolKind || 'other' }} · {{ toolMetadata.toolStatus || 'pending' }}</span>
          <span v-if="usageLabel">usage {{ usageLabel }}</span>
        </div>
      </div>
    </div>
  </article>
</template>

<style scoped>
.markdown-body {
  overflow-wrap: anywhere;
}

.message-shell {
  max-width: min(920px, 82%);
}

.message-shell--user {
  max-width: min(720px, 70%);
}

.message-shell--tool {
  max-width: min(980px, 86%);
}

.role-badge {
  border: 1px solid var(--qq-border);
  font-weight: 700;
}

.role-badge--user {
  background: rgba(226, 232, 240, 0.12);
  color: var(--qq-text-primary);
}

.role-badge--assistant {
  background: rgba(34, 211, 238, 0.12);
  color: var(--qq-accent);
}

.message-bubble {
  border-color: var(--qq-border);
}

.message-bubble--user {
  background: rgba(226, 232, 240, 0.10);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.30);
}

.message-bubble--assistant {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.25);
}

.message-bubble--tool {
  box-shadow: 0 14px 32px rgba(0, 0, 0, 0.22);
}

.tool-icon--completed {
  color: #86efac;
}

.tool-icon--failed {
  color: #fca5a5;
}

.tool-message-toggle {
  transition:
    background-color 160ms ease,
    color 160ms ease;
}

.tool-message-toggle:hover {
  background: rgba(255, 255, 255, 0.06);
  color: rgb(226 232 240);
}

.tool-message-toggle:focus-visible {
  outline: 1px solid rgba(54, 220, 200, 0.58);
  outline-offset: 2px;
}

.tool-message-toggle__label {
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 4px;
  padding: 0.12rem 0.4rem;
  color: rgb(210 255 245);
}

.markdown-body :deep(p),
.markdown-body :deep(ul),
.markdown-body :deep(ol),
.markdown-body :deep(blockquote),
.markdown-body :deep(pre),
.markdown-body :deep(.markdown-table-scroll) {
  margin: 0.7rem 0;
}

.markdown-body :deep(p:first-child),
.markdown-body :deep(ul:first-child),
.markdown-body :deep(ol:first-child),
.markdown-body :deep(blockquote:first-child),
.markdown-body :deep(pre:first-child),
.markdown-body :deep(.markdown-table-scroll:first-child) {
  margin-top: 0;
}

.markdown-body :deep(p:last-child),
.markdown-body :deep(ul:last-child),
.markdown-body :deep(ol:last-child),
.markdown-body :deep(blockquote:last-child),
.markdown-body :deep(pre:last-child),
.markdown-body :deep(.markdown-table-scroll:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4) {
  margin: 1rem 0 0.45rem;
  color: rgb(248 250 252);
  font-weight: 700;
  line-height: 1.35;
}

.markdown-body :deep(h1) {
  font-size: 1.35rem;
}

.markdown-body :deep(h2) {
  font-size: 1.15rem;
}

.markdown-body :deep(h3) {
  font-size: 1rem;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  padding-left: 1.25rem;
}

.markdown-body :deep(ul) {
  list-style: disc;
}

.markdown-body :deep(ol) {
  list-style: decimal;
}

.markdown-body :deep(li + li) {
  margin-top: 0.25rem;
}

.markdown-body :deep(strong) {
  color: rgb(255 255 255);
  font-weight: 700;
}

.markdown-body :deep(a) {
  color: var(--qq-accent);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.markdown-body :deep(.markdown-table-scroll) {
  width: 100%;
  max-width: 100%;
  overflow-x: auto;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 6px;
  background:
    linear-gradient(90deg, rgba(54, 220, 200, 0.08), transparent 35%),
    rgba(3, 24, 22, 0.24);
}

.markdown-body :deep(table) {
  width: 100%;
  min-width: max-content;
  border-collapse: separate;
  border-spacing: 0;
}

.markdown-body :deep(thead) {
  background: rgba(54, 220, 200, 0.14);
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  border-right: 1px solid rgba(255, 255, 255, 0.06);
  padding: 0.55rem 0.75rem;
  text-align: left;
  vertical-align: top;
  white-space: normal;
  word-break: break-word;
}

.markdown-body :deep(th:last-child),
.markdown-body :deep(td:last-child) {
  border-right: 0;
}

.markdown-body :deep(th) {
  color: rgb(225 255 250);
  font-size: 0.78rem;
  font-weight: 700;
}

.markdown-body :deep(td) {
  color: rgb(241 245 249);
}

.markdown-body :deep(tbody tr:last-child td) {
  border-bottom: 0;
}

.markdown-body :deep(tbody tr:hover) {
  background: rgba(255, 255, 255, 0.04);
}

.markdown-body :deep(table code) {
  white-space: nowrap;
}

.markdown-body :deep(code) {
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 4px;
  background: rgba(4, 20, 18, 0.42);
  padding: 0.1rem 0.32rem;
  color: rgb(210 255 245);
  font-size: 0.9em;
}

.markdown-body :deep(pre) {
  overflow-x: auto;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 6px;
  background: rgba(4, 20, 18, 0.52);
  padding: 0.8rem;
}

.markdown-body :deep(pre code) {
  border: 0;
  background: transparent;
  padding: 0;
  color: inherit;
}

.markdown-body :deep(blockquote) {
  border-left: 3px solid var(--qq-accent);
  padding-left: 0.8rem;
  color: var(--qq-text-secondary);
}

:global([data-theme="light"]) .message-bubble--user {
  background: rgba(8, 125, 167, 0.08);
  border-color: rgba(8, 125, 167, 0.22);
  box-shadow: 0 2px 10px rgba(15, 23, 42, 0.08);
}

:global([data-theme="light"]) .message-bubble--assistant {
  box-shadow: 0 10px 26px rgba(15, 23, 42, 0.10);
}

:global([data-theme="light"]) .message-bubble--tool {
  box-shadow: 0 14px 28px rgba(15, 23, 42, 0.12);
}

:global([data-theme="light"]) .role-badge--user {
  background: rgba(8, 125, 167, 0.09);
  border-color: rgba(8, 125, 167, 0.22);
  color: #075985;
}

:global([data-theme="light"]) .role-badge--assistant {
  background: rgba(8, 125, 167, 0.08);
  border-color: rgba(8, 125, 167, 0.18);
  color: var(--qq-accent-strong);
}

:global([data-theme="light"]) .tool-icon--completed {
  color: #047857;
}

:global([data-theme="light"]) .tool-icon--failed {
  color: #be123c;
}

:global([data-theme="light"]) .tool-message-toggle:hover {
  background: rgba(15, 23, 42, 0.05);
  color: var(--qq-text-primary);
}

:global([data-theme="light"]) .tool-message-toggle__label {
  border-color: rgba(15, 23, 42, 0.14);
  color: var(--qq-accent-strong);
}

:global([data-theme="light"]) .markdown-body :deep(h1),
:global([data-theme="light"]) .markdown-body :deep(h2),
:global([data-theme="light"]) .markdown-body :deep(h3),
:global([data-theme="light"]) .markdown-body :deep(h4),
:global([data-theme="light"]) .markdown-body :deep(strong) {
  color: var(--qq-text-primary);
}

:global([data-theme="light"]) .markdown-body :deep(td) {
  color: var(--qq-text-primary);
}

:global([data-theme="light"]) .markdown-body :deep(th) {
  color: var(--qq-accent-strong);
}

:global([data-theme="light"]) .markdown-body :deep(.markdown-table-scroll) {
  border-color: rgba(15, 23, 42, 0.14);
  background:
    linear-gradient(90deg, rgba(8, 125, 167, 0.055), transparent 35%),
    rgba(255, 255, 255, 0.72);
}

:global([data-theme="light"]) .markdown-body :deep(th),
:global([data-theme="light"]) .markdown-body :deep(td) {
  border-color: rgba(15, 23, 42, 0.10);
}

:global([data-theme="light"]) .markdown-body :deep(code) {
  border-color: rgba(15, 23, 42, 0.14);
  background: rgba(15, 23, 42, 0.055);
  color: #075985;
}

:global([data-theme="light"]) .markdown-body :deep(pre) {
  border-color: rgba(15, 23, 42, 0.14);
  background: rgba(15, 23, 42, 0.055);
}
</style>
