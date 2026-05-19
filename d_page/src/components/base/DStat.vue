<script setup>
defineProps({
  label: {
    type: String,
    default: 'Metric',
  },
  value: {
    type: [String, Number],
    default: '-',
  },
  suffix: {
    type: String,
    default: '',
  },
  hint: {
    type: String,
    default: '',
  },
  trend: {
    type: String,
    default: '',
  },
  tone: {
    type: String,
    default: 'default',
  },
})
</script>

<template>
  <section class="d-page-stat" :class="`d-page-stat--${tone}`">
    <span class="d-page-stat__label">{{ label }}</span>
    <strong class="d-page-stat__value">{{ value }}<span v-if="suffix">{{ suffix }}</span></strong>
    <span v-if="trend || hint" class="d-page-stat__meta">{{ trend || hint }}</span>
  </section>
</template>

<style scoped>
.d-page-stat {
  display: grid;
  gap: 6px;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  background: var(--d-page-surface, #ffffff);
  color: var(--d-page-text, #172033);
  padding: 13px;
}

.d-page-stat__label {
  color: var(--d-page-muted, #64748b);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.3;
}

.d-page-stat__value {
  font-size: 26px;
  line-height: 1.15;
  letter-spacing: 0;
}

.d-page-stat__meta {
  color: var(--d-page-muted, #64748b);
  font-size: 12px;
  line-height: 1.4;
}

.d-page-stat--accent .d-page-stat__value {
  color: var(--d-page-accent, #14b8a6);
}

.d-page-stat--danger .d-page-stat__value {
  color: var(--d-page-danger, #e11d48);
}
</style>
