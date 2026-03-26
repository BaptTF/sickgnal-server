package handlers

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/BaptTF/sickgnal-server/models"
	"github.com/BaptTF/sickgnal-server/protocol"
)

// RelayManager manages instant relay connections.
type RelayManager struct {
	mu      sync.RWMutex
	clients map[string]*relayEntry // userID -> relay entry
}

type relayEntry struct {
	writer *protocol.ConnWriter
}

// NewRelayManager creates a new relay manager.
func NewRelayManager() *RelayManager {
	return &RelayManager{
		clients: make(map[string]*relayEntry),
	}
}

// Register registers a connection for instant relay.
func (rm *RelayManager) Register(userID string, writer *protocol.ConnWriter) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.clients[userID] = &relayEntry{writer: writer}
	log.Printf("Instant relay enabled for user %s", userID)
}

// Unregister removes a connection from instant relay.
func (rm *RelayManager) Unregister(userID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.clients, userID)
	log.Printf("Instant relay disabled for user %s", userID)
}

// IsRegistered checks if a user has an active relay connection.
func (rm *RelayManager) IsRegistered(userID string) bool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	_, ok := rm.clients[userID]
	return ok
}

// Push attempts to push a raw JSON message to a user via instant relay.
// Returns true if the message was successfully pushed, false if the user
// is not registered for instant relay.
func (rm *RelayManager) Push(userID string, payload json.RawMessage) bool {
	rm.mu.RLock()
	entry, ok := rm.clients[userID]
	rm.mu.RUnlock()

	if !ok {
		return false
	}

	// Send with request ID 0 (unsolicited server push)
	if err := protocol.WriteRawPacket(entry.writer, 0, payload); err != nil {
		log.Printf("Instant relay push failed for user %s: %v", userID, err)
		return false
	}

	return true
}

// HandleEnableInstantRelay processes enable instant relay requests (ty=180).
func (h *Handler) HandleEnableInstantRelay(ctx *Context, msg *protocol.EnableInstantRelay) interface{} {
	userID, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	// Register this connection for instant relay
	h.relay.Register(userID, ctx.ConnWriter)

	// Update connection auth state
	ctx.SetAuth(userID, msg.Token)

	// Flush all stored messages to the client
	h.flushStoredMessages(ctx, userID)

	// No explicit response needed — the flushed messages serve as confirmation
	// But we should send Ok to acknowledge the request
	return protocol.NewOk()
}

// HandleDisableInstantRelay processes disable instant relay requests (ty=181).
func (h *Handler) HandleDisableInstantRelay(ctx *Context) interface{} {
	if ctx.UserID != "" {
		h.relay.Unregister(ctx.UserID)
	}
	return protocol.NewOk()
}

// flushStoredMessages sends all stored messages for a user via instant relay.
func (h *Handler) flushStoredMessages(ctx *Context, userID string) {
	// Flush initial messages first
	var initialMsgs []models.StoredInitialMessage
	h.db.Where("recipient_id = ?", userID).
		Order("created_at ASC").
		Find(&initialMsgs)

	if len(initialMsgs) > 0 {
		ids := make([]uint, len(initialMsgs))
		for i, msg := range initialMsgs {
			ids[i] = msg.ID
			if err := protocol.WriteRawPacket(ctx.ConnWriter, 0, json.RawMessage(msg.Payload)); err != nil {
				log.Printf("Flush initial message failed for user %s: %v", userID, err)
				return
			}
		}
		h.db.Where("id IN ?", ids).Delete(&models.StoredInitialMessage{})
	}

	// Flush regular messages
	var regularMsgs []models.StoredMessage
	h.db.Where("recipient_id = ?", userID).
		Order("created_at ASC").
		Find(&regularMsgs)

	if len(regularMsgs) > 0 {
		ids := make([]uint, len(regularMsgs))
		for i, msg := range regularMsgs {
			ids[i] = msg.ID
			if err := protocol.WriteRawPacket(ctx.ConnWriter, 0, json.RawMessage(msg.Payload)); err != nil {
				log.Printf("Flush regular message failed for user %s: %v", userID, err)
				return
			}
		}
		h.db.Where("id IN ?", ids).Delete(&models.StoredMessage{})
	}
}
