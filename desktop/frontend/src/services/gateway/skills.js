import { fetchJSON } from './http'

export async function listSkills(baseUrl) {
  const payload = await fetchJSON(baseUrl, '/v1/skills')
  return (payload?.skills || []).map(normalizeSkill)
}

export async function createSkill(baseUrl, input) {
  const payload = await fetchJSON(baseUrl, '/v1/skills', {
    method: 'POST',
    body: skillPayload(input, { includeId: true }),
  })
  return normalizeSkill(payload)
}

export async function updateSkill(baseUrl, skillId, input) {
  const payload = await fetchJSON(baseUrl, `/v1/skills/${encodeURIComponent(skillId)}`, {
    method: 'PATCH',
    body: skillPayload(input),
  })
  return normalizeSkill(payload)
}

export async function deleteSkill(baseUrl, skillId) {
  await fetchJSON(baseUrl, `/v1/skills/${encodeURIComponent(skillId)}`, {
    method: 'DELETE',
  })
}

function skillPayload(input, options = {}) {
  const body = {
    name: input.name || '',
    description: input.description || '',
    path: input.path || input.name || '',
    content: input.content || '',
    version: input.version || '',
    source: input.source || '',
    allowed_tools: normalizeList(input.allowedTools),
    metadata: input.metadata || {},
    files: normalizeFiles(input.files),
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

function normalizeFiles(files) {
  if (!Array.isArray(files)) {
    return []
  }
  return files
    .map((file) => ({
      path: String(file?.path || '').trim(),
      content: String(file?.content || ''),
    }))
    .filter((file) => file.path)
}

function normalizeSkill(skill) {
  return {
    id: skill.id,
    name: skill.name,
    description: skill.description,
    path: skill.path,
    content: skill.content || '',
    version: skill.version,
    status: skill.status,
    source: skill.source || '',
    allowedTools: skill.allowed_tools || [],
    metadata: skill.metadata || {},
    createdAt: skill.created_at,
    updatedAt: skill.updated_at,
  }
}
