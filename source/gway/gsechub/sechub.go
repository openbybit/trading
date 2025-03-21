package gsechub

import (
	"errors"
	"os"

	"git.bybit.com/codesec/sechub-sdk-go/api"
	"git.bybit.com/codesec/sechub-sdk-go/client"
)

var ErrNilClient = errors.New("sec client is nil")

var secHubCli *api.Sechub

type Config = client.Config

func init() {
	Init(nil)
}

// Init 初始化client, conf可以传nil和空,默认会使用环境变量初始化
func Init(conf *Config) {
	if conf == nil || conf.AppSignKey == "" {
		// 本地没有环境变量配置
		if os.Getenv(api.EnvSechubAppSignKey) == "" {
			return
		}
	}

	secHubCli = api.NewClient(conf)
}

func Decrypt(in string) (string, error) {
	if secHubCli != nil {
		return secHubCli.DecryptData(in)
	}

	return in, ErrNilClient
}
