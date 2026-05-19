<script setup>
import { reactive, watch } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import QqButton from '@/components/ued/QqButton.vue'
import QqFormField from '@/components/ued/QqFormField.vue'
import QqFormSection from '@/components/ued/QqFormSection.vue'
import QqInput from '@/components/ued/QqInput.vue'
import QqSelect from '@/components/ued/QqSelect.vue'
import QqSwitch from '@/components/ued/QqSwitch.vue'
import { chooseDirectory, chooseGatewayConfig, chooseGatewayProgram } from '@/services/wails/config'
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

async function pickGatewayProgram() {
  const value = await chooseGatewayProgram()
  if (value) {
    form.gateway.programPath = value
  }
}

async function pickGatewayConfig() {
  const value = await chooseGatewayConfig()
  if (value) {
    form.gateway.configPath = value
  }
}

async function save() {
  await settingsStore.save(mergeSettings(form))
  await appStore.refreshGatewayData()
}

function agentOptions() {
  return [
    { label: '请选择 Agent', value: '' },
    ...agentsStore.items.map((agent) => ({
      label: `${agent.name} (${agent.id})`,
      value: agent.id,
    })),
  ]
}
</script>

<template>
  <section class="scrollbar-thin h-full overflow-y-auto px-5 py-5">
    <div class="mx-auto max-w-5xl space-y-5">
      <section class="qq-panel-strong rounded-[8px] px-5 py-5">
        <p class="text-xs uppercase tracking-[0.24em] text-[color:var(--qq-text-tertiary)]">Local Settings</p>
        <h2 class="mt-3 text-3xl font-semibold text-slate-50">桌面端配置</h2>
        <p class="mt-4 max-w-3xl text-sm leading-7 text-[color:var(--qq-text-secondary)]">
          当前只保留聊天主链路需要的本地设置。网关地址、默认 Agent 和工作目录使用 TOML 写入本机配置文件。
        </p>
      </section>

      <div class="grid gap-5 xl:grid-cols-[1.1fr_0.9fr]">
        <QqFormSection
          eyebrow="Gateway"
          title="网关与默认 Agent"
          description="连接地址、默认 Agent 和工作目录都从这里管理，保持桌面端与网关配置一致。"
        >
          <div class="grid gap-5">
            <QqFormField label="Gateway URL" helper="桌面端所有 HTTP 和 WebSocket 请求都走这个地址。">
              <QqInput v-model="form.gateway.baseUrl" type="text" />
            </QqFormField>

            <QqFormField label="Default Agent" helper="默认进入聊天时优先使用的 Agent。">
              <QqSelect v-model="form.gateway.defaultAgentId" :options="agentOptions()" />
            </QqFormField>

            <QqFormField label="Gateway Program Path" helper="可选。填写后会优先启动这个网关程序，而不是 bundled gateway。">
              <div class="flex flex-col gap-3 md:flex-row">
                <QqInput v-model="form.gateway.programPath" class="flex-1" type="text" />
                <QqButton variant="secondary" @click="pickGatewayProgram">选择程序</QqButton>
              </div>
            </QqFormField>

            <QqFormField label="Gateway Config Path" helper="可选。填写后会在启动自定义网关程序时作为 --config 参数传入。">
              <div class="flex flex-col gap-3 md:flex-row">
                <QqInput v-model="form.gateway.configPath" class="flex-1" type="text" />
                <QqButton variant="secondary" @click="pickGatewayConfig">选择配置</QqButton>
              </div>
            </QqFormField>

            <QqFormField label="Workspace Directory" helper="当前版本不消费该目录，只为后续项目上下文预留入口。">
              <div class="flex flex-col gap-3 md:flex-row">
                <QqInput v-model="form.workspace.rootDir" class="flex-1" type="text" />
                <QqButton variant="secondary" @click="pickDirectory">浏览</QqButton>
              </div>
            </QqFormField>
          </div>
        </QqFormSection>

        <QqFormSection
          eyebrow="Behavior"
          title="界面行为"
          description="先把聊天主链路需要的可见行为统一收口，后面再扩展更细的偏好项。"
        >
          <div class="grid gap-3">
            <QqSwitch
              v-model="form.ui.showTimestamps"
              label="显示消息时间"
              description="控制聊天消息中是否显示时间。"
            />
          </div>

          <div class="mt-5 flex flex-wrap items-center gap-3">
            <QqButton @click="save">保存设置</QqButton>
            <QqButton variant="secondary" @click="appStore.refreshGatewayData">
              <RefreshCw class="h-4 w-4" />
              刷新网关数据
            </QqButton>
          </div>
        </QqFormSection>
      </div>

      <QqFormSection eyebrow="Runtime" title="运行信息" description="用于确认当前本地配置路径和运行时环境。">
        <div class="grid gap-4 md:grid-cols-2">
          <div class="rounded-[6px] border border-white/10 bg-[rgba(9,32,28,0.22)] px-4 py-3 text-sm text-[color:var(--qq-text-secondary)]">
            <p class="text-xs uppercase tracking-[0.16em] text-[color:var(--qq-text-tertiary)]">Config Path</p>
            <p class="mt-2 break-all">{{ settingsStore.path || '未加载' }}</p>
          </div>
          <div class="rounded-[6px] border border-white/10 bg-[rgba(9,32,28,0.22)] px-4 py-3 text-sm text-[color:var(--qq-text-secondary)]">
            <p class="text-xs uppercase tracking-[0.16em] text-[color:var(--qq-text-tertiary)]">Runtime</p>
            <p class="mt-2 break-all">
              {{ appStore.appInfo?.name || 'Icoo Claw' }} {{ appStore.appInfo?.version || '' }}
            </p>
            <p class="mt-1 break-all">
              {{ appStore.appInfo?.os || '' }} / {{ appStore.appInfo?.arch || '' }} / {{ appStore.appInfo?.goVersion || '' }}
            </p>
          </div>
        </div>
      </QqFormSection>
    </div>
  </section>
</template>
