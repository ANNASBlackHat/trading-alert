package target

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type InMemoryStore struct {
	mu      sync.RWMutex
	targets map[string]Target
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		targets: make(map[string]Target),
	}
}

func (s *InMemoryStore) Save(target Target) (Target, error) {
	if target.Symbol == "" {
		return Target{}, fmt.Errorf("symbol is required")
	}

	target.Symbol = strings.ToUpper(target.Symbol)
	target.Direction = Direction(strings.ToLower(string(target.Direction)))
	if !IsValidDirection(string(target.Direction)) {
		return Target{}, fmt.Errorf("invalid direction: must be 'up' or 'down'")
	}

	if target.Type == "" {
		target.Type = TargetTypePrice
	}

	if target.Type == TargetTypePrice {
		if target.TargetPrice <= 0 {
			return Target{}, fmt.Errorf("target_price must be greater than zero for price alerts")
		}
	} else if target.Type == TargetTypeTrailing {
		if target.TrailingPercent <= 0 && target.TrailingValue <= 0 {
			return Target{}, fmt.Errorf("either trailing_percent or trailing_value must be greater than zero for trailing stop alerts")
		}
		if target.ActivationPrice <= 0 {
			target.IsActive = true
		} else {
			target.IsActive = false
		}
	} else {
		return Target{}, fmt.Errorf("invalid target type: %s", target.Type)
	}

	target.ID = generateID(target.Symbol)
	target.CreatedAt = time.Now().UTC()
	target.LastState = StateUnknown

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.targets[target.ID]; exists {
		// A tiny chance of collision; regenerate once.
		target.ID = generateID(target.Symbol)
	}
	s.targets[target.ID] = target
	return target, nil
}

func (s *InMemoryStore) Get(id string) (Target, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	target, ok := s.targets[id]
	if !ok {
		return Target{}, fmt.Errorf("target not found")
	}
	return target, nil
}

func (s *InMemoryStore) ListAll() ([]Target, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]Target, 0, len(s.targets))
	for _, t := range s.targets {
		list = append(list, t)
	}
	return list, nil
}

func (s *InMemoryStore) ListBySymbol(symbol string) ([]Target, error) {
	symbol = strings.ToUpper(symbol)
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]Target, 0)
	for _, t := range s.targets {
		if strings.EqualFold(t.Symbol, symbol) {
			list = append(list, t)
		}
	}
	return list, nil
}

func (s *InMemoryStore) ListByBotName(botName string) ([]Target, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]Target, 0)
	for _, t := range s.targets {
		if t.BotName == botName {
			list = append(list, t)
		}
	}
	return list, nil
}

func (s *InMemoryStore) Update(target Target) error {
	if target.ID == "" {
		return fmt.Errorf("target id is required")
	}
	target.Symbol = strings.ToUpper(target.Symbol)

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.targets[target.ID]; !ok {
		return fmt.Errorf("target not found")
	}
	s.targets[target.ID] = target
	return nil
}

func (s *InMemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.targets[id]; !ok {
		return fmt.Errorf("target not found")
	}
	delete(s.targets, id)
	return nil
}

func generateID(symbol string) string {
	return fmt.Sprintf("%s-%d", strings.ToLower(strings.ReplaceAll(symbol, " ", "_")), time.Now().UnixNano())
}
