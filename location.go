package voipinfra

import (
	"context"
	"net"
	"sync"
	"time"
)

type Location struct {
	IP        string
	Port      int
	ExpiresAt time.Time
}

type SIPMessage struct {
	Method     string
	RequestURI string
	Version    string
	Headers    map[string]string
	Body       string
}

type Registration struct {
	Username  string
	Contact   string
	IP        net.IP
	Port      int
	ExpiresAt time.Time
	From      string
	To        string
	Via       string
}

type Registrar struct {
	mu            sync.RWMutex
	registrations map[string]Registration
}

func NewRegistrar() *Registrar {
	return &Registrar{
		registrations: make(map[string]Registration),
	}
}
func (r *Registrar) Register(registration Registration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registrations[registration.Username] = registration
}

func (r *Registrar) Get(username string) (Registration, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	registration, ok := r.registrations[username]
	return registration, ok
}
func (r *Registrar) Remove(username string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.registrations, username)
}
func (r *Registrar) removeExpired() {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	for key, registration := range r.registrations {
		if now.After(registration.ExpiresAt) {
			delete(r.registrations, key)
		}
	}
}
func (r *Registrar) Cleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.removeExpired()

		case <-ctx.Done():
			return
		}
	}
}
