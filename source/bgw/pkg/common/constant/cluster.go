package constant

// Load Balance Strategies
const (
	LabelEnvAZ = "BSM_SERVICE_AZ"

	SelectorRandom         = "SELECTOR_RANDOM"
	SelectorRoundRobin     = "SELECTOR_ROUND_ROBIN"
	SelectorWhitelist      = "SELECTOR_WHITELIST"
	SelectorZoneRaft       = "SELECTOR_ZONE_RAFT"
	SelectorZoneRoundRobin = "SELECTOR_ZONE_ROUND_ROBIN"
	SelectorGrey           = "SELECTOR_GREY"
	SelectorEtcdPartition  = "SELECTOR_ETCD_PARTITION"
	SelectorRaft           = "SELECTOR_RAFT"
	SelectorZoneVIPRaft    = "SELECTOR_ZONE_VIP_RAFT"
	SymbolsRoundRobin      = "SELECTOR_SYMBOLS_ROUND_ROBIN"
	MetasRoundRobin        = "SELECTOR_METAS_ROUND_RANDOM"
	SelectorConsistentHash = "SELECTOR_CONSISTENT_HASH"
	SelectorMultiRegistry  = "SELECTOR_MULTI_REGISTRY"
)
