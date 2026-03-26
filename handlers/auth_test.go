package handlers

import (
	"testing"

	"github.com/BaptTF/sickgnal-server/protocol"
)

func TestCreateAccountHappyPath(t *testing.T) {
	h := setupTestHandler(t)

	userID, token, _ := createTestAccount(t, h, "alice")

	if userID == "" {
		t.Error("user ID should not be empty")
	}
	if token == "" {
		t.Error("token should not be empty")
	}
}

func TestCreateAccountDuplicateUsername(t *testing.T) {
	h := setupTestHandler(t)

	// Create first account
	createTestAccount(t, h, "alice")

	// Try to create another with the same username
	b64IK2, edPriv2 := generateTestIdentityKeys(t)
	sig2 := signUsername(t, edPriv2, "alice")

	ctx := testContext(t)
	resp := h.HandleCreateAccount(ctx, &protocol.CreateAccount{
		Ty:        "128",
		IK:        b64IK2,
		Username:  "alice",
		Signature: sig2,
	})

	assertError(t, resp, protocol.ErrUsernameUnavailable)
}

func TestCreateAccountInvalidSignature(t *testing.T) {
	h := setupTestHandler(t)

	b64IK, _ := generateTestIdentityKeys(t)
	// Generate a different key for signing (wrong key)
	_, wrongKey := generateTestIdentityKeys(t)
	sig := signUsername(t, wrongKey, "alice")

	ctx := testContext(t)
	resp := h.HandleCreateAccount(ctx, &protocol.CreateAccount{
		Ty:        "128",
		IK:        b64IK,
		Username:  "alice",
		Signature: sig,
	})

	assertError(t, resp, protocol.ErrInvalidAuthentication)
}

func TestCreateAccountSetsAuthState(t *testing.T) {
	h := setupTestHandler(t)

	b64IK, edPriv := generateTestIdentityKeys(t)
	sig := signUsername(t, edPriv, "alice")

	ctx, capturedUserID, capturedToken := testContextWithAuth(t)
	resp := h.HandleCreateAccount(ctx, &protocol.CreateAccount{
		Ty:        "128",
		IK:        b64IK,
		Username:  "alice",
		Signature: sig,
	})

	authToken := resp.(*protocol.AuthToken)

	if *capturedUserID != authToken.ID {
		t.Errorf("SetAuth userID=%s, response ID=%s", *capturedUserID, authToken.ID)
	}
	if *capturedToken != authToken.Token {
		t.Errorf("SetAuth token=%s, response token=%s", *capturedToken, authToken.Token)
	}
}

func TestAuthChallengeFlowHappyPath(t *testing.T) {
	h := setupTestHandler(t)

	// Create account first
	_, _, edPriv := createTestAccount(t, h, "alice")

	// Step 1: Request challenge
	ctx := testContext(t)
	resp := h.HandleAuthChallengeRequest(ctx, &protocol.AuthChallengeRequest{
		Ty:       "130",
		Username: "alice",
	})

	challMsg, ok := resp.(*protocol.AuthChallengeMsg)
	if !ok {
		t.Fatalf("expected AuthChallengeMsg, got %T: %+v", resp, resp)
	}
	if challMsg.Ty != "131" {
		t.Errorf("expected ty=131, got %s", challMsg.Ty)
	}
	if challMsg.Challenge == "" {
		t.Fatal("challenge nonce is empty")
	}

	// Step 2: Solve challenge
	solve := signChallenge(t, edPriv, challMsg.Challenge, "alice")

	ctx2, capturedUserID, capturedToken := testContextWithAuth(t)
	resp2 := h.HandleAuthChallengeSolve(ctx2, &protocol.AuthChallengeSolve{
		Ty:        "132",
		Challenge: challMsg.Challenge,
		Solve:     solve,
	})

	authToken, ok := resp2.(*protocol.AuthToken)
	if !ok {
		t.Fatalf("expected AuthToken, got %T: %+v", resp2, resp2)
	}
	if authToken.ID == "" {
		t.Error("auth token ID is empty")
	}
	if authToken.Token == "" {
		t.Error("auth token is empty")
	}

	// Check auth state was set
	if *capturedUserID == "" || *capturedToken == "" {
		t.Error("SetAuth was not called")
	}
}

func TestAuthChallengeRequestUnknownUser(t *testing.T) {
	h := setupTestHandler(t)

	ctx := testContext(t)
	resp := h.HandleAuthChallengeRequest(ctx, &protocol.AuthChallengeRequest{
		Ty:       "130",
		Username: "nonexistent",
	})

	assertError(t, resp, protocol.ErrUserNotFound)
}

func TestAuthChallengeSolveInvalidSignature(t *testing.T) {
	h := setupTestHandler(t)

	createTestAccount(t, h, "alice")

	// Request challenge
	ctx := testContext(t)
	resp := h.HandleAuthChallengeRequest(ctx, &protocol.AuthChallengeRequest{
		Ty:       "130",
		Username: "alice",
	})
	challMsg := resp.(*protocol.AuthChallengeMsg)

	// Solve with wrong key
	_, wrongKey := generateTestIdentityKeys(t)
	wrongSolve := signChallenge(t, wrongKey, challMsg.Challenge, "alice")

	ctx2 := testContext(t)
	resp2 := h.HandleAuthChallengeSolve(ctx2, &protocol.AuthChallengeSolve{
		Ty:        "132",
		Challenge: challMsg.Challenge,
		Solve:     wrongSolve,
	})

	assertError(t, resp2, protocol.ErrInvalidAuthentication)
}

func TestAuthChallengeSolveUnknownNonce(t *testing.T) {
	h := setupTestHandler(t)

	ctx := testContext(t)
	resp := h.HandleAuthChallengeSolve(ctx, &protocol.AuthChallengeSolve{
		Ty:        "132",
		Challenge: "dW5rbm93bg==", // "unknown" in base64
		Solve:     "c2lnbmF0dXJl",
	})

	assertError(t, resp, protocol.ErrInvalidAuthentication)
}

func TestTokenValidation(t *testing.T) {
	h := setupTestHandler(t)

	_, token, _ := createTestAccount(t, h, "alice")

	// Valid token
	userID, errResp := h.validateToken(token)
	if errResp != nil {
		t.Fatalf("valid token rejected: %+v", errResp)
	}
	if userID == "" {
		t.Error("userID should not be empty for valid token")
	}

	// Invalid token
	_, errResp = h.validateToken("invalid_token_12345")
	if errResp == nil {
		t.Error("invalid token should be rejected")
	}

	// Empty token
	_, errResp = h.validateToken("")
	if errResp == nil {
		t.Error("empty token should be rejected")
	}
}

func TestNewTokenAfterChallengeInvalidatesOld(t *testing.T) {
	h := setupTestHandler(t)

	_, oldToken, edPriv := createTestAccount(t, h, "alice")

	// Do challenge-response to get new token
	ctx := testContext(t)
	resp := h.HandleAuthChallengeRequest(ctx, &protocol.AuthChallengeRequest{
		Ty:       "130",
		Username: "alice",
	})
	challMsg := resp.(*protocol.AuthChallengeMsg)

	solve := signChallenge(t, edPriv, challMsg.Challenge, "alice")
	ctx2 := testContext(t)
	resp2 := h.HandleAuthChallengeSolve(ctx2, &protocol.AuthChallengeSolve{
		Ty:        "132",
		Challenge: challMsg.Challenge,
		Solve:     solve,
	})
	newAuthToken := resp2.(*protocol.AuthToken)

	// Old token should no longer work
	_, errResp := h.validateToken(oldToken)
	if errResp == nil {
		t.Error("old token should be invalidated after challenge-response")
	}

	// New token should work
	_, errResp = h.validateToken(newAuthToken.Token)
	if errResp != nil {
		t.Fatalf("new token should be valid: %+v", errResp)
	}
}
