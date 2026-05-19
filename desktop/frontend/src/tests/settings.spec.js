import { describe, expect, it } from 'vitest'
import { defaultSettings, mergeSettings } from '@/services/settings/schema'

describe('settings schema helpers', () => {
  it('provides desktop defaults', () => {
    expect(defaultSettings()).toEqual({
      gateway: {
        baseUrl: 'http://127.0.0.1:8080',
        defaultAgentId: '',
      },
      workspace: {
        rootDir: '',
      },
      ui: {
        showTimestamps: true,
      },
    })
  })

  it('merges partial values over defaults', () => {
    expect(
      mergeSettings({
        gateway: { defaultAgentId: 'agent_1' },
      }),
    ).toEqual({
      gateway: {
        baseUrl: 'http://127.0.0.1:8080',
        defaultAgentId: 'agent_1',
      },
      workspace: {
        rootDir: '',
      },
      ui: {
        showTimestamps: true,
      },
    })
  })
})
