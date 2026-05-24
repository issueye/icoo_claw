<script setup>
import { computed } from 'vue'
import { renderMarkdown } from '@/services/utils/markdown'
import { LoaderCircle } from 'lucide-vue-next'

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
const toolLabel = computed(() => {
  const meta = props.message.metadata || {}
  if (!meta.toolCallId && !meta.toolStatus && !meta.toolKind) {
    return ''
  }
  return [meta.toolKind, meta.toolStatus].filter(Boolean).join(' · ')
})

const usageLabel = computed(() => {
  const usage = props.message.metadata?.usage
  if (!usage) {
    return ''
  }
  const parts = []
  if (Number.isFinite(usage.inputTokens)) parts.push(`in ${usage.inputTokens}`)
  if (Number.isFinite(usage.outputTokens)) parts.push(`out ${usage.outputTokens}`)
  if (Number.isFinite(usage.totalTokens)) parts.push(`total ${usage.totalTokens}`)
  return parts.join(' · ')
})

const isToolMessage = computed(() => Boolean(props.message.metadata?.toolCallId))
const statusLabel = computed(() => props.message.draft ? 'streaming' : '')

function roleLabel(role) {
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
    class="flex px-4 py-3"
    :class="message.role === 'user' ? 'justify-end' : 'justify-start'"
    :data-testid="`chat-message-${message.role}`"
  >
    <div class="max-w-[min(820px,78%)]">
      <header class="mb-2 flex items-center gap-2" :class="message.role === 'user' ? 'justify-end' : 'justify-start'">
        <span
          class="inline-flex rounded-[4px] px-2 py-0.5 text-[11px] uppercase tracking-[0.12em]"
          :class="
            message.role === 'user'
              ? 'bg-[rgba(255,255,255,0.14)] text-white'
              : 'bg-[rgba(54,220,200,0.16)] text-[var(--qq-accent)]'
          "
        >
          {{ roleLabel(message.role) }}
        </span>
        <span v-if="message.draft" class="text-xs text-[color:var(--qq-text-tertiary)]">streaming</span>
        <span
          v-if="toolLabel"
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
        class="border border-white/10 px-3 py-2.5"
        :class="message.role === 'user' ? 'bg-[rgba(255,255,255,0.12)]' : 'bg-[rgba(18,58,51,0.36)]'"
        style="border-radius: 6px;"
      >
        <div
          class="markdown-body text-sm leading-7"
          :class="message.error ? 'text-rose-100' : 'text-slate-50'"
          v-html="renderedContent"
        />
        <div
          v-if="isToolMessage || usageLabel"
          class="mt-2 border-t border-white/10 pt-2 text-xs text-[color:var(--qq-text-tertiary)]"
        >
          <span v-if="isToolMessage" class="mr-3">tool {{ message.metadata.toolKind || 'other' }} · {{ message.metadata.toolStatus || 'pending' }}</span>
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

.markdown-body :deep(p),
.markdown-body :deep(ul),
.markdown-body :deep(ol),
.markdown-body :deep(blockquote),
.markdown-body :deep(pre),
.markdown-body :deep(table) {
  margin: 0.7rem 0;
}

.markdown-body :deep(p:first-child),
.markdown-body :deep(ul:first-child),
.markdown-body :deep(ol:first-child),
.markdown-body :deep(blockquote:first-child),
.markdown-body :deep(pre:first-child),
.markdown-body :deep(table:first-child) {
  margin-top: 0;
}

.markdown-body :deep(p:last-child),
.markdown-body :deep(ul:last-child),
.markdown-body :deep(ol:last-child),
.markdown-body :deep(blockquote:last-child),
.markdown-body :deep(pre:last-child),
.markdown-body :deep(table:last-child) {
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
</style>
