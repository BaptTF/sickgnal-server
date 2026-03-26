package handlers

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/BaptTF/sickgnal-server/protocol"
	"github.com/BaptTF/sickgnal-server/store"
	"golang.org/x/crypto/curve25519"
)

// setupTestHandler creates a handler with an in-memory SQLite database.
func setupTestHandler(t *testing.T) *Handler {
	t.Helper()
	db, err := store.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	return New(db)
}

// testContext creates a handler context for testing.
func testContext(t *testing.T) *Context {
	t.Helper()
	var buf bytes.Buffer
	return &Context{
		RequestID:  1,
		Writer:     protocol.NewConnWriter(&buf),
		ConnWriter: protocol.NewConnWriter(&buf),
		SetAuth:    func(userID, token string) {},
	}
}

// testContextWithRaw creates a handler context with raw JSON attached.
func testContextWithRaw(t *testing.T, msg interface{}) *Context {
	t.Helper()
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal raw message: %v", err)
	}
	var buf bytes.Buffer
	return &Context{
		RequestID:  1,
		Writer:     protocol.NewConnWriter(&buf),
		ConnWriter: protocol.NewConnWriter(&buf),
		SetAuth:    func(userID, token string) {},
		RawMessage: raw,
	}
}

// testContextWithAuth creates a handler context that captures auth state changes.
func testContextWithAuth(t *testing.T) (*Context, *string, *string) {
	t.Helper()
	var buf bytes.Buffer
	var userID, token string
	ctx := &Context{
		RequestID:  1,
		Writer:     protocol.NewConnWriter(&buf),
		ConnWriter: protocol.NewConnWriter(&buf),
		SetAuth: func(uid, tok string) {
			userID = uid
			token = tok
		},
	}
	return ctx, &userID, &token
}

// generateTestIdentityKeys generates a test identity key pair.
// Returns base64 public key (64 bytes: x25519 || ed25519), ed25519 private key.
func generateTestIdentityKeys(t *testing.T) (string, ed25519.PrivateKey) {
	t.Helper()

	// Generate Ed25519 keypair
	edPub, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519: %v", err)
	}

	// Generate X25519 keypair (32-byte random scalar)
	var x25519Priv [32]byte
	if _, err := rand.Read(x25519Priv[:]); err != nil {
		t.Fatalf("generate x25519: %v", err)
	}
	x25519Pub, err := curve25519.X25519(x25519Priv[:], curve25519.Basepoint)
	if err != nil {
		t.Fatalf("x25519 scalar mult: %v", err)
	}

	// Combine: first 32 bytes x25519, last 32 bytes ed25519
	combined := make([]byte, 64)
	copy(combined[:32], x25519Pub)
	copy(combined[32:], edPub)

	return base64.StdEncoding.EncodeToString(combined), edPriv
}

// signUsername signs a username with the given ed25519 private key, returns base64 signature.
func signUsername(t *testing.T, privKey ed25519.PrivateKey, username string) string {
	t.Helper()
	sig := ed25519.Sign(privKey, []byte(username))
	return base64.StdEncoding.EncodeToString(sig)
}

// signChallenge signs a challenge nonce for auth: sig(SHA512(nonce) || username).
func signChallenge(t *testing.T, privKey ed25519.PrivateKey, b64Nonce, username string) string {
	t.Helper()
	nonce, err := base64.StdEncoding.DecodeString(b64Nonce)
	if err != nil {
		t.Fatalf("decode nonce: %v", err)
	}
	hash := sha512.Sum512(nonce)
	msg := append(hash[:], []byte(username)...)
	sig := ed25519.Sign(privKey, msg)
	return base64.StdEncoding.EncodeToString(sig)
}

// createTestAccount is a helper that creates a test account and returns its token and user ID.
func createTestAccount(t *testing.T, h *Handler, username string) (userID, token string, edPriv ed25519.PrivateKey) {
	t.Helper()
	b64IK, edPrivKey := generateTestIdentityKeys(t)
	sig := signUsername(t, edPrivKey, username)

	ctx, capturedUserID, capturedToken := testContextWithAuth(t)
	resp := h.HandleCreateAccount(ctx, &protocol.CreateAccount{
		Ty:        "128",
		IK:        b64IK,
		Username:  username,
		Signature: sig,
	})

	authToken, ok := resp.(*protocol.AuthToken)
	if !ok {
		t.Fatalf("expected AuthToken, got %T: %+v", resp, resp)
	}
	if authToken.Ty != "129" {
		t.Fatalf("expected ty=129, got %s", authToken.Ty)
	}
	if authToken.ID == "" {
		t.Fatal("empty user ID")
	}
	if authToken.Token == "" {
		t.Fatal("empty token")
	}

	return *capturedUserID, *capturedToken, edPrivKey
}

// assertOk checks that a response is a protocol.OkMsg.
func assertOk(t *testing.T, resp interface{}) {
	t.Helper()
	okMsg, ok := resp.(*protocol.OkMsg)
	if !ok {
		data, _ := json.Marshal(resp)
		t.Fatalf("expected OkMsg, got %T: %s", resp, string(data))
	}
	if okMsg.Ty != "254" {
		t.Errorf("expected ty=254, got %s", okMsg.Ty)
	}
}

// assertError checks that a response is a protocol.ErrorMsg with the expected code.
func assertError(t *testing.T, resp interface{}, expectedCode protocol.ErrorCode) {
	t.Helper()
	errMsg, ok := resp.(*protocol.ErrorMsg)
	if !ok {
		data, _ := json.Marshal(resp)
		t.Fatalf("expected ErrorMsg, got %T: %s", resp, string(data))
	}
	if errMsg.Code != expectedCode {
		t.Errorf("expected error code %s, got %s", expectedCode, errMsg.Code)
	}
}
