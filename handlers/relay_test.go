package handlers

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/BaptTF/sickgnal-server/protocol"
)

func TestRelayManagerRegisterPushUnregister(t *testing.T) {
	rm := NewRelayManager()

	var buf bytes.Buffer
	writer := protocol.NewConnWriter(&buf)

	rm.Register("user1", writer)

	if !rm.IsRegistered("user1") {
		t.Error("user1 should be registered")
	}
	if rm.IsRegistered("user2") {
		t.Error("user2 should not be registered")
	}

	// Push a message
	payload := json.RawMessage(`{"ty":"2","sndr_id":"sender","nonce":"n","kid":"k","msg":"hello"}`)
	ok := rm.Push("user1", payload)
	if !ok {
		t.Error("push to registered user should succeed")
	}

	// Verify the message was written to the buffer
	pkt, err := protocol.ReadPacket(&buf)
	if err != nil {
		t.Fatalf("read pushed packet: %v", err)
	}
	if pkt.RequestID != 0 {
		t.Errorf("pushed packet should have reqid=0, got %d", pkt.RequestID)
	}

	// Push to non-registered user
	ok = rm.Push("user2", payload)
	if ok {
		t.Error("push to unregistered user should return false")
	}

	// Unregister
	rm.Unregister("user1")
	if rm.IsRegistered("user1") {
		t.Error("user1 should be unregistered")
	}

	// Push after unregister
	ok = rm.Push("user1", payload)
	if ok {
		t.Error("push after unregister should return false")
	}
}

func TestEnableInstantRelayFlushesMessages(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	bobID, bobToken, _ := createTestAccount(t, h, "bob")

	// Alice sends messages to Bob (Bob is offline)
	sendInitialMsg(t, h, aliceToken, bobID)
	sendMsg(t, h, aliceToken, bobID, "regular_msg")

	// Bob enables instant relay - should flush pending messages
	var bobBuf bytes.Buffer
	bobWriter := protocol.NewConnWriter(&bobBuf)
	bobCtx := &Context{
		RequestID:  5,
		Writer:     bobWriter,
		ConnWriter: bobWriter,
		SetAuth:    func(uid, tok string) {},
	}

	resp := h.HandleEnableInstantRelay(bobCtx, &protocol.EnableInstantRelay{
		Ty:    "180",
		Token: bobToken,
	})
	assertOk(t, resp)

	// Read flushed messages from Bob's buffer
	pkt1, err := protocol.ReadPacket(&bobBuf)
	if err != nil {
		t.Fatalf("read first flushed message: %v", err)
	}
	if pkt1.RequestID != 0 {
		t.Errorf("flushed message should have reqid=0, got %d", pkt1.RequestID)
	}

	var msg1 map[string]interface{}
	json.Unmarshal(pkt1.Message, &msg1)
	if msg1["ty"] != "1" {
		t.Errorf("first flushed message should be type 1, got %v", msg1["ty"])
	}

	pkt2, err := protocol.ReadPacket(&bobBuf)
	if err != nil {
		t.Fatalf("read second flushed message: %v", err)
	}
	if pkt2.RequestID != 0 {
		t.Errorf("flushed message should have reqid=0, got %d", pkt2.RequestID)
	}

	var msg2 map[string]interface{}
	json.Unmarshal(pkt2.Message, &msg2)
	if msg2["ty"] != "2" {
		t.Errorf("second flushed message should be type 2, got %v", msg2["ty"])
	}

	// Verify messages are removed from storage
	ctx3 := testContext(t)
	resp2 := h.HandleGetInitialMessages(ctx3, &protocol.GetInitialMessages{Ty: "160", Token: bobToken, Limit: 100})
	if len(resp2.(*protocol.MessagesList).Msgs) != 0 {
		t.Error("expected 0 initial messages after flush")
	}

	ctx4 := testContext(t)
	resp3 := h.HandleGetMessages(ctx4, &protocol.GetMessages{Ty: "161", Token: bobToken, Limit: 100})
	if len(resp3.(*protocol.MessagesList).Msgs) != 0 {
		t.Error("expected 0 regular messages after flush")
	}
}

func TestInstantRelayPushesNewMessages(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	bobID, bobToken, _ := createTestAccount(t, h, "bob")

	// Bob enables instant relay
	var bobBuf bytes.Buffer
	bobWriter := protocol.NewConnWriter(&bobBuf)
	bobCtx := &Context{
		RequestID:  1,
		Writer:     bobWriter,
		ConnWriter: bobWriter,
		SetAuth:    func(uid, tok string) {},
	}
	h.HandleEnableInstantRelay(bobCtx, &protocol.EnableInstantRelay{
		Ty:    "180",
		Token: bobToken,
	})

	// Alice sends a message while Bob has relay enabled
	msg := &protocol.SendMessage{
		Ty:          "151",
		Token:       aliceToken,
		RecipientID: bobID,
		Nonce:       "n",
		KID:         "k",
		Msg:         "realtime_msg",
	}
	ctx := testContextWithRaw(t, msg)
	resp := h.HandleSendMessage(ctx, msg)
	assertOk(t, resp)

	// Bob should have received it immediately via relay
	pkt, err := protocol.ReadPacket(&bobBuf)
	if err != nil {
		t.Fatalf("read relayed message: %v", err)
	}
	if pkt.RequestID != 0 {
		t.Errorf("relayed message should have reqid=0, got %d", pkt.RequestID)
	}

	var parsed map[string]interface{}
	json.Unmarshal(pkt.Message, &parsed)
	if parsed["msg"] != "realtime_msg" {
		t.Errorf("expected msg=realtime_msg, got %v", parsed["msg"])
	}

	// Message should NOT be in storage
	ctx2 := testContext(t)
	resp2 := h.HandleGetMessages(ctx2, &protocol.GetMessages{Ty: "161", Token: bobToken, Limit: 100})
	if len(resp2.(*protocol.MessagesList).Msgs) != 0 {
		t.Error("expected 0 stored messages (was relayed)")
	}
}

func TestDisableInstantRelay(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	bobID, bobToken, _ := createTestAccount(t, h, "bob")

	// Bob enables relay
	var bobBuf bytes.Buffer
	bobWriter := protocol.NewConnWriter(&bobBuf)
	bobCtx := &Context{
		RequestID:  1,
		Writer:     bobWriter,
		ConnWriter: bobWriter,
		UserID:     bobID,
		SetAuth:    func(uid, tok string) {},
	}
	h.HandleEnableInstantRelay(bobCtx, &protocol.EnableInstantRelay{
		Ty:    "180",
		Token: bobToken,
	})

	// Disable relay
	resp := h.HandleDisableInstantRelay(bobCtx)
	assertOk(t, resp)

	// Send message - should be stored, not relayed
	sendMsg(t, h, aliceToken, bobID, "stored_msg")

	// Should be in storage
	ctx2 := testContext(t)
	resp2 := h.HandleGetMessages(ctx2, &protocol.GetMessages{Ty: "161", Token: bobToken, Limit: 100})
	if len(resp2.(*protocol.MessagesList).Msgs) != 1 {
		t.Errorf("expected 1 stored message after relay disabled, got %d", len(resp2.(*protocol.MessagesList).Msgs))
	}
}
