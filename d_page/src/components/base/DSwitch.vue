<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false,
  },
  label: {
    type: String,
    default: '',
  },
  name: {
    type: String,
    default: '',
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  hint: {
    type: String,
    default: '',
  },
  onText: {
    type: String,
    default: 'On',
  },
  offText: {
    type: String,
    default: 'Off',
  },
  emptyText: {
    type: String,
    default: 'Switch',
  },
})

const emit = defineEmits(['update:modelValue', 'change', 'interact'])

const displayLabel = computed(() => props.label || props.emptyText)
const stateText = computed(() => (props.modelValue ? props.onText : props.offText))

function handleChange(event) {
  const value = event.target.checked
  const payload = {
    type: 'change',
    source: 'switch',
    name: props.name,
    value,
  }

  emit('update:modelValue', value)
  emit('change', payload)
  emit('interact', payload)
}
</script>

<template>
  <label class="d-page-switch" :class="{ 'd-page-switch--disabled': disabled }">
    <input
      class="d-page-switch__input"
      type="checkbox"
      role="switch"
      :name="name || undefined"
      :checked="modelValue"
      :disabled="disabled"
      @change="handleChange"
    >
    <span class="d-page-switch__track" aria-hidden="true">
      <span class="d-page-switch__thumb" />
    </span>
    <span class="d-page-switch__content">
      <span class="d-page-switch__label">{{ displayLabel }}</span>
      <span class="d-page-switch__hint">{{ hint || stateText }}</span>
    </span>
  </label>
</template>

<style scoped>
.d-page-switch {
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
  color: var(--d-page-text, #172033);
  cursor: pointer;
  font-size: 14px;
}

.d-page-switch__input {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}

.d-page-switch__track {
  display: inline-flex;
  width: 42px;
  height: 24px;
  align-items: center;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: 999px;
  background: #e5eaf0;
  padding: 2px;
  transition: background-color 120ms ease, border-color 120ms ease, box-shadow 120ms ease;
}

.d-page-switch__thumb {
  width: 18px;
  height: 18px;
  border-radius: 999px;
  background: #ffffff;
  box-shadow: 0 1px 2px rgb(15 23 42 / 18%);
  transition: transform 120ms ease;
}

.d-page-switch__input:checked + .d-page-switch__track {
  border-color: var(--d-page-accent, #14b8a6);
  background: var(--d-page-accent, #14b8a6);
}

.d-page-switch__input:checked + .d-page-switch__track .d-page-switch__thumb {
  transform: translateX(18px);
}

.d-page-switch__input:focus-visible + .d-page-switch__track {
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--d-page-accent, #14b8a6) 16%, transparent);
}

.d-page-switch__content {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.d-page-switch__label {
  font-weight: 650;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.d-page-switch__hint {
  color: var(--d-page-muted, #64748b);
  font-size: 12px;
  line-height: 1.4;
}

.d-page-switch--disabled {
  cursor: not-allowed;
  opacity: 0.64;
}
</style>
