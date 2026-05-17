package repository

func sessionMetaKey(sessionID string) string {
	return "sess:" + sessionID + ":meta"
}

func sessionMessagesKey(sessionID string) string {
	return "sess:" + sessionID + ":messages"
}

func sessionRunsKey(sessionID string) string {
	return "sess:" + sessionID + ":runs"
}

func userSessionsKey(userID string) string {
	return "idx:user:" + userID + ":sessions"
}

func agentSessionsKey(agentID string) string {
	return "idx:agent:" + agentID + ":sessions"
}
