package config

import (
	"strings"
	"time"
)

const DefaultTimeOut = 2 * time.Second

type RemoteConfig struct {
	Protocol string              `json:"protocol,omitempty,optional"`
	Address  string              `json:"address,omitempty,optional"`
	Timeout  string              `json:"timeout,omitempty,optional"`
	Username string              `json:"username,omitempty,optional"`
	Password string              `json:"password,omitempty,optional"`
	Options  map[string][]string `json:"options,omitempty,optional"`
}

// GetTimeout return timeout duration.
// if the configure is invalid, or missing, the default value 5s will be returned
func (rc *RemoteConfig) GetTimeout(def time.Duration) time.Duration {
	if res, err := time.ParseDuration(rc.Timeout); err == nil {
		return res
	}

	// check if the default value is valid
	if def > 0 {
		return def
	}
	return DefaultTimeOut
}

// GetOptions will return the value of the key. If not found, def will be return;
// def => default value
func (rc *RemoteConfig) GetOptions(key string, def string) string {
	param, ok := rc.Options[key]
	if !ok {
		return def
	}
	return strings.Join(param, ",")
}

func (rc *RemoteConfig) GetArrayOptions(key string, def []string) []string {
	param, ok := rc.Options[key]
	if !ok {
		return def
	}
	return param
}

// GetAddresses get addresses
func (rc *RemoteConfig) GetAddresses() []string {
	return strings.Split(rc.Address, ",")
}

// GetAddress get address
func (rc *RemoteConfig) GetAddress() string {
	return rc.Address
}
