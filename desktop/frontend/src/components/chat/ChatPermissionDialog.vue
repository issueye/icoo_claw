<script setup>
import { computed } from 'vue'
import { Check, ShieldAlert, ShieldCheck, X } from 'lucide-vue-next'
import QqButton from '@/components/ued/QqButton.vue'

const props = defineProps({
  permission: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['select', 'cancel'])

const toolCall = computed(() => props.permission?.toolCall || {})
const title = computed(() => toolCall.value.title || toolCall.value.kind || '工具权限请求')
const rawInput = computed(() => formatPayload(toolCall.value.rawInput))
const allowOptions = computed(() => optionsByKind('allow'))
const rejectOptions = computed(() => optionsByKind('reject'))
const displayOptions = computed(() => props.permission?.options || [])

function optionsByKind(kind) {
  return (props.permission?.options || []).filter((option) => String(option.kind || '').startsWith(kind))
}

function choose(option) {
  emit('select', option)
}

function cancel() {
  emit('cancel')
}

function formatPayload(value) {
  if (value === null || value === undefined || value === '') {
    return ''
  }
  if (typeof value === 'string') {
    return value
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

function shortOptionLabel(option) {
  switch (option?.kind) {
    case 'allow_once':
      return '允许本次'
    case 'allow_always':
      return '始终允许'
    case 'reject_once':
      return '拒绝本次'
    case 'reject_always':
      return '始终拒绝'
    default:
      return option?.name || '选择'
  }
}

function optionKindLabel(kind) {
  switch (kind) {
    case 'allow_once':
      return '允许本次'
    case 'allow_always':
      return '始终允许'
    case 'reject_once':
      return '拒绝本次'
    case 'reject_always':
      return '始终拒绝'
    default:
      return kind || '自定义'
  }
}

function optionIcon(option) {
  if (String(option?.kind || '').startsWith('allow_always')) {
    return ShieldCheck
  }
  if (String(option?.kind || '').startsWith('allow')) {
    return Check
  }
  return X
}
</script>

<template>
  <section
    v-if="permission"
    class="permission-card mx-auto mb-3 w-[min(920px,calc(100%-2rem))] rounded-[8px] border border-[color:var(--qq-border-strong)] bg-[var(--qq-surface)] p-3 shadow-[0_18px_48px_rgba(0,0,0,0.28)] backdrop-blur-xl"
  >
    <div class="mb-3 flex items-start justify-between gap-3">
      <div>
        <div class="flex items-center gap-2">
          <ShieldAlert class="h-4 w-4 text-[var(--qq-accent)]" />
          <h3 class="text-sm font-semibold text-[color:var(--qq-text-primary)]">需要授权</h3>
        </div>
        <p class="mt-1 text-xs leading-5 text-[color:var(--qq-text-tertiary)]">Agent 想执行一个受保护的操作，请在当前对话中确认。</p>
      </div>
      <button
        class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-[4px] text-[color:var(--qq-text-tertiary)] transition hover:bg-[var(--qq-fill-soft)] hover:text-[color:var(--qq-text-primary)]"
        type="button"
        title="取消授权"
        @click="cancel"
      >
        <X class="h-4 w-4" />
      </button>
    </div>

    <div class="permission-dialog-body grid gap-3 md:grid-cols-[minmax(0,1fr)_minmax(16rem,0.8fr)]">
      <div class="flex items-start gap-3 rounded-[6px] border border-[color:var(--qq-border)] bg-[var(--qq-fill-medium)] p-3">
        <ShieldAlert class="mt-0.5 h-5 w-5 shrink-0 text-[var(--qq-accent)]" />
        <div class="min-w-0 flex-1">
          <div class="truncate text-sm font-semibold text-[color:var(--qq-text-primary)]">{{ title }}</div>
          <div class="mt-1 flex flex-wrap gap-2 text-[11px] text-[color:var(--qq-text-tertiary)]">
            <span v-if="toolCall.kind">类型：{{ toolCall.kind }}</span>
            <span v-if="toolCall.toolCallId">调用：{{ toolCall.toolCallId }}</span>
          </div>
        </div>
      </div>

      <div
        v-if="displayOptions.length"
        class="rounded-[6px] border border-[color:var(--qq-border)] bg-[var(--qq-fill-soft)]"
      >
        <div class="border-b border-[color:var(--qq-border)] px-3 py-2 text-xs font-semibold text-[color:var(--qq-text-primary)]">
          授权选项说明
        </div>
        <div class="max-h-28 space-y-2 overflow-auto p-3">
          <div
            v-for="option in displayOptions"
            :key="option.optionId"
            class="grid grid-cols-[5rem_1fr] gap-2 text-xs leading-5"
          >
            <span class="text-[color:var(--qq-accent-strong)]">{{ optionKindLabel(option.kind) }}</span>
            <span class="min-w-0 break-words text-[color:var(--qq-text-secondary)]">{{ option.name || option.optionId }}</span>
          </div>
        </div>
      </div>

      <div
        v-if="rawInput"
        class="rounded-[6px] border border-[color:var(--qq-border)] bg-[rgba(3,24,22,0.34)] md:col-span-2"
      >
        <div class="border-b border-[color:var(--qq-border)] px-3 py-2 text-xs font-semibold text-[color:var(--qq-text-primary)]">
          操作参数
        </div>
        <pre class="max-h-40 overflow-auto p-3 text-xs leading-5 text-[color:var(--qq-text-secondary)]">{{ rawInput }}</pre>
      </div>
    </div>

    <div class="mt-3 flex flex-col gap-3 border-t border-[color:var(--qq-border)] pt-3 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex flex-wrap gap-2">
        <QqButton
          v-for="option in rejectOptions"
          :key="option.optionId"
          variant="ghost"
          :title="option.name || option.optionId"
          @click="choose(option)"
        >
          <component :is="optionIcon(option)" class="h-4 w-4" />
          {{ shortOptionLabel(option) }}
        </QqButton>
        <QqButton v-if="rejectOptions.length === 0" variant="ghost" @click="cancel">取消</QqButton>
      </div>
      <div class="flex flex-wrap gap-2">
        <QqButton
          v-for="option in allowOptions"
          :key="option.optionId"
          :title="option.name || option.optionId"
          @click="choose(option)"
        >
          <component :is="optionIcon(option)" class="h-4 w-4" />
          {{ shortOptionLabel(option) }}
        </QqButton>
      </div>
    </div>
  </section>
</template>

<style scoped>
.permission-card {
  animation: permission-card-enter 180ms ease-out both;
}

.permission-dialog-body {
  padding-right: 0.15rem;
}

.permission-dialog-body pre {
  white-space: pre-wrap;
  word-break: break-word;
}

@keyframes permission-card-enter {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
