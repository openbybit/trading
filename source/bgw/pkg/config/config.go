package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"bgw/pkg/common/constant"

	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
)

var (
	Global globalConfig
)

type globalConfig struct {
	AppConfig
	MiddlewareConfig
	ComponentConfig
}

func init() {
	// initUnitTestConfig for ut
	initUnitTestConfig()
	initAppCfg()
	initMiddlewareCfg()
	initComponentCfg()
}

func AppCfg() *App {
	return &Global.App
}

const secureTokenPrefix = "secure-token"

func GetWingAddr() string {
	port := Global.App.BatWing
	shardPort := GetShardPort(port)
	return fmt.Sprintf(":%d", shardPort)
}

// GetNamespace get namespace
func GetNamespace() string {
	ns := os.Getenv(constant.BGWNamespace)
	if ns != "" {
		return ns
	}
	if ns := Global.App.Namespace; ns != "" {
		return ns
	}

	return constant.DEFAULT_NAMESPACE
}

// GetRegistryNamespace get registry namespace
func GetRegistryNamespace() string {
	ns := GetNamespace()
	if env.IsProduction() {
		ns = constant.DEFAULT_NAMESPACE
	}
	return ns
}

// GetGroup get group
func GetGroup() string {
	if gp := Global.App.Group; gp != "" {
		return gp
	}

	return constant.BGW_GROUP
}

// GetSecureTokenKey get secret token key
//
//	qa&sitnet: secure-token-{project_env}
//	testnet:   secure-token-testnet
//	mainnet:   secure-token
func GetSecureTokenKey() (st string) {
	st = secureTokenPrefix
	if !env.IsProduction() {
		pe := env.ProjectEnvName()
		if pe != "" {
			st += "-" + pe
		}
	} else if env.IsTestnet() {
		st += "-testnet"
	}
	return
}

func initUnitTestConfig() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Println("getwd err:", err)
		return
	}
	targetString := "pkg"
	index := strings.Index(currentPath, targetString)
	if index == -1 {
		log.Println("未找到目标字符串:", currentPath)
		return
	}
	appCfgFile = currentPath[:index] + appCfgFile
	log.Println("appCfgFile:", appCfgFile)

	midCfgFile = currentPath[:index] + midCfgFile
	log.Println("midCfgFile:", midCfgFile)

	compCfgFile = currentPath[:index] + compCfgFile
	log.Println("compCfgFile:", compCfgFile)
}
