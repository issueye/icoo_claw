<script setup>
import { computed } from 'vue'

const props = defineProps({
  items: {
    type: Array,
    default: () => [],
  },
  title: {
    type: String,
    default: '',
  },
  emptyText: {
    type: String,
    default: 'No items',
  },
})

const normalizedItems = computed(() => props.items.map((item, index) => {
  if (item && typeof item === 'object') {
    return {
      key: item.id || item.key || index,
      label: String(item.label || item.title || item.name || item.value || ''),
      description: item.description || item.subtitle || '',
    }
  }

  return {
    key: index,
    label: String(item),
    description: '',
  }
}))
</script>

<template>
  <section class="d-page-list">
    <h3 v-if="title" class="d-page-list__title">{{ title }}</h3>
    <ul v-if="normalizedItems.length" class="d-page-list__items">
      <li v-for="item in normalizedItems" :key="item.key" class="d-page-list__item">
        <span class="d-page-list__label">{{ item.label }}</span>
        <span v-if="item.description" class="d-page-list__description">{{ item.description }}</span>
      </li>
    </ul>
    <p v-else class="d-page-list__empty">{{ emptyText }}</p>
  </section>
</template>

<style scoped>
.d-page-list {
  display: grid;
  gap: 8px;
  color: var(--d-page-text, #172033);
}

.d-page-list__title,
.d-page-list__empty {
  margin: 0;
}

.d-page-list__title {
  font-size: 14px;
  font-weight: 700;
  line-height: 1.35;
}

.d-page-list__items {
  display: grid;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.d-page-list__item {
  display: grid;
  gap: 2px;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  background: var(--d-page-surface, #ffffff);
  padding: 10px;
}

.d-page-list__label {
  font-size: 13px;
  font-weight: 650;
  line-height: 1.35;
}

.d-page-list__description,
.d-page-list__empty {
  color: var(--d-page-muted, #64748b);
  font-size: 12px;
  line-height: 1.45;
}

.d-page-list__empty {
  border: 1px dashed var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  padding: 12px;
  text-align: center;
}
</style>
