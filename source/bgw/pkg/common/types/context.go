package types

import (
	"bgw/pkg/common/constant"
)

// // GetContext get jaeger context from fasthttp context
// func GetContext(ctx context.Context) context.Context {
// 	if c, ok := ctx.(*Ctx); ok {
// 		if val, ok := c.UserValue(constant.ContextKey).(context.Context); ok && val != nil {
// 			return val
// 		}
// 		return context.Background()
// 	}
// 	return ctx
// }

// RequestParsedFromContext get request value from context
func RequestParsedFromContext(ctx *Ctx, key string) interface{} {
	if v := ctx.UserValue(constant.BgwRequestParsed); v != nil {
		if handled, ok := v.(map[string]interface{}); ok {
			if vv, ok := handled[key]; ok {
				return vv
			}
		}
	}
	return nil
}

// ContextWithRequestSave set request key has been parsed
func ContextWithRequestSave(ctx *Ctx, key string, value interface{}) {
	var (
		m  map[string]interface{}
		ok bool
	)
	if v := ctx.UserValue(constant.BgwRequestParsed); v != nil {
		if m, ok = v.(map[string]interface{}); ok {
			m[key] = value
		} else {
			m = map[string]interface{}{
				key: value,
			}
		}
	} else {
		m = map[string]interface{}{
			key: value,
		}
	}
	ctx.SetUserValue(constant.BgwRequestParsed, m)
}
