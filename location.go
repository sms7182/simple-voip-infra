package voipinfra

import "time"

type Location struct {
	IP        string
	Port      int
	ExpiresAt time.Time
}
