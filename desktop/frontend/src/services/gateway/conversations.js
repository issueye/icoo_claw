import { fetchJSON } from './http'

export async function listConversations(baseUrl) {
  const payload = await fetchJSON(baseUrl, '/v1/conversations')
  return (payload?.conversations || []).map(normalizeConversation)
}

export async function createConversation(baseUrl, input) {
  const payload = await fetchJSON(baseUrl, '/v1/conversations', {
    method: 'POST',
    body: {
      agent_id: input.agentId,
      user_id: input.userId || '',
      title: input.title || '',
    },
  })
  return normalizeConversation(payload)
}

export async function listConversationMessages(baseUrl, conversationId) {
  const payload = await fetchJSON(baseUrl, `/v1/conversations/${conversationId}/messages`)
  return (payload?.messages || []).map(normalizeMessage)
}

export async function deleteConversation(baseUrl, conversationId) {
  await fetchJSON(baseUrl, `/v1/conversations/${conversationId}`, {
    method: 'DELETE',
  })
}

export function normalizeConversation(conversation) {
  return {
    id: conversation.id,
    sessionId: conversation.session_id,
    agentId: conversation.agent_id,
    userId: conversation.user_id,
    title: conversation.title,
    status: conversation.status,
    lastMessageAt: conversation.last_message_at,
    createdAt: conversation.created_at,
    updatedAt: conversation.updated_at,
  }
}

export function normalizeMessage(message) {
  return {
    id: message.id,
    role: message.role,
    content: message.content,
    metadata: message.metadata || {},
    createdAt: message.created_at,
  }
}
