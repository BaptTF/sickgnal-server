package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"log"
	"time"

	"github.com/BaptTF/sickgnal-server/crypto"
	"github.com/BaptTF/sickgnal-server/models"
	"github.com/BaptTF/sickgnal-server/protocol"
	"github.com/google/uuid"
)

// HandleCreateAccount processes account creation requests (ty=128).
func (h *Handler) HandleCreateAccount(ctx *Context, msg *protocol.CreateAccount) interface{} {
	// Validate the signature: sig = Ed25519Sign(identity_key, username)
	if err := crypto.VerifyAccountCreation(msg.IK, msg.Username, msg.Signature); err != nil {
		log.Printf("CreateAccount: signature verification failed: %v", err)
		return protocol.NewError(protocol.ErrInvalidAuthentication)
	}

	// Check if username is already taken
	var count int64
	h.db.Model(&models.User{}).Where("username = ?", msg.Username).Count(&count)
	if count > 0 {
		return protocol.NewError(protocol.ErrUsernameUnavailable)
	}

	// Generate UUID and token
	userID := uuid.New().String()
	token := generateToken()

	// Create user
	user := &models.User{
		ID:          userID,
		Username:    msg.Username,
		IdentityKey: msg.IK,
		Token:       token,
	}

	if err := h.db.Create(user).Error; err != nil {
		log.Printf("CreateAccount: db error: %v", err)
		// Race condition: username was taken between check and insert
		return protocol.NewError(protocol.ErrUsernameUnavailable)
	}

	// Update connection auth state
	ctx.SetAuth(userID, token)

	log.Printf("Account created: user=%s id=%s", msg.Username, userID)
	return protocol.NewAuthToken(userID, token)
}

// HandleAuthChallengeRequest processes authentication challenge requests (ty=130).
func (h *Handler) HandleAuthChallengeRequest(ctx *Context, msg *protocol.AuthChallengeRequest) interface{} {
	// Check if user exists
	var user models.User
	if err := h.db.Where("username = ?", msg.Username).First(&user).Error; err != nil {
		return protocol.NewError(protocol.ErrUserNotFound)
	}

	// Generate a 24-byte random nonce
	nonce := make([]byte, 24)
	if _, err := rand.Read(nonce); err != nil {
		log.Printf("AuthChallengeRequest: failed to generate nonce: %v", err)
		return protocol.NewError(protocol.ErrInternalError)
	}

	b64Nonce := base64.StdEncoding.EncodeToString(nonce)

	// Store or update the challenge (upsert)
	challenge := &models.AuthChallenge{
		Username:  msg.Username,
		Nonce:     b64Nonce,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	// Delete any existing challenge for this user, then create new one
	h.db.Where("username = ?", msg.Username).Delete(&models.AuthChallenge{})
	if err := h.db.Create(challenge).Error; err != nil {
		log.Printf("AuthChallengeRequest: db error: %v", err)
		return protocol.NewError(protocol.ErrInternalError)
	}

	return protocol.NewAuthChallenge(b64Nonce)
}

// HandleAuthChallengeSolve processes authentication challenge solutions (ty=132).
func (h *Handler) HandleAuthChallengeSolve(ctx *Context, msg *protocol.AuthChallengeSolve) interface{} {
	// Look up the pending challenge by nonce
	var challenge models.AuthChallenge
	if err := h.db.Where("nonce = ?", msg.Challenge).First(&challenge).Error; err != nil {
		log.Printf("AuthChallengeSolve: challenge not found")
		return protocol.NewError(protocol.ErrInvalidAuthentication)
	}

	// Check expiry
	if time.Now().After(challenge.ExpiresAt) {
		h.db.Delete(&challenge)
		return protocol.NewError(protocol.ErrInvalidAuthentication)
	}

	// Look up user
	var user models.User
	if err := h.db.Where("username = ?", challenge.Username).First(&user).Error; err != nil {
		return protocol.NewError(protocol.ErrUserNotFound)
	}

	// Verify the signature: sig(SHA512(nonce) || username)
	if err := crypto.VerifyAuthChallenge(user.IdentityKey, user.Username, msg.Challenge, msg.Solve); err != nil {
		log.Printf("AuthChallengeSolve: verification failed: %v", err)
		// Clean up the challenge
		h.db.Delete(&challenge)
		return protocol.NewError(protocol.ErrInvalidAuthentication)
	}

	// Delete the used challenge
	h.db.Delete(&challenge)

	// Generate new token
	token := generateToken()
	h.db.Model(&user).Update("token", token)

	// Update connection auth state
	ctx.SetAuth(user.ID, token)

	log.Printf("Auth challenge solved: user=%s id=%s", user.Username, user.ID)
	return protocol.NewAuthToken(user.ID, token)
}

// generateToken generates a random 32-byte hex token.
func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random token: " + err.Error())
	}
	return hex.EncodeToString(b)
}
