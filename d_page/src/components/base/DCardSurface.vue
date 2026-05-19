<script setup>
import { computed, useSlots } from 'vue'

const props = defineProps({
  title: {
    type: String,
    default: '',
  },
  subtitle: {
    type: String,
    default: '',
  },
  padded: {
    type: Boolean,
    default: true,
  },
  emptyText: {
    type: String,
    default: 'No content',
  },
})

const slots = useSlots()
const hasContent = computed(() => Boolean(slots.default))
</script>

<template>
  <section class="d-page-card-surface" :class="{ 'd-page-card-surface--padded': padded }">
    <header v-if="title || subtitle" class="d-page-card-surface__header">
      <h3 v-if="title" class="d-page-card-surface__title">{{ title }}</h3>
      <p v-if="subtitle" class="d-page-card-surface__subtitle">{{ subtitle }}</p>
    </header>
    <div v-if="hasContent" class="d-page-card-surface__body">
      <slot />
    </div>
    <p v-else class="d-page-card-surface__empty">{{ emptyText }}</p>
  </section>
</template>

<style scoped>
.d-page-card-surface {
  display: grid;
  gap: 12px;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  background: var(--d-page-surface, #ffffff);
  color: var(--d-page-text, #172033);
}

.d-page-card-surface--padded {
  padding: 14px;
}

.d-page-card-surface__header {
  display: grid;
  gap: 3px;
}

.d-page-card-surface__title,
.d-page-card-surface__subtitle,
.d-page-card-surface__empty {
  margin: 0;
}

.d-page-card-surface__title {
  font-size: 15px;
  font-weight: 700;
  line-height: 1.35;
}

.d-page-card-surface__subtitle,
.d-page-card-surface__empty {
  color: var(--d-page-muted, #64748b);
  font-size: 13px;
  line-height: 1.45;
}

.d-page-card-surface__body {
  min-width: 0;
}

.d-page-card-surface__empty {
  border: 1px dashed var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  padding: 14px;
  text-align: center;
}
</style>
