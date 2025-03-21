package ban

import (
	"context"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
	jsoniter "github.com/json-iterator/go"
	"google.golang.org/protobuf/proto"
)

func registerAdmin() {
	if defaultBanService == nil {
		return
	}
	// curl 'http://localhost:6480/admin?cmd=queryBanStatusCache&params={{uid}}'
	gapp.RegisterAdmin("queryBanStatusCache", "query ban status in cache", OnQueryBanCache)
	// curl 'http://localhost:6480/admin?cmd=deleteBanStatusCache&params={{uid}}'
	gapp.RegisterAdmin("deleteBanStatusCache", "delete ban status in cache", OnDeleteBanCache)
	// curl 'http://localhost:6480/admin?cmd=queryBanStatus&params={{uid}}'
	gapp.RegisterAdmin("queryBanStatus", "query ban status", OnQueryBan)
}

func OnQueryBanCache(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64At(0)
	key := fmt.Sprintf("%d_banned_info", uid)
	byteUserStatus, err := defaultBanService.(*banService).cache.Get([]byte(key))
	if err == nil {
		cachedMsg := &userStatusInternal{}
		if err = jsoniter.Unmarshal(byteUserStatus, cachedMsg); cachedMsg.UserState == nil || err != nil {
			return nil, err
		}
		us := &UserStatus{}
		err = proto.Unmarshal(cachedMsg.UserState, us)
		if err != nil {
			return nil, err
		}
		return UserStatusWrap{
			LoginStatus:    cachedMsg.LoginStatus,
			WithdrawStatus: cachedMsg.WithdrawStatus,
			TradeStatus:    cachedMsg.TradeStatus,
			LoginBanType:   cachedMsg.LoginBanType,
			UserState:      us,
		}, nil
	}

	return "empty ban status", nil
}

func OnDeleteBanCache(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64At(0)
	key := fmt.Sprintf("%d_banned_info", uid)
	defaultBanService.(*banService).cache.Del([]byte(key))
	return "success", nil
}

func OnQueryBan(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64At(0)
	return defaultBanService.GetMemberStatus(context.Background(), uid)
}
