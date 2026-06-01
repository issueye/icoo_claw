package service

import (
	"testing"
	"time"

	"icoo_claw/server/gateway/internal/config"
)

func TestNewGatewaySyncPublisherReturnsNoopWhenDisabled(t *testing.T) {
	publisher, err := NewGatewaySyncPublisher(config.Config{})
	if err != nil {
		t.Fatalf("NewGatewaySyncPublisher() error = %v", err)
	}
	if _, ok := publisher.(noopSyncPublisher); !ok {
		t.Fatalf("publisher = %T, want noopSyncPublisher", publisher)
	}
}

func TestNewMQTTSyncServiceRequiresBrokerURL(t *testing.T) {
	_, err := NewMQTTSyncService(config.MQTTConfig{
		Enabled:        true,
		ConnectTimeout: time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected missing broker error")
	}
}

func TestNormalizeTopicPrefix(t *testing.T) {
	if got := normalizeTopicPrefix(" /icoo gateway/+/events/# "); got != "icoo_gateway/_/events/_" {
		t.Fatalf("normalizeTopicPrefix() = %q", got)
	}
	if got := normalizeTopicPrefix(""); got != "icoo/gateway" {
		t.Fatalf("default topic prefix = %q", got)
	}
}
