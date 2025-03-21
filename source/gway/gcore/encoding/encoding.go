package encoding

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
)

// Base64Encode base64 encode
func Base64Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// Base64Decode base64 decode
func Base64Decode(data string) string {
	result, _ := base64.StdEncoding.DecodeString(data)
	return string(result)
}

func MD5Hex(bs []byte) string {
	h := md5.New()
	h.Write(bs)
	return hex.EncodeToString(h.Sum(nil))
}
