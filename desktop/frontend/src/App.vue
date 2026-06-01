<script setup>
import { onMounted, watch } from 'vue'
import { RouterView } from 'vue-router'
import AppNotifications from '@/components/common/AppNotifications.vue'
import { applyTheme, getStoredTheme } from '@/services/theme'
import { useAgentsStore } from '@/stores/agents'
import { useAppStore } from './stores/app'
import { useChatStore } from './stores/chat'
import { useConversationsStore } from './stores/conversations'
import { useNotificationsStore } from './stores/notifications'
import { useSettingsStore } from './stores/settings'

const appStore = useAppStore()
const agentsStore = useAgentsStore()
const chatStore = useChatStore()
const conversationsStore = useConversationsStore()
const notificationsStore = useNotificationsStore()
const settingsStore = useSettingsStore()

onMounted(() => {
  applyTheme(getStoredTheme())
  appStore.bootstrap()
})

watch(
  () => settingsStore.settings.ui.theme,
  (theme) => {
    applyTheme(theme)
  },
)

watch(
  () => appStore.error,
  (message) => {
    if (!message) {
      return
    }
    notificationsStore.error(message, { title: '网关连接失败' })
    appStore.error = ''
  },
)

watch(
  () => chatStore.error,
  (message) => {
    if (!message) {
      return
    }
    notificationsStore.error(message, { title: '聊天请求失败' })
    chatStore.error = ''
  },
)

watch(
  () => conversationsStore.error,
  (message) => {
    if (!message) {
      return
    }
    notificationsStore.error(message, { title: '会话操作失败' })
    conversationsStore.error = ''
  },
)

watch(
  () => agentsStore.error,
  (message) => {
    if (!message) {
      return
    }
    notificationsStore.error(message, { title: 'Agent 读取失败' })
    agentsStore.error = ''
  },
)
</script>

<template>
  <RouterView />
  <AppNotifications />
</template>
