import { fetchJSON } from './http'

export async function listAgentInstances(baseUrl) {
  const payload = await fetchJSON(baseUrl, '/v1/agent-instances')
  return (payload?.instances || []).map(normalizeInstance)
}

export async function startAgentInstance(baseUrl, input) {
  const payload = await fetchJSON(baseUrl, '/v1/agent-instances', {
    method: 'POST',
    body: {
      agent_id: input.agentId,
      name: input.name || '',
      transport: input.transport || '',
    },
  })
  return normalizeInstance(payload)
}

export async function stopAgentInstance(baseUrl, instanceId) {
  await fetchJSON(baseUrl, `/v1/agent-instances/${encodeURIComponent(instanceId)}/stop`, {
    method: 'POST',
  })
}

export async function restartAgentInstance(baseUrl, instanceId) {
  const payload = await fetchJSON(baseUrl, `/v1/agent-instances/${encodeURIComponent(instanceId)}/restart`, {
    method: 'POST',
  })
  return normalizeInstance(payload)
}

export async function drainAgentInstance(baseUrl, instanceId) {
  const payload = await fetchJSON(baseUrl, `/v1/agent-instances/${encodeURIComponent(instanceId)}/drain`, {
    method: 'POST',
  })
  return normalizeInstance(payload)
}

export async function deleteAgentInstance(baseUrl, instanceId) {
  await fetchJSON(baseUrl, `/v1/agent-instances/${encodeURIComponent(instanceId)}`, {
    method: 'DELETE',
  })
}

function normalizeInstance(instance) {
  return {
    id: instance.id,
    agentId: instance.agent_id,
    name: instance.name || '',
    status: instance.status,
    pid: instance.pid || 0,
    host: instance.host || '',
    port: instance.port || 0,
    baseUrl: instance.base_url || '',
    transport: instance.transport || 'http',
    providerId: instance.provider_id || '',
    modelProvider: instance.model_provider || '',
    modelName: instance.model_name || '',
    modelBaseUrl: instance.model_base_url || '',
    apiKeySet: Boolean(instance.api_key_set),
    lastHeartbeatAt: instance.last_heartbeat_at || '',
    lastError: instance.last_error || '',
    inflight: instance.inflight || 0,
    createdAt: instance.created_at,
    updatedAt: instance.updated_at,
  }
}
