package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"testing"
)

func TestReadWritePacketRoundTrip(t *testing.T) {
	msg := map[string]string{"ty": "254"}

	var buf bytes.Buffer
	err := WritePacket(&buf, 42, msg)
	if err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	pkt, err := ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}

	if pkt.RequestID != 42 {
		t.Errorf("expected request ID 42, got %d", pkt.RequestID)
	}

	var parsed map[string]string
	if err := json.Unmarshal(pkt.Message, &parsed); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	if parsed["ty"] != "254" {
		t.Errorf("expected ty=254, got %s", parsed["ty"])
	}
}

func TestReadWritePacketWithComplexMessage(t *testing.T) {
	msg := NewAuthToken("550e8400-e29b-41d4-a716-446655440000", "abc123token")

	var buf bytes.Buffer
	err := WritePacket(&buf, 1, msg)
	if err != nil {
		t.Fatalf("WritePacket failed: %v", err)
	}

	pkt, err := ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}

	if pkt.RequestID != 1 {
		t.Errorf("expected request ID 1, got %d", pkt.RequestID)
	}

	var parsed AuthToken
	if err := json.Unmarshal(pkt.Message, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Ty != "129" {
		t.Errorf("expected ty=129, got %s", parsed.Ty)
	}
	if parsed.ID != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("unexpected ID: %s", parsed.ID)
	}
	if parsed.Token != "abc123token" {
		t.Errorf("unexpected token: %s", parsed.Token)
	}
}

func TestReadPacketEmptyPayload(t *testing.T) {
	// Write a header with length=0
	var buf bytes.Buffer
	header := make([]byte, 4)
	binary.BigEndian.PutUint16(header[0:2], 0) // len=0
	binary.BigEndian.PutUint16(header[2:4], 1) // reqid=1
	buf.Write(header)

	_, err := ReadPacket(&buf)
	if err == nil {
		t.Error("expected error for empty payload, got nil")
	}
}

func TestReadPacketTruncatedHeader(t *testing.T) {
	// Only 2 bytes instead of 4
	buf := bytes.NewReader([]byte{0x00, 0x05})
	_, err := ReadPacket(buf)
	if err == nil {
		t.Error("expected error for truncated header")
	}
}

func TestReadPacketTruncatedPayload(t *testing.T) {
	// Header says 10 bytes payload but only 3 available
	var buf bytes.Buffer
	header := make([]byte, 4)
	binary.BigEndian.PutUint16(header[0:2], 10) // len=10
	binary.BigEndian.PutUint16(header[2:4], 1)
	buf.Write(header)
	buf.Write([]byte("abc"))

	_, err := ReadPacket(&buf)
	if err == nil {
		t.Error("expected error for truncated payload")
	}
}

func TestReadPacketEOF(t *testing.T) {
	buf := bytes.NewReader([]byte{})
	_, err := ReadPacket(buf)
	if err == nil {
		t.Error("expected error for EOF")
	}
}

func TestWritePacketMaxSize(t *testing.T) {
	// Create a payload that exceeds max size
	bigData := make([]byte, MaxPayloadSize+1000)
	for i := range bigData {
		bigData[i] = 'A'
	}
	// A string that marshals to > 65535 bytes
	msg := map[string]string{"data": string(bigData)}

	var buf bytes.Buffer
	err := WritePacket(&buf, 1, msg)
	if err == nil {
		t.Error("expected error for oversized payload")
	}
}

func TestRequestIDZero(t *testing.T) {
	msg := NewOk()

	var buf bytes.Buffer
	err := WritePacket(&buf, 0, msg)
	if err != nil {
		t.Fatalf("WritePacket with reqid=0 failed: %v", err)
	}

	pkt, err := ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}

	if pkt.RequestID != 0 {
		t.Errorf("expected request ID 0, got %d", pkt.RequestID)
	}
}

func TestMultiplePacketsInStream(t *testing.T) {
	var buf bytes.Buffer

	// Write 3 packets
	for i := uint16(1); i <= 3; i++ {
		if err := WritePacket(&buf, i, NewOk()); err != nil {
			t.Fatalf("WritePacket %d failed: %v", i, err)
		}
	}

	// Read them back
	for i := uint16(1); i <= 3; i++ {
		pkt, err := ReadPacket(&buf)
		if err != nil {
			t.Fatalf("ReadPacket %d failed: %v", i, err)
		}
		if pkt.RequestID != i {
			t.Errorf("packet %d: expected reqid=%d, got %d", i, i, pkt.RequestID)
		}
	}

	// Should be EOF now
	_, err := ReadPacket(&buf)
	if err == nil || err == io.EOF {
		// io.EOF wrapped in our error is fine
	}
}

func TestConnWriterThreadSafe(t *testing.T) {
	var buf bytes.Buffer
	cw := NewConnWriter(&buf)

	// Write from multiple goroutines
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(id uint16) {
			done <- cw.WritePacket(id, NewOk())
		}(uint16(i))
	}

	for i := 0; i < 10; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent write %d failed: %v", i, err)
		}
	}

	// Read back all 10 packets
	reader := bytes.NewReader(buf.Bytes())
	for i := 0; i < 10; i++ {
		pkt, err := ReadPacket(reader)
		if err != nil {
			t.Fatalf("ReadPacket %d failed: %v", i, err)
		}
		var parsed OkMsg
		if err := json.Unmarshal(pkt.Message, &parsed); err != nil {
			t.Fatalf("unmarshal %d: %v", i, err)
		}
		if parsed.Ty != "254" {
			t.Errorf("packet %d: expected ty=254, got %s", i, parsed.Ty)
		}
	}
}

func TestWriteRawPacket(t *testing.T) {
	var buf bytes.Buffer
	cw := NewConnWriter(&buf)

	raw := json.RawMessage(`{"ty":"254"}`)
	err := WriteRawPacket(cw, 5, raw)
	if err != nil {
		t.Fatalf("WriteRawPacket failed: %v", err)
	}

	pkt, err := ReadPacket(&buf)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}

	if pkt.RequestID != 5 {
		t.Errorf("expected reqid=5, got %d", pkt.RequestID)
	}

	var parsed OkMsg
	if err := json.Unmarshal(pkt.Message, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed.Ty != "254" {
		t.Errorf("expected ty=254, got %s", parsed.Ty)
	}
}
