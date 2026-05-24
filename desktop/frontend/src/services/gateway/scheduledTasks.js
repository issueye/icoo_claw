import { fetchJSON } from './http'

export async function listScheduledTasks(baseUrl) {
  const payload = await fetchJSON(baseUrl, '/v1/scheduled-tasks')
  return (payload?.tasks || []).map(normalizeTask)
}

export async function listScheduledTaskRuns(baseUrl, taskId) {
  const payload = await fetchJSON(baseUrl, `/v1/scheduled-tasks/${encodeURIComponent(taskId)}/runs`)
  return (payload?.runs || []).map(normalizeTaskRun)
}

export async function createScheduledTask(baseUrl, input) {
  const payload = await fetchJSON(baseUrl, '/v1/scheduled-tasks', {
    method: 'POST',
    body: taskPayload(input, { includeId: true }),
  })
  return normalizeTask(payload)
}

export async function updateScheduledTask(baseUrl, taskId, input) {
  const payload = await fetchJSON(baseUrl, `/v1/scheduled-tasks/${encodeURIComponent(taskId)}`, {
    method: 'PATCH',
    body: taskPayload(input),
  })
  return normalizeTask(payload)
}

export async function deleteScheduledTask(baseUrl, taskId) {
  await fetchJSON(baseUrl, `/v1/scheduled-tasks/${encodeURIComponent(taskId)}`, {
    method: 'DELETE',
  })
}

function taskPayload(input, options = {}) {
  const body = {
    name: input.name || '',
    description: input.description || '',
    agent_id: input.agentId || '',
    schedule_type: input.scheduleType || 'interval',
    schedule_value: input.scheduleValue || '',
    action_type: input.actionType || 'webhook',
    payload: parsePayload(input.payloadText),
    enabled: Boolean(input.enabled),
  }
  if (options.includeId) {
    body.id = input.id || ''
  }
  return body
}

function parsePayload(value) {
  const text = String(value || '').trim()
  if (!text) {
    return {}
  }
  return JSON.parse(text)
}

function normalizeTask(task) {
  return {
    id: task.id,
    name: task.name,
    description: task.description || '',
    agentId: task.agent_id || '',
    scheduleType: task.schedule_type || 'interval',
    scheduleValue: task.schedule_value || '',
    actionType: task.action_type || 'webhook',
    payload: task.payload || {},
    payloadText: JSON.stringify(task.payload || {}, null, 2),
    enabled: Boolean(task.enabled),
    status: task.status || '',
    lastRunAt: task.last_run_at || '',
    nextRunAt: task.next_run_at || '',
    runCount: task.run_count || 0,
    lastError: task.last_error || '',
    createdAt: task.created_at,
    updatedAt: task.updated_at,
  }
}

function normalizeTaskRun(run) {
  return {
    id: run.id,
    taskId: run.task_id,
    agentId: run.agent_id || '',
    status: run.status || '',
    summary: run.summary || '',
    error: run.error || '',
    executedAt: run.executed_at || '',
    createdAt: run.created_at || '',
    updatedAt: run.updated_at || '',
  }
}
