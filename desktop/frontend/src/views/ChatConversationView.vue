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
import { useProjectsStore } from '@/stores/projects'
import { useSettingsStore } from '@/stores/settings'

const route = useRoute()
const appStore = useAppStore()
const agentsStore = useAgentsStore()
const chatStore = useChatStore()
const conversationsStore = useConversationsStore()
const projectsStore = useProjectsStore()
const settingsStore = useSettingsStore()
const draft = computed({
  get: () => chatStore.composerDraftFor(conversationId.value),
  set: (value) => {
    chatStore.setComposerDraft(conversationId.value, value)
  },
})

const conversationId = computed(() => String(route.params.id || ''))
const conversation = computed(() => conversationsStore.byId(conversationId.value))
const messages = computed(() => conversationsStore.messagesFor(conversationId.value))
const selectedAgentName = computed(() => {
  const agentId = conversation.value?.agentId || settingsStore.settings.gateway.defaultAgentId
  return agentsStore.items.find((agent) => agent.id === agentId)?.name || agentId || '未选择'
})
const currentProjectContext = computed(() => projectsStore.currentProjectContext)
const isConversationStreaming = computed(() => chatStore.isStreaming(conversationId.value))
const conversationSocketState = computed(() => chatStore.socketStateFor(conversationId.value))

watch(
  conversationId,
  async (value) => {
    if (value) {
      if (chatStore.isStreaming(value)) {
        return
      }
      await conversationsStore.loadMessages(settingsStore.settings.gateway.baseUrl, value, { force: true })
    }
  },
  { immediate: true },
)

async function submit() {
  const payload = draft.value
  try {
    draft.value = ''
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
      :project-context="currentProjectContext"
      :socket-status="conversationSocketState"
    />

    <header class="border-b border-white/10 bg-[rgba(18,58,51,0.34)] px-4 py-3 backdrop-blur-xl">
      <p class="text-xs uppercase tracking-[0.2em] text-[color:var(--qq-text-tertiary)]">Conversation</p>
      <h2 class="mt-1 text-lg font-semibold text-slate-50">
        <span data-testid="conversation-header-title">{{ conversation?.title || 'Untitled Conversation' }}</span>
      </h2>
    </header>

    <div class="min-h-0 flex-1 bg-transparent">
      <ChatMessageList :messages="messages" :show-timestamps="settingsStore.settings.ui.showTimestamps" />
    </div>

    <ChatComposer
      v-model="draft"
      :busy="isConversationStreaming"
      :disabled="!draft.trim()"
      :project-context="currentProjectContext"
      @cancel="chatStore.cancelStream(conversationId)"
      @send="submit"
    />
  </section>
</template>
