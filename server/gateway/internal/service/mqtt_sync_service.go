package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"icoo_claw/common/id"
	"icoo_claw/server/gateway/internal/config"
	"icoo_claw/server/gateway/internal/dto"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type SyncPublisher interface {
	Publish(ctx context.Context, event dto.SyncEvent) error
	Close() error
}

type noopSyncPublisher struct{}

func NewNoopSyncPublisher() SyncPublisher {
	return noopSyncPublisher{}
}

func (noopSyncPublisher) Publish(context.Context, dto.SyncEvent) error {
	return nil
}

func (noopSyncPublisher) Close() error {
	return nil
}

type MQTTSyncService struct {
	client      mqtt.Client
	topicPrefix string
	qos         byte
	retained    bool
}

func NewGatewaySyncPublisher(cfg config.Config) (SyncPublisher, error) {
	if !cfg.MQTT.Enabled {
		return NewNoopSyncPublisher(), nil
	}
	return NewMQTTSyncService(cfg.MQTT)
}

func NewMQTTSyncService(cfg config.MQTTConfig) (*MQTTSyncService, error) {
	brokerURL := strings.TrimSpace(cfg.BrokerURL)
	if brokerURL == "" {
		return nil, fmt.Errorf("mqtt broker url is required when mqtt is enabled")
	}

	clientID := strings.TrimSpace(cfg.ClientID)
	if clientID == "" {
		clientID = "icoo-gateway-" + id.Random()
	}

	options := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetClientID(clientID).
		SetCleanSession(true).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(2 * time.Second)

	if cfg.Username != "" {
		options.SetUsername(cfg.Username)
	}
	if cfg.Password != "" {
		options.SetPassword(cfg.Password)
	}

	client := mqtt.NewClient(options)
	token := client.Connect()
	if !token.WaitTimeout(cfg.ConnectTimeout) {
		return nil, fmt.Errorf("connect mqtt broker %s: timeout", brokerURL)
	}
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("connect mqtt broker %s: %w", brokerURL, err)
	}

	return &MQTTSyncService{
		client:      client,
		topicPrefix: normalizeTopicPrefix(cfg.TopicPrefix),
		qos:         byte(cfg.QoS),
		retained:    cfg.Retained,
	}, nil
}

func (s *MQTTSyncService) Publish(ctx context.Context, event dto.SyncEvent) error {
	if s == nil || s.client == nil {
		return nil
	}
	if event.ID == "" {
		event.ID = "sync_" + id.Random()
	}
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	if event.Source == "" {
		event.Source = "gateway"
	}
	if event.Protocol == "" {
		event.Protocol = "event"
	}
	if event.Type == "" {
		event.Type = "unknown"
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal sync event: %w", err)
	}

	topic := s.eventTopic(event)
	token := s.client.Publish(topic, s.qos, s.retained, payload)
	if deadline, ok := ctx.Deadline(); ok {
		if !token.WaitTimeout(time.Until(deadline)) {
			return context.DeadlineExceeded
		}
	} else {
		token.Wait()
	}
	if err := token.Error(); err != nil {
		return fmt.Errorf("publish mqtt topic %s: %w", topic, err)
	}
	return nil
}

func (s *MQTTSyncService) Subscribe(topic string, handler func(topic string, payload []byte)) error {
	if s == nil || s.client == nil || handler == nil {
		return nil
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		topic = s.topicPrefix + "/#"
	}
	token := s.client.Subscribe(topic, s.qos, func(_ mqtt.Client, message mqtt.Message) {
		handler(message.Topic(), message.Payload())
	})
	token.Wait()
	if err := token.Error(); err != nil {
		return fmt.Errorf("subscribe mqtt topic %s: %w", topic, err)
	}
	return nil
}

func (s *MQTTSyncService) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	s.client.Disconnect(250)
	return nil
}

func (s *MQTTSyncService) eventTopic(event dto.SyncEvent) string {
	return strings.Join([]string{
		s.topicPrefix,
		cleanTopicPart(event.Protocol),
		cleanTopicPath(event.Type),
	}, "/")
}

func normalizeTopicPrefix(prefix string) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		return "icoo/gateway"
	}
	return cleanTopicPath(prefix)
}

func cleanTopicPath(value string) string {
	parts := strings.Split(strings.Trim(strings.TrimSpace(value), "/"), "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		if clean := cleanTopicPart(part); clean != "" {
			cleaned = append(cleaned, clean)
		}
	}
	if len(cleaned) == 0 {
		return "unknown"
	}
	return strings.Join(cleaned, "/")
}

func cleanTopicPart(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "+", "_")
	value = strings.ReplaceAll(value, "#", "_")
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.Trim(value, "/")
	if value == "" {
		return "unknown"
	}
	return value
}
