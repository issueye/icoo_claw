export async function executeAction(actionRef, runtime, eventContext = {}) {
  try {
    if (Array.isArray(actionRef)) {
      const chain = runtime.actionRegistry.get('chain')

      if (!chain) {
        return actionError('ACTION_NOT_REGISTERED', 'Action type "chain" is not registered in this runtime.')
      }

      return chain({ type: 'chain', payload: actionRef }, runtime, eventContext)
    }

    const action = resolveAction(actionRef, runtime)

    if (!action.ok) {
      return action
    }

    const handler = runtime.actionRegistry.get(action.value.type)

    if (!handler) {
      return actionError(
        'ACTION_NOT_REGISTERED',
        `Action type "${action.value.type}" is not registered in this runtime.`,
      )
    }

    const result = await handler(action.value, runtime, eventContext)
    return result && typeof result === 'object' && 'ok' in result
      ? result
      : { ok: true, type: action.value.type, value: result }
  } catch (error) {
    return actionError('ACTION_FAILED', error?.message || 'Action execution failed.', error)
  }
}

function resolveAction(actionRef, runtime) {
  if (typeof actionRef === 'string') {
    const action = runtime.actionDefinitions[actionRef]

    if (!action) {
      return actionError('ACTION_NOT_FOUND', `Action "${actionRef}" was not found in schema actions.`)
    }

    return { ok: true, value: { ...action, id: actionRef } }
  }

  if (!actionRef || typeof actionRef !== 'object' || Array.isArray(actionRef)) {
    return actionError('INVALID_ACTION', 'Action must be an action id, an action object, or an action array.')
  }

  if (typeof actionRef.type !== 'string' || !actionRef.type.trim()) {
    return actionError('INVALID_ACTION', 'Action object requires a type.')
  }

  return { ok: true, value: actionRef }
}

function actionError(code, message, cause) {
  return {
    ok: false,
    error: {
      code,
      message,
      cause,
    },
  }
}
