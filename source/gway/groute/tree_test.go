package groute

import "testing"

func TestTree(t *testing.T) {
	h := func() {
		t.Log("handler")
	}
	root := node{}
	root.addRoute("/test/a", h)
	root.addRoute("/test/b", h)
	root.addRoute("/prefix/*any", h)
	root.addRoute("/params/:name/:id", h)
	params := Params{}
	_ = root.findRoute("/params/aa/1", &params)
	t.Log(params.IsZero())
	t.Log(params.Route())
	t.Log(params.ByName("name"))      // aa
	t.Log(params.ByName("id"))        // aa
	t.Log(params.ByName("not exist")) //

	params = Params{}
	res := root.findRoute("", &params)
	if res != nil {
		t.Error("should be nil")
	}

	params = Params{}
	res = root.findRoute("/", &params)
	if res != nil {
		t.Error("should be nil")
	}

	params = Params{}
	_ = root.findRoute("/prefix/bb", &params)
	t.Log(params.ByName("any"))
}
