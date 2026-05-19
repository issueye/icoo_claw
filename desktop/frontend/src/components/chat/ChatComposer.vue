<script setup>
import { computed } from 'vue'
import { SendHorizonal, Square } from 'lucide-vue-next'
import QqButton from '@/components/ued/QqButton.vue'

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
  <div class="border-t border-white/10 bg-[rgba(18,58,51,0.28)] px-4 py-3 backdrop-blur-xl">
    <div class="qq-panel rounded-[6px] p-3">
      <textarea
        :value="modelValue"
        data-testid="chat-composer-input"
        class="qq-field-control min-h-[84px] w-full resize-none px-3 py-2.5 text-sm leading-6 text-slate-100 outline-none placeholder:text-[color:var(--qq-text-tertiary)]"
        placeholder="输入你的问题，回车发送，Shift + Enter 换行"
        @input="emit('update:modelValue', $event.target.value)"
        @keydown="handleKeydown"
      />
      <div class="mt-3 flex items-center justify-between gap-3 border-t border-white/10 pt-3">
        <p class="min-w-0 text-xs text-[color:var(--qq-text-tertiary)]">
          <span v-if="projectContext" class="block truncate">
            当前项目：{{ projectContext.name }} · {{ projectContext.rootDir }}
          </span>
          <span v-else>
            {{ busy ? '正在通过 WebSocket 接收增量响应' : '聊天标题将使用首条用户输入在本地生成' }}
          </span>
        </p>
        <div class="flex items-center gap-2">
          <QqButton
            v-if="busy"
            variant="ghost"
            @click="emit('cancel')"
          >
            <Square class="h-4 w-4 fill-current" />
            停止
          </QqButton>
          <QqButton
            :disabled="!canSubmit"
            data-testid="chat-composer-send"
            @click="emit('send')"
          >
            <SendHorizonal class="h-4 w-4" />
            发送
          </QqButton>
        </div>
      </div>
    </div>
  </div>
</template>
