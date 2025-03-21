package user

import (
	"context"
	"fmt"

	"code.bydev.io/fbu/gateway/gway.git/gapp"
)

func registerAccountAdmin() {
	if accountService == nil {
		return
	}
	// http://localhost:6480/admin?cmd=queryMembertag&uid={{uid}}&tag={{tag}}
	gapp.RegisterAdmin("queryMembertag", "query member tag", OnQueryMembertag)
	// http://localhost:6480/admin?cmd=setMembertag&tag={{tag}}&val={{val}}
	gapp.RegisterAdmin("setMembertag", "set member tag", OnSetMembertag)
	// http://localhost:6480/admin?cmd=queryMembertagCache&uid={{uid}}&tag={{tag}}
	gapp.RegisterAdmin("queryMembertagCache", "query member tag in cache", OnQueryMembertagCache)
	// http://localhost:6480/admin?cmd=deleteMembertagCache&uid={{uid}}&tag={{tag}}
	gapp.RegisterAdmin("deleteMembertagCache", "delete member tag in cache", OnDeleteMembertagCache)
	// http://localhost:6480/admin?cmd=queryAccountID&uid={{uid}}&accountType={{accountType}}&bizType={{bizType}}
	gapp.RegisterAdmin("queryAccountID", "query accountid in cache", OnQueryAccountID)
	// http://localhost:6480/admin?cmd=deleteAccountIDCache={{uid}}&accountType={{accountType}}&bizType={{bizType}}
	gapp.RegisterAdmin("deleteAccountIDCache", "delete accountid in cache", OnDeleteAccountID)
}

func OnQueryMembertag(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64By("uid")
	tag := args.GetStringBy("tag")
	return accountService.QueryMemberTag(context.Background(), uid, tag)
}

func OnSetMembertag(args gapp.AdminArgs) (interface{}, error) {
	tag := args.GetStringBy("tag")
	val := args.GetStringBy("val")
	err := accountService.accountCache.Set([]byte(tag), []byte(val), 1000)
	return nil, err
}

func OnQueryMembertagCache(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64By("uid")
	tag := args.GetStringBy("tag")
	key := fmt.Sprintf("%dmember_tag_%s", uid, tag)
	val, err := accountService.accountCache.Get([]byte(key))
	if err == nil {
		return string(val), nil
	}
	return "empty tag", nil
}

func OnDeleteMembertagCache(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64By("uid")
	tag := args.GetStringBy("tag")
	key := fmt.Sprintf("%dmember_tag_%s", uid, tag)
	accountService.accountCache.Del([]byte(key))
	return "success", nil
}

func OnQueryAccountID(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64By("uid")
	accountType := args.GetStringBy("accountType")
	bizType := args.GetInt64By("bizType")
	key := fmt.Sprintf("%d%s%d", uid, accountType, bizType)
	return accountService.accountCache.Get([]byte(key))
}

func OnDeleteAccountID(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64By("uid")
	accountType := args.GetStringBy("accountType")
	bizType := args.GetInt64By("bizType")
	key := fmt.Sprintf("%d%s%d", uid, accountType, bizType)
	accountService.accountCache.Del([]byte(key))
	return "success", nil
}

func registerCopytradeAdmin() {
	if copyTradeService == nil {
		return
	}
	// http://localhost:6480/admin?cmd=queryCopytrade&uid=1673933
	gapp.RegisterAdmin("queryCopytrade", "query member tag", OnQueryCopytradeData)
	// http://localhost:6480/admin?cmd=deleteCopytrade&uid={{uid}}
	gapp.RegisterAdmin("deleteCopytrade", "delete member tag in cache", OnDeleteCopytradeData)
}

func OnQueryCopytradeData(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64By("uid")
	res, _ := copyTradeService.getCopyTradeData(uid)
	return res, nil
}

func OnDeleteCopytradeData(args gapp.AdminArgs) (interface{}, error) {
	uid := args.GetInt64By("uid")
	copyTradeService.DeleteCopyTradeData(uid)
	return "success", nil
}
