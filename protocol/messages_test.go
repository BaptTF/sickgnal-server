package protocol

import (
	"encoding/json"
	"testing"
)

func TestOkMsgJSON(t *testing.T) {
	msg := NewOk()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	expected := `{"ty":"254"}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestErrorMsgJSON(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{ErrInvalidMessage, `{"ty":"255","code":"InvalidMessage"}`},
		{ErrMessageTypeNotAccepted, `{"ty":"255","code":"MessageTypeNotAccepted"}`},
		{ErrInvalidAuthentication, `{"ty":"255","code":"InvalidAuthentication"}`},
		{ErrUsernameUnavailable, `{"ty":"255","code":"UsernameUnavailable"}`},
		{ErrUserNotFound, `{"ty":"255","code":"UserNotFound"}`},
		{ErrPreKeyLimit, `{"ty":"255","code":"PreKeyLimit"}`},
		{ErrNoAvailableKey, `{"ty":"255","code":"NoAvailableKey"}`},
		{ErrInternalError, `{"ty":"255","code":"InternalError"}`},
	}

	for _, tt := range tests {
		msg := NewError(tt.code)
		data, err := json.Marshal(msg)
		if err != nil {
			t.Fatalf("marshal %s: %v", tt.code, err)
		}
		if string(data) != tt.expected {
			t.Errorf("code %s: expected %s, got %s", tt.code, tt.expected, string(data))
		}
	}
}

func TestAuthTokenJSON(t *testing.T) {
	msg := NewAuthToken("550e8400-e29b-41d4-a716-446655440000", "mytoken")
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["ty"] != "129" {
		t.Errorf("expected ty=129, got %v", parsed["ty"])
	}
	if parsed["id"] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("unexpected id: %v", parsed["id"])
	}
	if parsed["token"] != "mytoken" {
		t.Errorf("unexpected token: %v", parsed["token"])
	}
}

func TestUserProfileJSON(t *testing.T) {
	msg := NewUserProfile("some-uuid", "alice")
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["ty"] != "10" {
		t.Errorf("expected ty=10, got %v", parsed["ty"])
	}
	if parsed["id"] != "some-uuid" {
		t.Errorf("unexpected id: %v", parsed["id"])
	}
	if parsed["username"] != "alice" {
		t.Errorf("unexpected username: %v", parsed["username"])
	}
}

func TestPreKeyBundleJSON(t *testing.T) {
	// With ephemeral key
	msg := NewPreKeyBundle("ik_b64", "pk_b64", "pksig_b64", &EphemeralKey{
		ID:  "key-uuid",
		Key: "ek_b64",
	})
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["ty"] != "0" {
		t.Errorf("expected ty=0, got %v", parsed["ty"])
	}
	if parsed["ik"] != "ik_b64" {
		t.Errorf("unexpected ik: %v", parsed["ik"])
	}
	if parsed["pk"] != "pk_b64" {
		t.Errorf("unexpected pk: %v", parsed["pk"])
	}
	if parsed["pksig"] != "pksig_b64" {
		t.Errorf("unexpected pksig: %v", parsed["pksig"])
	}

	ek := parsed["ek"].(map[string]interface{})
	if ek["id"] != "key-uuid" {
		t.Errorf("unexpected ek.id: %v", ek["id"])
	}
	if ek["ek"] != "ek_b64" {
		t.Errorf("unexpected ek.ek: %v", ek["ek"])
	}

	// Without ephemeral key (null)
	msg2 := NewPreKeyBundle("ik_b64", "pk_b64", "pksig_b64", nil)
	data2, err := json.Marshal(msg2)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed2 map[string]interface{}
	json.Unmarshal(data2, &parsed2)

	if parsed2["ek"] != nil {
		t.Errorf("expected ek=null, got %v", parsed2["ek"])
	}
}

func TestPreKeyStatusJSON(t *testing.T) {
	msg := NewPreKeyStatus(100, []string{"uuid1", "uuid2", "uuid3"})
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["ty"] != "136" {
		t.Errorf("expected ty=136, got %v", parsed["ty"])
	}
	if parsed["limit"].(float64) != 100 {
		t.Errorf("unexpected limit: %v", parsed["limit"])
	}
	keys := parsed["keys"].([]interface{})
	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestMessagesListJSON(t *testing.T) {
	msgs := []json.RawMessage{
		json.RawMessage(`{"ty":"1","sndr_id":"uuid"}`),
		json.RawMessage(`{"ty":"2","sndr_id":"uuid"}`),
	}
	msg := NewMessagesList(msgs)
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["ty"] != "170" {
		t.Errorf("expected ty=170, got %v", parsed["ty"])
	}
	list := parsed["msgs"].([]interface{})
	if len(list) != 2 {
		t.Errorf("expected 2 msgs, got %d", len(list))
	}
}

func TestMessagesListEmptyJSON(t *testing.T) {
	msg := NewMessagesList(nil)
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	list := parsed["msgs"].([]interface{})
	if len(list) != 0 {
		t.Errorf("expected 0 msgs, got %d", len(list))
	}
}

func TestAuthChallengeJSON(t *testing.T) {
	msg := NewAuthChallenge("base64nonce==")
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)

	if parsed["ty"] != "131" {
		t.Errorf("expected ty=131, got %v", parsed["ty"])
	}
	if parsed["chall"] != "base64nonce==" {
		t.Errorf("unexpected chall: %v", parsed["chall"])
	}
}

func TestParseMessageType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{`{"ty":"128","username":"alice"}`, "128", false},
		{`{"ty":"254"}`, "254", false},
		{`{"ty":"0"}`, "0", false},
		{`{"no_ty":"val"}`, "", true},
		{`invalid json`, "", true},
		{`{"ty":""}`, "", true},
	}

	for _, tt := range tests {
		ty, err := ParseMessageType(json.RawMessage(tt.input))
		if tt.wantErr {
			if err == nil {
				t.Errorf("input %s: expected error, got ty=%s", tt.input, ty)
			}
		} else {
			if err != nil {
				t.Errorf("input %s: unexpected error: %v", tt.input, err)
			}
			if ty != tt.expected {
				t.Errorf("input %s: expected ty=%s, got %s", tt.input, tt.expected, ty)
			}
		}
	}
}

func TestParseMessageServerTypes(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"CreateAccount", `{"ty":"128","ik":"key","username":"alice","sig":"sig"}`},
		{"AuthChallengeRequest", `{"ty":"130","username":"alice"}`},
		{"AuthChallengeSolve", `{"ty":"132","chall":"nonce","solve":"sig"}`},
		{"PreKeyUpload", `{"ty":"133","token":"tok","replace":false,"pk":null,"tks":null}`},
		{"PreKeyDelete", `{"ty":"134","token":"tok","keys":[]}`},
		{"PreKeyStatusRequest", `{"ty":"135","token":"tok"}`},
		{"PreKeyBundleRequest", `{"ty":"137","token":"tok","id":"uuid"}`},
		{"UserProfileByUsername", `{"ty":"140","token":"tok","username":"alice"}`},
		{"UserProfileById", `{"ty":"141","token":"tok","id":"uuid"}`},
		{"SendInitialMessage", `{"ty":"150","token":"tok","rcpt_id":"uuid","ik":"k","ek":"k","kid":null,"i":"u","j":"u","nonce":"n","msg":"m"}`},
		{"SendMessage", `{"ty":"151","token":"tok","rcpt_id":"uuid","nonce":"n","kid":"k","msg":"m"}`},
		{"GetInitialMessages", `{"ty":"160","token":"tok","limit":100}`},
		{"GetMessages", `{"ty":"161","token":"tok","limit":100}`},
		{"EnableInstantRelay", `{"ty":"180","token":"tok"}`},
		{"DisableInstantRelay", `{"ty":"181"}`},
	}

	for _, tt := range tests {
		msg, ty, err := ParseMessage(json.RawMessage(tt.input))
		if err != nil {
			t.Errorf("%s: parse error: %v", tt.name, err)
			continue
		}
		if msg == nil {
			t.Errorf("%s: msg is nil", tt.name)
		}
		_ = ty // just make sure it returned
	}
}

func TestParseMessageRejectsClientTypes(t *testing.T) {
	clientTypes := []string{
		`{"ty":"0"}`,  // PreKeyBundle
		`{"ty":"1"}`,  // ConversationOpen
		`{"ty":"2"}`,  // ConversationMessage
		`{"ty":"3"}`,  // KeyRotation
		`{"ty":"10"}`, // UserProfile
	}

	for _, input := range clientTypes {
		_, _, err := ParseMessage(json.RawMessage(input))
		if err == nil {
			t.Errorf("input %s: expected error for client-only type, got nil", input)
		}
	}
}
