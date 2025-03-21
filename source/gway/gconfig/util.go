package gconfig

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"unsafe"
)

// OnInit 数据unmarshal前调用,常用于初始化默认数据
type OnInit interface {
	OnInit()
}

// OnLoaded 配置加载完后调用,常用于数据校验,以及数据加工处理,如果返回error则不会替换内存数据
type OnLoaded interface {
	OnLoaded() error
}

// UnmarshalFunc define json/yaml unmarshal signature
type UnmarshalFunc func(in []byte, out interface{}) (err error)

// Unmarshal out必须是结构体的指针的指针,内部会使用atomic.SwapPointer保证原子性操作替换指针
// 同时如果实现了OnInit和OnLoaded接口,会自动调用对应接口
func Unmarshal(data []byte, out interface{}, unmarshal UnmarshalFunc) error {
	if unmarshal == nil {
		unmarshal = json.Unmarshal
	}
	rv := reflect.ValueOf(out)
	rt := rv.Type()
	if rt.Kind() != reflect.Ptr || rt.Elem().Kind() != reflect.Ptr || rt.Elem().Elem().Kind() != reflect.Struct {
		return ErrInvalidType
	}

	value := reflect.New(rt.Elem().Elem())
	iface := value.Interface()
	// used to initialize
	if obj, ok := iface.(OnInit); ok {
		obj.OnInit()
	}

	if len(data) != 0 {
		if err := unmarshal(data, value.Interface()); err != nil {
			return err
		}
	}

	// used to check data
	if obj, ok := iface.(OnLoaded); ok {
		if err := obj.OnLoaded(); err != nil {
			return err
		}
	}

	atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(rv.Pointer())), unsafe.Pointer(value.Pointer()))
	return nil
}

// LoadFile 从配置文件中加载, 默认从./conf文件夹中加载,如果是单元测试,则自动向上查找./conf目录
// filename通常为main.yaml
func LoadFile(filename string, out interface{}, unmarshal UnmarshalFunc) error {
	if unmarshal == nil {
		unmarshal = json.Unmarshal
	}
	confDir, err := FindConfDir()
	if err != nil {
		return err
	}

	path := filepath.Join(confDir, filename)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return unmarshal(data, out)
}

// FindConfDir 查找conf目录, 如果是执行go test,则会从当前工作目录向上查找./conf文件夹,直到找到或遇到go.mod则停止查找
func FindConfDir() (string, error) {
	if isDirExist("./conf") {
		return "./conf", nil
	}

	if !isGoTest() {
		return "", ErrNotFound
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for dir != "" {
		dir = filepath.Dir(dir)
		confDir := filepath.Join(dir, "conf")
		if isDirExist(confDir) {
			return confDir, nil
		}
		// 找到工程根目录则结束
		gomodPath := filepath.Join(dir, "go.mod")
		if isFileExist(gomodPath) {
			return "", ErrNotFound
		}
	}

	return "", ErrNotFound
}

func isDirExist(path string) bool {
	if info, err := os.Stat(path); (err == nil || os.IsExist(err)) && info.IsDir() {
		return true
	}

	return false
}

func isFileExist(path string) bool {
	if info, err := os.Stat(path); (err == nil || os.IsExist(err)) && !info.IsDir() {
		return true
	}

	return false
}

var _isTesting = -1

func isGoTest() bool {
	if _isTesting != -1 {
		return _isTesting == 1
	}

	_isTesting = 1
	if flag.Lookup("test.v") != nil {
		return true
	}

	// -test.timeout,-test.run,-test.bench,-test.v
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "-test.") {
			return true
		}
	}

	_isTesting = 0
	return false
}
