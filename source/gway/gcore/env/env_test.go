package env

import (
	"os"
	"testing"
)

func TestEnv(t *testing.T) {
	testAppName := "bgw-202301181905-testnet_auth_cookie-168bb4bb"
	testVersioName := "202301181905-testnet_auth_cookie-168bb4bb"
	testProjectName := "bgw"
	testProjectEnvName := "ls-asset"

	os.Setenv("MY_APP_NAME", testAppName)
	os.Setenv("MY_VERSION_NAME", testVersioName)
	os.Setenv("MY_PROJECT_NAME", testProjectName)
	os.Setenv("MY_PROJECT_ENV_NAME", testProjectEnvName)
	doInit()

	if IsProduction() {
		t.Error("should be false")
	}

	if AppName() != testAppName {
		t.Error("invalid app name")
	}

	if VersionName() != testVersioName {
		t.Error("invalid version name")
	}

	if ProjectName() != testProjectName {
		t.Error("invalid project name")
	}

	if ProjectEnvName() != testProjectEnvName {
		t.Error("invalid project env name")
	}

	os.Setenv("MY_ENV_NAME", "testnet")
	doInit()
	if !IsProduction() || EnvName() != "testnet" {
		t.Errorf("should be true, env=%v", EnvName())
	}

	os.Setenv("MY_ENV_NAME", "mainnet")
	doInit()
	if !IsProduction() || EnvName() != "mainnet" {
		t.Errorf("should be true, env=%v", EnvName())
	}
}
