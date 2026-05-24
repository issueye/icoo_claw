import { fetchJSON } from './http'

export async function listProviders(baseUrl) {
  const payload = await fetchJSON(baseUrl, '/v1/providers')
  return (payload?.providers || []).map(normalizeProvider)
}

export async function createProvider(baseUrl, input) {
  const payload = await fetchJSON(baseUrl, '/v1/providers', {
    method: 'POST',
    body: providerPayload(input, { includeId: true, includeSecret: true }),
  })
  return normalizeProvider(payload)
}

export async function updateProvider(baseUrl, providerId, input) {
  const payload = await fetchJSON(baseUrl, `/v1/providers/${encodeURIComponent(providerId)}`, {
    method: 'PATCH',
    body: providerPayload(input, { includeSecret: Boolean(input.apiKey) }),
  })
  return normalizeProvider(payload)
}

export async function deleteProvider(baseUrl, providerId) {
  await fetchJSON(baseUrl, `/v1/providers/${encodeURIComponent(providerId)}`, {
    method: 'DELETE',
  })
}

function providerPayload(input, options = {}) {
  const body = {
    name: input.name,
    type: input.type,
    base_url: input.baseUrl || '',
    default_model: input.defaultModel || '',
    enabled: Boolean(input.enabled),
  }
  if (options.includeId) {
    body.id = input.id || ''
  }
  if (options.includeSecret) {
    body.api_key = input.apiKey || ''
  }
  return body
}

function normalizeProvider(provider) {
  return {
    id: provider.id,
    name: provider.name,
    type: provider.type,
    baseUrl: provider.base_url || '',
    defaultModel: provider.default_model || '',
    enabled: Boolean(provider.enabled),
    apiKeySet: Boolean(provider.api_key_set),
    apiKeyPreview: provider.api_key_preview || '',
    createdAt: provider.created_at,
    updatedAt: provider.updated_at,
  }
}
