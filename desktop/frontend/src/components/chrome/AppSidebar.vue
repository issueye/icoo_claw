<script setup>
import { useRoute } from 'vue-router'
import ProjectSwitcher from '@/components/project/ProjectSwitcher.vue'

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
    <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-[4px] border border-white/10 bg-[rgba(0,242,254,0.14)] text-sm font-semibold text-[var(--qq-accent)]">
      IC
    </div>

    <nav class="flex w-full flex-1 flex-col items-center gap-1">
      <RouterLink
        v-for="item in items"
        :key="item.name"
        :to="item.to"
        class="group relative flex w-full flex-col items-center gap-1.5 rounded-[5px] px-2 py-2.5 text-[10px] transition-all duration-150"
        :class="
          isActive(item)
            ? 'bg-[rgba(0,242,254,0.10)] text-[color:var(--qq-accent)] shadow-[inset_0_0_12px_rgba(0,242,254,0.04)]'
            : 'text-[color:var(--qq-text-tertiary)] hover:bg-[var(--qq-fill-soft)] hover:text-[color:var(--qq-text-secondary)]'
        "
      >
        <!-- 活跃指示条 -->
        <span
          v-if="isActive(item)"
          class="absolute left-0 top-1/2 h-5 w-0.5 -translate-y-1/2 rounded-r-full bg-[var(--qq-accent)] shadow-[0_0_6px_var(--qq-accent)]"
        />
        <component :is="item.icon" class="h-4 w-4 transition-transform group-hover:scale-110" />
        <span class="text-center leading-tight font-medium">{{ item.label }}</span>
      </RouterLink>
    </nav>

    <ProjectSwitcher compact />
  </aside>
</template>
