package ws

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestRateLimit(t *testing.T) {
	r := rateLimit{}
	assert.True(t, r.Allow())
	r.Set(1, 0)
	assert.True(t, r.Allow())
	assert.False(t, r.Allow())

	// time.Sleep(time.Second)
	// assert.True(t, r.Allow())
	// assert.False(t, r.Allow())
}

func TestError(t *testing.T) {
	paramsErr := "Params Error"
	c1 := newCodeErr(1, paramsErr)
	c2 := newCodeErrFrom(c1, "detail: %s", "args1")
	assert.True(t, isError(c2, c1))
	assert.EqualValues(t, 1, c1.Code())
	assert.EqualValues(t, paramsErr, c1.Error())

	assert.Equal(t, c1.Code(), toCodeErr(c1).Code())
	assert.Equal(t, defaultErrorCode, toCodeErr(io.EOF).Code())
	assert.Nil(t, toCodeErr(nil))

	// code
	es := newEnvStore()
	es.SetMainnet()
	assert.Equal(t, paramsErr, c2.Error())
	es.Recovery()
}

func TestLogWriter(t *testing.T) {
	lw := logWriter{}
	_, _ = lw.Write([]byte("aaa"))
	assert.Equal(t, "aaa\n", lw.String())
}

func TestInt64SetUnmarshal(t *testing.T) {
	type LogConfig struct {
		Level  string   `yaml:"level" json:"level"` // glog日志等级
		UidMap Int64Set `yaml:"uids" json:"uids"`   // 根据uid打印log
	}
	t.Run("json ok", func(t *testing.T) {
		jsonData := `{"level": "warn", "uids":  [ 1  ]}`
		l := LogConfig{}
		assert.Nil(t, json.Unmarshal([]byte(jsonData), &l))
		assert.Equal(t, 1, len(l.UidMap))

		jsonData = `{"level": "warn", "uids":  { "1": true, "2": false}}`
		l = LogConfig{}
		assert.Nil(t, json.Unmarshal([]byte(jsonData), &l))
		assert.Equal(t, 1, len(l.UidMap))
	})

	t.Run("json fail", func(t *testing.T) {
		jsonData := `{"level": "warn", "uids":  [ "1"  ]}`
		l := LogConfig{}
		assert.NotNil(t, json.Unmarshal([]byte(jsonData), &l))

		jsonData = `{"level": "warn", "uids":  { 1: true}}`
		l = LogConfig{}
		assert.NotNil(t, json.Unmarshal([]byte(jsonData), &l))

		jsonData = `{"level": "warn", "uids":  { "1": "asdf"}}`
		l = LogConfig{}
		assert.NotNil(t, json.Unmarshal([]byte(jsonData), &l))
	})

	t.Run("yaml ok", func(t *testing.T) {
		yamlData := `{"level": "warn", "uids":  [ 1  ]}`

		l1 := LogConfig{}
		assert.Nil(t, yaml.Unmarshal([]byte(yamlData), &l1))
		assert.Equal(t, 1, len(l1.UidMap))

		yamlData1 := `{"level": "warn", "uids":  { 1: true, 2: false} }`

		l2 := LogConfig{}
		assert.Nil(t, yaml.Unmarshal([]byte(yamlData1), &l2))
		assert.Equal(t, 1, len(l2.UidMap))
	})

	t.Run("yaml fail", func(t *testing.T) {
		yamlData := `{"level": "warn", "uids":  [ "1"  ]}`

		l1 := LogConfig{}
		assert.NotNil(t, yaml.Unmarshal([]byte(yamlData), &l1))

		yamlData1 := `{"level": "warn", "uids":  { "1": true, 2: false} }`

		l2 := LogConfig{}
		assert.NotNil(t, yaml.Unmarshal([]byte(yamlData1), &l2))
	})

	t.Run("Int64Set Unmarshal fail", func(t *testing.T) {
		data := `{a:"b"}`
		var target = make(Int64Set)
		err := target.UnmarshalJSON([]byte(data))
		assert.NotNil(t, err)
	})
}

func TestStrintSetUnmarshal(t *testing.T) {
	type LogConfig struct {
		Level  string    `yaml:"level" json:"level"` // glog日志等级
		UidMap StringSet `yaml:"uids" json:"uids"`   // 根据uid打印log
	}

	t.Run("json ok", func(t *testing.T) {
		jsonData := `{"level": "warn", "uids":  [ "1"  ]}`
		l := LogConfig{}
		assert.Nil(t, json.Unmarshal([]byte(jsonData), &l))
		assert.Equal(t, 1, len(l.UidMap))

		jsonData = `{"level": "warn", "uids":  { "1": true}}`
		l = LogConfig{}
		assert.Nil(t, json.Unmarshal([]byte(jsonData), &l))
		assert.Equal(t, 1, len(l.UidMap))
	})

	t.Run("json fail", func(t *testing.T) {
		jsonData := `{"level": "warn", "uids":  [ 1  ]}`
		l := LogConfig{}
		assert.NotNil(t, json.Unmarshal([]byte(jsonData), &l))

		jsonData = `{"level": "warn", "uids":  { 1: true}}`
		l = LogConfig{}
		assert.NotNil(t, json.Unmarshal([]byte(jsonData), &l))
	})

	t.Run("yaml ok", func(t *testing.T) {
		yamlData := `{"level": "warn", "uids":  [ "1"  ]}`

		l1 := LogConfig{}
		assert.Nil(t, yaml.Unmarshal([]byte(yamlData), &l1))
		assert.Equal(t, 1, len(l1.UidMap))

		yamlData1 := `{"level": "warn", "uids":  { "1": true, "2": false} }`

		l2 := LogConfig{}
		assert.Nil(t, yaml.Unmarshal([]byte(yamlData1), &l2))
		assert.Equal(t, 1, len(l2.UidMap))
	})

	t.Run("yaml fail", func(t *testing.T) {
		// yaml unmarshal不会报错
		// yamlData := `{"level": "warn", "uids":  [ true  ]}`
		// l1 := LogConfig{}
		// assert.NotNil(t, yaml.Unmarshal([]byte(yamlData), &l1))

		yamlData1 := `{"level": "warn", "uids":  { "1": "aa"} }`
		l2 := LogConfig{}
		assert.NotNil(t, yaml.Unmarshal([]byte(yamlData1), &l2))
	})

	t.Run("StringSet Unmarshal json fail", func(t *testing.T) {
		data := `{a:"b"}`
		var target = make(StringSet)
		err := target.UnmarshalJSON([]byte(data))
		assert.NotNil(t, err)
	})

	t.Run("StringSet Unmarshal yaml fail", func(t *testing.T) {
		var target = make(StringSet)
		err := target.UnmarshalYAML(func(d interface{}) error {
			return fmt.Errorf("abc")
		})
		assert.NotNil(t, err)
	})
}
