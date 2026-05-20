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
  required: {
    type: Boolean,
    default: false,
  },
  hint: {
    type: String,
    default: '',
  },
  emptyText: {
    type: String,
    default: 'Checkbox',
  },
})

const emit = defineEmits(['update:modelValue', 'change', 'interact'])

const displayLabel = computed(() => props.label || props.emptyText)

function handleChange(event) {
  const value = event.target.checked
  const payload = {
    type: 'change',
    source: 'checkbox',
    name: props.name,
    value,
  }

  emit('update:modelValue', value)
  emit('change', payload)
  emit('interact', payload)
}
</script>

<template>
  <label class="d-page-checkbox" :class="{ 'd-page-checkbox--disabled': disabled }">
    <input
      class="d-page-checkbox__input"
      type="checkbox"
      :name="name || undefined"
      :checked="modelValue"
      :disabled="disabled"
      :required="required"
      @change="handleChange"
    >
    <span class="d-page-checkbox__box" aria-hidden="true" />
    <span class="d-page-checkbox__content">
      <span class="d-page-checkbox__label">{{ displayLabel }}</span>
      <span v-if="hint" class="d-page-checkbox__hint">{{ hint }}</span>
    </span>
  </label>
</template>

<style scoped>
.d-page-checkbox {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  gap: 9px;
  align-items: start;
  color: var(--d-page-text, #172033);
  cursor: pointer;
  font-size: 14px;
}

.d-page-checkbox__input {
  position: absolute;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}

.d-page-checkbox__box {
  display: inline-flex;
  width: 18px;
  height: 18px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: 4px;
  background: var(--d-page-surface, #ffffff);
  transition: background-color 120ms ease, border-color 120ms ease, box-shadow 120ms ease;
}

.d-page-checkbox__box::after {
  width: 8px;
  height: 4px;
  border-bottom: 2px solid #ffffff;
  border-left: 2px solid #ffffff;
  content: '';
  opacity: 0;
  transform: rotate(-45deg) translateY(-1px);
}

.d-page-checkbox__input:checked + .d-page-checkbox__box {
  border-color: var(--d-page-accent, #14b8a6);
  background: var(--d-page-accent, #14b8a6);
}

.d-page-checkbox__input:checked + .d-page-checkbox__box::after {
  opacity: 1;
}

.d-page-checkbox__input:focus-visible + .d-page-checkbox__box {
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--d-page-accent, #14b8a6) 16%, transparent);
}

.d-page-checkbox__content {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.d-page-checkbox__label {
  font-weight: 650;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.d-page-checkbox__hint {
  color: var(--d-page-muted, #64748b);
  font-size: 12px;
  line-height: 1.4;
}

.d-page-checkbox--disabled {
  cursor: not-allowed;
  opacity: 0.64;
}
</style>
