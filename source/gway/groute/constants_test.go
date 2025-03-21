package groute

import "testing"

func TestConstants(t *testing.T) {
	list := []string{"normal", "unified_margin", "unified_trading", "unified", "unknown"}
	for _, v := range list {
		t.Log(ToAccountType(v))
	}

	routeTypeList := []string{"default", "category", "account_type", "all_in_one", "unknown"}
	for _, v := range routeTypeList {
		t.Log(ParseRouteType(v))
	}

	t.Log(ROUTE_TYPE_DEFAULT.MarshalJSON())
}
