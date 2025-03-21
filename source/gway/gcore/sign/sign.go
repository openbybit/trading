package sign

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
)

type Type string

const (
	TypeHmac Type = "hmac"
	TypeRsa  Type = "rsa"
)

// Sign 签名,支持rsa,hmac
// rsa使用base64编码结果
// hmac使用hex编码结果
func Sign(typ Type, secret, content []byte) (string, error) {
	hash := crypto.SHA256
	switch typ {
	case TypeRsa:
		priKey, err := parsePrivateKey(string(secret))
		if err != nil {
			return "", err
		}
		h := hash.New()
		h.Write(content)
		hashed := h.Sum(nil)
		sign, err := rsa.SignPKCS1v15(rand.Reader, priKey, hash, hashed)
		if err != nil {
			return "", err
		}
		return base64.StdEncoding.EncodeToString(sign), nil
	default:
		mac := hmac.New(hash.New, []byte(secret))
		mac.Write(content)
		signature := hex.EncodeToString(mac.Sum(nil))
		return signature, nil
	}
}

// Verify 验证签名是否正确,默认签名为hmac
func Verify(typ Type, secret, content []byte, sign string) error {
	switch typ {
	case TypeRsa:
		hashed := sha256.Sum256([]byte(content))
		pubKey, err := parsePublicKey(string(secret))
		if err != nil {
			return err
		}
		sig, err := base64.StdEncoding.DecodeString(sign)
		if err != nil {
			return err
		}
		err = rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], sig)
		return err
	default:
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(content)
		signature := hex.EncodeToString(mac.Sum(nil))
		if !strings.EqualFold(signature, sign) {
			return fmt.Errorf("hmac sign not equal: real=%s, sign=%s", signature, sign)
		}

		return nil
	}
}

// parsePrivateKey 私钥加签
func parsePrivateKey(privateKey string) (*rsa.PrivateKey, error) {
	const (
		// PEMBEGIN 私钥 PEMBEGIN 开头
		PEMBEGIN = "-----BEGIN RSA PRIVATE KEY-----\n"
		// PEMEND 私钥 PEMEND 结尾
		PEMEND = "\n-----END RSA PRIVATE KEY-----"
	)

	if !strings.HasPrefix(privateKey, PEMBEGIN) {
		privateKey = PEMBEGIN + privateKey
	}
	if !strings.HasSuffix(privateKey, PEMEND) {
		privateKey += PEMEND
	}

	block, _ := pem.Decode([]byte(privateKey))
	if block == nil {
		return nil, errors.New("私钥信息错误！")
	}
	priKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return priKey, nil
	}
	priKey8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return priKey8.(*rsa.PrivateKey), nil
}

// parsePublicKey 公钥验签
func parsePublicKey(publicKey string) (*rsa.PublicKey, error) {
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

	block, _ := pem.Decode([]byte(publicKey))
	if block == nil {
		return nil, errors.New("公钥信息错误！")
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pubKey.(*rsa.PublicKey), nil
}
