<script setup>
import { computed, useSlots } from 'vue'

const props = defineProps({
  text: {
    type: [String, Number],
    default: '',
  },
  level: {
    type: [Number, String],
    default: 2,
  },
  overline: {
    type: String,
    default: '',
  },
  subtitle: {
    type: String,
    default: '',
  },
  emptyText: {
    type: String,
    default: 'Untitled',
  },
})

const slots = useSlots()
const headingLevel = computed(() => {
  const parsed = Number(props.level)
  return Number.isInteger(parsed) && parsed >= 1 && parsed <= 6 ? parsed : 2
})
const headingTag = computed(() => `h${headingLevel.value}`)
const displayText = computed(() => String(props.text ?? '').trim())
const hasDefaultSlot = computed(() => Boolean(slots.default))
const isEmpty = computed(() => !hasDefaultSlot.value && displayText.value.length === 0)
</script>

<template>
  <header class="d-page-heading" :class="`d-page-heading--h${headingLevel}`">
    <p v-if="overline" class="d-page-heading__overline">{{ overline }}</p>
    <component :is="headingTag" class="d-page-heading__title" :class="{ 'd-page-heading__title--empty': isEmpty }">
      <slot>{{ isEmpty ? emptyText : displayText }}</slot>
    </component>
    <p v-if="subtitle" class="d-page-heading__subtitle">{{ subtitle }}</p>
  </header>
</template>

<style scoped>
.d-page-heading {
  display: grid;
  gap: 4px;
  color: var(--d-page-text, #172033);
}

.d-page-heading__overline {
  margin: 0;
  color: var(--d-page-accent, #14b8a6);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.4;
  text-transform: uppercase;
}

.d-page-heading__title {
  margin: 0;
  font-weight: 700;
  letter-spacing: 0;
  line-height: 1.2;
  overflow-wrap: anywhere;
}

.d-page-heading--h1 .d-page-heading__title {
  font-size: 30px;
}

.d-page-heading--h2 .d-page-heading__title {
  font-size: 24px;
}

.d-page-heading--h3 .d-page-heading__title {
  font-size: 20px;
}

.d-page-heading--h4 .d-page-heading__title,
.d-page-heading--h5 .d-page-heading__title,
.d-page-heading--h6 .d-page-heading__title {
  font-size: 16px;
}

.d-page-heading__title--empty,
.d-page-heading__subtitle {
  color: var(--d-page-muted, #64748b);
}

.d-page-heading__title--empty {
  font-style: italic;
}

.d-page-heading__subtitle {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
  overflow-wrap: anywhere;
}
</style>
