import { defineStore } from 'pinia'
import { listAgents } from '@/services/gateway/agents'
import { useSettingsStore } from './settings'

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
        await this.ensureDefaultSelection()
        return this.items
      } catch (error) {
        this.error = error?.message || String(error)
        throw error
      } finally {
        this.loading = false
      }
    },

    async ensureDefaultSelection() {
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
      }
    },
  },
})
