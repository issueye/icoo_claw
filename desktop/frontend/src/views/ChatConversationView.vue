<script setup>
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import ChatMessageList from '@/components/chat/ChatMessageList.vue'
import ChatStatusBar from '@/components/chat/ChatStatusBar.vue'
import { useAgentsStore } from '@/stores/agents'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { useConversationsStore } from '@/stores/conversations'
import { useSettingsStore } from '@/stores/settings'

const route = useRoute()
const appStore = useAppStore()
const agentsStore = useAgentsStore()
const chatStore = useChatStore()
const conversationsStore = useConversationsStore()
const settingsStore = useSettingsStore()
const draft = computed({
  get: () => chatStore.composerDraft || '',
  set: (value) => {
    chatStore.composerDraft = value
  },
})

const conversationId = computed(() => String(route.params.id || ''))
const conversation = computed(() => conversationsStore.byId(conversationId.value))
const messages = computed(() => conversationsStore.messagesFor(conversationId.value))
const selectedAgentName = computed(() => agentsStore.selectedAgent?.name || conversation.value?.agentId || '未选择')

watch(
  conversationId,
  async (value) => {
    if (value) {
      await conversationsStore.loadMessages(settingsStore.settings.gateway.baseUrl, value, { force: true })
    }
  },
  { immediate: true },
)

async function submit() {
  const payload = draft.value
  draft.value = ''
  try {
    await chatStore.sendPrompt(payload, conversationId.value)
  } catch (error) {
    draft.value = payload
    chatStore.error = error?.message || String(error)
  }
}
</script>

<template>
  <section class="flex h-full min-h-0 flex-col">
    <ChatStatusBar
      :agent-name="selectedAgentName"
      :gateway-status="appStore.gatewayStatus"
      :socket-status="chatStore.socketState"
    />

    <header class="border-b border-line bg-panel px-5 py-4">
      <p class="text-xs uppercase tracking-[0.2em] text-slate-500">Conversation</p>
      <h2 class="mt-1 text-lg font-semibold text-slate-50">
        <span data-testid="conversation-header-title">{{ conversation?.title || 'Untitled Conversation' }}</span>
      </h2>
    </header>

    <div class="min-h-0 flex-1 bg-[linear-gradient(180deg,rgba(255,255,255,0.02),transparent)]">
      <ChatMessageList :messages="messages" :show-timestamps="settingsStore.settings.ui.showTimestamps" />
    </div>

    <ChatComposer
      v-model="draft"
      :busy="chatStore.streaming"
      :disabled="!draft.trim()"
      @cancel="chatStore.cancelStream"
      @send="submit"
    />
  </section>
</template>
