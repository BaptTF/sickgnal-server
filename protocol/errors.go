package protocol

// ErrorCode represents the error codes sent in Error messages.
// These are serialized as JSON strings matching the Rust enum variant names.
type ErrorCode string

const (
	ErrInvalidMessage         ErrorCode = "InvalidMessage"
	ErrMessageTypeNotAccepted ErrorCode = "MessageTypeNotAccepted"
	ErrInvalidAuthentication  ErrorCode = "InvalidAuthentication"
	ErrUsernameUnavailable    ErrorCode = "UsernameUnavailable"
	ErrUserNotFound           ErrorCode = "UserNotFound"
	ErrPreKeyLimit            ErrorCode = "PreKeyLimit"
	ErrNoAvailableKey         ErrorCode = "NoAvailableKey"
	ErrInternalError          ErrorCode = "InternalError"
)
