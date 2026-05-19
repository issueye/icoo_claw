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
  <aside class="flex w-20 shrink-0 flex-col items-center gap-4 border-r border-line bg-[#0b1017] px-3 py-4">
    <div class="flex h-11 w-11 items-center justify-center rounded-md border border-accent/20 bg-accent/10 text-sm font-semibold text-accent">
      IC
    </div>

    <nav class="flex w-full flex-1 flex-col items-center gap-2">
      <RouterLink
        v-for="item in items"
        :key="item.name"
        :to="item.to"
        class="group flex w-full flex-col items-center gap-2 rounded-md px-2 py-3 text-[11px] transition"
        :class="isActive(item) ? 'bg-panelSoft text-slate-50' : 'text-slate-400 hover:bg-panel hover:text-slate-200'"
      >
        <component :is="item.icon" class="h-4 w-4" />
        <span class="text-center leading-tight">{{ item.label }}</span>
      </RouterLink>
    </nav>
  </aside>
</template>
