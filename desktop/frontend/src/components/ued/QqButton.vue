<script setup>
import { computed } from 'vue'

const props = defineProps({
  variant: {
    type: String,
    default: 'primary',
  },
  size: {
    type: String,
    default: 'md',
  },
  block: {
    type: Boolean,
    default: false,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
})

const variantClassMap = {
  primary:
    'border-transparent bg-[linear-gradient(135deg,var(--qq-accent),var(--qq-accent-strong))] text-slate-950 hover:brightness-105',
  secondary:
    'border-[color:var(--qq-border)] bg-[rgba(255,255,255,0.12)] text-[color:var(--qq-text-primary)] hover:bg-[rgba(255,255,255,0.18)]',
  ghost:
    'border-transparent bg-transparent text-[color:var(--qq-text-secondary)] hover:bg-[rgba(255,255,255,0.08)] hover:text-[color:var(--qq-text-primary)]',
  danger:
    'border-transparent bg-[linear-gradient(135deg,#ff8fb8,#ff7584)] text-slate-950 hover:brightness-105',
}

const sizeClassMap = {
  sm: 'h-8 rounded-[4px] px-3 text-sm',
  md: 'h-9 rounded-[4px] px-4 text-sm',
  lg: 'h-10 rounded-[4px] px-5 text-base',
}

const classes = computed(() => [
  'inline-flex items-center justify-center gap-2 border font-medium transition duration-150 ease-out',
  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[rgba(72,255,209,0.28)]',
  'disabled:cursor-not-allowed disabled:opacity-55',
  variantClassMap[props.variant] || variantClassMap.primary,
  sizeClassMap[props.size] || sizeClassMap.md,
  props.block ? 'w-full' : '',
])
</script>

<template>
  <button :class="classes" :disabled="disabled" type="button">
    <slot />
  </button>
</template>
