package handlers

import (
	"testing"

	"github.com/BaptTF/sickgnal-server/protocol"
	"github.com/google/uuid"
)

func TestUserProfileByUsername(t *testing.T) {
	h := setupTestHandler(t)
	aliceID, aliceToken, _ := createTestAccount(t, h, "alice")
	createTestAccount(t, h, "bob")

	ctx := testContext(t)
	resp := h.HandleUserProfileByUsername(ctx, &protocol.UserProfileByUsername{
		Ty:       "140",
		Token:    aliceToken,
		Username: "bob",
	})

	profile, ok := resp.(*protocol.UserProfile)
	if !ok {
		t.Fatalf("expected UserProfile, got %T: %+v", resp, resp)
	}
	if profile.Ty != "10" {
		t.Errorf("expected ty=10, got %s", profile.Ty)
	}
	if profile.Username != "bob" {
		t.Errorf("expected username=bob, got %s", profile.Username)
	}
	if profile.ID == "" {
		t.Error("profile ID should not be empty")
	}
	if profile.ID == aliceID {
		t.Error("profile ID should be Bob's, not Alice's")
	}
}

func TestUserProfileByUsernameNotFound(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")

	ctx := testContext(t)
	resp := h.HandleUserProfileByUsername(ctx, &protocol.UserProfileByUsername{
		Ty:       "140",
		Token:    aliceToken,
		Username: "nonexistent",
	})

	assertError(t, resp, protocol.ErrUserNotFound)
}

func TestUserProfileById(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	bobID, _, _ := createTestAccount(t, h, "bob")

	ctx := testContext(t)
	resp := h.HandleUserProfileById(ctx, &protocol.UserProfileById{
		Ty:    "141",
		Token: aliceToken,
		ID:    bobID,
	})

	profile, ok := resp.(*protocol.UserProfile)
	if !ok {
		t.Fatalf("expected UserProfile, got %T: %+v", resp, resp)
	}
	if profile.Username != "bob" {
		t.Errorf("expected username=bob, got %s", profile.Username)
	}
	if profile.ID != bobID {
		t.Errorf("expected ID=%s, got %s", bobID, profile.ID)
	}
}

func TestUserProfileByIdNotFound(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")

	ctx := testContext(t)
	resp := h.HandleUserProfileById(ctx, &protocol.UserProfileById{
		Ty:    "141",
		Token: aliceToken,
		ID:    uuid.New().String(),
	})

	assertError(t, resp, protocol.ErrUserNotFound)
}

func TestUserProfileInvalidToken(t *testing.T) {
	h := setupTestHandler(t)

	ctx := testContext(t)
	resp := h.HandleUserProfileByUsername(ctx, &protocol.UserProfileByUsername{
		Ty:       "140",
		Token:    "bad_token",
		Username: "anyone",
	})

	assertError(t, resp, protocol.ErrInvalidAuthentication)
}
