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

	for _, p := range newProviders {
		log.Printf("[INFO] Provider registered: id=%s, url=%s, weight=%.1f, active=%v", p.ID, p.URL, p.Weight, p.Active)
	}
}

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
		log.Println("[WARN] No active providers available for routing")
		return nil
	}

	r := rand.Float64() * totalWeight
	for _, p := range active {
		r -= p.Weight
		if r <= 0 {
			log.Printf("[DEBUG] Provider selected: id=%s, weight=%.1f, total_weight=%.1f", p.ID, p.Weight, totalWeight)
			return &p
		}
	}
	log.Printf("[DEBUG] Fallback to first provider: id=%s", active[0].ID)
	return &active[0]
}
