package registry

import (
	"regexp"
	"strconv"
	"strings"

	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
)

type RaftRole string
type SwimLane string

const (
	RoleTypeLeader   RaftRole = "leader"
	RoleTypeFollower RaftRole = "follower"

	ZoneKey        = "zone"
	TermKey        = "term"
	WeightKey      = "weight"
	RoleKey        = "role"
	VIPZonePrefix  = "RZVIP"
	GrayZonePrefix = "RZGRAY"

	SwimEnv              = "lane-env"
	EnvLane     SwimLane = "env"
	BaseLane    SwimLane = "base"
	DefaultLane SwimLane = ""

	AZName    = "az"
	CloudName = "cloud_name"
	Symbols   = "symbols"
)

func (r RaftRole) IsLeader() bool {
	return r == RoleTypeLeader
}

type Metadata map[string]string

func (md Metadata) GetWeight() int64 {
	t, err := strconv.Atoi(md[WeightKey])
	if err == nil {
		return int64(t)
	}
	return -1
}

func (md Metadata) GetPartition() int {
	zone := md[ZoneKey]
	if zone == "" {
		return -1
	}
	zone = strings.TrimSpace(zone)
	if strings.HasPrefix(zone, VIPZonePrefix) || strings.HasPrefix(zone, GrayZonePrefix) {
		// skip vip and gray zone
		return -1
	}

	if strings.Contains(zone, "_") {
		a := strings.SplitN(zone, "_", 2)
		if len(a) == 2 {
			t, err := strconv.Atoi(a[1])
			if err == nil {
				return t
			}
			return -1
		}
	}

	re := regexp.MustCompile("[0-9]+$")
	match := re.FindString(zone)
	if match == "" {
		return -1
	}

	return cast.Atoi(match)
}

func (md Metadata) GetZoneName() string {
	return strings.TrimSpace(md[ZoneKey])
}

func (md Metadata) GetTerm() int {
	return cast.Atoi(md[TermKey])
}

func (md Metadata) GetRole() RaftRole {
	switch md[RoleKey] {
	case "leader":
		return RoleTypeLeader
	case "follower":
		return RoleTypeFollower
	default:
		return RoleTypeFollower
	}
}

func (md Metadata) GetSwimLane(env string) SwimLane {
	l := md[SwimEnv]
	if l == "" {
		return DefaultLane
	}
	switch l {
	case env:
		return EnvLane
	case string(BaseLane):
		return BaseLane
	default:
		return DefaultLane
	}
}

func (s SwimLane) IsEnvLane() bool {
	return s == EnvLane
}

func (s SwimLane) IsBaseLane() bool {
	return s == BaseLane
}

func (md Metadata) GetAZName() string {
	return strings.TrimSpace(md[AZName])
}
func (md Metadata) GetCloudName() string {
	return strings.TrimSpace(md[CloudName])
}

func (md Metadata) GetSymbolsName() string {
	return "," + strings.ToLower(strings.TrimSpace(md[Symbols])) + ","
}

func (md Metadata) GetDynamicName(key string) string {
	return strings.TrimSpace(md[key])
}
