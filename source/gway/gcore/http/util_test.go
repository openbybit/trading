package http

import "testing"

func TestReplacePath(t *testing.T) {
	res := replacePathParams("im/v1/chats/:chat_id/:not_found/:Upper", map[string]string{"chat_id": "aaa", "Upper": "upper"})
	t.Log(res)
}

func TestIsContextType(t *testing.T) {
	res := IsContentType(MIMEApplicationJSONCharsetUTF8, MIMEApplicationJSON)
	if !res {
		t.Errorf("invalid content type: %v", MIMEApplicationJSON)
	}
}
