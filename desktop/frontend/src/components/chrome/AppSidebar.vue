<script setup>
import { useRoute } from 'vue-router'

defineProps({
  items: {
    type: Array,
    required: true,
  },
})

const route = useRoute()

function isActive(item) {
  if (item.name === 'chat-home') {
    return route.path.startsWith('/chat')
  }
  return route.name === item.name
}
</script>

<template>
  <aside class="qq-panel-strong flex w-[76px] shrink-0 flex-col items-center gap-3 border-r border-white/10 px-2 py-3">
    <div class="flex h-10 w-10 items-center justify-center rounded-[4px] border border-white/10 bg-[rgba(54,220,200,0.14)] text-sm font-semibold text-[var(--qq-accent)]">
      IC
    </div>

    <nav class="flex w-full flex-1 flex-col items-center gap-1">
      <RouterLink
        v-for="item in items"
        :key="item.name"
        :to="item.to"
        class="group flex w-full flex-col items-center gap-1.5 rounded-[4px] px-2 py-2.5 text-[11px] transition"
        :class="
          isActive(item)
            ? 'bg-[rgba(255,255,255,0.12)] text-white'
            : 'text-[color:var(--qq-text-tertiary)] hover:bg-[rgba(255,255,255,0.06)] hover:text-[color:var(--qq-text-primary)]'
        "
      >
        <component :is="item.icon" class="h-4 w-4" />
        <span class="text-center leading-tight">{{ item.label }}</span>
      </RouterLink>
    </nav>
  </aside>
</template>
