package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

const (
	// HeaderSize is the total size of the frame header (2 bytes length + 2 bytes request ID).
	HeaderSize = 4
	// MaxPayloadSize is the maximum size of a JSON payload (limited by uint16).
	MaxPayloadSize = 65535
)

// Packet represents a framed message on the wire.
type Packet struct {
	RequestID uint16
	Message   json.RawMessage
}

// ReadPacket reads a single framed packet from the reader.
// Wire format: [2-byte BE uint16 length][2-byte BE uint16 request_id][JSON payload]
func ReadPacket(r io.Reader) (*Packet, error) {
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	payloadLen := binary.BigEndian.Uint16(header[0:2])
	requestID := binary.BigEndian.Uint16(header[2:4])

	if payloadLen == 0 {
		return nil, fmt.Errorf("empty payload")
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	return &Packet{
		RequestID: requestID,
		Message:   json.RawMessage(payload),
	}, nil
}

// WritePacket writes a framed packet to the writer.
// The message is serialized to JSON, then framed with length + request ID.
func WritePacket(w io.Writer, requestID uint16, message interface{}) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	if len(payload) > MaxPayloadSize {
		return fmt.Errorf("payload too large: %d > %d", len(payload), MaxPayloadSize)
	}

	header := make([]byte, HeaderSize)
	binary.BigEndian.PutUint16(header[0:2], uint16(len(payload)))
	binary.BigEndian.PutUint16(header[2:4], requestID)

	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}

// ConnWriter is a thread-safe writer for sending packets on a connection.
type ConnWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// NewConnWriter creates a new thread-safe connection writer.
func NewConnWriter(w io.Writer) *ConnWriter {
	return &ConnWriter{w: w}
}

// WritePacket sends a framed packet in a thread-safe manner.
func (cw *ConnWriter) WritePacket(requestID uint16, message interface{}) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	return WritePacket(cw.w, requestID, message)
}

// WriteRawPacket writes a pre-serialized JSON payload with framing.
// This is used for relaying stored messages that are already JSON.
func WriteRawPacket(cw *ConnWriter, requestID uint16, rawJSON json.RawMessage) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()

	payload := []byte(rawJSON)
	if len(payload) > MaxPayloadSize {
		return fmt.Errorf("payload too large: %d > %d", len(payload), MaxPayloadSize)
	}

	header := make([]byte, HeaderSize)
	binary.BigEndian.PutUint16(header[0:2], uint16(len(payload)))
	binary.BigEndian.PutUint16(header[2:4], requestID)

	if _, err := cw.w.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := cw.w.Write(payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}
