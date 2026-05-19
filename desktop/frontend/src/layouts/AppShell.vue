<script setup>
import { computed } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { LayoutTemplate, MessageSquareText, PlugZap, RefreshCw, Search, Settings2, Wrench } from 'lucide-vue-next'
import AppSidebar from '@/components/chrome/AppSidebar.vue'
import ConversationList from '@/components/conversation/ConversationList.vue'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { useConversationsStore } from '@/stores/conversations'
import { useSettingsStore } from '@/stores/settings'

const appStore = useAppStore()
const chatStore = useChatStore()
const conversationsStore = useConversationsStore()
const settingsStore = useSettingsStore()
const route = useRoute()
const router = useRouter()

const navItems = [
  { name: 'chat-home', label: 'Chat', icon: MessageSquareText, to: '/chat' },
  { name: 'ued', label: 'UED', icon: LayoutTemplate, to: '/ued' },
  { name: 'search', label: 'Search', icon: Search, to: '/search' },
  { name: 'skills', label: 'Skills', icon: Wrench, to: '/skills' },
  { name: 'plugins', label: 'Plugins', icon: PlugZap, to: '/plugins' },
  { name: 'settings', label: 'Settings', icon: Settings2, to: '/settings' },
]

const activeConversationId = computed(() => String(route.params.id || ''))
const shellStatusLabel = computed(() => {
  if (chatStore.streaming) {
    return '回答生成中'
  }
  if (appStore.gatewayStatus === 'connected') {
    return '网关已连接'
  }
  if (appStore.gatewayStatus === 'offline') {
    return '网关离线'
  }
  return '等待初始化'
})

async function refresh() {
  await appStore.refreshGatewayData()
}

function newChat() {
  router.push('/chat')
}

async function deleteConversation(conversationId) {
  await conversationsStore.deleteConversation(settingsStore.settings.gateway.baseUrl, conversationId)
  if (activeConversationId.value === conversationId) {
    router.push('/chat')
  }
}
</script>

<template>
  <div class="flex h-screen flex-col bg-ink text-slate-100">
    <header class="flex h-14 shrink-0 items-center justify-between border-b border-line bg-panel px-5">
      <div class="min-w-0">
        <p class="text-[11px] uppercase tracking-[0.2em] text-accent/80">Icoo Claw Desktop</p>
        <div class="flex items-center gap-3">
          <h1 class="truncate text-sm font-semibold text-slate-50">Gateway Chat Shell</h1>
          <span class="rounded-full border border-line bg-panelSoft px-2 py-0.5 text-[11px] text-slate-300">
            {{ shellStatusLabel }}
          </span>
        </div>
      </div>
      <div class="flex items-center gap-3">
        <span v-if="appStore.lastRefreshedAt" class="hidden text-xs text-slate-400 md:inline">
          {{ new Date(appStore.lastRefreshedAt).toLocaleTimeString() }}
        </span>
        <button
          class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-line bg-panelSoft text-slate-300 transition hover:border-accent/60 hover:text-accent"
          type="button"
          title="刷新网关状态"
          @click="refresh"
        >
          <RefreshCw class="h-4 w-4" />
        </button>
      </div>
    </header>

    <div class="flex min-h-0 flex-1">
      <AppSidebar :items="navItems" />

      <ConversationList
        class="hidden md:flex"
        :active-id="activeConversationId"
        :conversations="conversationsStore.items"
        :deleting-id="conversationsStore.deletingId"
        :loading="conversationsStore.loading"
        :streaming="chatStore.streaming"
        @delete="deleteConversation"
        @new-chat="newChat"
        @refresh="refresh"
      />

      <main class="relative min-w-0 flex-1 bg-[linear-gradient(180deg,rgba(255,255,255,0.02),transparent)]">
        <RouterView />
      </main>
    </div>
  </div>
</template>
