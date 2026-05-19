<script setup>
import { reactive, watch } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import { chooseDirectory } from '@/services/wails/config'
import { mergeSettings } from '@/services/settings/schema'
import { useAgentsStore } from '@/stores/agents'
import { useAppStore } from '@/stores/app'
import { useSettingsStore } from '@/stores/settings'

const appStore = useAppStore()
const settingsStore = useSettingsStore()
const agentsStore = useAgentsStore()

const form = reactive(mergeSettings())

watch(
  () => settingsStore.settings,
  (value) => {
    Object.assign(form.gateway, value.gateway)
    Object.assign(form.workspace, value.workspace)
    Object.assign(form.ui, value.ui)
  },
  { deep: true, immediate: true },
)

async function pickDirectory() {
  const value = await chooseDirectory()
  if (value) {
    form.workspace.rootDir = value
  }
}

async function save() {
  await settingsStore.save(mergeSettings(form))
  await appStore.refreshGatewayData()
}
</script>

<template>
  <section class="scrollbar-thin h-full overflow-y-auto px-6 py-6">
    <div class="mx-auto max-w-3xl">
      <p class="text-xs uppercase tracking-[0.24em] text-accent/70">Local Settings</p>
      <h2 class="mt-3 text-3xl font-semibold text-slate-50">桌面端配置</h2>
      <p class="mt-4 max-w-2xl text-sm leading-7 text-slate-400">
        当前只保留聊天主链路需要的本地设置。网关地址、默认 Agent 和工作目录使用 TOML 写入本机配置文件。
      </p>

      <div class="mt-8 space-y-6">
        <section class="border border-line bg-panel px-5 py-5">
          <label class="block text-sm font-medium text-slate-200">Gateway URL</label>
          <input
            v-model="form.gateway.baseUrl"
            class="mt-3 h-11 w-full rounded-md border border-line bg-[#0b1017] px-3 text-sm text-slate-100 outline-none transition focus:border-accent/60"
            type="text"
          />
          <p class="mt-3 text-xs text-slate-500">桌面端所有 HTTP 和 WebSocket 请求都走这个地址。</p>
        </section>

        <section class="border border-line bg-panel px-5 py-5">
          <label class="block text-sm font-medium text-slate-200">Default Agent</label>
          <select
            v-model="form.gateway.defaultAgentId"
            class="mt-3 h-11 w-full rounded-md border border-line bg-[#0b1017] px-3 text-sm text-slate-100 outline-none transition focus:border-accent/60"
          >
            <option value="">请选择 Agent</option>
            <option v-for="agent in agentsStore.items" :key="agent.id" :value="agent.id">
              {{ agent.name }} ({{ agent.id }})
            </option>
          </select>
        </section>

        <section class="border border-line bg-panel px-5 py-5">
          <label class="block text-sm font-medium text-slate-200">Workspace Directory</label>
          <div class="mt-3 flex flex-col gap-3 md:flex-row">
            <input
              v-model="form.workspace.rootDir"
              class="h-11 flex-1 rounded-md border border-line bg-[#0b1017] px-3 text-sm text-slate-100 outline-none transition focus:border-accent/60"
              type="text"
            />
            <button
              class="inline-flex h-11 items-center justify-center rounded-md border border-line bg-panelSoft px-4 text-sm text-slate-200 transition hover:border-accent/60 hover:text-accent"
              type="button"
              @click="pickDirectory"
            >
              浏览
            </button>
          </div>
          <p class="mt-3 text-xs text-slate-500">当前版本不消费该目录，只为后续项目上下文预留入口。</p>
        </section>

        <section class="border border-line bg-panel px-5 py-5">
          <div class="flex items-center justify-between gap-4">
            <div>
              <h3 class="text-sm font-medium text-slate-200">Show Timestamps</h3>
              <p class="mt-2 text-xs text-slate-500">控制聊天消息中是否显示时间。</p>
            </div>
            <label class="inline-flex items-center gap-3 text-sm text-slate-200">
              <input v-model="form.ui.showTimestamps" class="h-4 w-4 accent-emerald-400" type="checkbox" />
              启用
            </label>
          </div>
        </section>
      </div>

      <div class="mt-8 flex flex-wrap items-center gap-3">
        <button
          class="inline-flex h-11 items-center justify-center rounded-md bg-accent px-5 text-sm font-medium text-slate-950 transition hover:bg-accentStrong"
          type="button"
          @click="save"
        >
          保存设置
        </button>
        <button
          class="inline-flex h-11 items-center gap-2 rounded-md border border-line bg-panel px-4 text-sm text-slate-200 transition hover:border-accent/60 hover:text-accent"
          type="button"
          @click="appStore.refreshGatewayData"
        >
          <RefreshCw class="h-4 w-4" />
          刷新网关数据
        </button>
      </div>

      <section class="mt-10 border border-line bg-panel px-5 py-5 text-sm text-slate-400">
        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <p class="text-xs uppercase tracking-[0.16em] text-slate-500">Config Path</p>
            <p class="mt-2 break-all">{{ settingsStore.path || '未加载' }}</p>
          </div>
          <div>
            <p class="text-xs uppercase tracking-[0.16em] text-slate-500">Runtime</p>
            <p class="mt-2 break-all">
              {{ appStore.appInfo?.name || 'Icoo Claw' }} {{ appStore.appInfo?.version || '' }}
            </p>
            <p class="mt-1 break-all">
              {{ appStore.appInfo?.os || '' }} / {{ appStore.appInfo?.arch || '' }} / {{ appStore.appInfo?.goVersion || '' }}
            </p>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>
