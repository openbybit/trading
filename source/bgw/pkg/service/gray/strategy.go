package gray

import (
	"context"
)

const (
	GrayStrategyFullOn    = "full_on"
	GrayStrategyFullClose = "full_close"
	GrayStrategyUid       = "uid"
	GrayStrategyTail      = "uid_tail"
	GrayStrategyService   = "service"
	GrayStrategyPath      = "path"
	GrayStrategyIp        = "ip"
)

var globalStrategies = map[string]struct{}{
	GrayStrategyFullOn:    {},
	GrayStrategyFullClose: {},
}

type Strategies []*strategy

func (s *Strategies) grayCheck(ctx context.Context) (bool, error) {
	if s == nil || len(*s) == 0 {
		return false, nil
	}
	var (
		res          = true
		resErr error = nil
	)

	for _, sg := range *s {
		ok, err := sg.grayCheck(ctx)
		if _, in := globalStrategies[sg.Strags]; in {
			return ok, err
		}

		if !ok || err != nil {
			res = false
			resErr = err
		}
	}

	return res, resErr
}

type strategy struct {
	Strags string `yaml:"strategy"`
	Value  []any  `yaml:"value"`
}

func (s *strategy) grayCheck(ctx context.Context) (bool, error) {
	checker, ok := getChecker(s.Strags)
	if !ok {
		return false, nil
	}
	return checker(ctx, s.Value)
}
