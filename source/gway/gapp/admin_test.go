package gapp

import (
	"bytes"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestAdminGet(t *testing.T) {
	RegisterAdmin("test", "test", onTestAdmin)
	r := httptest.NewRequest("GET", "http://localhost:6480/admin?cmd=test&params=1,2&key1=2&key2=value2", nil)
	w := httptest.NewRecorder()
	onAdminHandler(w, r)
}

func TestAdminPost(t *testing.T) {
	RegisterAdmin("test", "test", onTestAdmin)
	// 方式一:
	body := `{"args": "test  	 1 2 key1=2 key2=value2"}`
	r := httptest.NewRequest("POST", "http://localhost:6480/admin", bytes.NewBuffer([]byte(body)))
	w := httptest.NewRecorder()
	onAdminHandler(w, r)
	// 方式二:
	body1 := `test 1 2 key1=2 key2=value2`
	r1 := httptest.NewRequest("POST", "http://localhost:6480/admin", bytes.NewBuffer([]byte(body1)))
	w2 := httptest.NewRecorder()
	onAdminHandler(w2, r1)
}

func onTestAdmin(args AdminArgs) (interface{}, error) {
	fmt.Println(args.ParamSize(), args.GetIntAt(0), args.GetIntBy("key1"), args.GetStringBy("key2"))
	return "test", nil
}
