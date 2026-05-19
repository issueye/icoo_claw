<script setup>
import { ChevronDown } from 'lucide-vue-next'

defineProps({
  modelValue: {
    type: [String, Number],
    default: '',
  },
  options: {
    type: Array,
    default: () => [],
  },
  disabled: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['update:modelValue'])

function handleChange(event) {
  emit('update:modelValue', event.target.value)
}
</script>

<template>
  <div class="relative">
    <select
      class="qq-field-control appearance-none px-3 pr-9 text-sm"
      :disabled="disabled"
      :value="modelValue"
      @change="handleChange"
    >
      <option v-for="option in options" :key="option.value" :value="option.value">
        {{ option.label }}
      </option>
    </select>
    <span class="pointer-events-none absolute inset-y-0 right-3 flex items-center text-[color:var(--qq-text-tertiary)]">
      <ChevronDown class="h-4 w-4" />
    </span>
  </div>
</template>
