package auth

import (
	"fmt"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
	"bgw/pkg/config"
	"bgw/pkg/server/metadata"
	"bgw/pkg/service"
	"bgw/pkg/service/masque"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

func GetToken(c *types.Ctx) string {
	secureToken := config.GetSecureTokenKey()
	token := string(c.Request.Header.Cookie(secureToken))
	if token != "" {
		glog.Debug(c, "get secure-token failed, user token from header", glog.String("secure-key", secureToken))
		return token
	}

	return util.DecodeHeaderValue(c.Request.Header.Peek(userToken))
}

func GetMemberID(c *types.Ctx, token string) (int64, bool, error) {
	if token == "" {
		return 0, false, berror.ErrAuthVerifyFailed
	}

	md := metadata.MDFromContext(c)

	svc, err := masque.GetMasqueService()
	if err != nil {
		return 0, false, err
	}

	pc := metadata.MDFromContext(c).GetPlatform()
	originUrl := md.Extension.Referer + md.Path

	rsp, err := svc.MasqueTokenInvoke(service.GetContext(c), pc, token, originUrl, masque.WeakAuth)
	if err != nil {
		return 0, false, err
	}
	if rsp.Error != nil {
		return 0, false, fmt.Errorf("masq error")
	}

	md.UID = rsp.UserId
	if pu, ok := rsp.ExtInfo[parentUID]; ok {
		md.ParentUID = pu
	}
	if t, ok := rsp.ExtInfo[subMemberTypeKey]; ok {
		md.MemberRelation = t
		md.IsDemoUID = t == demoSubMember && md.ParentUID != ""
	}
	return rsp.UserId, md.IsDemoUID, nil
}
