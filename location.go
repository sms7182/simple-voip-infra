package voipinfra

import "time"

type Location struct {
	IP        string    `json:"ip"`
	Port      int       `json:"port"`
	ExpiresAt time.Time `json:"expiresAt"`
}
