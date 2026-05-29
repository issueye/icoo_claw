package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Provider gives runtime access to lazily-instantiated models.
type Provider interface {
	Model(ctx context.Context) (Model, error)
}

// ProviderFunc is an adapter to allow use of ordinary functions as providers.
type ProviderFunc func(context.Context) (Model, error)

// Model implements Provider.
func (fn ProviderFunc) Model(ctx context.Context) (Model, error) {
	if fn == nil {
		return nil, errors.New("model provider function is nil")
	}
	return fn(ctx)
}

// AnthropicProvider caches anthropic clients with optional TTL.
type AnthropicProvider struct {
	APIKey      string
	BaseURL     string
	ModelName   string
	MaxTokens   int
	MaxRetries  int
	System      string
	Temperature *float64
	CacheTTL    time.Duration

	cache cachedModelSlot
}

// Model implements Provider with caching using double-checked locking.
func (p *AnthropicProvider) Model(ctx context.Context) (Model, error) {
	return p.cache.getOrCreate(ctx, p.CacheTTL, func(context.Context) (Model, error) {
		return NewAnthropic(AnthropicConfig{
			APIKey:      p.resolveAPIKey(),
			BaseURL:     strings.TrimSpace(p.BaseURL),
			Model:       strings.TrimSpace(p.ModelName),
			MaxTokens:   p.MaxTokens,
			MaxRetries:  p.MaxRetries,
			System:      p.System,
			Temperature: p.Temperature,
		})
	})
}

func (p *AnthropicProvider) resolveAPIKey() string {
	if key := strings.TrimSpace(p.APIKey); key != "" {
		return key
	}
	if key := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")); key != "" {
		return key
	}
	if key := strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN")); key != "" {
		return key
	}
	return ""
}

// OpenAIProvider caches OpenAI clients with optional TTL.
type OpenAIProvider struct {
	APIKey      string
	BaseURL     string // Optional: for Azure or proxies
	ModelName   string
	MaxTokens   int
	MaxRetries  int
	System      string
	Temperature *float64
	CacheTTL    time.Duration

	cache cachedModelSlot
}

// Model implements Provider with caching using double-checked locking.
func (p *OpenAIProvider) Model(ctx context.Context) (Model, error) {
	return p.cache.getOrCreate(ctx, p.CacheTTL, func(context.Context) (Model, error) {
		return NewOpenAI(OpenAIConfig{
			APIKey:      p.resolveAPIKey(),
			BaseURL:     strings.TrimSpace(p.BaseURL),
			Model:       strings.TrimSpace(p.ModelName),
			MaxTokens:   p.MaxTokens,
			MaxRetries:  p.MaxRetries,
			System:      p.System,
			Temperature: p.Temperature,
		})
	})
}

func (p *OpenAIProvider) resolveAPIKey() string {
	if key := strings.TrimSpace(p.APIKey); key != "" {
		return key
	}
	if key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY")); key != "" {
		return key
	}
	return ""
}

// MustProvider materialises a model immediately and panics on failure.
func MustProvider(p Provider) Model {
	if p == nil {
		panic("model provider is nil")
	}
	mdl, err := p.Model(context.Background())
	if err != nil {
		panic(fmt.Sprintf("model provider failed: %v", err))
	}
	return mdl
}

type cachedModelSlot struct {
	mu      sync.RWMutex
	model   Model
	expires time.Time
}

func (c *cachedModelSlot) getOrCreate(ctx context.Context, ttl time.Duration, build func(context.Context) (Model, error)) (Model, error) {
	if ttl <= 0 {
		return build(ctx)
	}
	if mdl := c.get(ttl); mdl != nil {
		return mdl, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.model != nil && time.Now().Before(c.expires) {
		return c.model, nil
	}
	mdl, err := build(ctx)
	if err != nil {
		return nil, err
	}
	c.model = mdl
	c.expires = time.Now().Add(ttl)
	return mdl, nil
}

func (c *cachedModelSlot) get(ttl time.Duration) Model {
	if ttl <= 0 {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.model == nil || time.Now().After(c.expires) {
		return nil
	}
	return c.model
}
