package future

import futenumsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/futenums/v1"

// IsValidByType 是否为已配置的的合约
func (s *Scmeta) IsValidByType(contractType futenumsv1.ContractType) bool {
	switch contractType {
	case futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL,
		futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES,
		futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL,
		futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES:
		return true
	default:
		return false
	}
}

// IsInverseByType 是否为已配置的反向合约
func (s *Scmeta) IsInverseByType(contractType futenumsv1.ContractType) bool {
	switch contractType {
	case futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL,
		futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES:
		return false
	case futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL,
		futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES:
		return true
	default:
		return false
	}
}

// IsInverseFutureByType 是否为已配置的反向交割合约
func (s *Scmeta) IsInverseFutureByType(contractType futenumsv1.ContractType) bool {
	return contractType == futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES
}

// IsInverseFutureByType 是否为已配置的反向永续合约
func (s *Scmeta) IsInversePerpetualByType(contractType futenumsv1.ContractType) bool {
	return contractType == futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL
}

// IsLinearByType 是否为已配置的正向合约
func (s *Scmeta) IsLinearByType(contractType futenumsv1.ContractType) bool {
	switch contractType {
	case futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES,
		futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL:
		return true
	case futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL,
		futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES:
		return false
	default:
		return false
	}
}
