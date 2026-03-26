package handlers

import (
	"encoding/json"
	"log"

	"github.com/BaptTF/sickgnal-server/models"
	"github.com/BaptTF/sickgnal-server/protocol"
)

// buildRelayMessage takes the raw incoming JSON, removes server-only fields
// (ty, token, rcpt_id) and adds the relay fields (ty, sndr_id).
// It uses json.RawMessage to preserve the exact structure of all other fields,
// including any duplicate keys that the Rust serde flatten produces.
func buildRelayMessage(raw json.RawMessage, newTy string, senderID string) (json.RawMessage, error) {
	// Decode the raw message preserving order and duplicates via ordered iteration.
	// Go's json.Decoder reads tokens in order, so we can reconstruct the JSON
	// with our modifications while preserving all fields.
	dec := json.NewDecoder(bytesReader(raw))
	dec.UseNumber()

	// Read opening brace
	t, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return nil, json.Unmarshal(raw, &struct{}{}) // trigger a proper error
	}

	// Build output manually
	out := []byte(`{"ty":`)
	tyJSON, _ := json.Marshal(newTy)
	out = append(out, tyJSON...)
	out = append(out, `,"sndr_id":`...)
	senderJSON, _ := json.Marshal(senderID)
	out = append(out, senderJSON...)

	// Skip fields: ty, token, rcpt_id — copy everything else
	for dec.More() {
		// Read key
		keyToken, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key := keyToken.(string)

		// Read value as raw
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}

		// Skip server-only fields
		if key == "ty" || key == "token" || key == "rcpt_id" {
			continue
		}

		// Append to output
		keyJSON, _ := json.Marshal(key)
		out = append(out, ',')
		out = append(out, keyJSON...)
		out = append(out, ':')
		out = append(out, val...)
	}

	out = append(out, '}')
	return json.RawMessage(out), nil
}

// bytesReader wraps a byte slice for json.NewDecoder.
func bytesReader(b []byte) *bytesReaderType {
	return &bytesReaderType{data: b, pos: 0}
}

type bytesReaderType struct {
	data []byte
	pos  int
}

func (r *bytesReaderType) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, nil
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// HandleSendInitialMessage processes initial message send requests (ty=150).
// This stores a type-1 ConversationOpen message for the recipient.
func (h *Handler) HandleSendInitialMessage(ctx *Context, msg *protocol.SendInitialMessage) interface{} {
	senderID, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	// Verify recipient exists
	var recipientCount int64
	h.db.Model(&models.User{}).Where("id = ?", msg.RecipientID).Count(&recipientCount)
	if recipientCount == 0 {
		return protocol.NewError(protocol.ErrUserNotFound)
	}

	// Build relay message preserving all fields exactly as sent by the client
	payload, err := buildRelayMessage(ctx.RawMessage, protocol.TyConversationOpen, senderID)
	if err != nil {
		log.Printf("SendInitialMessage: build relay error: %v", err)
		return protocol.NewError(protocol.ErrInternalError)
	}

	// Try instant relay, or store for later
	if h.relay.Push(msg.RecipientID, payload) {
		return protocol.NewOk()
	}

	stored := &models.StoredInitialMessage{
		RecipientID: msg.RecipientID,
		Payload:     string(payload),
	}
	if err := h.db.Create(stored).Error; err != nil {
		log.Printf("SendInitialMessage: db error: %v", err)
		return protocol.NewError(protocol.ErrInternalError)
	}

	return protocol.NewOk()
}

// HandleSendMessage processes regular message send requests (ty=151).
// This stores a type-2 ConversationMessage for the recipient.
func (h *Handler) HandleSendMessage(ctx *Context, msg *protocol.SendMessage) interface{} {
	senderID, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	// Verify recipient exists
	var recipientCount int64
	h.db.Model(&models.User{}).Where("id = ?", msg.RecipientID).Count(&recipientCount)
	if recipientCount == 0 {
		return protocol.NewError(protocol.ErrUserNotFound)
	}

	// Build relay message
	payload, err := buildRelayMessage(ctx.RawMessage, protocol.TyConversationMessage, senderID)
	if err != nil {
		log.Printf("SendMessage: build relay error: %v", err)
		return protocol.NewError(protocol.ErrInternalError)
	}

	// Try instant relay, or store for later
	if h.relay.Push(msg.RecipientID, payload) {
		return protocol.NewOk()
	}

	stored := &models.StoredMessage{
		RecipientID: msg.RecipientID,
		Payload:     string(payload),
	}
	if err := h.db.Create(stored).Error; err != nil {
		log.Printf("SendMessage: db error: %v", err)
		return protocol.NewError(protocol.ErrInternalError)
	}

	return protocol.NewOk()
}

// HandleGetInitialMessages processes requests to retrieve stored initial messages (ty=160).
func (h *Handler) HandleGetInitialMessages(ctx *Context, msg *protocol.GetInitialMessages) interface{} {
	userID, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	limit := msg.Limit
	if limit <= 0 {
		limit = 100
	}

	var stored []models.StoredInitialMessage
	h.db.Where("recipient_id = ?", userID).
		Order("created_at ASC").
		Limit(limit).
		Find(&stored)

	msgs := make([]json.RawMessage, len(stored))
	ids := make([]uint, len(stored))
	for i, s := range stored {
		msgs[i] = json.RawMessage(s.Payload)
		ids[i] = s.ID
	}

	if len(ids) > 0 {
		h.db.Where("id IN ?", ids).Delete(&models.StoredInitialMessage{})
	}

	return protocol.NewMessagesList(msgs)
}

// HandleGetMessages processes requests to retrieve stored regular messages (ty=161).
func (h *Handler) HandleGetMessages(ctx *Context, msg *protocol.GetMessages) interface{} {
	userID, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	limit := msg.Limit
	if limit <= 0 {
		limit = 100
	}

	var stored []models.StoredMessage
	h.db.Where("recipient_id = ?", userID).
		Order("created_at ASC").
		Limit(limit).
		Find(&stored)

	msgs := make([]json.RawMessage, len(stored))
	ids := make([]uint, len(stored))
	for i, s := range stored {
		msgs[i] = json.RawMessage(s.Payload)
		ids[i] = s.ID
	}

	if len(ids) > 0 {
		h.db.Where("id IN ?", ids).Delete(&models.StoredMessage{})
	}

	return protocol.NewMessagesList(msgs)
}
