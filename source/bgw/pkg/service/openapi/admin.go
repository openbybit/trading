package openapi

import (
	"context"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"google.golang.org/protobuf/proto"
)

func registerAdmin() {
	if defaultOpenapiService == nil {
		return
	}
	// curl 'http://localhost:6480/admin?cmd=queryAPIkeyCache&params={{apikey}}'
	gapp.RegisterAdmin("queryAPIkeyCache", "query APIkey in cache", OnQueryAPIkeyCache)
	// curl 'http://localhost:6480/admin?cmd=deleteAPIkeyCache&params={{apikey}}'
	gapp.RegisterAdmin("deleteAPIkeyCache", "delete APIkey in cache", OnDeleteAPIkeyCache)
	// curl 'http://localhost:6480/admin?cmd=queryAPIkeys&apikey={{apikey}}&xoriginfrom={{xOriginFrom}}'
	gapp.RegisterAdmin("queryAPIkey", "query APIkey", OnQueryAPIkey)
}

func OnQueryAPIkeyCache(args gapp.AdminArgs) (interface{}, error) {
	apikey := args.GetStringAt(0)
	data, err := defaultOpenapiService.(*openapiService).cache.Get([]byte(apikey))
	if err == nil {
		msg := &user.MemberLogin{}
		err = proto.Unmarshal(data, msg)
		if err != nil {
			return nil, err
		}
		return msg, nil
	}

	return "empty apikey", nil
}

func OnDeleteAPIkeyCache(args gapp.AdminArgs) (interface{}, error) {
	apikey := args.GetStringAt(0)
	defaultOpenapiService.(*openapiService).cache.Del([]byte(apikey))
	return "success", nil
}

func OnQueryAPIkey(args gapp.AdminArgs) (interface{}, error) {
	apikey := args.GetStringBy("apikey")
	xoriginfrom := args.GetStringBy("xoriginfrom")
	return defaultOpenapiService.GetAPIKey(context.Background(), apikey, xoriginfrom)
}
