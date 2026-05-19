<script setup>
import { computed, ref } from 'vue'
import ChatComposer from '@/components/chat/ChatComposer.vue'
import ChatEmptyState from '@/components/chat/ChatEmptyState.vue'
import ChatStatusBar from '@/components/chat/ChatStatusBar.vue'
import { useAgentsStore } from '@/stores/agents'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { useProjectsStore } from '@/stores/projects'

const appStore = useAppStore()
const chatStore = useChatStore()
const agentsStore = useAgentsStore()
const projectsStore = useProjectsStore()
const draft = ref('')

const selectedAgentName = computed(() => agentsStore.selectedAgent?.name || '未选择')
const currentProjectContext = computed(() => projectsStore.currentProjectContext)

async function submit() {
  const payload = draft.value
  draft.value = ''
  try {
    await chatStore.sendPrompt(payload)
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
      :socket-status="chatStore.socketState"
    />

    <div class="flex min-h-0 flex-1 flex-col">
      <ChatEmptyState />
      <ChatComposer
        v-model="draft"
        :busy="chatStore.streaming"
        :disabled="!draft.trim()"
        :project-context="currentProjectContext"
        @cancel="chatStore.cancelStream"
        @send="submit"
      />
    </div>
  </section>
</template>
