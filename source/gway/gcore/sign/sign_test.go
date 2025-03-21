package sign

import (
	"strings"
	"testing"
)

func TestSignHmac(t *testing.T) {
	secret := "IAMbTXzdhPfKf4LIV0TLadtBgNT9zQu1YEsh"
	content := `16733413355255FdeE4CnNztmXLC9HE100000{"name":"b","category":"option","symbol":"BTCUSDT"}`
	sign := "1c5e4598e0d7d71c5d47ec1f31bb0c4ed923b90220d7cf7eb9fd6346cc6a973d"
	res, err := Sign(TypeHmac, []byte(secret), []byte(content))
	if err != nil || res != sign {
		t.Errorf("sign hmac fail, err=%v, res=%v", err, res)
	}
}

func TestSignRsa(t *testing.T) {
	private_key := `MIICXgIBAAKBgQDcVlEN5NlRa2oW2Bi/84BnAJHiTendRj77bik9wSdHn3JXaFKv
	vngcY7Pspq1mWchYQYIEqLWryUztqhrkuBA9Im+tNka7Hma2eIcZ01tMCDgjewYK
	7cZX/r1xibKXO28CVl0Unnkru83sPtA+wkIT5GiNqHSMU7llWjki3P+cmQIDAQAB
	AoGBAKh3wv+pk9PiGjqfPcU+bFXVJLXwtrh+JlfeMeBK2Dq2Ghnk5RwEuReT0BVI
	l9pjGYEJjVz8lfNkNdKeNnPcnGSGeozAiFMtrVZ+eJBSjTEPL62/Hl8DFnU4MXvB
	R8gmU1XHWcy8JllPO1RMDEiJPl2eDGNTDPwdSPfnOsAQ0zmRAkEA93lBcM2bcwkh
	PtukZXo/LmcFdpim8N/eqHJm6qmEnBtF0fbz0w2kFPlVqm6fj5gbA7oaAfBTeT/Z
	X28UUlXB/QJBAOPtttxr+EkAgj8v4ajuCKeP9Dzw4b7MtqkZXkeyWktHWhtVDHoX
	dMLhQrXptsCO2ZVxddMYR/wHhWicdW0U6c0CQFnriDi5rMMmzRqu6lQpEC4HJvgJ
	zZb2cUwZjYW0pMeoLT12ku/cJAOu+U6dNYMSjLZU98A+l8YVyiEgFm04Ve0CQQDa
	YNyNzejB0Pn5nl+f4ghqutLwPH6dtzffRk39dZVrgL6FZ3Qf2i9ltDudXYJadcNk
	mqOgECiQAYjBlP4w+BOVAkEA9chXafFfOHWFL4cvPH9SUTSaHLjTQO9X9qKQnv7E
	TbZP78dwa+U+MZWw8Gsud3d0hNddRgVuWG93JCKfalbaGA==`

	content := "{\"app_name\":\"usvc\",\"data\":\"your request param\"}"
	sign := `ljhxWHg2u076lmmDEbvWbYUMdib9poU9Nqo/+L2GQpYLMlsyoeZbOadrTb88wviEp9tVm4TkciAKFKx68e97dCg7ezM3lg9+LTLTvohBHtCv2abcQVMTBXNTknBFddtMFI1oWA5eQJ6lm7IjSup/0a2XWxSDi5kDA3HYDocvpro=`

	res, err := Sign(TypeRsa, []byte(private_key), []byte(content))
	if err != nil || res != sign {
		t.Errorf("sign hmac fail, err=%v, res=%v", err, res)
	}
}

func TestVerifySignHmac(t *testing.T) {
	secret := "IAMbTXzdhPfKf4LIV0TLadtBgNT9zQu1YEsh"
	content := `16733413355255FdeE4CnNztmXLC9HE100000{"name":"b","category":"option","symbol":"BTCUSDT"}`
	sign := "1c5e4598e0d7d71c5d47ec1f31bb0c4ed923b90220d7cf7eb9fd6346cc6a973d"
	invalidSign := "invalid signature"

	// 正常hmac验签
	if err := Verify(TypeHmac, []byte(secret), []byte(content), sign); err != nil {
		t.Error("invalid hmac signature")
	}

	// 未指定类型,默认hmac
	if err := Verify("", []byte(secret), []byte(content), sign); err != nil {
		t.Error("invalid hmac signature")
	}

	// 异常hmac验签
	if err := Verify(TypeHmac, []byte(secret), []byte(content), invalidSign); err == nil {
		t.Error("invalid hmac signature")
	}
}

func TestVerifySignRsa(t *testing.T) {
	// private_key := `MIICXgIBAAKBgQDcVlEN5NlRa2oW2Bi/84BnAJHiTendRj77bik9wSdHn3JXaFKv
	// vngcY7Pspq1mWchYQYIEqLWryUztqhrkuBA9Im+tNka7Hma2eIcZ01tMCDgjewYK
	// 7cZX/r1xibKXO28CVl0Unnkru83sPtA+wkIT5GiNqHSMU7llWjki3P+cmQIDAQAB
	// AoGBAKh3wv+pk9PiGjqfPcU+bFXVJLXwtrh+JlfeMeBK2Dq2Ghnk5RwEuReT0BVI
	// l9pjGYEJjVz8lfNkNdKeNnPcnGSGeozAiFMtrVZ+eJBSjTEPL62/Hl8DFnU4MXvB
	// R8gmU1XHWcy8JllPO1RMDEiJPl2eDGNTDPwdSPfnOsAQ0zmRAkEA93lBcM2bcwkh
	// PtukZXo/LmcFdpim8N/eqHJm6qmEnBtF0fbz0w2kFPlVqm6fj5gbA7oaAfBTeT/Z
	// X28UUlXB/QJBAOPtttxr+EkAgj8v4ajuCKeP9Dzw4b7MtqkZXkeyWktHWhtVDHoX
	// dMLhQrXptsCO2ZVxddMYR/wHhWicdW0U6c0CQFnriDi5rMMmzRqu6lQpEC4HJvgJ
	// zZb2cUwZjYW0pMeoLT12ku/cJAOu+U6dNYMSjLZU98A+l8YVyiEgFm04Ve0CQQDa
	// YNyNzejB0Pn5nl+f4ghqutLwPH6dtzffRk39dZVrgL6FZ3Qf2i9ltDudXYJadcNk
	// mqOgECiQAYjBlP4w+BOVAkEA9chXafFfOHWFL4cvPH9SUTSaHLjTQO9X9qKQnv7E
	// TbZP78dwa+U+MZWw8Gsud3d0hNddRgVuWG93JCKfalbaGA==`

	publicKey := `MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDcVlEN5NlRa2oW2Bi/84BnAJHi
    TendRj77bik9wSdHn3JXaFKvvngcY7Pspq1mWchYQYIEqLWryUztqhrkuBA9Im+t
    Nka7Hma2eIcZ01tMCDgjewYK7cZX/r1xibKXO28CVl0Unnkru83sPtA+wkIT5GiN
    qHSMU7llWjki3P+cmQIDAQAB`

	publicKey = formatPublicKey(publicKey)
	content := "{\"app_name\":\"usvc\",\"data\":\"your request param\"}"
	sign := `ljhxWHg2u076lmmDEbvWbYUMdib9poU9Nqo/+L2GQpYLMlsyoeZbOadrTb88wviEp9tVm4TkciAKFKx68e97dCg7ezM3lg9+LTLTvohBHtCv2abcQVMTBXNTknBFddtMFI1oWA5eQJ6lm7IjSup/0a2XWxSDi5kDA3HYDocvpro=`
	invalidSign := "invalid signature"

	// 正常rsa验签
	if err := Verify(TypeRsa, []byte(publicKey), []byte(content), sign); err != nil {
		t.Error("verify signature failed")
	}

	// 异常rsa验签
	if err := Verify(TypeRsa, []byte(publicKey), []byte(content), invalidSign); err == nil {
		t.Error("invalid rsa signature")
	}
}

// formatPublicKey 组装公钥
func formatPublicKey(publicKey string) string {
	const (
		// PUBPEMBEGIN 公钥 PEMBEGIN 开头
		PUBPEMBEGIN = "-----BEGIN PUBLIC KEY-----\n"
		// PUBPEMEND 公钥 PEMEND 结尾
		PUBPEMEND = "\n-----END PUBLIC KEY-----"
	)
	if !strings.HasPrefix(publicKey, PUBPEMBEGIN) {
		publicKey = PUBPEMBEGIN + publicKey
	}
	if !strings.HasSuffix(publicKey, PUBPEMEND) {
		publicKey += PUBPEMEND
	}
	return publicKey
}

func BenchmarkSignHmac(b *testing.B) {
	secret := "IAMbTXzdhPfKf4LIV0TLadtBgNT9zQu1YEsh"
	content := `16733413355255FdeE4CnNztmXLC9HE100000{"name":"b","category":"option","symbol":"BTCUSDT"}`
	sign := "1c5e4598e0d7d71c5d47ec1f31bb0c4ed923b90220d7cf7eb9fd6346cc6a973d"

	for i := 0; i < b.N; i++ {
		_ = Verify(TypeHmac, []byte(secret), []byte(content), sign)
	}
	// goos: darwin
	// goarch: arm64
	// pkg: bgw/pkg/common/util
	// BenchmarkSignHmac
	// BenchmarkSignHmac-8   	 2174733	       549.3 ns/op	     784 B/op	      10 allocs/op
	// PASS
}

func BenchmarkSignRsa(b *testing.B) {
	publicKey := `MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDcVlEN5NlRa2oW2Bi/84BnAJHi
    TendRj77bik9wSdHn3JXaFKvvngcY7Pspq1mWchYQYIEqLWryUztqhrkuBA9Im+t
    Nka7Hma2eIcZ01tMCDgjewYK7cZX/r1xibKXO28CVl0Unnkru83sPtA+wkIT5GiN
    qHSMU7llWjki3P+cmQIDAQAB`

	publicKey = formatPublicKey(publicKey)
	content := "{\"app_name\":\"usvc\",\"data\":\"your request param\"}"
	sign := `ljhxWHg2u076lmmDEbvWbYUMdib9poU9Nqo/+L2GQpYLMlsyoeZbOadrTb88wviEp9tVm4TkciAKFKx68e97dCg7ezM3lg9+LTLTvohBHtCv2abcQVMTBXNTknBFddtMFI1oWA5eQJ6lm7IjSup/0a2XWxSDi5kDA3HYDocvpro=`

	for i := 0; i < b.N; i++ {
		_ = Verify(TypeRsa, []byte(publicKey), []byte(content), sign)
	}
	// goos: darwin
	// goarch: arm64
	// pkg: bgw/pkg/common/util
	// BenchmarkSignRsa
	// BenchmarkSignRsa-8   	   83574	     13944 ns/op	    4866 B/op	      40 allocs/op
}
