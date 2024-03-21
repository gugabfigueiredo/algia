package domain

import "github.com/nbd-wtf/go-nostr"

// Event is
type Event struct {
	Event   *nostr.Event `json:"event"`
	Profile Profile      `json:"profile"`
}
