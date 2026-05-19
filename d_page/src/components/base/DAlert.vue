<script setup>
const props = defineProps({
  title: {
    type: String,
    default: '',
  },
  message: {
    type: String,
    default: '',
  },
  tone: {
    type: String,
    default: 'info',
  },
  emptyText: {
    type: String,
    default: 'No message',
  },
})
</script>

<template>
  <section class="d-page-alert" :class="`d-page-alert--${props.tone}`" role="status">
    <strong v-if="title" class="d-page-alert__title">{{ title }}</strong>
    <p class="d-page-alert__message">{{ message || emptyText }}</p>
  </section>
</template>

<style scoped>
.d-page-alert {
  display: grid;
  gap: 4px;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  background: #f8fafc;
  color: var(--d-page-text, #172033);
  padding: 11px 12px;
}

.d-page-alert__title {
  font-size: 13px;
  line-height: 1.35;
}

.d-page-alert__message {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.d-page-alert--success {
  border-color: #bbf7d0;
  background: #f0fdf4;
  color: #166534;
}

.d-page-alert--warning {
  border-color: #fde68a;
  background: #fffbeb;
  color: #92400e;
}

.d-page-alert--danger {
  border-color: #fecdd3;
  background: #fff1f2;
  color: #be123c;
}

.d-page-alert--info {
  border-color: #bae6fd;
  background: #f0f9ff;
  color: #075985;
}
</style>
