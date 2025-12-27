package config

import (
	"log"
	"math/rand"
	"sync"
)

type Provider struct {
	ID     string  `json:"id"`
	URL    string  `json:"url"`
	Weight float64 `json:"weight"`
	Active bool    `json:"active"`
}

type RouterConfig struct {
	sync.RWMutex
	Providers []Provider
}

// UpdateProviders allows the Admin API to update weights/providers without a restart
func (c *RouterConfig) UpdateProviders(newProviders []Provider) {
	c.Lock()
	defer c.Unlock()

	oldCount := len(c.Providers)
	c.Providers = newProviders
	log.Printf("[INFO] Providers updated: old_count=%d, new_count=%d", oldCount, len(newProviders))
}

// GetProviders returns a thread-safe copy for the Admin Stats API
func (c *RouterConfig) GetProviders() []Provider {
	c.RLock()
	defer c.RUnlock()
	
	dst := make([]Provider, len(c.Providers))
	copy(dst, c.Providers)
	return dst
}

// GetProvider selects a provider based on weighted random selection
func (c *RouterConfig) GetProvider() *Provider {
	c.RLock()
	defer c.RUnlock()

	var active []Provider
	totalWeight := 0.0
	for _, p := range c.Providers {
		if p.Active {
			active = append(active, p)
			totalWeight += p.Weight
		}
	}

	if len(active) == 0 {
		return nil
	}

	r := rand.Float64() * totalWeight
	for i := range active {
		r -= active[i].Weight
		if r <= 0 {
			return &active[i]
		}
	}
	return &active[0]
}