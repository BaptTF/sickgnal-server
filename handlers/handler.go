package handlers

import (
	"encoding/json"
	"log"

	"github.com/BaptTF/sickgnal-server/protocol"
	"gorm.io/gorm"
)

// Context holds per-request state passed to handlers.
type Context struct {
	RequestID  uint16
	Writer     *protocol.ConnWriter
	UserID     string // Current authenticated user ID (empty if not authenticated)
	Token      string // Current auth token (empty if not authenticated)
	SetAuth    func(userID, token string)
	ConnWriter *protocol.ConnWriter
	RawMessage json.RawMessage // The raw JSON of the current message
}

// Handler dispatches messages to the appropriate handler functions.
type Handler struct {
	db    *gorm.DB
	relay *RelayManager
}

// New creates a new Handler.
func New(db *gorm.DB) *Handler {
	return &Handler{
		db:    db,
		relay: NewRelayManager(),
	}
}

// Relay returns the relay manager for connection management.
func (h *Handler) Relay() *RelayManager {
	return h.relay
}

// Handle processes a parsed message and returns a response (or nil if no response needed).
func (h *Handler) Handle(ctx *Context, msg interface{}, ty string) interface{} {
	switch ty {
	case protocol.TyCreateAccount:
		return h.HandleCreateAccount(ctx, msg.(*protocol.CreateAccount))

	case protocol.TyAuthChallengeRequest:
		return h.HandleAuthChallengeRequest(ctx, msg.(*protocol.AuthChallengeRequest))

	case protocol.TyAuthChallengeSolve:
		return h.HandleAuthChallengeSolve(ctx, msg.(*protocol.AuthChallengeSolve))

	case protocol.TyPreKeyUpload:
		return h.HandlePreKeyUpload(ctx, msg.(*protocol.PreKeyUpload))

	case protocol.TyPreKeyDelete:
		return h.HandlePreKeyDelete(ctx, msg.(*protocol.PreKeyDelete))

	case protocol.TyPreKeyStatusRequest:
		return h.HandlePreKeyStatusRequest(ctx, msg.(*protocol.PreKeyStatusRequest))

	case protocol.TyPreKeyBundleRequest:
		return h.HandlePreKeyBundleRequest(ctx, msg.(*protocol.PreKeyBundleRequest))

	case protocol.TyUserProfileByUsername:
		return h.HandleUserProfileByUsername(ctx, msg.(*protocol.UserProfileByUsername))

	case protocol.TyUserProfileById:
		return h.HandleUserProfileById(ctx, msg.(*protocol.UserProfileById))

	case protocol.TySendInitialMessage:
		return h.HandleSendInitialMessage(ctx, msg.(*protocol.SendInitialMessage))

	case protocol.TySendMessage:
		return h.HandleSendMessage(ctx, msg.(*protocol.SendMessage))

	case protocol.TyGetInitialMessages:
		return h.HandleGetInitialMessages(ctx, msg.(*protocol.GetInitialMessages))

	case protocol.TyGetMessages:
		return h.HandleGetMessages(ctx, msg.(*protocol.GetMessages))

	case protocol.TyEnableInstantRelay:
		return h.HandleEnableInstantRelay(ctx, msg.(*protocol.EnableInstantRelay))

	case protocol.TyDisableInstantRelay:
		return h.HandleDisableInstantRelay(ctx)

	default:
		log.Printf("Unhandled message type: %s", ty)
		return protocol.NewError(protocol.ErrMessageTypeNotAccepted)
	}
}

// validateToken checks if the provided token is valid and returns the user ID.
// Returns an error response if invalid.
func (h *Handler) validateToken(token string) (string, interface{}) {
	if token == "" {
		return "", protocol.NewError(protocol.ErrInvalidAuthentication)
	}

	var user struct {
		ID string
	}
	result := h.db.Table("users").Select("id").Where("token = ?", token).First(&user)
	if result.Error != nil {
		return "", protocol.NewError(protocol.ErrInvalidAuthentication)
	}

	return user.ID, nil
}

// getUsernameByToken looks up the username associated with an auth token.
// Returns an empty string if the token is invalid or the user is not found.
func (h *Handler) getUsernameByToken(token string) string {
	var user struct {
		Username string
	}
	h.db.Table("users").Select("username").Where("token = ?", token).First(&user)
	return user.Username
}
