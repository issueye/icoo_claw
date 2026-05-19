<script setup>
import { computed } from 'vue'

const props = defineProps({
  label: {
    type: [String, Number],
    default: '',
  },
  name: {
    type: String,
    default: '',
  },
  value: {
    type: null,
    default: null,
  },
  variant: {
    type: String,
    default: 'primary',
  },
  size: {
    type: String,
    default: 'md',
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  emptyText: {
    type: String,
    default: 'Button',
  },
})

const emit = defineEmits(['click', 'interact'])

const displayLabel = computed(() => String(props.label ?? '').trim())
const isDisabled = computed(() => props.disabled || props.loading)

function handleClick() {
  if (isDisabled.value) return

  const payload = {
    type: 'click',
    source: 'button',
    name: props.name,
    value: props.value,
  }

  emit('click', payload)
  emit('interact', payload)
}
</script>

<template>
  <button
    class="d-page-button"
    :class="[`d-page-button--${variant}`, `d-page-button--${size}`, { 'd-page-button--loading': loading }]"
    type="button"
    :disabled="isDisabled"
    @click="handleClick"
  >
    <span v-if="loading" class="d-page-button__spinner" aria-hidden="true" />
    <span>{{ displayLabel || emptyText }}</span>
  </button>
</template>

<style scoped>
.d-page-button {
  display: inline-flex;
  min-height: 36px;
  max-width: 100%;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px solid transparent;
  border-radius: var(--d-page-radius, 6px);
  padding: 0 14px;
  font: inherit;
  font-size: 14px;
  font-weight: 650;
  line-height: 1;
  letter-spacing: 0;
  cursor: pointer;
  transition: background-color 120ms ease, border-color 120ms ease, color 120ms ease, box-shadow 120ms ease;
}

.d-page-button:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--d-page-accent, #14b8a6) 55%, white);
  outline-offset: 2px;
}

.d-page-button:disabled {
  cursor: not-allowed;
  opacity: 0.62;
}

.d-page-button--primary {
  background: var(--d-page-accent, #14b8a6);
  color: #ffffff;
}

.d-page-button--primary:not(:disabled):hover {
  background: #0f9f92;
}

.d-page-button--secondary {
  border-color: var(--d-page-border, #dbe3ea);
  background: var(--d-page-surface, #ffffff);
  color: var(--d-page-text, #172033);
}

.d-page-button--secondary:not(:disabled):hover,
.d-page-button--ghost:not(:disabled):hover {
  border-color: color-mix(in srgb, var(--d-page-accent, #14b8a6) 45%, var(--d-page-border, #dbe3ea));
  background: color-mix(in srgb, var(--d-page-accent, #14b8a6) 8%, white);
}

.d-page-button--danger {
  background: var(--d-page-danger, #e11d48);
  color: #ffffff;
}

.d-page-button--danger:not(:disabled):hover {
  background: #be123c;
}

.d-page-button--ghost {
  border-color: transparent;
  background: transparent;
  color: var(--d-page-text, #172033);
}

.d-page-button--sm {
  min-height: 30px;
  padding: 0 10px;
  font-size: 12px;
}

.d-page-button--lg {
  min-height: 42px;
  padding: 0 18px;
  font-size: 15px;
}

.d-page-button__spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 999px;
  animation: d-page-button-spin 800ms linear infinite;
}

@keyframes d-page-button-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
