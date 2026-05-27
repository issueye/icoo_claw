import { fetchJSON } from './http'

export async function listAgents(baseUrl) {
  const payload = await fetchJSON(baseUrl, '/v1/agents')
  return (payload?.agents || []).map(normalizeAgent)
}

export async function createAgent(baseUrl, input) {
  const payload = await fetchJSON(baseUrl, '/v1/agents', {
    method: 'POST',
    body: agentPayload(input, { includeId: true }),
  })
  return normalizeAgent(payload)
}

export async function updateAgent(baseUrl, agentId, input) {
  const payload = await fetchJSON(baseUrl, `/v1/agents/${encodeURIComponent(agentId)}`, {
    method: 'PATCH',
    body: agentPayload(input),
  })
  return normalizeAgent(payload)
}

export async function deleteAgent(baseUrl, agentId) {
  await fetchJSON(baseUrl, `/v1/agents/${encodeURIComponent(agentId)}`, {
    method: 'DELETE',
  })
}

function agentPayload(input, options = {}) {
  const body = {
    name: input.name,
    provider_id: input.providerId || '',
    model_provider: input.modelProvider || 'openai',
    model_name: input.modelName || '',
    base_url: input.baseUrl || '',
    transport: input.transport || 'http',
    command_args: normalizeCommandArgs(input.commandArgs),
    system_prompt: input.systemPrompt || '',
    max_iterations: Number(input.maxIterations) || 0,
    tool_whitelist: normalizeList(input.toolWhitelist),
    network_allow: normalizeList(input.networkAllow),
    mcp_server_ids: normalizeList(input.mcpServerIds),
    skill_names: normalizeList(input.skillNames),
    enabled: Boolean(input.enabled),
  }
  if (options.includeId) {
    body.id = input.id || ''
  }
  return body
}

function normalizeList(value) {
  if (Array.isArray(value)) {
    return value.map((item) => String(item || '').trim()).filter(Boolean)
  }
  return String(value || '')
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function normalizeCommandArgs(value) {
  if (Array.isArray(value)) {
    return value.map((item) => String(item || '').trim()).filter(Boolean)
  }
  return String(value || '')
    .split(/\n/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function normalizeAgent(agent) {
  return {
    id: agent.id,
    name: agent.name,
    providerId: agent.provider_id || '',
    modelProvider: agent.model_provider,
    modelName: agent.model_name,
    baseUrl: agent.base_url,
    transport: agent.transport || 'http',
    commandArgs: agent.command_args || [],
    systemPrompt: agent.system_prompt,
    maxIterations: agent.max_iterations,
    toolWhitelist: agent.tool_whitelist || [],
    networkAllow: agent.network_allow || [],
    mcpServerIds: agent.mcp_server_ids || [],
    skillNames: agent.skill_names || [],
    enabled: agent.enabled,
    createdAt: agent.created_at,
    updatedAt: agent.updated_at,
  }
}
