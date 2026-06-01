<script setup>
import { computed } from 'vue'
import { ShieldAlert } from 'lucide-vue-next'
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
</script>

<template>
  <QqModal
    :model-value="open"
    title="需要授权"
    description="Agent 想执行一个受保护的操作。"
    @update:model-value="(value) => { if (!value) cancel() }"
  >
    <div v-if="permission" class="space-y-4">
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

      <pre
        v-if="rawInput"
        class="max-h-64 overflow-auto rounded-[6px] border border-[color:var(--qq-border)] bg-[rgba(3,24,22,0.34)] p-3 text-xs leading-5 text-[color:var(--qq-text-secondary)]"
      >{{ rawInput }}</pre>
    </div>

    <template #footer>
      <div class="flex flex-1 flex-wrap justify-between gap-3">
        <div class="flex flex-wrap gap-2">
          <QqButton
            v-for="option in rejectOptions"
            :key="option.optionId"
            variant="ghost"
            @click="choose(option)"
          >
            {{ option.name || '拒绝' }}
          </QqButton>
          <QqButton v-if="rejectOptions.length === 0" variant="ghost" @click="cancel">取消</QqButton>
        </div>
        <div class="flex flex-wrap gap-2">
          <QqButton
            v-for="option in allowOptions"
            :key="option.optionId"
            @click="choose(option)"
          >
            {{ option.name || '允许' }}
          </QqButton>
        </div>
      </div>
    </template>
  </QqModal>
</template>

