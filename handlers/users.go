package handlers

import (
	"github.com/BaptTF/sickgnal-server/models"
	"github.com/BaptTF/sickgnal-server/protocol"
)

// HandleUserProfileByUsername processes user profile lookup by username (ty=140).
func (h *Handler) HandleUserProfileByUsername(ctx *Context, msg *protocol.UserProfileByUsername) interface{} {
	_, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	var user models.User
	if err := h.db.Where("username = ?", msg.Username).First(&user).Error; err != nil {
		return protocol.NewError(protocol.ErrUserNotFound)
	}

	return protocol.NewUserProfile(user.ID, user.Username)
}

// HandleUserProfileById processes user profile lookup by UUID (ty=141).
func (h *Handler) HandleUserProfileById(ctx *Context, msg *protocol.UserProfileById) interface{} {
	_, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	var user models.User
	if err := h.db.Where("id = ?", msg.ID).First(&user).Error; err != nil {
		return protocol.NewError(protocol.ErrUserNotFound)
	}

	return protocol.NewUserProfile(user.ID, user.Username)
}
