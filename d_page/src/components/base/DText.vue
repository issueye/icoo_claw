<script setup>
import { computed, useSlots } from 'vue'

const props = defineProps({
  text: {
    type: [String, Number],
    default: '',
  },
  tone: {
    type: String,
    default: 'default',
  },
  align: {
    type: String,
    default: 'left',
  },
  emptyText: {
    type: String,
    default: 'No text',
  },
})

const slots = useSlots()
const hasDefaultSlot = computed(() => Boolean(slots.default))
const displayText = computed(() => String(props.text ?? '').trim())
const isEmpty = computed(() => !hasDefaultSlot.value && displayText.value.length === 0)
</script>

<template>
  <p
    class="d-page-text"
    :class="[`d-page-text--${tone}`, { 'd-page-text--empty': isEmpty }]"
    :style="{ textAlign: align }"
  >
    <slot>{{ isEmpty ? emptyText : displayText }}</slot>
  </p>
</template>

<style scoped>
.d-page-text {
  margin: 0;
  color: var(--d-page-text, #172033);
  font-size: 14px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.d-page-text--muted {
  color: var(--d-page-muted, #64748b);
}

.d-page-text--accent {
  color: var(--d-page-accent, #14b8a6);
}

.d-page-text--danger {
  color: var(--d-page-danger, #e11d48);
}

.d-page-text--empty {
  color: var(--d-page-muted, #64748b);
  font-style: italic;
}
</style>
