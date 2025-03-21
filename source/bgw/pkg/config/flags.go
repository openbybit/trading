package config

// 启动参数
var Flags struct {
	ShardIndex int
}

func GetShardPort(port int) int {
	if Flags.ShardIndex >= 0 {
		return port + Flags.ShardIndex
	}

	return port
}
