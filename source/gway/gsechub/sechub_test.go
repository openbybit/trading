package gsechub

import "testing"

func TestDecrypt(t *testing.T) {
	passwd := "abc"
	s, err := Decrypt(passwd)
	t.Log(s, err)
}
