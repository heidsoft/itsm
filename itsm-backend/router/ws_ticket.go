package router

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// wsTicket holds the user identity data stored when a short-lived
// WebSocket ticket is created. The ticket is single-use and expires
// after a short TTL.
type wsTicket struct {
	UserID   int
	TenantID int
	ExpiresAt time.Time
}

// WSTicketStore manages short-lived, single-use tickets for WebSocket
// authentication. Tickets replace the insecure practice of passing JWT
// tokens via query strings.
type WSTicketStore struct {
	tickets sync.Map
	ttl     time.Duration
}

const (
	// WSTicketBytes is the number of random bytes in a ticket value.
	WSTicketBytes = 32
	// DefaultWSTicketTTL is the default time-to-live for a ticket.
	DefaultWSTicketTTL = 30 * time.Second
)

// NewWSTicketStore creates a ticket store with the given TTL.
func NewWSTicketStore(ttl time.Duration) *WSTicketStore {
	store := &WSTicketStore{
		ttl: ttl,
	}
	go store.cleanup()
	return store
}

// Generate creates a new short-lived ticket associated with the given
// user and tenant IDs. The returned string is a hex-encoded random
// token that the client passes as a query parameter when opening a
// WebSocket connection.
func (s *WSTicketStore) Generate(userID, tenantID int) (string, error) {
	raw := make([]byte, WSTicketBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}

	ticketStr := hex.EncodeToString(raw)
	s.tickets.Store(ticketStr, wsTicket{
		UserID:    userID,
		TenantID:  tenantID,
		ExpiresAt: time.Now().Add(s.ttl),
	})
	return ticketStr, nil
}

// Redeem looks up a ticket by value. If found and not expired, it
// returns the stored user/tenant IDs and deletes the ticket (single-use).
// Returns false if the ticket is missing or expired.
func (s *WSTicketStore) Redeem(ticketStr string) (userID int, tenantID int, ok bool) {
	val, loaded := s.tickets.LoadAndDelete(ticketStr)
	if !loaded {
		return 0, 0, false
	}

	t := val.(wsTicket)
	if time.Now().After(t.ExpiresAt) {
		return 0, 0, false
	}

	return t.UserID, t.TenantID, true
}

// cleanup periodically removes expired tickets from the store so
// memory does not grow unbounded over time.
func (s *WSTicketStore) cleanup() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		s.tickets.Range(func(key, value interface{}) bool {
			t := value.(wsTicket)
			if now.After(t.ExpiresAt) {
				s.tickets.Delete(key)
			}
			return true
		})
	}
}
