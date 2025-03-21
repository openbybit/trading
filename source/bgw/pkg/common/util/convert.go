package util

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unsafe"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"github.com/buger/jsonparser"
	jsoniter "github.com/json-iterator/go"
	"github.com/json-iterator/go/extra"
	"github.com/oliveagle/jsonpath"
	"golang.org/x/text/encoding/charmap"
	"gopkg.in/yaml.v3"
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

// Base64EncodeByte base64 encode byte
func Base64EncodeByte(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64DecodeByte base64 decode byte
func Base64DecodeByte(data []byte) string {
	result, _ := base64.StdEncoding.DecodeString(string(data))
	return string(result)
}

// ToJSON json marshal, no error
func ToJSON(data interface{}) []byte {
	b, err := jsoniter.Marshal(data)
	if err != nil {
		return nil
	}
	return b
}

// NewJSONEncoder returns a new json encoder.
func NewJSONEncoder(w io.Writer) *jsoniter.Encoder {
	return jsoniter.NewEncoder(w)
}

// NewJSONDecoder returns a new json decoder.
func NewJSONDecoder(r io.Reader) *jsoniter.Decoder {
	return jsoniter.NewDecoder(r)
}

func init() {
	extra.RegisterFuzzyDecoders()
	jsoniter.RegisterTypeDecoder("string", &fuzzyStringDecoder{})
}

type fuzzyStringDecoder struct{}

// Decode fuzzyStringDecoder decode
func (decoder *fuzzyStringDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	valueType := iter.WhatIsNext()
	switch valueType {
	case jsoniter.NumberValue:
		var number json.Number
		iter.ReadVal(&number)
		*((*string)(ptr)) = string(number)
	case jsoniter.StringValue:
		*((*string)(ptr)) = iter.ReadString()
	case jsoniter.NilValue:
		iter.Skip()
		*((*string)(ptr)) = ""
	case jsoniter.BoolValue:
		*((*string)(ptr)) = strconv.FormatBool(iter.ReadBool())
	default:
		iter.ReportError("fuzzyStringDecoder", "not number or string")
	}
}

// JsonMarshal json marshal
func JsonMarshal(data interface{}) ([]byte, error) {
	return jsoniter.Marshal(data)
}

// JsonUnmarshal json unmarshal from byte
func JsonUnmarshal(data []byte, target interface{}) error {
	return jsoniter.Unmarshal(data, target)
}

// JsonUnmarshalString json unmarshal from string
func JsonUnmarshalString(data string, target interface{}) error {
	return jsoniter.UnmarshalFromString(data, target)
}

// JsonGetString json get from byte
func JsonGetString(data []byte, key string) string {
	v, err := jsonparser.GetString(data, key)
	if err != nil {
		return ""
	}
	return v
}

// JsonGetAllVal get all value for given key for situation that duplicate key in json
func JsonGetAllVal(data []byte, key string) [][]byte {
	var (
		res [][]byte
		f   = func(k []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			if string(k) == key {
				res = append(res, value)
			}
			return nil
		}
	)
	_ = jsonparser.ObjectEach(data, f)
	return res
}

// JsonGetBool json get from byte
func JsonGetBool(data []byte, key string) (bool, error) {
	return jsonparser.GetBoolean(data, key)
}

// JsonGetInt64 json get from byte
func JsonGetInt64(data []byte, key string) (int64, error) {
	return jsonparser.GetInt(data, key)
}

// YamlMarshal yaml marshal
func YamlMarshal(data interface{}) ([]byte, error) {
	return yaml.Marshal(data)
}

// YamlUnmarshal yaml unmarshal from byte
func YamlUnmarshal(data []byte, target interface{}) error {
	return yaml.Unmarshal(data, target)
}

// YamlUnmarshalString yaml unmarshal from string
func YamlUnmarshalString(data string, target interface{}) error {
	return yaml.Unmarshal([]byte(data), target)
}

// ToJSONString data convert to json string
func ToJSONString(data interface{}) string {
	return string(ToJSON(data))
}

// ToMD5 get string md5sum
func ToMD5(str string) string {
	hash := md5.New()
	hash.Write([]byte(str))
	cipherStr := hash.Sum(nil)

	return hex.EncodeToString(cipherStr)
}

// ToMD5Byte get byte md5sum
func ToMD5Byte(data []byte) string {
	hash := md5.New()
	hash.Write(data)
	cipherStr := hash.Sum(nil)

	return hex.EncodeToString(cipherStr)
}

var latin1 = charmap.ISO8859_1.NewDecoder()

// DecodeHeaderValue decode header value
func DecodeHeaderValue(v []byte) string {
	bs, err := latin1.Bytes(v)
	if err != nil {
		return ""
	}

	return cast.UnsafeBytesToString(bs)
}

func ToValidateGrpcHeader(key, msg string) string {
	findIdx := -1
	for i := 0; i < len(msg); i++ {
		if msg[i] < 0x20 || msg[i] > 0x7E {
			findIdx = i
			break
		}
	}
	if findIdx == -1 {
		return msg
	}
	glog.Info(context.TODO(), "invalid value", glog.String("key", key), glog.Any("value", msg))

	buf := strings.Builder{}
	buf.WriteString(msg[:findIdx])
	buf.WriteString(" ")
	for i := findIdx + 1; i < len(msg); i++ {
		if msg[i] < 0x20 || msg[i] > 0x7E {
			continue
		}
		buf.WriteByte(msg[i])
	}

	return buf.String()
}

func JsonpathGet(data []byte, patten string) (res interface{}, err error) {
	var object interface{}
	err = jsoniter.Unmarshal(data, &object)
	if err != nil {
		return nil, err
	}
	if object == nil {
		return nil, errors.New("nil object")
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("josnpath lookup panic, %v", r)
		}
	}()

	return jsonpath.JsonPathLookup(object, patten)
}
