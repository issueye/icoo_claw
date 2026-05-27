<script setup>
import { computed } from 'vue'
import { Send, Square } from 'lucide-vue-next'
import QqButton from '@/components/ued/QqButton.vue'

const props = defineProps({
  modelValue: {
    type: String,
    default: '',
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  busy: {
    type: Boolean,
    default: false,
  },
  projectContext: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['update:modelValue', 'send', 'cancel'])
const canSubmit = computed(() => !props.disabled && props.modelValue.trim() && !props.busy)

function handleKeydown(event) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    if (canSubmit.value) {
      emit('send')
    }
  }
}
</script>

<template>
  <div class="border-t border-white/8 qq-chat-composer-bg px-3 py-2.5 backdrop-blur-xl">
    <div class="qq-panel rounded-[8px] overflow-hidden">
      <textarea
        :value="modelValue"
        data-testid="chat-composer-input"
        class="qq-field-control min-h-[72px] w-full resize-none border-0 bg-transparent px-3.5 pt-3 pb-1.5 text-sm leading-6 text-[color:var(--qq-text-primary)] outline-none placeholder:text-[color:var(--qq-text-tertiary)]"
        placeholder="输入你的问题，回车发送，Shift + Enter 换行"
        @input="emit('update:modelValue', $event.target.value)"
        @keydown="handleKeydown"
      />
      <div class="flex items-center justify-between gap-3 border-t border-white/6 px-3 py-2">
        <p class="min-w-0 text-[11px] text-[color:var(--qq-text-tertiary)]">
          <span v-if="projectContext" class="block truncate">
            {{ projectContext.name }} · {{ projectContext.rootDir }}
          </span>
          <span v-else class="opacity-70">
            {{ busy ? '正在接收流式响应...' : '回车发送 · Shift+Enter 换行' }}
          </span>
        </p>
        <div class="flex shrink-0 items-center gap-2">
          <button
            v-if="busy"
            type="button"
            class="inline-flex items-center gap-1.5 rounded-[6px] border border-white/10 bg-white/6 px-3 py-1.5 text-xs text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-white/10 hover:text-white"
            @click="emit('cancel')"
          >
            <Square class="h-3 w-3" />
            停止
          </button>
          <button
            :disabled="!canSubmit"
            data-testid="chat-composer-submit"
            type="button"
            class="inline-flex items-center gap-1.5 rounded-[6px] px-3.5 py-1.5 text-xs font-semibold transition"
            :class="canSubmit
              ? 'bg-[linear-gradient(135deg,var(--qq-accent),var(--qq-accent-strong))] text-slate-950 shadow-[0_0_12px_rgba(0,242,254,0.25)] hover:brightness-110 hover:shadow-[0_0_18px_rgba(0,242,254,0.35)]'
              : 'bg-white/6 text-[color:var(--qq-text-tertiary)] cursor-not-allowed opacity-50'
            "
            @click="emit('send')"
          >
            <Send class="h-3 w-3" />
            发送
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
