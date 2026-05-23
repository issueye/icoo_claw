import { defineStore } from 'pinia'
import { getGatewayHealth } from '@/services/gateway/health'
import { getAppInfo } from '@/services/wails/config'
import { useAgentsStore } from './agents'
import { useConversationsStore } from './conversations'
import { useSettingsStore } from './settings'

export const useAppStore = defineStore('app', {
  state: () => ({
    booting: false,
    ready: false,
    error: '',
    gatewayStatus: 'unknown',
    gatewayInfo: null,
    lastRefreshedAt: '',
    appInfo: null,
  }),

  actions: {
    async bootstrap() {
      if (this.booting) {
        return
      }
      this.booting = true
      this.error = ''

      const settingsStore = useSettingsStore()
      try {
        await settingsStore.load()
        this.appInfo = await getAppInfo()
      } catch (error) {
        this.error = error?.message || String(error)
      }

      try {
        await this.refreshGatewayData()
      } finally {
        this.ready = true
        this.booting = false
      }
    },

    async refreshGatewayData() {
      const settingsStore = useSettingsStore()
      const agentsStore = useAgentsStore()
      const conversationsStore = useConversationsStore()
      const baseUrl = String(settingsStore.settings.gateway.baseUrl || '').trim()

      if (!baseUrl) {
        this.gatewayStatus = 'unconfigured'
        this.gatewayInfo = null
        this.error = ''
        return
      }

      try {
        await this.loadGatewayData(baseUrl, agentsStore, conversationsStore)
        this.gatewayStatus = 'connected'
        this.lastRefreshedAt = new Date().toISOString()
        this.error = ''
      } catch (error) {
        this.gatewayStatus = 'offline'
        this.error = error?.code === 'gateway_unreachable' ? error?.message || String(error) : ''
      }
    },

    async loadGatewayData(baseUrl, agentsStore, conversationsStore) {
      this.gatewayInfo = await getGatewayHealth(baseUrl)
      await Promise.all([
        agentsStore.fetchAgents(baseUrl),
        conversationsStore.fetchConversations(baseUrl),
      ])
    },
  },
})
