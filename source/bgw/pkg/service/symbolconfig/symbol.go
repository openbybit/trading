package symbolconfig

import (
	"bytes"
	"errors"

	futenumsv1 "code.bydev.io/fbu/future/bufgen.git/pkg/bybit/future/futenums/v1"
	"code.bydev.io/fbu/future/sdk.git/pkg/future"
	"code.bydev.io/fbu/future/sdk.git/pkg/scmeta"
	"code.bydev.io/fbu/gateway/gway.git/gcore/cast"
	"code.bydev.io/fbu/gateway/gway.git/glog"

	"bgw/pkg/common/bhttp"
	"bgw/pkg/common/types"
	"bgw/pkg/common/util"
)

const (
	Symbol = "symbol"
)

// HandleSymbol handle symbol mapping
func HandleSymbol(symbol string, sc scmeta.Module) int32 {
	return int32(sc.SymbolFromCache(symbol))
}

// GetSymbol parse symbol from request, default filed name is symbol
func GetSymbol(c *types.Ctx) string {
	return GetSymbolWithSymbolFiledName(c, Symbol)
}

// GetSymbolWithSymbolFiledName parse symbol from request with specify field
func GetSymbolWithSymbolFiledName(c *types.Ctx, symbolFiledName string) (symbol string) {
	if v := types.RequestParsedFromContext(c, symbolFiledName); v != nil {
		s, ok := v.(string)
		if ok {
			glog.Debug(c, "GetSymbol from ctx", glog.String("meth", cast.UnsafeBytesToString(c.Method())),
				glog.String("path", cast.UnsafeBytesToString(c.Path())), glog.String("symbol", s),
			)
			symbol = s
			return
		}
	}

	defer func() {
		glog.Debug(c, "GetSymbol save ctx", glog.String("meth", cast.UnsafeBytesToString(c.Method())),
			glog.String("path", cast.UnsafeBytesToString(c.Path())),
			glog.String("symbol", symbol),
			glog.String("symbolFiledName", symbolFiledName),
		)
		types.ContextWithRequestSave(c, symbolFiledName, symbol)
	}()

	if !c.IsGet() && !bytes.HasPrefix(c.Request.Header.ContentType(), bhttp.ContentTypePostForm) {
		symbol = util.JsonGetString(c.PostBody(), symbolFiledName)
	} else if c.IsGet() {
		symbol = string(c.QueryArgs().Peek(symbolFiledName))
	} else {
		symbol = string(c.PostArgs().Peek(symbolFiledName))
	}

	return
}

func GetContractType(symbol string, sc scmeta.Module) futenumsv1.ContractType {
	contract, _ := sc.GetContractType(sc.SymbolFromCache(symbol))
	switch contract {
	case futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL.String():
		return futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_PERPETUAL
	case futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL.String():
		return futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_PERPETUAL
	case futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES.String():
		return futenumsv1.ContractType_CONTRACT_TYPE_INVERSE_FUTURES
	case futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES.String():
		return futenumsv1.ContractType_CONTRACT_TYPE_LINEAR_FUTURES
	default:
		return futenumsv1.ContractType_CONTRACT_TYPE_UNSPECIFIED
	}
}

func GetSymbolEnum(symbol string, sc scmeta.Module) future.Symbol {
	return sc.SymbolFromCache(symbol)
}

func IsLinearUsdcSymbol(symbol string, sc scmeta.Module) bool {
	return sc.IsLinearUsdcSymbol(sc.SymbolFromCache(symbol))
}

var errBatchReqInvalidParameter = errors.New("batch request invalid parameter")

// GetBatchSymbol get symbols from aio batch order create request body
func GetBatchSymbol(ctx *types.Ctx) (symbols []string, err error) {
	return GetBatchSymbolByFieldName(ctx, "$.request[:].symbol")
}

// GetBatchSymbolByFieldName get symbols from batch order create request body with specify field
func GetBatchSymbolByFieldName(ctx *types.Ctx, field string) (symbols []string, err error) {
	if v := types.RequestParsedFromContext(ctx, field); v != nil {
		s, ok := v.([]string)
		if ok {
			glog.Debug(ctx, "GetSymbol from ctx", glog.String("method", cast.UnsafeBytesToString(ctx.Method())),
				glog.String("path", cast.UnsafeBytesToString(ctx.Path())), glog.Any("symbols", s), glog.Any("field", field),
			)
			symbols = s
			return
		}
	}

	defer func() {
		glog.Debug(ctx, "GetSymbol save ctx", glog.String("method", cast.UnsafeBytesToString(ctx.Method())),
			glog.String("path", cast.UnsafeBytesToString(ctx.Path())), glog.Any("symbols", symbols), glog.Any("field", field),
		)
		if err == nil {
			types.ContextWithRequestSave(ctx, field, symbols)
		}
	}()

	l, _ := util.JsonpathGet(ctx.Request.Body(), field)
	items, ok := l.([]interface{})
	if !ok {
		err = errBatchReqInvalidParameter
		return
	}
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			err = errBatchReqInvalidParameter
			return
		}
		symbols = append(symbols, s)
	}
	return
}
