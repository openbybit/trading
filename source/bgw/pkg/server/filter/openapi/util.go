package openapi

import (
	"bytes"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/types"
	"bgw/pkg/server/metadata"
	ropenapi "bgw/pkg/service/openapi"
)

func GetMemberID(ctx *types.Ctx, apikey string) (int64, error) {
	if apikey == "" {
		return 0, berror.WithMessage(berror.ErrParams, "only support v3")
	}

	md := metadata.MDFromContext(ctx)
	if md == nil {
		return 0, berror.ErrDefault
	}

	ros, err := ropenapi.GetOpenapiService()
	if err != nil {
		return 0, err
	}

	member, err := ros.GetAPIKey(ctx, apikey, md.Extension.XOriginFrom)
	if err != nil {
		return 0, err
	}

	if member.MemberId == 0 {
		return 0, berror.WithMessage(berror.ErrOpenAPIApiKey, "invalid member id")
	}

	md.UID = member.MemberId
	return member.MemberId, nil
}

// queryParse http query parse, not covert
func queryParse(query []byte) map[string]string {
	params := make(map[string]string, 10)
	ss := bytes.Split(query, []byte("&"))
	for _, s := range ss {
		kv := bytes.SplitN(s, []byte("="), 2)
		if len(kv) != 2 {
			continue
		}
		params[string(kv[0])] = string(kv[1])
	}
	return params
}
