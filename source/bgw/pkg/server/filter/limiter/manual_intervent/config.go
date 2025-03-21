package manual_intervent

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"bgw/pkg/server/filter/biz_limiter/rate"
	"code.bydev.io/fbu/gateway/gway.git/gcore/container"
	"code.bydev.io/fbu/gateway/gway.git/gcore/env"
	"code.bydev.io/fbu/gateway/gway.git/glog"
	"golang.org/x/net/idna"
)

// define rule type
const (
	clientIpRuleType    = "clientIp"
	requestUrlRule      = "requestUrl"
	requestHostRuleType = "requestHost"
	clientOpFromRule    = "clientOpFrom"

	timeLayOut = "2006-01-02 15:04:05"
)

var (
	maxOpFromLength = 32

	allowedMethods    = []string{"GET", "POST", "PUT", "DELETE", "HEAD", "PATCH"}
	allowedMethodsSet = container.NewSet()

	ruleTypeMap = map[string]int{
		clientIpRuleType:    1,
		requestHostRuleType: 2,
		clientOpFromRule:    3,
		requestUrlRule:      4,
	}
)

func init() {
	for _, v := range allowedMethods {
		allowedMethodsSet.Add(v)
	}
}

type limiter struct {
	startInUTC time.Time
	endInUTC   time.Time
	ruleType   string
	limit      *rate.Limiter
}

type config struct {
	Enable bool    `json:"enable"` // Enable indicates whether to effect
	Rules  []*rule `json:"rules,omitempty"`
}

type rule struct {
	Enable           bool            `json:"enable"`
	RuleType         string          `json:"ruleType"`         // rule type,defines clientIp|requestHost|clientOpFrom|requestUrl
	EffectiveEnvName string          `json:"effectiveEnvName"` // EffectiveEnvName comes from environment variables ${MY_PROJECT_ENV_NAME}, it can be split by "," and will affect on multiple env
	EffectPeriod     *StandardPeriod `json:"effectPeriod,omitempty"`
	ExtData          []*extData      `json:"extData,omitempty"`
	Limit            int             `json:"limit"`
}

type extData struct {
	ClientIp     string     `json:"clientIp"`
	RequestHost  string     `json:"requestHost"`
	ClientOpFrom string     `json:"clientOpFrom"`
	RequestUrl   requestUrl `json:"requestUrl"`
}

type requestUrl struct {
	Path       string `json:"path"`
	HttpMethod string `json:"httpMethod"` // "*" means all methods
	Limit      int    `json:"limit"`
}

// StandardPeriod is a struct that defines for what's the period of manual_intervention_config.
// It'll be invalid if startDate or endDate isn't YYYY-MM-DD hh:mm:ss format
type StandardPeriod struct {
	StartDateInUTC string `json:"startDateInUTC"` // StartDateInUTC in YYYY-MM-DD hh:mm:ss format,in order to understand easily
	EndDateInUTC   string `json:"endDateInUTC"`   // EndDateInUTC in YYYY-MM-DD hh:mm:ss format,in order to understand easily
}

func (config *config) validConfig() error {
	if config == nil {
		return errors.New("config can't affect")
	}

	for index, rule := range config.Rules {
		if !rule.Enable {
			glog.Warn(context.Background(), "[global interveneLimiter]config rule can't affect",
				glog.Int("index", index), glog.Any("rule", rule))
		}

		//check if ruleType valid
		if _, exist := ruleTypeMap[rule.RuleType]; !exist {
			return errors.New(fmt.Sprintf("ruleType:%s is invalid,index:%d,", rule.RuleType, index))
		}

		// check if effectPeriod valid
		if err := rule.EffectPeriod.isValid(); err != nil {
			return fmt.Errorf("effectPeriodInUtc is invalid, index: %d, error: %w", index, err)
		}

		if err := rule.checkExtData(); err != nil {
			return fmt.Errorf("extData is invalid, index: %d, error: %w", index, err)
		}

		// check if limit valid
		if rule.Limit < 0 {
			return errors.New(fmt.Sprintf("limit is negative,index:%d", index))
		}
	}

	return nil
}

func (r *rule) checkExtData() error {
	if r.ExtData == nil || len(r.ExtData) == 0 {
		return errors.New("extData is empty")
	}

	switch r.RuleType {
	case clientIpRuleType:
		for _, ext := range r.ExtData {
			if ip := net.ParseIP(ext.ClientIp); ip == nil {
				return errors.New("ip is invalid")
			}
			//if cidr.ParseNoError(ext.ClientIp) == nil {
			//	return errors.New("ip is invalid")
			//}
		}
		return nil
	case requestHostRuleType:
		for _, ext := range r.ExtData {
			if _, err := idna.Lookup.ToASCII(ext.RequestHost); err != nil {
				return errors.New("host is invalid")
			}
		}
		return nil
	case clientOpFromRule:
		for _, ext := range r.ExtData {
			if ext.ClientOpFrom == "" || len(ext.ClientOpFrom) > maxOpFromLength {
				return errors.New("opFrom is invalid")
			}
		}

		return nil
	case requestUrlRule:

		for _, ext := range r.ExtData {

			if !allowedMethodsSet.Contains(ext.RequestUrl.HttpMethod) && ext.RequestUrl.HttpMethod != "*" {
				return errors.New("httpMethod is invalid")
			}

			if ext.RequestUrl.Limit < 0 {
				return errors.New("limit is invalid")
			}

			if _, err := url.Parse(ext.RequestUrl.Path); err != nil {
				return err
			}
		}
		return nil
	default:
		return errors.New("ruleType is invalid")
	}
}

func (r *rule) matchEffectiveEnvName() bool {
	envNames := strings.Split(r.EffectiveEnvName, ",")

	for _, envName := range envNames {
		if env.ProjectEnvName() == envName {
			return true
		}
	}
	return false
}

func (period *StandardPeriod) isValid() error {

	_, err := time.Parse(timeLayOut, period.StartDateInUTC)
	if err != nil {
		return errors.New("startDate is invalid")
	}

	_, err = time.Parse(timeLayOut, period.EndDateInUTC)
	if err != nil {
		return errors.New("endDate is invalid")
	}

	return nil
}

func (period *StandardPeriod) inPeriod() bool {

	startTimeInUTC, err := time.Parse(timeLayOut, period.StartDateInUTC)
	if err != nil {
		return false
	}

	endTimeInUTC, err := time.Parse(timeLayOut, period.EndDateInUTC)
	if err != nil {
		return false
	}

	return startTimeInUTC.Before(endTimeInUTC) && endTimeInUTC.After(time.Now().UTC())
}
