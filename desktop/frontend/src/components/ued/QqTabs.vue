<script setup>
defineProps({
  modelValue: {
    type: String,
    default: '',
  },
  tabs: {
    type: Array,
    default: () => [],
  },
})

const emit = defineEmits(['update:modelValue'])
</script>

<template>
  <div class="qq-panel inline-flex flex-wrap gap-2 rounded-[24px] p-2">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      class="rounded-[18px] px-4 py-2.5 text-sm font-medium transition"
      :class="
        modelValue === tab.value
          ? 'bg-[rgba(255,255,255,0.18)] text-[color:var(--qq-text-primary)] shadow-[0_10px_24px_rgba(8,30,25,0.12)]'
          : 'text-[color:var(--qq-text-secondary)] hover:bg-[rgba(255,255,255,0.08)] hover:text-[color:var(--qq-text-primary)]'
      "
      type="button"
      @click="emit('update:modelValue', tab.value)"
    >
      {{ tab.label }}
    </button>
  </div>
</template>
