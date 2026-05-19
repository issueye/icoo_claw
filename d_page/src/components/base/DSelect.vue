<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: {
    type: [String, Number, Boolean],
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
  options: {
    type: Array,
    default: () => [],
  },
  placeholder: {
    type: String,
    default: 'Select an option',
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
  error: {
    type: String,
    default: '',
  },
})

const emit = defineEmits(['update:modelValue', 'change', 'interact'])

const normalizedOptions = computed(() => props.options.map((option) => {
  if (option && typeof option === 'object') {
    return {
      label: String(option.label ?? option.value ?? ''),
      value: option.value ?? '',
      disabled: Boolean(option.disabled),
    }
  }

  return {
    label: String(option),
    value: option,
    disabled: false,
  }
}))

function handleChange(event) {
  const value = event.target.value
  const payload = {
    type: 'change',
    source: 'select',
    name: props.name,
    value,
  }

  emit('update:modelValue', value)
  emit('change', payload)
  emit('interact', payload)
}
</script>

<template>
  <label class="d-page-select" :class="{ 'd-page-select--invalid': error, 'd-page-select--disabled': disabled }">
    <span v-if="label" class="d-page-select__label">
      {{ label }}<span v-if="required" aria-hidden="true"> *</span>
    </span>
    <select
      class="d-page-select__field"
      :name="name || undefined"
      :value="modelValue"
      :disabled="disabled"
      :required="required"
      :aria-invalid="error ? 'true' : undefined"
      @change="handleChange"
    >
      <option value="" :disabled="required">{{ placeholder }}</option>
      <option
        v-for="option in normalizedOptions"
        :key="`${option.value}`"
        :value="option.value"
        :disabled="option.disabled"
      >
        {{ option.label }}
      </option>
    </select>
    <span v-if="error" class="d-page-select__message d-page-select__message--error">{{ error }}</span>
    <span v-else-if="hint" class="d-page-select__message">{{ hint }}</span>
  </label>
</template>

<style scoped>
.d-page-select {
  display: grid;
  gap: 6px;
  color: var(--d-page-text, #172033);
  font-size: 14px;
}

.d-page-select__label {
  font-weight: 650;
  line-height: 1.35;
}

.d-page-select__field {
  width: 100%;
  min-height: 38px;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  background: var(--d-page-surface, #ffffff);
  color: var(--d-page-text, #172033);
  font: inherit;
  line-height: 1.4;
  padding: 8px 10px;
  transition: border-color 120ms ease, box-shadow 120ms ease;
}

.d-page-select__field:focus {
  border-color: var(--d-page-accent, #14b8a6);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--d-page-accent, #14b8a6) 16%, transparent);
  outline: none;
}

.d-page-select__message {
  color: var(--d-page-muted, #64748b);
  font-size: 12px;
  line-height: 1.4;
}

.d-page-select__message--error,
.d-page-select--invalid .d-page-select__label {
  color: var(--d-page-danger, #e11d48);
}

.d-page-select--invalid .d-page-select__field {
  border-color: var(--d-page-danger, #e11d48);
}

.d-page-select--disabled {
  opacity: 0.64;
}
</style>
