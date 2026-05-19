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
  placeholder: {
    type: String,
    default: '',
  },
  rows: {
    type: [Number, String],
    default: 4,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
  required: {
    type: Boolean,
    default: false,
  },
  maxLength: {
    type: [Number, String],
    default: null,
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
    default: 'Enter details',
  },
})

const emit = defineEmits(['update:modelValue', 'input', 'change', 'interact'])

const textareaValue = computed(() => props.modelValue ?? '')
const fallbackPlaceholder = computed(() => props.placeholder || props.emptyPlaceholder)
const normalizedRows = computed(() => {
  const parsed = Number(props.rows)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 4
})

function emitValue(type, value) {
  const payload = {
    type,
    source: 'textarea',
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
</script>

<template>
  <label class="d-page-textarea" :class="{ 'd-page-textarea--invalid': error, 'd-page-textarea--disabled': disabled }">
    <span v-if="label" class="d-page-textarea__label">
      {{ label }}<span v-if="required" aria-hidden="true"> *</span>
    </span>
    <textarea
      class="d-page-textarea__field"
      :name="name || undefined"
      :value="textareaValue"
      :placeholder="fallbackPlaceholder"
      :rows="normalizedRows"
      :disabled="disabled"
      :required="required"
      :maxlength="maxLength || undefined"
      :aria-invalid="error ? 'true' : undefined"
      @input="handleInput"
      @change="handleChange"
    />
    <span v-if="error" class="d-page-textarea__message d-page-textarea__message--error">{{ error }}</span>
    <span v-else-if="hint" class="d-page-textarea__message">{{ hint }}</span>
  </label>
</template>

<style scoped>
.d-page-textarea {
  display: grid;
  gap: 6px;
  color: var(--d-page-text, #172033);
  font-size: 14px;
}

.d-page-textarea__label {
  font-weight: 650;
  line-height: 1.35;
}

.d-page-textarea__field {
  width: 100%;
  min-height: 92px;
  resize: vertical;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  background: var(--d-page-surface, #ffffff);
  color: var(--d-page-text, #172033);
  font: inherit;
  line-height: 1.5;
  padding: 9px 10px;
  transition: border-color 120ms ease, box-shadow 120ms ease;
}

.d-page-textarea__field::placeholder {
  color: var(--d-page-muted, #64748b);
}

.d-page-textarea__field:focus {
  border-color: var(--d-page-accent, #14b8a6);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--d-page-accent, #14b8a6) 16%, transparent);
  outline: none;
}

.d-page-textarea__message {
  color: var(--d-page-muted, #64748b);
  font-size: 12px;
  line-height: 1.4;
}

.d-page-textarea__message--error,
.d-page-textarea--invalid .d-page-textarea__label {
  color: var(--d-page-danger, #e11d48);
}

.d-page-textarea--invalid .d-page-textarea__field {
  border-color: var(--d-page-danger, #e11d48);
}

.d-page-textarea--disabled {
  opacity: 0.64;
}
</style>
