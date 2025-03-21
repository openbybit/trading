package response

import (
	"bytes"

	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	gmetadata "bgw/pkg/server/metadata"

	"code.bydev.io/fbu/gateway/gway.git/glog"
)

type translate struct {
	msgSource msgSourceType // translate message source
	msgTag    int64         // translate message tag, if msgSource is not 0, then msgTag must be set
}

func (t *translate) do(ctx *types.Ctx, source metadataCarrier, target Target, batch bool) {
	var ar apiResponse

	ar = newSingleAPIResponseInfo()
	ar.SetCode(target.GetCode())
	ar.SetMsg(target.GetMessage())
	ar.SetExtInfo(target.GetExtInfo())

	if source != nil {
		gmd := source.Metadata()
		if _, ok := gmd[constant.BgwAPIResponseCodes]; ok {
			ar = getCodeFromCtx(ctx, gmd, batch)
		}
	}

	noTranslate := func() {
		target.SetCode(ar.GetCode())
		target.SetMessage(ar.GetMsg())

		if batch {
			if a, ok := ar.(*batchAPIResponseInfo); ok {
				target.SetExtInfo(a.GetBatchExtInfo())
			}
		} else {
			target.SetExtInfo(ar.GetExtInfo())
		}

		if ext := ar.GetExtMap(); len(ext) > 0 && !bytes.Equal(ext, emptyJSON) {
			target.SetExtMap(ar.GetExtMap())
		}
	}

	// not translate
	if t.msgSource == MsgSource_Unknown {
		noTranslate()
		return
	}

	// translate
	md := gmetadata.MDFromContext(ctx)
	app := md.Route.GetAppName(ctx)
	codeLoader, ok := codeLoader(app)
	if !ok {
		noTranslate() // not translate
		glog.Error(ctx, "codeLoader can't found", glog.String("app", app))
		return
	}

	lang := md.GetLanguage()
	// common code and msg
	code := ar.GetCode()
	gmetadata.ContextWithUpstreamCode(ctx, code) // observe code
	glog.Debug(ctx, "common handleConvertV2")

	targetCode, targetMsg, ext := t.translate(ctx, codeLoader, code, ar.GetMsg(), app, lang, ar.GetExtMap(), md)
	target.SetCode(targetCode)
	target.SetMessage(targetMsg)
	if len(ext) > 0 && !bytes.Equal(ext, emptyJSON) {
		target.SetExtMap(ext)
	}

	if !batch {
		target.SetExtInfo(ar.GetExtInfo())
		return
	}

	// batch code and msg
	bar, ok := ar.(*batchAPIResponseInfo)
	if !ok {
		glog.Error(ctx, "batch code and message error", glog.Any("info", ar))
		return
	}

	for i := 0; i < len(bar.Codes); i++ {
		bcode := bar.GetCodeByIdx(i)
		glog.Debug(ctx, "batch handleConvertV2")
		bar.Codes[i], bar.Messages[i], _ = t.translate(ctx, codeLoader, bcode, bar.GetMessageByIdx(i), app, lang, bar.GetExtMapByIdx(i), md)
	}
	target.SetExtInfo(bar.GetBatchExtInfo())
}

func (t *translate) translate(ctx *types.Ctx, codeLoader *CodeListener, code int64, msg, app, lang string, ext []byte, md *gmetadata.Metadata) (int64, string, []byte) {
	glog.Debug(ctx, "handleConvertV2", glog.String("app", app), glog.Int64("source", int64(t.msgSource)),
		glog.Int64("tag", t.msgTag), glog.Any("source-code", code), glog.String("lang", lang))

	targetCode, targetMsg, hasParse := codeLoader.parseCodeMessage(ctx, ext, t.msgTag, lang, code, msg, t.msgSource, md.Route.Registry)
	glog.Debug(ctx, "parseCodeMessage over", glog.Any("target", targetCode), glog.Bool("hasParse", hasParse))

	if hasParse {
		return targetCode, targetMsg, nil
	}
	return targetCode, targetMsg, ext
}
