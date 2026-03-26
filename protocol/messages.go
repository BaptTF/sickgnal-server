package protocol

import (
	"encoding/json"
	"fmt"
)

// Message type constants (string values matching Rust serde rename).
const (
	TyPreKeyBundle          = "0"
	TyConversationOpen      = "1"
	TyConversationMessage   = "2"
	TyKeyRotation           = "3"
	TyUserProfile           = "10"
	TyCreateAccount         = "128"
	TyAuthToken             = "129"
	TyAuthChallengeRequest  = "130"
	TyAuthChallenge         = "131"
	TyAuthChallengeSolve    = "132"
	TyPreKeyUpload          = "133"
	TyPreKeyDelete          = "134"
	TyPreKeyStatusRequest   = "135"
	TyPreKeyStatus          = "136"
	TyPreKeyBundleRequest   = "137"
	TyUserProfileByUsername = "140"
	TyUserProfileById       = "141"
	TySendInitialMessage    = "150"
	TySendMessage           = "151"
	TyGetInitialMessages    = "160"
	TyGetMessages           = "161"
	TyMessagesList          = "170"
	TyEnableInstantRelay    = "180"
	TyDisableInstantRelay   = "181"
	TyOk                    = "254"
	TyError                 = "255"
)

// --- Server -> Client messages ---

// PreKeyBundle (ty=0) - sent by server in response to a bundle request.
type PreKeyBundle struct {
	Ty               string        `json:"ty"`
	IdentityKey      string        `json:"ik"`
	MidtermPreKey    string        `json:"pk"`
	MidtermPreKeySig string        `json:"pksig"`
	EphemeralPreKey  *EphemeralKey `json:"ek"`
}

type EphemeralKey struct {
	ID  string `json:"id"`
	Key string `json:"ek"`
}

func NewPreKeyBundle(ik, pk, pksig string, ek *EphemeralKey) *PreKeyBundle {
	return &PreKeyBundle{
		Ty:               TyPreKeyBundle,
		IdentityKey:      ik,
		MidtermPreKey:    pk,
		MidtermPreKeySig: pksig,
		EphemeralPreKey:  ek,
	}
}

// ConversationOpen (ty=1) - initial key exchange message relayed between peers.
type ConversationOpen struct {
	Ty       string  `json:"ty"`
	SenderID string  `json:"sndr_id"`
	IK       string  `json:"ik"`
	EK       string  `json:"ek"`
	KID      *string `json:"kid"`
	I        string  `json:"i"`
	J        string  `json:"j"`
	Nonce    string  `json:"nonce"`
	Msg      string  `json:"msg"`
}

// ConversationMessage (ty=2) - encrypted message relayed between peers.
type ConversationMessage struct {
	Ty       string `json:"ty"`
	SenderID string `json:"sndr_id"`
	Nonce    string `json:"nonce"`
	KID      string `json:"kid"`
	Msg      string `json:"msg"`
}

// KeyRotation (ty=3) - key rotation message relayed between peers.
type KeyRotation struct {
	Ty       string            `json:"ty"`
	SenderID string            `json:"sndr_id"`
	Nonce    string            `json:"nonce"`
	KID      string            `json:"kid"`
	Msg      *KeyRotationInner `json:"msg"`
}

type KeyRotationInner struct {
	Nonce string `json:"nonce"`
	KID   string `json:"kid"`
	Msg   string `json:"msg"`
}

// UserProfile (ty=10) - user profile response.
type UserProfile struct {
	Ty       string `json:"ty"`
	ID       string `json:"id"`
	Username string `json:"username"`
}

func NewUserProfile(id, username string) *UserProfile {
	return &UserProfile{
		Ty:       TyUserProfile,
		ID:       id,
		Username: username,
	}
}

// AuthToken (ty=129) - auth token response.
type AuthToken struct {
	Ty    string `json:"ty"`
	ID    string `json:"id"`
	Token string `json:"token"`
}

func NewAuthToken(id, token string) *AuthToken {
	return &AuthToken{
		Ty:    TyAuthToken,
		ID:    id,
		Token: token,
	}
}

// AuthChallengeMsg (ty=131) - auth challenge nonce.
type AuthChallengeMsg struct {
	Ty        string `json:"ty"`
	Challenge string `json:"chall"`
}

func NewAuthChallenge(challenge string) *AuthChallengeMsg {
	return &AuthChallengeMsg{
		Ty:        TyAuthChallenge,
		Challenge: challenge,
	}
}

// PreKeyStatus (ty=136) - pre-key status response.
type PreKeyStatusMsg struct {
	Ty    string   `json:"ty"`
	Limit int      `json:"limit"`
	Keys  []string `json:"keys"`
}

func NewPreKeyStatus(limit int, keys []string) *PreKeyStatusMsg {
	return &PreKeyStatusMsg{
		Ty:    TyPreKeyStatus,
		Limit: limit,
		Keys:  keys,
	}
}

// MessagesList (ty=170) - list of messages response.
type MessagesList struct {
	Ty   string            `json:"ty"`
	Msgs []json.RawMessage `json:"msgs"`
}

func NewMessagesList(msgs []json.RawMessage) *MessagesList {
	if msgs == nil {
		msgs = []json.RawMessage{}
	}
	return &MessagesList{
		Ty:   TyMessagesList,
		Msgs: msgs,
	}
}

// OkMsg (ty=254) - success acknowledgment.
type OkMsg struct {
	Ty string `json:"ty"`
}

func NewOk() *OkMsg {
	return &OkMsg{Ty: TyOk}
}

// ErrorMsg (ty=255) - error response.
type ErrorMsg struct {
	Ty   string    `json:"ty"`
	Code ErrorCode `json:"code"`
}

func NewError(code ErrorCode) *ErrorMsg {
	return &ErrorMsg{
		Ty:   TyError,
		Code: code,
	}
}

// --- Client -> Server messages ---

// CreateAccount (ty=128).
type CreateAccount struct {
	Ty        string `json:"ty"`
	IK        string `json:"ik"`
	Username  string `json:"username"`
	Signature string `json:"sig"`
}

// AuthChallengeRequest (ty=130).
type AuthChallengeRequest struct {
	Ty       string `json:"ty"`
	Username string `json:"username"`
}

// AuthChallengeSolve (ty=132).
type AuthChallengeSolve struct {
	Ty        string `json:"ty"`
	Challenge string `json:"chall"`
	Solve     string `json:"solve"`
}

// PreKeyUpload (ty=133).
type PreKeyUpload struct {
	Ty      string               `json:"ty"`
	Token   string               `json:"token"`
	Replace bool                 `json:"replace"`
	PK      *SignedPreKeyUpload  `json:"pk"`
	TKs     []EphemeralKeyUpload `json:"tks"`
}

type SignedPreKeyUpload struct {
	Key       string `json:"key"`
	Signature string `json:"sig"`
}

type EphemeralKeyUpload struct {
	ID  string `json:"id"`
	Key string `json:"ek"`
}

// PreKeyDelete (ty=134).
type PreKeyDelete struct {
	Ty    string   `json:"ty"`
	Token string   `json:"token"`
	Keys  []string `json:"keys"`
}

// PreKeyStatusRequest (ty=135).
type PreKeyStatusRequest struct {
	Ty    string `json:"ty"`
	Token string `json:"token"`
}

// PreKeyBundleRequest (ty=137).
type PreKeyBundleRequest struct {
	Ty    string `json:"ty"`
	Token string `json:"token"`
	ID    string `json:"id"`
}

// UserProfileByUsername (ty=140).
type UserProfileByUsername struct {
	Ty       string `json:"ty"`
	Token    string `json:"token"`
	Username string `json:"username"`
}

// UserProfileById (ty=141).
type UserProfileById struct {
	Ty    string `json:"ty"`
	Token string `json:"token"`
	ID    string `json:"id"`
}

// SendInitialMessage (ty=150).
type SendInitialMessage struct {
	Ty          string  `json:"ty"`
	Token       string  `json:"token"`
	RecipientID string  `json:"rcpt_id"`
	IK          string  `json:"ik"`
	EK          string  `json:"ek"`
	KID         *string `json:"kid"`
	I           string  `json:"i"`
	J           string  `json:"j"`
	Nonce       string  `json:"nonce"`
	Msg         string  `json:"msg"`
}

// SendMessage (ty=151).
type SendMessage struct {
	Ty          string `json:"ty"`
	Token       string `json:"token"`
	RecipientID string `json:"rcpt_id"`
	Nonce       string `json:"nonce"`
	KID         string `json:"kid"`
	Msg         string `json:"msg"`
}

// GetInitialMessages (ty=160).
type GetInitialMessages struct {
	Ty    string `json:"ty"`
	Token string `json:"token"`
	Limit int    `json:"limit"`
}

// GetMessages (ty=161).
type GetMessages struct {
	Ty    string `json:"ty"`
	Token string `json:"token"`
	Limit int    `json:"limit"`
}

// EnableInstantRelay (ty=180).
type EnableInstantRelay struct {
	Ty    string `json:"ty"`
	Token string `json:"token"`
}

// DisableInstantRelay (ty=181).
type DisableInstantRelay struct {
	Ty string `json:"ty"`
}

// ParseMessageType extracts the "ty" field from a raw JSON message.
func ParseMessageType(raw json.RawMessage) (string, error) {
	var envelope struct {
		Ty string `json:"ty"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", fmt.Errorf("parse message type: %w", err)
	}
	if envelope.Ty == "" {
		return "", fmt.Errorf("missing or empty 'ty' field")
	}
	return envelope.Ty, nil
}

// ParseMessage parses a raw JSON message into the appropriate typed struct
// based on the "ty" field. Only parses server-bound message types (128+).
func ParseMessage(raw json.RawMessage) (interface{}, string, error) {
	ty, err := ParseMessageType(raw)
	if err != nil {
		return nil, "", err
	}

	var msg interface{}
	switch ty {
	case TyCreateAccount:
		msg = &CreateAccount{}
	case TyAuthChallengeRequest:
		msg = &AuthChallengeRequest{}
	case TyAuthChallengeSolve:
		msg = &AuthChallengeSolve{}
	case TyPreKeyUpload:
		msg = &PreKeyUpload{}
	case TyPreKeyDelete:
		msg = &PreKeyDelete{}
	case TyPreKeyStatusRequest:
		msg = &PreKeyStatusRequest{}
	case TyPreKeyBundleRequest:
		msg = &PreKeyBundleRequest{}
	case TyUserProfileByUsername:
		msg = &UserProfileByUsername{}
	case TyUserProfileById:
		msg = &UserProfileById{}
	case TySendInitialMessage:
		msg = &SendInitialMessage{}
	case TySendMessage:
		msg = &SendMessage{}
	case TyGetInitialMessages:
		msg = &GetInitialMessages{}
	case TyGetMessages:
		msg = &GetMessages{}
	case TyEnableInstantRelay:
		msg = &EnableInstantRelay{}
	case TyDisableInstantRelay:
		msg = &DisableInstantRelay{}
	default:
		return nil, ty, fmt.Errorf("unknown or client-only message type: %s", ty)
	}

	if err := json.Unmarshal(raw, msg); err != nil {
		return nil, ty, fmt.Errorf("unmarshal message type %s: %w", ty, err)
	}

	return msg, ty, nil
}
