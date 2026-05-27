<script setup>
import { ref } from 'vue'
import { useRoute } from 'vue-router'
import { Sun, Moon } from 'lucide-vue-next'
import ProjectSwitcher from '@/components/project/ProjectSwitcher.vue'

defineProps({
  items: {
    type: Array,
    required: true,
  },
})

const route = useRoute()
const theme = ref(localStorage.getItem('qq-theme') || 'dark')

function isActive(item) {
  if (item.name === 'chat-home') {
    return route.path.startsWith('/chat')
  }
  return route.name === item.name
}

function toggleTheme() {
  const nextTheme = theme.value === 'dark' ? 'light' : 'dark'
  theme.value = nextTheme
  localStorage.setItem('qq-theme', nextTheme)
  document.documentElement.setAttribute('data-theme', nextTheme)
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

    <!-- 一键切换主题悬浮按钮 (太阳/月亮微交互) -->
    <button
      class="group flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-white/10 bg-[var(--qq-fill-soft)] text-[color:var(--qq-text-secondary)] transition hover:scale-105 hover:border-[color:var(--qq-accent)] hover:bg-[rgba(0,242,254,0.08)] hover:text-[color:var(--qq-accent)] active:scale-95"
      type="button"
      @click="toggleTheme"
    >
      <Sun v-if="theme === 'dark'" class="h-4.5 w-4.5 transition-transform group-hover:rotate-45" />
      <Moon v-else class="h-4.5 w-4.5 transition-transform group-hover:-rotate-12" />
    </button>

    <ProjectSwitcher compact />
  </aside>
</template>
