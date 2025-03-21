package env

import "os"

type EnvType = uint8

const (
	EnvTypeUnknown EnvType = 0
	EnvTypeDev     EnvType = 0x01 // 开发环境
	EnvTypeSitnet  EnvType = 0x02 // 测试环境,qa/sitnet
	EnvTypeTestnet EnvType = 0x04 // 预发环境
	EnvTypeMainnet EnvType = 0x08 // 线上环境
)

var (
	envType        EnvType = 0  // 环境名映射到的枚举值
	envName                = "" // 环境名, 例如 MY_ENV_NAME=dev,testnet,mainnet
	appName                = "" // 应用名, 例如 MY_APP_NAME=bgw-202301181905-testnet_auth_cookie-168bb4bb
	versionName            = "" // 应用版本: 例如 MY_VERSION_NAME=202301181905-testnet_auth_cookie-168bb4bb
	projectName            = "" // 应用名: 例如 MY_PROJECT_NAME=bgw
	projectEnvName         = "" // 应用流水线名: 例如 MY_PROJECT_ENV_NAME=ls-asset

	availableZone    = "" // 可用区: BSM_SERVICE_AZ
	availableZoneID  = "" // 可用区: MY_AVAILABLE_ZONE_ID
	cloudProvider    = "" // 云厂商: MY_CLOUD_PROVIDER
	multiCloudSwitch = ""
	serviceName      = ""
)

func init() {
	doInit()
}

func doInit() {
	// 线上有两套环境命名规范,目前主推MY_{XXX}
	envName = os.Getenv("MY_ENV_NAME")
	appName = os.Getenv("MY_APP_NAME")
	versionName = os.Getenv("MY_VERSION_NAME")
	projectName = os.Getenv("MY_PROJECT_NAME")
	projectEnvName = os.Getenv("MY_PROJECT_ENV_NAME")

	availableZone = os.Getenv("BSM_SERVICE_AZ")
	availableZoneID = os.Getenv("MY_AVAILABLE_ZONE_ID")
	cloudProvider = os.Getenv("MY_CLOUD_PROVIDER")
	multiCloudSwitch = os.Getenv("MY_MULTI_CLOUD_SWITCH")
	serviceName = os.Getenv("BSM_SERVICE_NAME")

	// 使用BSM_SERVICE_{XXX}兜底兼容
	if envName == "" {
		envName = os.Getenv("BSM_SERVICE_STAGE")
	}

	// 统一命名qa,sitnet
	if envName == "qa" {
		envName = "sitnet"
	}

	envType = parseEnvType(envName)
}

func parseEnvType(name string) EnvType {
	switch name {
	case "dev":
		return EnvTypeDev
	case "qa", "sitnet":
		return EnvTypeSitnet
	case "testnet":
		return EnvTypeTestnet
	case "mainnet":
		return EnvTypeMainnet
	default:
		return EnvTypeUnknown
	}
}

// IsDev is dev
func IsDev() bool {
	return envType == EnvTypeDev
}

// IsSitnet is sitnet
func IsSitnet() bool {
	return envType == EnvTypeSitnet
}

// IsTestnet is testnet
func IsTestnet() bool {
	return envType == EnvTypeTestnet
}

// IsMainnet is mainnet
func IsMainnet() bool {
	return envType == EnvTypeMainnet
}

// IsProduction testnet和mainnet都认为是生产环境
func IsProduction() bool {
	return envType == EnvTypeTestnet || envType == EnvTypeMainnet
}

// EnvName env name
func EnvName() string {
	return envName
}

// SetEnvName set env name
func SetEnvName(x string) {
	envName = x
	envType = parseEnvType(x)
}

// AppName app name
func AppName() string {
	return appName
}

// SetAppName set app name
func SetAppName(x string) {
	appName = x
}

// VersionName version name
func VersionName() string {
	return versionName
}

// SetVersionName set version name
func SetVersionName(x string) {
	versionName = x
}

// ProjectName project name
func ProjectName() string {
	return projectName
}

// SetProjectName set project name
func SetProjectName(x string) {
	projectName = x
}

// ProjectEnvName project env name
func ProjectEnvName() string {
	return projectEnvName
}

// SetProjectEnvName 设置projectEnvName,常用于测试
func SetProjectEnvName(x string) {
	projectEnvName = x
}

// AvailableZone BSM_SERVICE_AZ
func AvailableZone() string {
	return availableZone
}

// SetAvailableZone set available zone
func SetAvailableZone(x string) {
	availableZone = x
}

// AvailableZoneID MY_AVAILABLE_ZONE_ID
func AvailableZoneID() string {
	return availableZoneID
}

// CloudProvider MY_CLOUD_PROVIDER
func CloudProvider() string {
	return cloudProvider
}

func IsSupportCloud() bool {
	if multiCloudSwitch == "off" {
		return false
	}
	return len(cloudProvider) > 0
}

func ServiceName() string {
	return serviceName
}
