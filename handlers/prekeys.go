package handlers

import (
	"log"

	"github.com/BaptTF/sickgnal-server/models"
	"github.com/BaptTF/sickgnal-server/protocol"
)

const PreKeyLimit = 100

// HandlePreKeyUpload processes pre-key upload requests (ty=133).
func (h *Handler) HandlePreKeyUpload(ctx *Context, msg *protocol.PreKeyUpload) interface{} {
	userID, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	tx := h.db.Begin()

	// Handle signed pre-key update
	if msg.PK != nil {
		// Delete existing signed pre-key
		tx.Where("user_id = ?", userID).Delete(&models.SignedPreKey{})

		spk := &models.SignedPreKey{
			UserID:    userID,
			Key:       msg.PK.Key,
			Signature: msg.PK.Signature,
		}
		if err := tx.Create(spk).Error; err != nil {
			tx.Rollback()
			log.Printf("PreKeyUpload: failed to create signed pre-key: %v", err)
			return protocol.NewError(protocol.ErrInternalError)
		}
	}

	// Handle ephemeral pre-keys
	if msg.TKs != nil {
		if msg.Replace {
			// Delete all existing ephemeral pre-keys
			tx.Where("user_id = ?", userID).Delete(&models.EphemeralPreKey{})
		}

		// Check current count
		var currentCount int64
		tx.Model(&models.EphemeralPreKey{}).Where("user_id = ?", userID).Count(&currentCount)

		if int(currentCount)+len(msg.TKs) > PreKeyLimit {
			tx.Rollback()
			return protocol.NewError(protocol.ErrPreKeyLimit)
		}

		// Insert new ephemeral pre-keys
		for _, tk := range msg.TKs {
			epk := &models.EphemeralPreKey{
				ID:     tk.ID,
				UserID: userID,
				Key:    tk.Key,
			}
			if err := tx.Create(epk).Error; err != nil {
				tx.Rollback()
				log.Printf("PreKeyUpload: failed to create ephemeral pre-key: %v", err)
				return protocol.NewError(protocol.ErrInternalError)
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("PreKeyUpload: commit failed: %v", err)
		return protocol.NewError(protocol.ErrInternalError)
	}

	return protocol.NewOk()
}

// HandlePreKeyDelete processes pre-key deletion requests (ty=134).
func (h *Handler) HandlePreKeyDelete(ctx *Context, msg *protocol.PreKeyDelete) interface{} {
	userID, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	// Delete specified ephemeral pre-keys (silently ignore unknown IDs)
	if len(msg.Keys) > 0 {
		h.db.Where("id IN ? AND user_id = ?", msg.Keys, userID).Delete(&models.EphemeralPreKey{})
	}

	return protocol.NewOk()
}

// HandlePreKeyStatusRequest processes pre-key status requests (ty=135).
func (h *Handler) HandlePreKeyStatusRequest(ctx *Context, msg *protocol.PreKeyStatusRequest) interface{} {
	_, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	// Get user ID from token
	userID, _ := h.validateToken(msg.Token)

	// Get all ephemeral pre-key IDs for this user
	var keys []models.EphemeralPreKey
	h.db.Select("id").Where("user_id = ?", userID).Find(&keys)

	keyIDs := make([]string, len(keys))
	for i, k := range keys {
		keyIDs[i] = k.ID
	}

	return protocol.NewPreKeyStatus(PreKeyLimit, keyIDs)
}

// HandlePreKeyBundleRequest processes pre-key bundle requests (ty=137).
func (h *Handler) HandlePreKeyBundleRequest(ctx *Context, msg *protocol.PreKeyBundleRequest) interface{} {
	_, errResp := h.validateToken(msg.Token)
	if errResp != nil {
		return errResp
	}

	// Look up target user
	var targetUser models.User
	if err := h.db.Where("id = ?", msg.ID).First(&targetUser).Error; err != nil {
		return protocol.NewError(protocol.ErrUserNotFound)
	}

	// Get signed pre-key
	var spk models.SignedPreKey
	if err := h.db.Where("user_id = ?", msg.ID).First(&spk).Error; err != nil {
		return protocol.NewError(protocol.ErrNoAvailableKey)
	}

	// Try to get one ephemeral pre-key (and consume it)
	var ek *protocol.EphemeralKey
	var ephKey models.EphemeralPreKey
	if err := h.db.Where("user_id = ?", msg.ID).First(&ephKey).Error; err == nil {
		ek = &protocol.EphemeralKey{
			ID:  ephKey.ID,
			Key: ephKey.Key,
		}
		// Delete the consumed ephemeral pre-key
		h.db.Delete(&ephKey)
	}
	// ek is nil if no ephemeral pre-key available (this is OK per spec)

	bundle := protocol.NewPreKeyBundle(targetUser.IdentityKey, spk.Key, spk.Signature, ek)
	return bundle
}
