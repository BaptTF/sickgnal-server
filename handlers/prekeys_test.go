package handlers

import (
	"testing"

	"github.com/BaptTF/sickgnal-server/protocol"
	"github.com/google/uuid"
)

func TestPreKeyUploadHappyPath(t *testing.T) {
	h := setupTestHandler(t)
	_, token, _ := createTestAccount(t, h, "alice")

	ctx := testContext(t)
	resp := h.HandlePreKeyUpload(ctx, &protocol.PreKeyUpload{
		Ty:      "133",
		Token:   token,
		Replace: false,
		PK: &protocol.SignedPreKeyUpload{
			Key:       "base64_midterm_key",
			Signature: "base64_signature",
		},
		TKs: []protocol.EphemeralKeyUpload{
			{ID: uuid.New().String(), Key: "base64_ek1"},
			{ID: uuid.New().String(), Key: "base64_ek2"},
			{ID: uuid.New().String(), Key: "base64_ek3"},
		},
	})

	assertOk(t, resp)
}

func TestPreKeyUploadReplace(t *testing.T) {
	h := setupTestHandler(t)
	_, token, _ := createTestAccount(t, h, "alice")

	// Upload initial keys
	ctx := testContext(t)
	h.HandlePreKeyUpload(ctx, &protocol.PreKeyUpload{
		Ty:      "133",
		Token:   token,
		Replace: false,
		TKs: []protocol.EphemeralKeyUpload{
			{ID: uuid.New().String(), Key: "base64_ek1"},
			{ID: uuid.New().String(), Key: "base64_ek2"},
		},
	})

	// Check status - should have 2
	ctx2 := testContext(t)
	resp := h.HandlePreKeyStatusRequest(ctx2, &protocol.PreKeyStatusRequest{
		Ty:    "135",
		Token: token,
	})
	status := resp.(*protocol.PreKeyStatusMsg)
	if len(status.Keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(status.Keys))
	}

	// Upload with replace=true
	ctx3 := testContext(t)
	h.HandlePreKeyUpload(ctx3, &protocol.PreKeyUpload{
		Ty:      "133",
		Token:   token,
		Replace: true,
		TKs: []protocol.EphemeralKeyUpload{
			{ID: uuid.New().String(), Key: "base64_new_ek1"},
		},
	})

	// Check status - should have 1 (old ones replaced)
	ctx4 := testContext(t)
	resp2 := h.HandlePreKeyStatusRequest(ctx4, &protocol.PreKeyStatusRequest{
		Ty:    "135",
		Token: token,
	})
	status2 := resp2.(*protocol.PreKeyStatusMsg)
	if len(status2.Keys) != 1 {
		t.Errorf("expected 1 key after replace, got %d", len(status2.Keys))
	}
}

func TestPreKeyUploadLimitEnforced(t *testing.T) {
	h := setupTestHandler(t)
	_, token, _ := createTestAccount(t, h, "alice")

	// Upload 100 keys (at the limit)
	tks := make([]protocol.EphemeralKeyUpload, PreKeyLimit)
	for i := range tks {
		tks[i] = protocol.EphemeralKeyUpload{
			ID:  uuid.New().String(),
			Key: "base64_key",
		}
	}

	ctx := testContext(t)
	resp := h.HandlePreKeyUpload(ctx, &protocol.PreKeyUpload{
		Ty:      "133",
		Token:   token,
		Replace: true,
		TKs:     tks,
	})
	assertOk(t, resp)

	// Try to upload one more
	ctx2 := testContext(t)
	resp2 := h.HandlePreKeyUpload(ctx2, &protocol.PreKeyUpload{
		Ty:      "133",
		Token:   token,
		Replace: false,
		TKs: []protocol.EphemeralKeyUpload{
			{ID: uuid.New().String(), Key: "base64_one_too_many"},
		},
	})
	assertError(t, resp2, protocol.ErrPreKeyLimit)
}

func TestPreKeyUploadInvalidToken(t *testing.T) {
	h := setupTestHandler(t)

	ctx := testContext(t)
	resp := h.HandlePreKeyUpload(ctx, &protocol.PreKeyUpload{
		Ty:    "133",
		Token: "invalid_token",
	})

	assertError(t, resp, protocol.ErrInvalidAuthentication)
}

func TestPreKeyDelete(t *testing.T) {
	h := setupTestHandler(t)
	_, token, _ := createTestAccount(t, h, "alice")

	key1 := uuid.New().String()
	key2 := uuid.New().String()
	key3 := uuid.New().String()

	// Upload 3 keys
	ctx := testContext(t)
	h.HandlePreKeyUpload(ctx, &protocol.PreKeyUpload{
		Ty:      "133",
		Token:   token,
		Replace: true,
		TKs: []protocol.EphemeralKeyUpload{
			{ID: key1, Key: "k1"},
			{ID: key2, Key: "k2"},
			{ID: key3, Key: "k3"},
		},
	})

	// Delete 2 of them
	ctx2 := testContext(t)
	resp := h.HandlePreKeyDelete(ctx2, &protocol.PreKeyDelete{
		Ty:    "134",
		Token: token,
		Keys:  []string{key1, key3},
	})
	assertOk(t, resp)

	// Check status - should have 1 remaining
	ctx3 := testContext(t)
	resp2 := h.HandlePreKeyStatusRequest(ctx3, &protocol.PreKeyStatusRequest{
		Ty:    "135",
		Token: token,
	})
	status := resp2.(*protocol.PreKeyStatusMsg)
	if len(status.Keys) != 1 {
		t.Errorf("expected 1 key remaining, got %d", len(status.Keys))
	}
	if status.Keys[0] != key2 {
		t.Errorf("expected remaining key to be %s, got %s", key2, status.Keys[0])
	}
}

func TestPreKeyDeleteUnknownIDsSilentlyIgnored(t *testing.T) {
	h := setupTestHandler(t)
	_, token, _ := createTestAccount(t, h, "alice")

	ctx := testContext(t)
	resp := h.HandlePreKeyDelete(ctx, &protocol.PreKeyDelete{
		Ty:    "134",
		Token: token,
		Keys:  []string{"nonexistent-uuid-1", "nonexistent-uuid-2"},
	})
	assertOk(t, resp)
}

func TestPreKeyStatusRequest(t *testing.T) {
	h := setupTestHandler(t)
	_, token, _ := createTestAccount(t, h, "alice")

	// Empty initially
	ctx := testContext(t)
	resp := h.HandlePreKeyStatusRequest(ctx, &protocol.PreKeyStatusRequest{
		Ty:    "135",
		Token: token,
	})
	status := resp.(*protocol.PreKeyStatusMsg)
	if status.Ty != "136" {
		t.Errorf("expected ty=136, got %s", status.Ty)
	}
	if status.Limit != PreKeyLimit {
		t.Errorf("expected limit=%d, got %d", PreKeyLimit, status.Limit)
	}
	if len(status.Keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(status.Keys))
	}
}

func TestPreKeyBundleRequest(t *testing.T) {
	h := setupTestHandler(t)

	// Create Bob with pre-keys
	bobID, bobToken, _ := createTestAccount(t, h, "bob")
	ctx := testContext(t)
	ekID := uuid.New().String()
	h.HandlePreKeyUpload(ctx, &protocol.PreKeyUpload{
		Ty:      "133",
		Token:   bobToken,
		Replace: true,
		PK: &protocol.SignedPreKeyUpload{
			Key:       "bob_midterm_key",
			Signature: "bob_midterm_sig",
		},
		TKs: []protocol.EphemeralKeyUpload{
			{ID: ekID, Key: "bob_ek1"},
		},
	})

	// Create Alice and request Bob's bundle
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	ctx2 := testContext(t)
	resp := h.HandlePreKeyBundleRequest(ctx2, &protocol.PreKeyBundleRequest{
		Ty:    "137",
		Token: aliceToken,
		ID:    bobID,
	})

	bundle, ok := resp.(*protocol.PreKeyBundle)
	if !ok {
		t.Fatalf("expected PreKeyBundle, got %T: %+v", resp, resp)
	}
	if bundle.Ty != "0" {
		t.Errorf("expected ty=0, got %s", bundle.Ty)
	}
	if bundle.MidtermPreKey != "bob_midterm_key" {
		t.Errorf("unexpected midterm key: %s", bundle.MidtermPreKey)
	}
	if bundle.MidtermPreKeySig != "bob_midterm_sig" {
		t.Errorf("unexpected midterm sig: %s", bundle.MidtermPreKeySig)
	}
	if bundle.EphemeralPreKey == nil {
		t.Fatal("expected ephemeral pre-key, got nil")
	}
	if bundle.EphemeralPreKey.ID != ekID {
		t.Errorf("unexpected ek ID: %s", bundle.EphemeralPreKey.ID)
	}

	// Request again - ephemeral key should be consumed
	ctx3 := testContext(t)
	resp2 := h.HandlePreKeyBundleRequest(ctx3, &protocol.PreKeyBundleRequest{
		Ty:    "137",
		Token: aliceToken,
		ID:    bobID,
	})
	bundle2 := resp2.(*protocol.PreKeyBundle)
	if bundle2.EphemeralPreKey != nil {
		t.Error("ephemeral key should have been consumed after first request")
	}
}

func TestPreKeyBundleRequestNoSignedKey(t *testing.T) {
	h := setupTestHandler(t)

	// Create Bob without any pre-keys
	bobID, _, _ := createTestAccount(t, h, "bob")

	// Create Alice and request Bob's bundle
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	ctx := testContext(t)
	resp := h.HandlePreKeyBundleRequest(ctx, &protocol.PreKeyBundleRequest{
		Ty:    "137",
		Token: aliceToken,
		ID:    bobID,
	})

	assertError(t, resp, protocol.ErrNoAvailableKey)
}

func TestPreKeyBundleRequestUserNotFound(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")

	ctx := testContext(t)
	resp := h.HandlePreKeyBundleRequest(ctx, &protocol.PreKeyBundleRequest{
		Ty:    "137",
		Token: aliceToken,
		ID:    uuid.New().String(), // Non-existent user
	})

	assertError(t, resp, protocol.ErrUserNotFound)
}
