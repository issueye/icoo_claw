import { fetchJSON } from './http'

export async function listAgents(baseUrl) {
  const payload = await fetchJSON(baseUrl, '/v1/agents')
  return (payload?.agents || []).map(normalizeAgent)
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
