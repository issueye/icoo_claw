<script setup>
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import ChatMessageList from '@/components/chat/ChatMessageList.vue'
import ChatPermissionDialog from '@/components/chat/ChatPermissionDialog.vue'
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
const pendingPermission = computed(() => chatStore.pendingPermissionFor(conversationId.value))

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

function selectPermissionOption(option) {
  if (!pendingPermission.value || !option?.optionId) {
    return
  }
  chatStore.decidePermission(conversationId.value, pendingPermission.value.id, 'selected', option.optionId)
}

function cancelPermission() {
  if (!pendingPermission.value) {
    return
  }
  chatStore.decidePermission(conversationId.value, pendingPermission.value.id, 'cancelled')
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

    <header class="border-b border-white/8 qq-conv-header-bg px-5 py-2.5 backdrop-blur-xl flex items-center gap-3 min-w-0">
      <div class="min-w-0 flex-1">
        <h2 class="truncate text-sm font-semibold text-[color:var(--qq-text-primary)] leading-tight" data-testid="conversation-header-title">
          {{ conversation?.title || 'Untitled Conversation' }}
        </h2>
        <p v-if="conversation?.agentId" class="mt-0.5 text-[11px] text-[color:var(--qq-text-tertiary)] truncate">
          {{ conversation.agentId }}
        </p>
      </div>
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

    <ChatPermissionDialog
      :permission="pendingPermission"
      @cancel="cancelPermission"
      @select="selectPermissionOption"
    />
  </section>
</template>
