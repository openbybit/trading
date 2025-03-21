package constant

import "fmt"

const (
	Version            = "2.3.2"
	Name               = "BGW"
	RootPath           = "/BGW"                  // version controll root path
	RootDataPath       = "/BGW_data"             // biz metadata root path
	LimiterQuota       = "quota"                 // for redis_limiter of quota
	LimiterQuotaV2     = "quota_v2"              // for redis_limiter_v2 of quota
	OpenapiIpWhiteList = "api_key_white_list_ip" // for openapi ip white list
	ZoneIDWhiteList    = "zone_ids"              // for option zone id white list, vip and gray
	GWSource           = "bgw"
	BGWG               = "BGWG"
)

func GetAppName() string {
	return fmt.Sprintf("%s(%s)", Name, Version)
}
