package gconfig

import (
	"testing"
)

type Config struct {
	Name string `json:"name" yaml:"name"`
}

var testData = `{"name": "test"}`

func TestUnmarshal(t *testing.T) {
	conf := &Config{}
	if err := Unmarshal([]byte(testData), &conf, nil); err != nil {
		t.Error(err)
	} else if conf.Name != "test" {
		t.Error("unmarshal fail")
	} else {
		t.Log(conf)
	}
}

func TestIsTesting(t *testing.T) {
	if !isGoTest() {
		t.Error("should be true")
	}
}

func TestFindConf(t *testing.T) {
	dir, err := FindConfDir()
	if err != nil {
		t.Error(err)
	} else {
		t.Log(dir)
	}
}
