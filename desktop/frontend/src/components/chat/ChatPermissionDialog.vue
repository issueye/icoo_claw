<script setup>
import { computed } from 'vue'
import { Check, ShieldAlert, ShieldCheck, X } from 'lucide-vue-next'
import QqButton from '@/components/ued/QqButton.vue'
import QqModal from '@/components/ued/QqModal.vue'

const props = defineProps({
  permission: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['select', 'cancel'])

const open = computed(() => Boolean(props.permission))
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
  <QqModal
    :model-value="open"
    title="需要授权"
    description="Agent 想执行一个受保护的操作。"
    @update:model-value="(value) => { if (!value) cancel() }"
  >
    <div v-if="permission" class="permission-dialog-body space-y-3">
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
        class="rounded-[6px] border border-[color:var(--qq-border)] bg-[rgba(3,24,22,0.34)]"
      >
        <div class="border-b border-[color:var(--qq-border)] px-3 py-2 text-xs font-semibold text-[color:var(--qq-text-primary)]">
          操作参数
        </div>
        <pre class="max-h-56 overflow-auto p-3 text-xs leading-5 text-[color:var(--qq-text-secondary)]">{{ rawInput }}</pre>
      </div>
    </div>

    <template #footer>
      <div class="flex flex-1 flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
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
    </template>
  </QqModal>
</template>

<style scoped>
.permission-dialog-body {
  max-height: min(56vh, 520px);
  overflow: hidden auto;
  padding-right: 0.15rem;
}

.permission-dialog-body pre {
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
