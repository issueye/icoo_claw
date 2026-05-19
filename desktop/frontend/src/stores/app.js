import { defineStore } from 'pinia'
import { getGatewayHealth } from '@/services/gateway/health'
import { ensureBundledGateway, getAppInfo } from '@/services/wails/config'
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
      const baseUrl = settingsStore.settings.gateway.baseUrl

      try {
        await this.loadGatewayData(baseUrl, agentsStore, conversationsStore)
        this.gatewayStatus = 'connected'
        this.lastRefreshedAt = new Date().toISOString()
        this.error = ''
      } catch (error) {
        if (error?.code === 'gateway_unreachable') {
          try {
            const started = await ensureBundledGateway(baseUrl)
            if (started) {
              await this.loadGatewayData(baseUrl, agentsStore, conversationsStore)
              this.gatewayStatus = 'connected'
              this.lastRefreshedAt = new Date().toISOString()
              this.error = ''
              return
            }
          } catch (wakeError) {
            this.gatewayStatus = 'offline'
            this.error = wakeError?.message || String(wakeError)
            return
          }
        }
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
