<script setup>
import { computed } from 'vue'
import { SendHorizonal, Square } from 'lucide-vue-next'

const props = defineProps({
  modelValue: {
    type: String,
    default: '',
  },
  busy: {
    type: Boolean,
    default: false,
  },
  disabled: {
    type: Boolean,
    default: false,
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
  <div class="border-t border-line bg-panel px-5 py-4">
    <div class="rounded-md border border-line bg-[#0c1118] p-3 shadow-shell">
      <textarea
        :value="modelValue"
        data-testid="chat-composer-input"
        class="min-h-[92px] w-full resize-none border-0 bg-transparent text-sm leading-6 text-slate-100 outline-none placeholder:text-slate-500"
        placeholder="输入你的问题，回车发送，Shift + Enter 换行"
        @input="emit('update:modelValue', $event.target.value)"
        @keydown="handleKeydown"
      />
      <div class="mt-3 flex items-center justify-between gap-3 border-t border-line pt-3">
        <p class="text-xs text-slate-500">
          {{ busy ? '正在通过 WebSocket 接收增量响应' : '聊天标题将使用首条用户输入在本地生成' }}
        </p>
        <div class="flex items-center gap-2">
          <button
            v-if="busy"
            class="inline-flex h-10 items-center gap-2 rounded-md border border-danger/30 bg-danger/10 px-3 text-sm text-rose-200 transition hover:border-danger/60"
            type="button"
            @click="emit('cancel')"
          >
            <Square class="h-4 w-4 fill-current" />
            停止
          </button>
          <button
            class="inline-flex h-10 items-center gap-2 rounded-md bg-accent px-4 text-sm font-medium text-slate-950 transition hover:bg-accentStrong disabled:cursor-not-allowed disabled:bg-slate-700 disabled:text-slate-300"
            :disabled="!canSubmit"
            data-testid="chat-composer-send"
            type="button"
            @click="emit('send')"
          >
            <SendHorizonal class="h-4 w-4" />
            发送
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
