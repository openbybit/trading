package galert

import (
	"os"

	"code.bydev.io/fbu/gateway/gway.git/gcore/nets"
)

type fieldType uint8

const (
	fieldTypeBasic       fieldType = iota // 普通kv数据
	fieldTypeCurrentTime                  // 当前时间
	fieldTypeLink                         // 链接
)

type linkInfo struct {
	Show string
	Link string
}

type Field struct {
	typ     fieldType
	key     string
	value   interface{}
	isShort bool
}

func (f *Field) SetShort() {
	f.isShort = true
}

func DefaultFields() []*Field {
	result := []*Field{
		CurrentTimeField("trigger", ""),
	}

	result = append(result, BasicField("ip", nets.GetLocalIP()))
	myEnv := os.Getenv("MY_ENV_NAME")
	if myEnv != "" {
		result = append(result, BasicField("env", myEnv))
	}
	myProjectName := os.Getenv("MY_PROJECT_NAME")
	if myProjectName != "" {
		result = append(result, BasicField("env", myProjectName))
	}
	return result
}

func BasicField(key string, value interface{}) *Field {
	return &Field{
		typ:   fieldTypeBasic,
		key:   key,
		value: value,
	}
}

func LinkField(key string, value string, link string) *Field {
	if value == "" {
		value = link
	}

	return &Field{
		typ:   fieldTypeLink,
		key:   key,
		value: &linkInfo{Show: value, Link: link},
	}
}

func CurrentTimeField(key string, layout string) *Field {
	if layout == "" {
		layout = "2006-01-02T15:04:05.000Z"
	}
	return &Field{
		typ:   fieldTypeCurrentTime,
		key:   key,
		value: layout,
	}
}
