export class GatewayError extends Error {
  constructor(message, options = {}) {
    super(message)
    this.name = 'GatewayError'
    this.status = options.status ?? 500
    this.code = options.code ?? 'gateway_error'
    this.data = options.data ?? null
  }
}

export function normalizeBaseUrl(value) {
  const normalized = String(value || '').trim().replace(/\/+$/, '')
  return normalized || 'http://127.0.0.1:8080'
}

export function toWebSocketURL(baseUrl, path = '/v1/ws/chat') {
  const target = new URL(path, `${normalizeBaseUrl(baseUrl)}/`)
  target.protocol = target.protocol === 'https:' ? 'wss:' : 'ws:'
  return target.toString()
}

export async function fetchJSON(baseUrl, path, options = {}) {
  const init = {
    method: options.method || 'GET',
    headers: {
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...(options.headers || {}),
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
  }

  const requestURL = `${normalizeBaseUrl(baseUrl)}${path}`
  let response
  try {
    response = await fetch(requestURL, init)
  } catch (error) {
    throw new GatewayError(
      `无法连接到网关 ${normalizeBaseUrl(baseUrl)}。请先启动测试服务，或确认该地址可以访问。`,
      {
        status: 0,
        code: 'gateway_unreachable',
        data: {
          cause: error?.message || String(error),
          url: requestURL,
        },
      },
    )
  }

  const text = await response.text()
  const payload = tryParseJSON(text)

  if (!response.ok) {
    throw new GatewayError(payload?.error || response.statusText || 'gateway request failed', {
      status: response.status,
      code: payload?.code || 'gateway_error',
      data: payload,
    })
  }

  return payload
}

function tryParseJSON(text) {
  if (!text) {
    return null
  }
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}
