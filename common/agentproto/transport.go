package agentproto

import "strings"

const (
	TransportHTTP = "http"
	TransportACP  = "acp"
)

func NormalizeTransport(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case TransportACP:
		return TransportACP
	default:
		return TransportHTTP
	}
}
