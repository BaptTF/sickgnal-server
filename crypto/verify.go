package crypto

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
)

// ParsePublicIdentityKeys parses a base64-encoded 64-byte identity key
// into its x25519 (first 32 bytes) and ed25519 (last 32 bytes) components.
func ParsePublicIdentityKeys(b64Key string) (x25519Key, ed25519Key []byte, err error) {
	raw, err := base64.StdEncoding.DecodeString(b64Key)
	if err != nil {
		return nil, nil, fmt.Errorf("decode identity key: %w", err)
	}
	if len(raw) != 64 {
		return nil, nil, fmt.Errorf("identity key must be 64 bytes, got %d", len(raw))
	}
	return raw[:32], raw[32:], nil
}

// VerifySignature verifies an Ed25519 signature.
// publicKey must be 32 bytes, signature must be 64 bytes.
func VerifySignature(publicKey ed25519.PublicKey, message, signature []byte) bool {
	if len(publicKey) != ed25519.PublicKeySize {
		return false
	}
	if len(signature) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(publicKey, message, signature)
}

// VerifyAccountCreation verifies the signature on a CreateAccount message.
// The signature is over the raw username bytes, signed by the Ed25519 identity key.
func VerifyAccountCreation(b64IdentityKey, username, b64Signature string) error {
	_, ed25519Key, err := ParsePublicIdentityKeys(b64IdentityKey)
	if err != nil {
		return fmt.Errorf("parse identity key: %w", err)
	}

	sig, err := base64.StdEncoding.DecodeString(b64Signature)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	if !VerifySignature(ed25519Key, []byte(username), sig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// VerifyAuthChallenge verifies the auth challenge response.
// The client signs SHA512(nonce) || username_bytes.
func VerifyAuthChallenge(b64IdentityKey, username, b64Nonce, b64Solve string) error {
	_, ed25519Key, err := ParsePublicIdentityKeys(b64IdentityKey)
	if err != nil {
		return fmt.Errorf("parse identity key: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(b64Nonce)
	if err != nil {
		return fmt.Errorf("decode nonce: %w", err)
	}

	solve, err := base64.StdEncoding.DecodeString(b64Solve)
	if err != nil {
		return fmt.Errorf("decode solve: %w", err)
	}

	// Compute SHA-512 of the nonce
	nonceHash := sha512.Sum512(nonce)

	// The signed message is SHA512(nonce) || username
	signedMsg := append(nonceHash[:], []byte(username)...)

	if !VerifySignature(ed25519Key, signedMsg, solve) {
		return fmt.Errorf("challenge signature verification failed")
	}

	return nil
}
