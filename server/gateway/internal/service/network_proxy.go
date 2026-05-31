package service

import (
	"encoding/json"
	"strings"

	"icoo_claw/common/agentproto"
	"icoo_claw/server/gateway/internal/dto"
)

func cleanNetworkProxy(proxy dto.NetworkProxy) dto.NetworkProxy {
	return dto.NetworkProxy{
		HTTPProxy:  strings.TrimSpace(proxy.HTTPProxy),
		HTTPSProxy: strings.TrimSpace(proxy.HTTPSProxy),
		NoProxy:    strings.TrimSpace(proxy.NoProxy),
	}
}

func marshalNetworkProxy(proxy dto.NetworkProxy) string {
	proxy = cleanNetworkProxy(proxy)
	if proxy.HTTPProxy == "" && proxy.HTTPSProxy == "" && proxy.NoProxy == "" {
		return ""
	}
	payload, err := json.Marshal(proxy)
	if err != nil {
		return ""
	}
	return string(payload)
}

func unmarshalNetworkProxy(raw string) dto.NetworkProxy {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return dto.NetworkProxy{}
	}
	var proxy dto.NetworkProxy
	if err := json.Unmarshal([]byte(raw), &proxy); err != nil {
		return dto.NetworkProxy{}
	}
	return cleanNetworkProxy(proxy)
}

func toAgentProtoNetworkProxy(proxy dto.NetworkProxy) agentproto.NetworkProxyConfig {
	proxy = cleanNetworkProxy(proxy)
	return agentproto.NetworkProxyConfig{
		HTTPProxy:  proxy.HTTPProxy,
		HTTPSProxy: proxy.HTTPSProxy,
		NoProxy:    proxy.NoProxy,
	}
}
