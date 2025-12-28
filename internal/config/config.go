package config

import (
	"sync"
	"sync/atomic"
)

type Provider struct {
	ID     string  `json:"id"`
	URL    string  `json:"url"`
	Weight float64 `json:"weight"`
	Active bool    `json:"active"`
}

type RouterConfig struct {
	mu           sync.RWMutex
	Providers    []Provider
	requestCount uint64 // Used for deterministic selection
}

// SelectProvider implements a deterministic weighted selection
func (s *RouterConfig) SelectProvider() *Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var activeProviders []Provider
	for _, p := range s.Providers {
		if p.Active {
			activeProviders = append(activeProviders, p)
		}
	}

	if len(activeProviders) == 0 {
		return nil
	}

	// 1. Increment total requests atomically
	current := atomic.AddUint64(&s.requestCount, 1)
	
	// 2. Use modulo to find position in a 100-slot window
	// This ensures that over 100 requests, the distribution matches weights exactly.
	pos := current % 100

	var cumulative float64
	for _, p := range activeProviders {
		cumulative += p.Weight
		if float64(pos) < cumulative {
			return &p
		}
	}

	return &activeProviders[0]
}

func (s *RouterConfig) UpdateProviders(newProviders []Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Providers = newProviders
	// Reset counter on config change to start fresh window
	atomic.StoreUint64(&s.requestCount, 0)
}

func (s *RouterConfig) GetProviders() []Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Providers
}