<script setup>
import { computed } from 'vue'

const props = defineProps({
  label: {
    type: [String, Number, Boolean],
    default: '',
  },
  tone: {
    type: String,
    default: 'default',
  },
  size: {
    type: String,
    default: 'md',
  },
  emptyText: {
    type: String,
    default: 'Tag',
  },
})

const displayLabel = computed(() => String(props.label ?? '').trim() || props.emptyText)
</script>

<template>
  <span class="d-page-tag" :class="[`d-page-tag--${tone}`, `d-page-tag--${size}`]">
    {{ displayLabel }}
  </span>
</template>

<style scoped>
.d-page-tag {
  display: inline-flex;
  max-width: 100%;
  min-height: 24px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: 999px;
  background: var(--d-page-bg, #f8fafc);
  color: var(--d-page-text, #172033);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.2;
  overflow-wrap: anywhere;
  padding: 3px 9px;
}

.d-page-tag--sm {
  min-height: 20px;
  padding: 2px 7px;
  font-size: 11px;
}

.d-page-tag--lg {
  min-height: 28px;
  padding: 4px 11px;
  font-size: 13px;
}

.d-page-tag--accent {
  border-color: color-mix(in srgb, var(--d-page-accent, #14b8a6) 35%, white);
  background: color-mix(in srgb, var(--d-page-accent, #14b8a6) 12%, white);
  color: #0f766e;
}

.d-page-tag--success {
  border-color: #bbf7d0;
  background: #f0fdf4;
  color: #166534;
}

.d-page-tag--warning {
  border-color: #fde68a;
  background: #fffbeb;
  color: #92400e;
}

.d-page-tag--danger {
  border-color: #fecdd3;
  background: #fff1f2;
  color: #be123c;
}

.d-page-tag--muted {
  color: var(--d-page-muted, #64748b);
}
</style>
