package api

import (
	"testing"

	"icoo_claw/common/core/agent_sdk/config"
)

func TestNetworkToolOptionsUseExplicitProxy(t *testing.T) {
	opts := networkToolOptions(NetworkProxyOptions{
		HTTPProxy:  "http://proxy.local:8080",
		HTTPSProxy: "http://secure-proxy.local:8080",
		NoProxy:    "localhost",
	}, nil)

	if opts.HTTPProxy != "http://proxy.local:8080" || opts.HTTPSProxy != "http://secure-proxy.local:8080" || opts.NoProxy != "localhost" {
		t.Fatalf("network options = %+v", opts)
	}
}

func TestNetworkToolOptionsUseSandboxProxyPorts(t *testing.T) {
	httpPort := 18080
	settings := &config.Settings{Sandbox: &config.SandboxConfig{Network: &config.SandboxNetworkConfig{HTTPProxyPort: &httpPort}}}

	opts := networkToolOptions(NetworkProxyOptions{}, settings)
	if opts.HTTPProxy != "http://127.0.0.1:18080" || opts.HTTPSProxy != "http://127.0.0.1:18080" {
		t.Fatalf("network options = %+v", opts)
	}
}
