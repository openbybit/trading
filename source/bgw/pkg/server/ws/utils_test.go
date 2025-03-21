package ws

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"code.bydev.io/frameworks/byone/core/nacos"
	"github.com/stretchr/testify/assert"
)

func TestIsNil(t *testing.T) {
	assert.True(t, isNil(nil), "nil")
	var acceptor Acceptor
	assert.True(t, isNil(acceptor), "acceptor")
	var user User
	assert.True(t, isNil(user), "user")
	assert.False(t, isNil(User(newUser(1))), "convert to interface")
	assert.False(t, isNil(newUser(2)))
}

func TestJsonDecode(t *testing.T) {
	data := `{"a": "a"}`
	out := map[string]interface{}{}
	if err := jsonDecode(strings.NewReader(data), &out); err != nil {
		t.Error("decode fail")
	}
	assert.EqualValues(t, map[string]interface{}{"a": "a"}, out)
}

func TestJsonMarshal(t *testing.T) {
	data, _ := jsonMarshal(map[string]interface{}{"a": "a"})
	assert.Equal(t, `{"a":"a"}`, string(data))
}

func TestToJsonString(t *testing.T) {
	data := toJsonString(map[string]interface{}{"a": "a"})
	assert.Equal(t, `{"a":"a"}`, data)
}

func TestNewUUID(t *testing.T) {
	assert.NotEmpty(t, newUUID())
}

func TestToInt64(t *testing.T) {
	assert.EqualValues(t, 1, toInt64("1"))
}

func TestToString(t *testing.T) {
	assert.EqualValues(t, "1", toString(1))
}

func TestToStringList(t *testing.T) {
	assert.Equal(t, []string{"1", "2"}, toStringList([]interface{}{"1", "2"}))
}

func TestNowUnixNano(t *testing.T) {
	assert.NotZero(t, nowUnixNano())
}

func TestDecodeUserID(t *testing.T) {
	token := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2NjU2MjkwNjksInVzZXJfaWQiOjM3MTk0MTcsIm5vbmNlIjoiNjI4YjgyOWYifQ.UKwCUjUtuIjVM3aK8_Y2ZgF7DVggnFe5x32JC0OF6LQt9O-7d1shJg5AJ8kQWDAOSrfi0o3Al1c0bQP1YcpB8g"
	userId, err := decodeUserIDFromToken(token)
	assert.Nil(t, err)
	assert.EqualValues(t, 3719417, userId)
	userId, err = decodeUserIDFromToken("3699159")
	assert.Nil(t, err)
	assert.EqualValues(t, 3699159, userId)

	_, err = decodeUserIDFromToken("")
	assert.NotNil(t, err)
	_, err = decodeUserIDFromToken("a.b")
	assert.NotNil(t, err)
	_, err = decodeUserIDFromToken("a.b.c")
	assert.NotNil(t, err)

	buildToken := func(payload string) string {
		n := base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(payload))
		return fmt.Sprintf("a.%s.c", n)
	}

	invalidJson := buildToken(`{"user_id": "1"`)
	_, err = decodeUserIDFromToken(invalidJson)
	assert.NotNil(t, err)

	zeroUid := buildToken(`{"user_id": "0"}`)
	_, err = decodeUserIDFromToken(zeroUid)
	assert.NotNil(t, err)
	assert.Truef(t, strings.Contains(err.Error(), "invalid userid"), err.Error())
}

func TestTrimSuffixIndex(t *testing.T) {
	assert.Equal(t, "ins.linear", trimSuffixIndex("ins.linear_1"))
	assert.Equal(t, "aaa_aaa", trimSuffixIndex("aaa_aaa"))
}

func TestPlatform(t *testing.T) {
	assert.Equal(t, "UNKNOWN", verifyPlatform(""))
	assert.Equal(t, "UNKNOWN", verifyPlatform("0"))
	assert.Equal(t, "STREAM", verifyPlatform("1"))
	assert.Equal(t, "WEB", verifyPlatform("2"))
	assert.Equal(t, "APP", verifyPlatform("3"))
	assert.Equal(t, "4", verifyPlatform("4"))
}

func TestVersion(t *testing.T) {
	assert.Equal(t, "UNKNOWN", verifyVersion(""))
	assert.Equal(t, "11.22.33", verifyVersion("11.22.33"))
	assert.Equal(t, "UNKNOWN", verifyVersion("aaa"))
}

func TestSource(t *testing.T) {
	assert.Equal(t, "UNKNOWN", verifySource(""))
	assert.Equal(t, "2", verifySource("2"))
}

func TestDistinctString(t *testing.T) {
	assert.Equal(t, []string{"2", "3"}, distinctString([]string{"2", "2", "3"}))
}

func TestVerifySource(t *testing.T) {
	if verifySource("2&timestamp=1670707841325") != "2" {
		t.Error("invalid source")
	}

	if verifySource("2") != "2" {
		t.Error("invalid source")
	}
}

func TestContainsInOrderedList(t *testing.T) {
	list := []string{"a", "d", "c", "b"}
	sort.Strings(list)
	assert.Equal(t, []string{"a", "b", "c", "d"}, list, "sort string list")

	for _, x := range list {
		assert.True(t, containsInOrderedList(list, x), "exist")
	}

	notExist := []string{"e", "1", "A"}
	for _, x := range notExist {
		assert.False(t, containsInOrderedList(list, x), "not exist")
	}

	if !containsAnyInOrderedList(list, []string{"d", "e"}) {
		t.Error("containsAnyInOrderedList,should be true")
	}

	if containsAnyInOrderedList(list, []string{"e", "f"}) {
		t.Error("containsAnyInOrderedList,should be false")
	}

	assert.False(t, containsAnyInOrderedList(nil, []string{"x"}))
}

func TestToNano(t *testing.T) {
	now := time.Now()
	assert.Equalf(t, toUnixNano(now.UnixNano()), now.UnixNano(), "nano")
	assert.Equalf(t, toUnixNano(now.UnixMicro()), now.UnixMicro()*1e3, "micro: %v, %v", now.UnixMicro(), strconv.Itoa(1e16))
	assert.Equalf(t, toUnixNano(now.UnixMilli()), now.UnixMilli()*1e6, "milli: %v, %v", now.UnixMilli(), strconv.Itoa(1e13))
	assert.Equalf(t, toUnixNano(now.Unix()), now.Unix()*1e9, "unix: %v, %v", now.Unix(), 1e9)
}

func TestEncodeMapToString(t *testing.T) {
	assert.Equal(t, "", encodeMapToString(nil))
	assert.Equal(t, "a=1,b=2", encodeMapToString(map[string]string{"a": "1", "b": "2"}))
}

func TestSetDefaultNacosConfig(t *testing.T) {
	nc := &nacos.NacosConf{}
	setDefaultNacosConfig(nc)

	es := newEnvStore()
	defer es.Recovery()
	es.SetMainnet()

	nc = &nacos.NacosConf{}
	setDefaultNacosConfig(nc)
	assert.Equal(t, "public", nc.NamespaceId)
}
