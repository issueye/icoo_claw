<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { Activity, Bot, CalendarClock, Check, KeyRound, MessageSquareText, Minus, Palette, PlugZap, RefreshCw, Search, Settings2, Square, Wrench, X } from 'lucide-vue-next'
import { Window } from '@wailsio/runtime'
import AppSidebar from '@/components/chrome/AppSidebar.vue'
import ConversationList from '@/components/conversation/ConversationList.vue'
import { THEME_OPTIONS, applyTheme } from '@/services/theme'
import QqButton from '@/components/ued/QqButton.vue'
import QqFormField from '@/components/ued/QqFormField.vue'
import QqInput from '@/components/ued/QqInput.vue'
import QqModal from '@/components/ued/QqModal.vue'
import { openACPMonitorWindow } from '@/services/wails/windows'
import { useAgentsStore } from '@/stores/agents'
import { useAcpMonitorStore } from '@/stores/acpMonitor'
import { useAppStore } from '@/stores/app'
import { useChatStore } from '@/stores/chat'
import { useConversationsStore } from '@/stores/conversations'
import { useNotificationsStore } from '@/stores/notifications'
import { useProjectsStore } from '@/stores/projects'
import { useSettingsStore } from '@/stores/settings'

const appStore = useAppStore()
const acpMonitorStore = useAcpMonitorStore()
const agentsStore = useAgentsStore()
const chatStore = useChatStore()
const conversationsStore = useConversationsStore()
const notificationsStore = useNotificationsStore()
const projectsStore = useProjectsStore()
const settingsStore = useSettingsStore()
const route = useRoute()
const router = useRouter()
const gatewayDialog = reactive({
  open: false,
  draftBaseUrl: '',
  saving: false,
  connecting: false,
})
const conversationListCollapsed = ref(false)
const conversationDeleteDialog = reactive({
  open: false,
  conversationId: '',
})
const themePopover = reactive({
  open: false,
  draft: '',
  saving: false,
})

const navItems = [
  { name: 'chat-home', label: 'AI 会话', icon: MessageSquareText, to: '/chat' },
  { name: 'search', label: '全局搜索', icon: Search, to: '/search' },
  { name: 'agents', label: '智能体', icon: Bot, to: '/agents' },
  { name: 'providers', label: '服务商', icon: KeyRound, to: '/providers' },
  { name: 'scheduled-tasks', label: '任务调度', icon: CalendarClock, to: '/scheduled-tasks' },
  { name: 'skills', label: '技能', icon: Wrench, to: '/skills' },
  { name: 'plugins', label: '扩展插件', icon: PlugZap, to: '/plugins' },
  { name: 'settings', label: '设置', icon: Settings2, to: '/settings' },
]

const activeConversationId = computed(() => String(route.params.id || ''))
const isChatRoute = computed(() => route.path === '/chat' || route.path.startsWith('/chat/'))
const currentSectionLabel = computed(() => navItems.find((item) => route.path === item.to || route.path.startsWith(`${item.to}/`))?.label || 'Workspace')
const shellStatusLabel = computed(() => {
  if (chatStore.anyStreaming) {
    return chatStore.runningConversationIds.length > 1 ? `${chatStore.runningConversationIds.length} 个会话生成中` : '回答生成中'
  }
  if (appStore.gatewayStatus === 'connected') {
    return '网关已连接'
  }
  if (appStore.gatewayStatus === 'unconfigured') {
    return '网关未配置'
  }
  if (appStore.gatewayStatus === 'offline') {
    return '网关离线'
  }
  return '等待初始化'
})
const statusToneClass = computed(() => {
  if (appStore.gatewayStatus === 'connected') {
    return 'bg-emerald-300'
  }
  if (appStore.gatewayStatus === 'offline') {
    return 'bg-rose-300'
  }
  if (appStore.gatewayStatus === 'unconfigured') {
    return 'bg-amber-300'
  }
  return 'bg-slate-300'
})
const gatewayBaseUrlLabel = computed(() => settingsStore.settings.gateway.baseUrl || '未配置网关地址')
const defaultAgentLabel = computed(() => agentsStore.selectedAgent?.name || settingsStore.settings.gateway.defaultAgentId || '未选择 Agent')
const currentProjectLabel = computed(() => projectsStore.currentProject?.name || '无项目')
const lastRefreshedLabel = computed(() => (appStore.lastRefreshedAt ? new Date(appStore.lastRefreshedAt).toLocaleTimeString() : '未刷新'))
const gatewayDialogBusy = computed(() => gatewayDialog.saving || gatewayDialog.connecting)
const gatewayHealthLabel = computed(() => {
  if (!appStore.gatewayInfo) {
    return '未获取'
  }
  return [appStore.gatewayInfo.service, appStore.gatewayInfo.status].filter(Boolean).join(' / ') || '已响应'
})
const savedTheme = computed(() => settingsStore.settings.ui.theme || 'dark')
const selectedTheme = computed(() => THEME_OPTIONS.find((theme) => theme.value === themePopover.draft) || THEME_OPTIONS[0])

async function refresh() {
  await appStore.refreshGatewayData()
}

async function minimiseWindow() {
  try {
    await Window.Minimise()
  } catch {
    // Browser preview has no native window bridge.
  }
}

async function toggleMaximiseWindow() {
  try {
    await Window.ToggleMaximise()
  } catch {
    // Browser preview has no native window bridge.
  }
}

async function closeWindow() {
  try {
    await Window.Close()
  } catch {
    // Browser preview has no native window bridge.
  }
}

function newChat() {
  router.push('/chat')
}

function deleteConversation(conversationId) {
  conversationDeleteDialog.conversationId = conversationId
  conversationDeleteDialog.open = true
}

function closeConversationDeleteDialog() {
  conversationDeleteDialog.open = false
  conversationDeleteDialog.conversationId = ''
}

async function confirmDeleteConversation() {
  const conversationId = conversationDeleteDialog.conversationId
  if (!conversationId) {
    return
  }
  await conversationsStore.deleteConversation(settingsStore.settings.gateway.baseUrl, conversationId)
  if (activeConversationId.value === conversationId) {
    router.push('/chat')
  }
  closeConversationDeleteDialog()
}

function openGatewayDialog() {
  gatewayDialog.draftBaseUrl = settingsStore.settings.gateway.baseUrl || ''
  gatewayDialog.open = true
}

function closeGatewayDialog() {
  gatewayDialog.open = false
}

async function reconnectGateway() {
  gatewayDialog.connecting = true
  try {
    await appStore.refreshGatewayData()
    if (appStore.gatewayStatus === 'connected') {
      notificationsStore.notify({
        title: '网关已连接',
        message: '网关状态恢复正常，已重新拉取基础数据。',
        tone: 'success',
      })
      gatewayDialog.open = false
      return
    }

    if (appStore.gatewayStatus === 'unconfigured') {
      notificationsStore.error('请先填写网关地址，再执行重连。', { title: '网关未配置' })
      gatewayDialog.open = true
      return
    }

    notificationsStore.error('重连失败，请检查网关地址或确认服务是否已经启动。', { title: '网关离线' })
    gatewayDialog.open = true
  } catch (error) {
    notificationsStore.error(error?.message || String(error), { title: '网关重连失败' })
    gatewayDialog.open = true
  } finally {
    gatewayDialog.connecting = false
  }
}

async function saveGatewaySettings() {
  gatewayDialog.saving = true
  try {
    const nextBaseUrl = String(gatewayDialog.draftBaseUrl || '').trim()
    await settingsStore.patch({
      gateway: {
        baseUrl: nextBaseUrl,
      },
    })
    notificationsStore.notify({
      title: '网关配置已保存',
      message: nextBaseUrl || '已清空网关地址。',
      tone: 'success',
    })
  } catch (error) {
    notificationsStore.error(error?.message || String(error), { title: '网关配置保存失败' })
    throw error
  } finally {
    gatewayDialog.saving = false
  }
}

function openThemePopover() {
  themePopover.draft = savedTheme.value
  themePopover.open = true
}

function closeThemePopover() {
  themePopover.open = false
  themePopover.draft = savedTheme.value
  applyTheme(savedTheme.value)
}

function previewTheme(value) {
  themePopover.draft = applyTheme(value)
}

async function saveTheme() {
  themePopover.saving = true
  try {
    const theme = applyTheme(themePopover.draft)
    await settingsStore.patch({ ui: { theme } })
    notificationsStore.notify({
      title: '主题已保存',
      message: selectedTheme.value.label,
      tone: 'success',
    })
    themePopover.open = false
  } catch (error) {
    notificationsStore.error(error?.message || String(error), { title: '主题保存失败' })
  } finally {
    themePopover.saving = false
  }
}

async function connectGateway() {
  try {
    await saveGatewaySettings()
    await reconnectGateway()
  } catch {
    gatewayDialog.open = true
  }
}

watch(
  () => settingsStore.settings.gateway.baseUrl,
  (value) => {
    gatewayDialog.draftBaseUrl = value || ''
  },
  { immediate: true },
)

watch(
  () => settingsStore.settings.ui.theme,
  (value) => {
    if (!themePopover.open) {
      themePopover.draft = value || 'dark'
    }
  },
  { immediate: true },
)

watch(
  () => appStore.gatewayStatus,
  (status, previous) => {
    if (status === 'unconfigured') {
      gatewayDialog.open = true
      return
    }

    if (status === 'offline' && previous !== 'offline' && !gatewayDialog.open) {
      gatewayDialog.open = true
    }
  },
)
</script>

<template>
  <div class="qq-theme qq-mesh relative flex h-screen flex-col text-[color:var(--qq-text-primary)]">
    <header
      class="qq-panel-strong relative z-[90] flex h-12 shrink-0 items-center justify-between border-b border-white/10 border-t-0 pl-4"
      style="--wails-draggable: drag"
    >
      <div class="flex min-w-0 items-center gap-3">
        <div class="flex h-7 w-7 shrink-0 items-center justify-center rounded-[4px] border border-white/10 bg-[var(--qq-fill-medium)] text-xs font-semibold text-[color:var(--qq-accent)]">
          IC
        </div>
        <div class="min-w-0">
          <p class="text-[10px] uppercase leading-4 tracking-[0.18em] text-[color:var(--qq-text-tertiary)]">Icoo Claw Desktop</p>
          <div class="flex min-w-0 items-center gap-2">
            <h1 class="truncate text-sm font-semibold text-[color:var(--qq-text-primary)]">{{ currentSectionLabel }}</h1>
            <span class="qq-badge hidden rounded-[4px] px-2 py-0.5 text-[11px] md:inline-flex">
              {{ shellStatusLabel }}
            </span>
          </div>
        </div>
      </div>
      <div class="flex h-full items-center gap-2" style="--wails-draggable: no-drag">
        <span v-if="appStore.lastRefreshedAt" class="hidden text-xs text-[color:var(--qq-text-tertiary)] md:inline">
          {{ new Date(appStore.lastRefreshedAt).toLocaleTimeString() }}
        </span>
        <QqButton variant="secondary" size="sm" @click="openGatewayDialog">
          网关管理
        </QqButton>
        <button
          class="relative inline-flex h-9 w-9 items-center justify-center rounded-[4px] border border-white/10 bg-[var(--qq-fill-medium)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[var(--qq-fill-strong)] hover:text-[color:var(--qq-accent)]"
          type="button"
          title="ACP 事件监控"
          @click="openACPMonitorWindow"
        >
          <Activity class="h-4 w-4" />
          <span
            v-if="acpMonitorStore.total"
            class="absolute -right-1 -top-1 min-w-4 rounded-full bg-[var(--qq-accent)] px-1 text-[10px] font-semibold leading-4 text-slate-950"
          >
            {{ acpMonitorStore.total > 99 ? '99+' : acpMonitorStore.total }}
          </span>
        </button>
        <div class="relative">
          <button
            class="inline-flex h-9 w-9 items-center justify-center rounded-[4px] border border-white/10 bg-[var(--qq-fill-medium)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[var(--qq-fill-strong)] hover:text-[color:var(--qq-accent)]"
            type="button"
            title="选择主题"
            @click="themePopover.open ? closeThemePopover() : openThemePopover()"
          >
            <Palette class="h-4 w-4" />
          </button>

          <div
            v-if="themePopover.open"
            class="theme-popover qq-panel-strong absolute right-0 top-[calc(100%+8px)] z-[120] w-[360px] rounded-[8px] border border-[color:var(--qq-border)] p-4 shadow-[var(--qq-shadow)]"
          >
            <div class="mb-3 flex items-start justify-between gap-3">
              <div>
                <p class="text-sm font-semibold text-[color:var(--qq-text-primary)]">界面主题</p>
                <p class="mt-1 text-xs text-[color:var(--qq-text-tertiary)]">选择后会立即预览，保存后下次启动继续使用。</p>
              </div>
              <button
                class="inline-flex h-7 w-7 items-center justify-center rounded-[4px] text-[color:var(--qq-text-tertiary)] hover:bg-[var(--qq-fill-soft)] hover:text-[color:var(--qq-text-primary)]"
                type="button"
                @click="closeThemePopover"
              >
                <X class="h-4 w-4" />
              </button>
            </div>

            <div class="grid gap-2">
              <button
                v-for="theme in THEME_OPTIONS"
                :key="theme.value"
                class="theme-choice flex w-full items-center gap-3 rounded-[6px] border px-3 py-2 text-left transition"
                :class="themePopover.draft === theme.value ? 'is-active' : ''"
                type="button"
                @click="previewTheme(theme.value)"
              >
                <span class="grid h-8 w-16 shrink-0 grid-cols-5 overflow-hidden rounded-[4px] border border-[color:var(--qq-border)]">
                  <span v-for="color in theme.colors" :key="color" :style="{ backgroundColor: color }" />
                </span>
                <span class="min-w-0 flex-1">
                  <span class="block truncate text-sm font-medium text-[color:var(--qq-text-primary)]">{{ theme.label }}</span>
                  <span class="block truncate text-xs text-[color:var(--qq-text-tertiary)]">{{ theme.description }}</span>
                </span>
                <Check v-if="themePopover.draft === theme.value" class="h-4 w-4 shrink-0 text-[color:var(--qq-accent)]" />
              </button>
            </div>

            <div class="mt-4 grid grid-cols-5 overflow-hidden rounded-[6px] border border-[color:var(--qq-border)]">
              <div v-for="color in selectedTheme.colors" :key="color" class="h-10" :style="{ backgroundColor: color }" />
            </div>

            <div class="mt-4 flex justify-end gap-2">
              <QqButton variant="ghost" size="sm" :disabled="themePopover.saving" @click="closeThemePopover">取消</QqButton>
              <QqButton size="sm" :disabled="themePopover.saving" @click="saveTheme">
                {{ themePopover.saving ? '保存中...' : '保存主题' }}
              </QqButton>
            </div>
          </div>
        </div>
        <button
          class="inline-flex h-9 w-9 items-center justify-center rounded-[4px] border border-white/10 bg-[var(--qq-fill-medium)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[var(--qq-fill-strong)] hover:text-[color:var(--qq-text-primary)]"
          type="button"
          title="刷新网关状态"
          @click="refresh"
        >
          <RefreshCw class="h-4 w-4" />
        </button>
        <div class="ml-1 flex h-full border-l border-white/10">
          <button
            class="inline-flex h-full w-11 items-center justify-center text-[color:var(--qq-text-secondary)] transition hover:bg-[var(--qq-fill-medium)] hover:text-[color:var(--qq-text-primary)]"
            type="button"
            title="最小化"
            @click="minimiseWindow"
          >
            <Minus class="h-4 w-4" />
          </button>
          <button
            class="inline-flex h-full w-11 items-center justify-center text-[color:var(--qq-text-secondary)] transition hover:bg-[var(--qq-fill-medium)] hover:text-[color:var(--qq-text-primary)]"
            type="button"
            title="最大化"
            @click="toggleMaximiseWindow"
          >
            <Square class="h-3.5 w-3.5" />
          </button>
          <button
            class="inline-flex h-full w-11 items-center justify-center text-[color:var(--qq-text-secondary)] transition hover:bg-rose-500/80 hover:text-white"
            type="button"
            title="关闭"
            @click="closeWindow"
          >
            <X class="h-4 w-4" />
          </button>
        </div>
      </div>
    </header>

    <div class="relative z-10 flex min-h-0 flex-1">
      <AppSidebar :items="navItems" />

      <ConversationList
        v-if="isChatRoute"
        class="hidden md:flex"
        :active-id="activeConversationId"
        :collapsed="conversationListCollapsed"
        :conversations="conversationsStore.items"
        :deleting-id="conversationsStore.deletingId"
        :loading="conversationsStore.loading"
        :running-conversation-ids="chatStore.runningConversationIds"
        :streaming="chatStore.anyStreaming"
        @delete="deleteConversation"
        @new-chat="newChat"
        @refresh="refresh"
        @toggle-collapse="conversationListCollapsed = !conversationListCollapsed"
      />


      <main class="qq-main-surface relative min-h-0 min-w-0 flex-1 overflow-hidden">
        <RouterView />
      </main>
    </div>

    <footer class="qq-panel-strong relative z-10 flex h-7 shrink-0 items-center justify-between gap-3 border-t border-white/8 px-3 text-[10px] text-[color:var(--qq-text-tertiary)]">
      <div class="flex min-w-0 items-center gap-2.5">
        <span class="inline-flex items-center gap-1.5">
          <span class="h-1.5 w-1.5 rounded-full shrink-0" :class="statusToneClass" />
          <span>{{ shellStatusLabel }}</span>
        </span>
        <span class="hidden truncate opacity-70 md:inline">{{ gatewayBaseUrlLabel }}</span>
        <span class="hidden lg:inline opacity-70">{{ currentProjectLabel }}</span>
      </div>
      <div class="flex shrink-0 items-center gap-2.5">
        <span class="hidden md:inline opacity-80">{{ defaultAgentLabel }}</span>
        <span class="hidden sm:inline opacity-70">{{ conversationsStore.items.length }} 会话</span>
        <span v-if="chatStore.anyStreaming" class="text-[color:var(--qq-accent)] animate-pulse">{{ chatStore.runningConversationIds.length }} 运行中</span>
        <span class="opacity-60">{{ lastRefreshedLabel }}</span>
      </div>
    </footer>

    <QqModal
      v-model="gatewayDialog.open"
      description="在这里管理桌面端连接的网关地址，主动发起连接，并查看当前网关返回的信息。"
      title="网关管理"
      @confirm="connectGateway"
    >
      <div class="grid gap-4">
        <div
          class="rounded-[6px] border border-white/10 bg-[var(--qq-fill-soft)] px-3 py-3 text-sm leading-6 text-[color:var(--qq-text-secondary)]"
        >
          <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
            <div>
              <p class="font-medium text-[color:var(--qq-text-primary)]">{{ shellStatusLabel }}</p>
              <p class="mt-1 break-all">{{ gatewayBaseUrlLabel }}</p>
            </div>
            <span class="qq-badge w-fit rounded-[4px] px-2 py-0.5 text-[11px]">刷新 {{ lastRefreshedLabel }}</span>
          </div>
          <p v-if="appStore.error" class="mt-3 text-[var(--qq-danger)]">
            {{ appStore.error }}
          </p>
          <p v-else class="mt-3">
            你可以先手动启动网关，再主动连接；也可以调整网关地址后重新连接。
          </p>
        </div>

        <QqFormField label="Gateway URL" helper="例如：http://127.0.0.1:8080">
          <QqInput v-model="gatewayDialog.draftBaseUrl" placeholder="请输入网关地址" />
        </QqFormField>

        <div class="grid gap-3 md:grid-cols-2">
          <div class="rounded-[6px] border border-white/10 bg-[var(--qq-fill-subtle)] px-3 py-3 text-sm">
            <p class="text-xs uppercase tracking-[0.16em] text-[color:var(--qq-text-tertiary)]">Health</p>
            <p class="mt-2 break-all text-[color:var(--qq-text-primary)]">{{ gatewayHealthLabel }}</p>
          </div>
          <div class="rounded-[6px] border border-white/10 bg-[var(--qq-fill-subtle)] px-3 py-3 text-sm">
            <p class="text-xs uppercase tracking-[0.16em] text-[color:var(--qq-text-tertiary)]">Resources</p>
            <p class="mt-2 text-[color:var(--qq-text-primary)]">Agent {{ agentsStore.items.length }} · 会话 {{ conversationsStore.items.length }}</p>
          </div>
        </div>
      </div>

      <template #footer>
        <QqButton variant="ghost" :disabled="gatewayDialogBusy" @click="closeGatewayDialog">关闭</QqButton>
        <QqButton variant="secondary" :disabled="gatewayDialogBusy" @click="saveGatewaySettings">
          {{ gatewayDialog.saving ? '保存中...' : '保存地址' }}
        </QqButton>
        <QqButton :disabled="gatewayDialogBusy" @click="connectGateway">
          {{ gatewayDialog.connecting ? '连接中...' : '连接网关' }}
        </QqButton>
      </template>
    </QqModal>

    <QqModal
      v-model="conversationDeleteDialog.open"
      description="删除后会话记录会从网关移除。"
      title="删除会话"
    >
      <div class="rounded-[6px] border border-white/10 bg-[var(--qq-fill-subtle)] px-3 py-3 text-sm leading-6 text-[color:var(--qq-text-secondary)]">
        <p class="font-medium text-[color:var(--qq-text-primary)]">
          {{ conversationsStore.byId(conversationDeleteDialog.conversationId)?.title || 'Untitled Conversation' }}
        </p>
        <p class="mt-1 break-all">ID {{ conversationDeleteDialog.conversationId || '-' }}</p>
      </div>

      <template #footer>
        <QqButton variant="ghost" :disabled="Boolean(conversationsStore.deletingId)" @click="closeConversationDeleteDialog">取消</QqButton>
        <QqButton data-testid="conversation-delete-confirm" variant="danger" :disabled="Boolean(conversationsStore.deletingId)" @click="confirmDeleteConversation">
          删除会话
        </QqButton>
      </template>
    </QqModal>
  </div>
</template>

<style scoped>
.theme-popover {
  backdrop-filter: blur(28px);
}

.theme-choice {
  border-color: var(--qq-border);
  background: var(--qq-fill-subtle);
}

.theme-choice:hover {
  border-color: var(--qq-border-strong);
  background: var(--qq-fill-soft);
}

.theme-choice.is-active {
  border-color: color-mix(in srgb, var(--qq-accent) 42%, var(--qq-border));
  background: color-mix(in srgb, var(--qq-accent) 12%, transparent);
}
</style>
