package nets

import "testing"

func TestClientIP(t *testing.T) {
	t.Log(GetLocalIP())
}
