<script setup>
import { computed } from 'vue'
import { FolderOpen, Plus, Send, Square } from 'lucide-vue-next'

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
const contextLabel = computed(() => props.projectContext?.name || 'local bridge')

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
  <div class="composer-dock border-white/8 qq-chat-composer-bg px-4 py-3 backdrop-blur-xl">
    <div class="composer-shell mx-auto">
      <textarea
        :value="modelValue"
        data-testid="chat-composer-input"
        class="composer-input"
        rows="3"
        placeholder="要求后续变更"
        @input="emit('update:modelValue', $event.target.value)"
        @keydown="handleKeydown"
      />
      <div class="composer-toolbar">
        <div class="flex min-w-0 items-center gap-2">
          <button
            type="button"
            class="composer-icon-button"
            aria-label="添加上下文（即将支持）"
            title="添加上下文（即将支持）"
            disabled
          >
            <Plus class="h-4 w-4" />
          </button>
          <span
            class="composer-context-chip"
            :title="projectContext ? projectContext.rootDir : 'local bridge'"
          >
            <FolderOpen v-if="projectContext" class="h-3.5 w-3.5" />
            <span class="min-w-0 truncate">{{ contextLabel }}</span>
          </span>
        </div>
        <div class="flex shrink-0 items-center gap-2">
          <button
            v-if="busy"
            type="button"
            class="composer-stop-button"
            @click="emit('cancel')"
          >
            <Square class="h-3 w-3" />
            停止
          </button>
          <button
            :disabled="!canSubmit"
            data-testid="chat-composer-submit"
            type="button"
            class="composer-send-button"
            aria-label="发送"
            title="发送"
            @click="emit('send')"
          >
            <Send class="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.composer-dock {
  box-shadow: 0 -18px 38px rgba(0, 0, 0, 0.18);
}

.composer-shell {
  max-width: min(940px, 100%);
  border: 1px solid rgba(226, 232, 240, 0.14);
  border-radius: 18px;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.045), rgba(255, 255, 255, 0.022)),
    rgba(9, 14, 25, 0.82);
  box-shadow:
    0 16px 42px rgba(0, 0, 0, 0.28),
    inset 0 1px 0 rgba(255, 255, 255, 0.06);
  overflow: hidden;
}

.composer-shell:focus-within {
  border-color: rgba(34, 211, 238, 0.42);
  box-shadow:
    0 18px 46px rgba(0, 0, 0, 0.32),
    0 0 0 3px rgba(34, 211, 238, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.08);
}

.composer-input {
  display: block;
  min-height: 76px;
  max-height: 180px;
  width: 100%;
  resize: none;
  border: 0;
  background: transparent;
  padding: 1rem 1rem 0.35rem;
  color: var(--qq-text-primary);
  font-size: 0.9rem;
  line-height: 1.65;
  outline: none;
}

.composer-input::placeholder {
  color: var(--qq-text-tertiary);
}

.composer-input::-webkit-resizer {
  display: none;
}

.composer-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  min-height: 46px;
  padding: 0.45rem 0.55rem 0.55rem 0.7rem;
}

.composer-icon-button,
.composer-send-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 0;
  transition:
    background-color 150ms ease,
    color 150ms ease,
    opacity 150ms ease,
    transform 150ms ease;
}

.composer-icon-button {
  width: 30px;
  height: 30px;
  border-radius: 999px;
  background: transparent;
  color: var(--qq-text-tertiary);
}

.composer-icon-button:hover {
  background: var(--qq-fill-medium);
  color: var(--qq-text-primary);
}

.composer-icon-button:disabled {
  cursor: default;
  color: var(--qq-text-tertiary);
  opacity: 0.72;
}

.composer-icon-button:disabled:hover {
  background: transparent;
  color: var(--qq-text-tertiary);
}

.composer-context-chip {
  display: inline-flex;
  min-width: 0;
  max-width: min(34vw, 280px);
  align-items: center;
  gap: 0.35rem;
  border-radius: 999px;
  background: var(--qq-fill-medium);
  padding: 0.35rem 0.55rem;
  color: var(--qq-text-secondary);
  font-size: 0.75rem;
  line-height: 1;
}

.composer-stop-button {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border: 1px solid rgba(248, 113, 113, 0.24);
  border-radius: 999px;
  background: rgba(248, 113, 113, 0.12);
  padding: 0.45rem 0.7rem;
  color: #fecaca;
  font-size: 0.75rem;
  line-height: 1;
  transition:
    background-color 150ms ease,
    border-color 150ms ease,
    color 150ms ease;
}

.composer-stop-button:hover {
  border-color: rgba(248, 113, 113, 0.42);
  background: rgba(248, 113, 113, 0.18);
  color: #fee2e2;
}

.composer-send-button {
  width: 32px;
  height: 32px;
  border-radius: 999px;
  background: var(--qq-text-primary);
  color: var(--qq-bg-top);
}

.composer-send-button:not(:disabled):hover {
  transform: translateY(-1px);
  background: #ffffff;
}

.composer-send-button:disabled {
  cursor: not-allowed;
  background: var(--qq-fill-medium);
  color: var(--qq-text-tertiary);
  opacity: 0.72;
}

.composer-icon-button:focus-visible,
.composer-stop-button:focus-visible,
.composer-send-button:focus-visible {
  outline: 2px solid rgba(34, 211, 238, 0.42);
  outline-offset: 2px;
}

html[data-theme="light"] .composer-dock {
  box-shadow: 0 -12px 28px rgba(15, 23, 42, 0.06);
}

html[data-theme="light"] .composer-shell {
  border-color: rgba(15, 23, 42, 0.12);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.96)),
    #ffffff;
  box-shadow:
    0 12px 34px rgba(15, 23, 42, 0.1),
    inset 0 1px 0 rgba(255, 255, 255, 0.95);
}

html[data-theme="light"] .composer-shell:focus-within {
  border-color: rgba(8, 125, 167, 0.34);
  box-shadow:
    0 14px 36px rgba(15, 23, 42, 0.12),
    0 0 0 3px rgba(8, 125, 167, 0.1);
}

html[data-theme="light"] .composer-send-button {
  background: #7a7f88;
  color: #ffffff;
}

html[data-theme="light"] .composer-send-button:not(:disabled):hover {
  background: #111827;
}

html[data-theme="light"] .composer-send-button:disabled {
  background: var(--qq-fill-medium);
  color: var(--qq-text-tertiary);
}

html[data-theme="light"] .composer-stop-button {
  border-color: rgba(190, 18, 60, 0.18);
  background: rgba(190, 18, 60, 0.08);
  color: #be123c;
}

@media (max-width: 720px) {
  .composer-dock {
    padding: 0.65rem;
  }

  .composer-shell {
    border-radius: 14px;
  }

  .composer-context-chip {
    max-width: 46vw;
  }
}
</style>
