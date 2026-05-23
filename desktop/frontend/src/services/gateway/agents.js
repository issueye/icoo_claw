import { fetchJSON } from './http'

export async function listAgents(baseUrl) {
  const payload = await fetchJSON(baseUrl, '/v1/agents')
  return (payload?.agents || []).map(normalizeAgent)
}

export async function createAgent(baseUrl, input) {
  const payload = await fetchJSON(baseUrl, '/v1/agents', {
    method: 'POST',
    body: {
      id: input.id,
      name: input.name,
      model_provider: input.modelProvider,
      model_name: input.modelName,
      base_url: input.baseUrl || '',
      system_prompt: input.systemPrompt || '',
      max_iterations: input.maxIterations,
      tool_whitelist: input.toolWhitelist || [],
      network_allow: input.networkAllow || [],
      mcp_server_ids: input.mcpServerIds || [],
      skill_ids: input.skillIds || [],
      enabled: input.enabled,
    },
  })
  return normalizeAgent(payload)
}

function normalizeAgent(agent) {
  return {
    id: agent.id,
    name: agent.name,
    modelProvider: agent.model_provider,
    modelName: agent.model_name,
    baseUrl: agent.base_url,
    systemPrompt: agent.system_prompt,
    maxIterations: agent.max_iterations,
    enabled: agent.enabled,
    createdAt: agent.created_at,
    updatedAt: agent.updated_at,
  }
}
