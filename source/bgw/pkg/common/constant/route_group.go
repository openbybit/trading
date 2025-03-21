package constant

const (
	ResourceGroupInvalid               = "RESOURCE_GROUP_UNSPECIFIED"
	ResourceGroupAll                   = "RESOURCE_GROUP_ALL"
	ResourceGroupOrder                 = "RESOURCE_GROUP_ORDER"
	ResourceGroupPosition              = "RESOURCE_GROUP_POSITION"
	ResourceGroupSpotTrade             = "RESOURCE_GROUP_SPOT_TRADE"
	ResourceGroupCloudContract         = "RESOURCE_GROUP_CLOUD_CONTRACT"
	ResourceGroupBlockTrade            = "RESOURCE_GROUP_BLOCK_TRADE"
	ResourceGroupOptionsTrade          = "RESOURCE_GROUP_OPTIONS_TRADE"
	ResourceGroupDerivativesTrade      = "RESOURCE_GROUP_DERIVATIVES_TRADE"
	ResourceGroupCopyTrade             = "RESOURCE_GROUP_COPY_TRADE"
	ResourceGroupExchangeHistory       = "RESOURCE_GROUP_EXCHANGE_HISTORY"
	ResourceGroupNFTQueryProductList   = "RESOURCE_GROUP_NFT_QUERY_PRODUCT_LIST"
	ResourceGroupAccountTransfer       = "RESOURCE_GROUP_ACCOUNT_TRANSFER"
	ResourceGroupSubMemberTransfer     = "RESOURCE_GROUP_SUB_MEMBER_TRANSFER"
	ResourceGroupWithdraw              = "RESOURCE_GROUP_WITHDRAW"
	ResourceGroupSubMemberTransferList = "RESOURCE_GROUP_SUB_MEMBER_TRANSFER_LIST"
	ResourceGroupAffiliate             = "RESOURCE_GROUP_AFFILIATE"
)

const (
	Order                 = "Order"
	Position              = "Position"
	SpotTrade             = "SpotTrade"
	OptionsTrade          = "OptionsTrade"
	CloudContract         = "CloudContract"
	BlockTrade            = "BlockTrade"
	DerivativesTrade      = "DerivativesTrade"
	CopyTrade             = "CopyTrading"
	ExchangeHistory       = "ExchangeHistory"
	NFTQueryProductList   = "NFTQueryProductList"
	AccountTransfer       = "AccountTransfer"
	SubMemberTransfer     = "SubMemberTransfer"
	Withdraw              = "Withdraw"
	SubMemberTransferList = "SubMemberTransferList"
	Affiliate             = "Affiliate"
	All                   = "All"
)

const (
	PermissionInvalid   = "PERMISSION_UNSPECIFIED"
	PermissionRead      = "PERMISSION_READ"
	PermissionWrite     = "PERMISSION_WRITE"
	PermissionReadWrite = "PERMISSION_READ_WRITE"
)

func GetRouteGroup(group string) string {
	switch group {
	case All:
		return ResourceGroupAll
	case Order:
		return ResourceGroupOrder
	case Position:
		return ResourceGroupPosition
	case SpotTrade:
		return ResourceGroupSpotTrade
	case BlockTrade:
		return ResourceGroupBlockTrade
	case CloudContract:
		return ResourceGroupCloudContract
	case OptionsTrade:
		return ResourceGroupOptionsTrade
	case DerivativesTrade:
		return ResourceGroupDerivativesTrade
	case CopyTrade:
		return ResourceGroupCopyTrade
	case ExchangeHistory:
		return ResourceGroupExchangeHistory
	case NFTQueryProductList:
		return ResourceGroupNFTQueryProductList
	case AccountTransfer:
		return ResourceGroupAccountTransfer
	case SubMemberTransfer:
		return ResourceGroupSubMemberTransfer
	case Withdraw:
		return ResourceGroupWithdraw
	case SubMemberTransferList:
		return ResourceGroupSubMemberTransferList
	case Affiliate:
		return ResourceGroupAffiliate
	}
	return ResourceGroupInvalid
}
