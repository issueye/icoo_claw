package api

import (
	"context"
	"sort"
	"strings"
	"sync"

	"icoo_claw/common/core/agent_sdk/tool"
)

type deferredToolState struct {
	mu       sync.RWMutex
	tools    map[string]tool.Tool
	sessions map[string]map[string]struct{}
	saver    func(sessionID string, activeTools []string) error
}

func newDeferredToolState(registry *tool.Registry) *deferredToolState {
	if registry == nil {
		return nil
	}
	tools := map[string]tool.Tool{}
	for _, impl := range registry.List() {
		if impl == nil || !tool.ShouldDefer(impl) {
			continue
		}
		key := canonicalToolName(impl.Name())
		if key == "" {
			continue
		}
		tools[key] = impl
	}
	if len(tools) == 0 {
		return nil
	}
	return &deferredToolState{
		tools:    tools,
		sessions: map[string]map[string]struct{}{},
	}
}

func (s *deferredToolState) shouldExpose(name, sessionID string) bool {
	if s == nil {
		return true
	}
	key := canonicalToolName(name)
	if _, ok := s.tools[key]; !ok {
		return true
	}
	if sessionID == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if active := s.sessions[sessionID]; active != nil {
		_, ok := active[key]
		return ok
	}
	return false
}

func (s *deferredToolState) activate(sessionID string, names []string) {
	if s == nil || sessionID == "" || len(names) == 0 {
		return
	}
	s.mu.Lock()
	active := s.sessions[sessionID]
	if active == nil {
		active = map[string]struct{}{}
		s.sessions[sessionID] = active
	}
	changed := false
	for _, name := range names {
		key := canonicalToolName(name)
		if key == "" {
			continue
		}
		if _, ok := s.tools[key]; ok {
			if _, ok := active[key]; !ok {
				active[key] = struct{}{}
				changed = true
			}
		}
	}
	var activeNames []string
	if changed && s.saver != nil {
		activeNames = make([]string, 0, len(active))
		for k := range active {
			if t, ok := s.tools[k]; ok {
				activeNames = append(activeNames, t.Name())
			} else {
				activeNames = append(activeNames, k)
			}
		}
	}
	s.mu.Unlock()

	if changed && s.saver != nil {
		_ = s.saver(sessionID, activeNames)
	}
}

func (s *deferredToolState) evict(sessionID string) {
	if s == nil || sessionID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func (s *deferredToolState) loadIfMissing(ctx context.Context, sessionID string, loader func(context.Context, string) ([]string, error)) error {
	if s == nil || sessionID == "" || loader == nil {
		return nil
	}
	s.mu.Lock()
	if _, ok := s.sessions[sessionID]; ok {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	tools, err := loader(ctx, sessionID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[sessionID]; ok {
		return nil
	}
	active := map[string]struct{}{}
	for _, toolName := range tools {
		key := canonicalToolName(toolName)
		if key != "" {
			active[key] = struct{}{}
		}
	}
	s.sessions[sessionID] = active
	return nil
}

func (s *deferredToolState) inactiveNames(sessionID string, whitelist map[string]struct{}) []string {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	active := s.sessions[sessionID]
	names := make([]string, 0, len(s.tools))
	for key, impl := range s.tools {
		if len(whitelist) > 0 {
			if _, ok := whitelist[key]; !ok {
				continue
			}
		}
		if active != nil {
			if _, ok := active[key]; ok {
				continue
			}
		}
		names = append(names, strings.TrimSpace(impl.Name()))
	}
	sort.Strings(names)
	return names
}

func deferredToolSection(names []string) string {
	if len(names) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<available-deferred-tools>\n")
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		b.WriteString(name)
		b.WriteByte('\n')
	}
	b.WriteString("</available-deferred-tools>")
	return b.String()
}
