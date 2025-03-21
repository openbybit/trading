package compliance

import (
	"context"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
)

// curl 'http://localhost:6480/admin?cmd=complianceGetUser&params=33433753'
// curl 'http://localhost:6480/admin?cmd=complianceGetStrategy&params={{strategy}}'
// curl 'http://localhost:6480/admin?cmd=complianceDelStrategy&params={{uid}}'
// curl 'http://localhost:6480/admin?cmd=complianceGetSiteConfig'
func registerAdmin() {
	if gw == nil {
		return
	}
	gapp.RegisterAdmin("complianceGetUser", "get user info", OnGetUserInfo)
	gapp.RegisterAdmin("complianceGetStrategy", "get strategy", OnGetStrategy)
	gapp.RegisterAdmin("complianceDelStrategy", "del user info", OnRemoveUserInfo)
	gapp.RegisterAdmin("complianceGetSiteConfig", "get site config", OnQuerySiteConfig)
}

func OnGetUserInfo(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64At(0)
	ui, err := gw.GetUserInfo(context.Background(), uid)
	if err != nil {
		return nil, err
	}
	return ui, nil
}

func OnGetStrategy(args gapp.AdminArgs) (interface{}, error) {
	strategy := args.GetStringAt(0)
	cfg := gw.GetStrategy(context.Background(), strategy)
	return cfg, nil
}

func OnRemoveUserInfo(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64At(0)
	gw.RemoveUserInfo(context.Background(), uid)
	return "success", nil
}

func OnQuerySiteConfig(args gapp.AdminArgs) (interface{}, error) {
	cfg := gw.QuerySiteConfig(context.Background())
	return cfg, nil
}
