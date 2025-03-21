package bizmetedata

import (
	"context"

	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	gmetadata "bgw/pkg/server/metadata"
	"bgw/pkg/service/user"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"google.golang.org/grpc/metadata"
)

type copyTrade struct {
}

func init() {
	gmetadata.Register(copyTradeKey, &copyTrade{})
}

type CopyTrade = user.CopyTrade

var copyTradeKey = "copytrade"

type copyTradeCtxKey struct{}

type leaderInfo struct {
	// 网关获取当前登录用户下的所有子账号关系（包括登录用户）
	// 通过MemberRelationType 与 MemberTag.CopyTradeLeaderUpgrade == suceess 筛选leader信息
	// 1.入股当前登录的是母账号，带单子账号已经完成升级
	//  此时通过母账号1，获取到带单子账号2的MemberRelations
	IsUpgraded   bool  `json:"is_upgraded"`    // 是否完成信号带单升级
	ParentUserId int64 `json:"parent_user_id"` // 当前leader user id 对应的parent user id
}

// WithCopyTradeMetadata set copyTrade metadata in context
func WithCopyTradeMetadata(ctx context.Context, data *CopyTrade) context.Context {
	if c, ok := ctx.(*types.Ctx); ok {
		c.SetUserValue(copyTradeKey, data)
	} else {
		return context.WithValue(ctx, copyTradeCtxKey{}, data)
	}
	return nil
}

// CopyTradeFromContext extracts the copy trade metadata from the context.
func CopyTradeFromContext(ctx context.Context) *CopyTrade {
	var v interface{}
	if c, ok := ctx.(*types.Ctx); ok {
		v = c.UserValue(copyTradeKey)
	} else {
		v = ctx.Value(copyTradeCtxKey{})
	}
	data, ok := v.(*CopyTrade)
	if !ok {
		return nil
	}
	return data
}

// Extract CopyTrade metadata from context
func (c *copyTrade) Extract(ctx context.Context) metadata.MD {
	data := CopyTradeFromContext(ctx)
	if data == nil {
		return nil
	}

	md := make(metadata.MD, 5)
	if data.LeaderID != 0 {
		md.Set("copytradeleader", cast.Int64toa(data.LeaderID))
	}
	if data.FollowerID != 0 {
		md.Set("copytradefollower", cast.Int64toa(data.FollowerID))
	}
	if data.TargetID != 0 {
		md.Set("copytradetarget", cast.Int64toa(data.TargetID))
	}

	for _, id := range data.LeaderIDs {
		md.Append("ctleaderlist", cast.Int64toa(id))
	}
	for _, id := range data.FollowerIDs {
		md.Append("ctfollowerlist", cast.Int64toa(id))
	}

	if len(data.LeaderIDs) <= 0 {
		return md
	}

	info := map[int64]leaderInfo{
		data.LeaderIDs[0]: {
			IsUpgraded:   data.IsUpgradeLeader,
			ParentUserId: data.ParentID,
		},
	}

	v, err := util.JsonMarshal(info)
	if err != nil {
		glog.Debug(ctx, "copytradeinfo JsonMarshal error", glog.NamedError("err", err), glog.Any("data", data))
	} else {
		md.Set("leaderinfom", string(v))
	}

	return md
}
