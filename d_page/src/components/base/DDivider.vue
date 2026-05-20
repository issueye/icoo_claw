<script setup>
const props = defineProps({
  label: {
    type: String,
    default: '',
  },
  orientation: {
    type: String,
    default: 'horizontal',
  },
  tone: {
    type: String,
    default: 'default',
  },
})
</script>

<template>
  <div
    class="d-page-divider"
    :class="[`d-page-divider--${props.orientation}`, `d-page-divider--${props.tone}`]"
    role="separator"
    :aria-orientation="props.orientation === 'vertical' ? 'vertical' : 'horizontal'"
  >
    <span v-if="label && orientation !== 'vertical'" class="d-page-divider__label">{{ label }}</span>
  </div>
</template>

<style scoped>
.d-page-divider {
  color: var(--d-page-muted, #64748b);
}

.d-page-divider--horizontal {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 10px;
  min-height: 1px;
}

.d-page-divider--horizontal::before,
.d-page-divider--horizontal::after {
  display: block;
  height: 1px;
  min-width: 16px;
  flex: 1;
  background: var(--d-page-border, #dbe3ea);
  content: '';
}

.d-page-divider--horizontal:not(:has(.d-page-divider__label))::after {
  display: none;
}

.d-page-divider--vertical {
  width: 1px;
  min-height: 24px;
  align-self: stretch;
  background: var(--d-page-border, #dbe3ea);
}

.d-page-divider__label {
  flex: none;
  font-size: 12px;
  font-weight: 700;
  line-height: 1.4;
}

.d-page-divider--accent::before,
.d-page-divider--accent::after,
.d-page-divider--vertical.d-page-divider--accent {
  background: color-mix(in srgb, var(--d-page-accent, #14b8a6) 38%, var(--d-page-border, #dbe3ea));
}
</style>
