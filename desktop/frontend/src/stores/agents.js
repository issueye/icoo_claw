import { defineStore } from 'pinia'
import { createAgent, listAgents } from '@/services/gateway/agents'
import { useSettingsStore } from './settings'

const desktopDefaultAgentId = 'agent_desktop_default'

export const useAgentsStore = defineStore('agents', {
  state: () => ({
    items: [],
    loading: false,
    error: '',
  }),

  getters: {
    selectedAgent(state) {
      const settingsStore = useSettingsStore()
      return state.items.find((item) => item.id === settingsStore.settings.gateway.defaultAgentId) || null
    },
  },

  actions: {
    async fetchAgents(baseUrl) {
      this.loading = true
      this.error = ''
      try {
        this.items = await listAgents(baseUrl)
        await this.ensureDefaultSelection(baseUrl)
        return this.items
      } catch (error) {
        this.error = error?.message || String(error)
        throw error
      } finally {
        this.loading = false
      }
    },

    async ensureDefaultSelection(baseUrl) {
      const settingsStore = useSettingsStore()
      const current = settingsStore.settings.gateway.defaultAgentId
      if (current && this.items.some((item) => item.id === current)) {
        return
      }
      if (this.items.length > 0) {
        await settingsStore.patch({
          gateway: {
            defaultAgentId: this.items[0].id,
          },
        })
        return
      }

      const agent = await this.createDesktopDefaultAgent(baseUrl, current || desktopDefaultAgentId)
      this.items = [agent]
      await settingsStore.patch({
        gateway: {
          defaultAgentId: agent.id,
        },
      })
    },

    async createDesktopDefaultAgent(baseUrl, agentId = desktopDefaultAgentId) {
      try {
        return await createAgent(baseUrl, {
          id: agentId,
          name: 'Desktop Default Agent',
          modelProvider: 'openai',
          modelName: 'fake',
          maxIterations: 1,
          toolWhitelist: [],
          networkAllow: [],
          mcpServerIds: [],
          skillIds: [],
          enabled: true,
        })
      } catch (error) {
        if (error?.status === 409 || error?.code === 'already_exists') {
          this.items = await listAgents(baseUrl)
          const existing = this.items.find((item) => item.id === agentId) || this.items[0]
          if (existing) {
            return existing
          }
        }
        throw error
      }
    },
  },
})
