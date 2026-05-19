<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: [String, Number],
    default: '',
  },
  label: {
    type: String,
    default: '',
  },
  name: {
    type: String,
    default: '',
  },
  type: {
    type: String,
    default: 'text',
  },
  placeholder: {
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
  clearable: {
    type: Boolean,
    default: false,
  },
  hint: {
    type: String,
    default: '',
  },
  error: {
    type: String,
    default: '',
  },
  emptyPlaceholder: {
    type: String,
    default: 'Enter text',
  },
})

const emit = defineEmits(['update:modelValue', 'input', 'change', 'interact'])

const inputValue = computed(() => props.modelValue ?? '')
const fallbackPlaceholder = computed(() => props.placeholder || props.emptyPlaceholder)
const describedBy = computed(() => {
  if (props.error) return `${props.name || 'd-page-input'}-error`
  if (props.hint) return `${props.name || 'd-page-input'}-hint`
  return undefined
})

function emitValue(type, value) {
  const payload = {
    type,
    source: 'input',
    name: props.name,
    value,
  }

  emit(type, payload)
  emit('interact', payload)
}

function handleInput(event) {
  const value = event.target.value
  emit('update:modelValue', value)
  emitValue('input', value)
}

function handleChange(event) {
  emitValue('change', event.target.value)
}

function clearValue() {
  if (props.disabled) return
  emit('update:modelValue', '')
  emitValue('input', '')
  emitValue('change', '')
}
</script>

<template>
  <label class="d-page-input" :class="{ 'd-page-input--invalid': error, 'd-page-input--disabled': disabled }">
    <span v-if="label" class="d-page-input__label">
      {{ label }}<span v-if="required" aria-hidden="true"> *</span>
    </span>
    <span class="d-page-input__control">
      <input
        class="d-page-input__field"
        :type="type"
        :name="name || undefined"
        :value="inputValue"
        :placeholder="fallbackPlaceholder"
        :disabled="disabled"
        :required="required"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="describedBy"
        @input="handleInput"
        @change="handleChange"
      >
      <button
        v-if="clearable && inputValue"
        class="d-page-input__clear"
        type="button"
        aria-label="Clear input"
        :disabled="disabled"
        @click="clearValue"
      >
        x
      </button>
    </span>
    <span v-if="error" :id="`${name || 'd-page-input'}-error`" class="d-page-input__message d-page-input__message--error">
      {{ error }}
    </span>
    <span v-else-if="hint" :id="`${name || 'd-page-input'}-hint`" class="d-page-input__message">
      {{ hint }}
    </span>
  </label>
</template>

<style scoped>
.d-page-input {
  display: grid;
  gap: 6px;
  color: var(--d-page-text, #172033);
  font-size: 14px;
}

.d-page-input__label {
  font-weight: 650;
  line-height: 1.35;
}

.d-page-input__control {
  position: relative;
  display: flex;
  align-items: center;
}

.d-page-input__field {
  width: 100%;
  min-height: 38px;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  background: var(--d-page-surface, #ffffff);
  color: var(--d-page-text, #172033);
  font: inherit;
  line-height: 1.4;
  padding: 8px 34px 8px 10px;
  transition: border-color 120ms ease, box-shadow 120ms ease;
}

.d-page-input__field::placeholder {
  color: var(--d-page-muted, #64748b);
}

.d-page-input__field:focus {
  border-color: var(--d-page-accent, #14b8a6);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--d-page-accent, #14b8a6) 16%, transparent);
  outline: none;
}

.d-page-input__clear {
  position: absolute;
  right: 6px;
  display: inline-flex;
  width: 24px;
  height: 24px;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--d-page-muted, #64748b);
  cursor: pointer;
  font: inherit;
  line-height: 1;
}

.d-page-input__clear:hover {
  background: color-mix(in srgb, var(--d-page-muted, #64748b) 12%, transparent);
  color: var(--d-page-text, #172033);
}

.d-page-input__message {
  color: var(--d-page-muted, #64748b);
  font-size: 12px;
  line-height: 1.4;
}

.d-page-input__message--error,
.d-page-input--invalid .d-page-input__label {
  color: var(--d-page-danger, #e11d48);
}

.d-page-input--invalid .d-page-input__field {
  border-color: var(--d-page-danger, #e11d48);
}

.d-page-input--disabled {
  opacity: 0.64;
}
</style>
