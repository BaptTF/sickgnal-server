package handlers

import (
	"encoding/json"
	"testing"

	"github.com/BaptTF/sickgnal-server/protocol"
)

func sendInitialMsg(t *testing.T, h *Handler, token, recipientID string) {
	t.Helper()
	msg := &protocol.SendInitialMessage{
		Ty:          "150",
		Token:       token,
		RecipientID: recipientID,
		IK:          "alice_ik",
		EK:          "alice_ek",
		KID:         nil,
		I:           "key_id_i",
		J:           "key_id_j",
		Nonce:       "nonce_b64",
		Msg:         "ciphertext_b64",
	}
	ctx := testContextWithRaw(t, msg)
	resp := h.HandleSendInitialMessage(ctx, msg)
	assertOk(t, resp)
}

func sendMsg(t *testing.T, h *Handler, token, recipientID, content string) {
	t.Helper()
	msg := &protocol.SendMessage{
		Ty:          "151",
		Token:       token,
		RecipientID: recipientID,
		Nonce:       "nonce",
		KID:         "kid",
		Msg:         content,
	}
	ctx := testContextWithRaw(t, msg)
	resp := h.HandleSendMessage(ctx, msg)
	assertOk(t, resp)
}

func TestSendInitialMessageAndRetrieve(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	bobID, bobToken, _ := createTestAccount(t, h, "bob")

	sendInitialMsg(t, h, aliceToken, bobID)

	// Bob retrieves initial messages
	ctx2 := testContext(t)
	resp2 := h.HandleGetInitialMessages(ctx2, &protocol.GetInitialMessages{
		Ty:    "160",
		Token: bobToken,
		Limit: 100,
	})

	msgList, ok := resp2.(*protocol.MessagesList)
	if !ok {
		t.Fatalf("expected MessagesList, got %T: %+v", resp2, resp2)
	}
	if msgList.Ty != "170" {
		t.Errorf("expected ty=170, got %s", msgList.Ty)
	}
	if len(msgList.Msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgList.Msgs))
	}

	// Parse the stored message - should have ty=1 and sndr_id
	var parsed map[string]interface{}
	if err := json.Unmarshal(msgList.Msgs[0], &parsed); err != nil {
		t.Fatalf("unmarshal stored msg: %v", err)
	}
	if parsed["ty"] != "1" {
		t.Errorf("expected ty=1, got %v", parsed["ty"])
	}
	if parsed["sndr_id"] == nil || parsed["sndr_id"] == "" {
		t.Error("expected sndr_id to be set")
	}
	// Original fields should be preserved
	if parsed["ik"] != "alice_ik" {
		t.Errorf("expected ik=alice_ik, got %v", parsed["ik"])
	}
	if parsed["msg"] != "ciphertext_b64" {
		t.Errorf("expected msg=ciphertext_b64, got %v", parsed["msg"])
	}
	// token and rcpt_id should be removed
	if _, exists := parsed["token"]; exists {
		t.Error("token should be removed from relayed message")
	}
	if _, exists := parsed["rcpt_id"]; exists {
		t.Error("rcpt_id should be removed from relayed message")
	}
}

func TestSendMessageAndRetrieve(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	bobID, bobToken, _ := createTestAccount(t, h, "bob")

	sendMsg(t, h, aliceToken, bobID, "encrypted_msg")

	// Bob retrieves messages
	ctx2 := testContext(t)
	resp2 := h.HandleGetMessages(ctx2, &protocol.GetMessages{
		Ty:    "161",
		Token: bobToken,
		Limit: 100,
	})

	msgList := resp2.(*protocol.MessagesList)
	if len(msgList.Msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgList.Msgs))
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(msgList.Msgs[0], &parsed); err != nil {
		t.Fatalf("unmarshal stored msg: %v", err)
	}
	if parsed["ty"] != "2" {
		t.Errorf("expected ty=2, got %v", parsed["ty"])
	}
	if parsed["msg"] != "encrypted_msg" {
		t.Errorf("expected msg=encrypted_msg, got %v", parsed["msg"])
	}
}

func TestMessagesDeletedAfterRetrieval(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	bobID, bobToken, _ := createTestAccount(t, h, "bob")

	sendMsg(t, h, aliceToken, bobID, "msg1")
	sendMsg(t, h, aliceToken, bobID, "msg2")

	// Bob retrieves
	ctx2 := testContext(t)
	resp := h.HandleGetMessages(ctx2, &protocol.GetMessages{
		Ty:    "161",
		Token: bobToken,
		Limit: 100,
	})
	msgList := resp.(*protocol.MessagesList)
	if len(msgList.Msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgList.Msgs))
	}

	// Retrieve again - should be empty
	ctx3 := testContext(t)
	resp2 := h.HandleGetMessages(ctx3, &protocol.GetMessages{
		Ty:    "161",
		Token: bobToken,
		Limit: 100,
	})
	msgList2 := resp2.(*protocol.MessagesList)
	if len(msgList2.Msgs) != 0 {
		t.Errorf("expected 0 messages after second retrieval, got %d", len(msgList2.Msgs))
	}
}

func TestGetMessagesEmptyInbox(t *testing.T) {
	h := setupTestHandler(t)
	_, token, _ := createTestAccount(t, h, "alice")

	ctx := testContext(t)
	resp := h.HandleGetMessages(ctx, &protocol.GetMessages{
		Ty:    "161",
		Token: token,
		Limit: 100,
	})

	msgList := resp.(*protocol.MessagesList)
	if len(msgList.Msgs) != 0 {
		t.Errorf("expected 0 messages for empty inbox, got %d", len(msgList.Msgs))
	}
}

func TestGetInitialMessagesEmptyInbox(t *testing.T) {
	h := setupTestHandler(t)
	_, token, _ := createTestAccount(t, h, "alice")

	ctx := testContext(t)
	resp := h.HandleGetInitialMessages(ctx, &protocol.GetInitialMessages{
		Ty:    "160",
		Token: token,
		Limit: 100,
	})

	msgList := resp.(*protocol.MessagesList)
	if len(msgList.Msgs) != 0 {
		t.Errorf("expected 0 messages for empty inbox, got %d", len(msgList.Msgs))
	}
}

func TestSendMessageToNonexistentUser(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")

	msg := &protocol.SendMessage{
		Ty:          "151",
		Token:       aliceToken,
		RecipientID: "nonexistent-uuid",
		Nonce:       "n",
		KID:         "k",
		Msg:         "m",
	}
	ctx := testContextWithRaw(t, msg)
	resp := h.HandleSendMessage(ctx, msg)
	assertError(t, resp, protocol.ErrUserNotFound)
}

func TestSendInitialMessageToNonexistentUser(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")

	msg := &protocol.SendInitialMessage{
		Ty:          "150",
		Token:       aliceToken,
		RecipientID: "nonexistent-uuid",
		IK:          "ik",
		EK:          "ek",
		I:           "i",
		J:           "j",
		Nonce:       "n",
		Msg:         "m",
	}
	ctx := testContextWithRaw(t, msg)
	resp := h.HandleSendInitialMessage(ctx, msg)
	assertError(t, resp, protocol.ErrUserNotFound)
}

func TestSendMessageInvalidToken(t *testing.T) {
	h := setupTestHandler(t)

	msg := &protocol.SendMessage{
		Ty:          "151",
		Token:       "bad_token",
		RecipientID: "uuid",
		Nonce:       "n",
		KID:         "k",
		Msg:         "m",
	}
	ctx := testContextWithRaw(t, msg)
	resp := h.HandleSendMessage(ctx, msg)
	assertError(t, resp, protocol.ErrInvalidAuthentication)
}

func TestMessageOrdering(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	bobID, bobToken, _ := createTestAccount(t, h, "bob")

	for i := 0; i < 5; i++ {
		sendMsg(t, h, aliceToken, bobID, string(rune('A'+i)))
	}

	ctx := testContext(t)
	resp := h.HandleGetMessages(ctx, &protocol.GetMessages{
		Ty:    "161",
		Token: bobToken,
		Limit: 100,
	})
	msgList := resp.(*protocol.MessagesList)
	if len(msgList.Msgs) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(msgList.Msgs))
	}

	for i, raw := range msgList.Msgs {
		var parsed map[string]interface{}
		json.Unmarshal(raw, &parsed)
		expected := string(rune('A' + i))
		if parsed["msg"] != expected {
			t.Errorf("message %d: expected msg=%s, got %v", i, expected, parsed["msg"])
		}
	}
}

func TestGetMessagesWithLimit(t *testing.T) {
	h := setupTestHandler(t)
	_, aliceToken, _ := createTestAccount(t, h, "alice")
	bobID, bobToken, _ := createTestAccount(t, h, "bob")

	for i := 0; i < 5; i++ {
		sendMsg(t, h, aliceToken, bobID, "m")
	}

	ctx := testContext(t)
	resp := h.HandleGetMessages(ctx, &protocol.GetMessages{
		Ty:    "161",
		Token: bobToken,
		Limit: 3,
	})
	msgList := resp.(*protocol.MessagesList)
	if len(msgList.Msgs) != 3 {
		t.Errorf("expected 3 messages with limit=3, got %d", len(msgList.Msgs))
	}

	ctx2 := testContext(t)
	resp2 := h.HandleGetMessages(ctx2, &protocol.GetMessages{
		Ty:    "161",
		Token: bobToken,
		Limit: 100,
	})
	msgList2 := resp2.(*protocol.MessagesList)
	if len(msgList2.Msgs) != 2 {
		t.Errorf("expected 2 remaining messages, got %d", len(msgList2.Msgs))
	}
}
