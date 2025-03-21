package gconfig

import "testing"

func TestLoad(t *testing.T) {
	c := NewMock(map[string]string{
		"test_key": `{"name": "test"}`,
	})

	type Config struct {
		Name string `json:"name"`
	}

	conf := &Config{}
	if err := Load(nil, c, "test_key", &conf, nil, nil); err != nil {
		t.Error(err)
	}

	if conf.Name != "test" {
		t.Error("load fail")
	}

	c.Put(nil, "test_key", `{"name": "changed"}`)

	t.Log("load success", conf)
}

func TestMock(t *testing.T) {
	key := "test_key"
	value := "value"
	c := NewMock(nil)
	x, _ := c.Get(nil, key)
	if x != "" {
		t.Error("should empty")
	}
	c.Delete(nil, key)
	c.Put(nil, key, value)
	x, _ = c.Get(nil, key)
	if x != value {
		t.Error("invalid value")
	}
}
