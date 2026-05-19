export function defaultSettings() {
  return {
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
  }
}

export function mergeSettings(value = {}) {
  const fallback = defaultSettings()
  return {
    gateway: {
      ...fallback.gateway,
      ...(value.gateway || {}),
    },
    workspace: {
      ...fallback.workspace,
      ...(value.workspace || {}),
    },
    ui: {
      ...fallback.ui,
      ...(value.ui || {}),
    },
  }
}
