package http

import (
	"bgw/pkg/server/filter/auth"
	"bgw/pkg/server/filter/openapi"
	"context"
	"errors"
	"github.com/valyala/fasthttp"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/tj/assert"

	"bgw/pkg/common/berror"
	"bgw/pkg/common/constant"
	"bgw/pkg/common/types"
	"bgw/pkg/service/user"
	"bgw/pkg/test/mock"
)

func TestGetAccountTypeByUID(t *testing.T) {
	ctx := context.Background()
	te, err := getAccountTypeByUID(ctx, 0)
	assert.Equal(t, berror.ErrParams, err)
	assert.Equal(t, constant.AccountTypeUnknown, te)

	p := gomonkey.ApplyFuncReturn(user.NewAccountService, nil, errors.New("111"))
	te, err = getAccountTypeByUID(ctx, 10)
	assert.Equal(t, "111,invalid account service", err.Error())
	assert.Equal(t, constant.AccountTypeUnknown, te)

	p.ApplyFuncReturn(user.NewAccountService, nil, nil)
	te, err = getAccountTypeByUID(ctx, 10)
	assert.Equal(t, "Internal System Error.,invalid account service", err.Error())
	assert.Equal(t, constant.AccountTypeUnknown, te)

	ctrl := gomock.NewController(t)
	as := mock.NewMockAccountIface(ctrl)

	as.EXPECT().
		QueryMemberTag(gomock.Any(),
			gomock.Eq(int64(10)),
			gomock.Eq(user.UnifiedTradingTag)).
		Return("", errors.New("xxx"))
	p.ApplyFuncReturn(user.NewAccountService, as, nil)
	te, err = getAccountTypeByUID(ctx, 10)
	assert.Equal(t, "xxx", err.Error())
	assert.Equal(t, constant.AccountTypeUnknown, te)

	as.EXPECT().
		QueryMemberTag(gomock.Any(),
			gomock.Eq(int64(10)),
			gomock.Eq(user.UnifiedTradingTag)).
		Return(user.UnifiedStateSuccess, nil)
	te, err = getAccountTypeByUID(ctx, 10)
	assert.NoError(t, err)
	assert.Equal(t, constant.AccountTypeUnifiedTrading, te)

	as.EXPECT().
		QueryMemberTag(gomock.Any(),
			gomock.Eq(int64(10)),
			gomock.Eq(user.UnifiedTradingTag)).
		Return(user.UnifiedStateFail, nil)
	as.EXPECT().
		QueryMemberTag(gomock.Any(),
			gomock.Eq(int64(10)),
			gomock.Eq(user.UnifiedMarginTag)).
		Return(user.UnifiedStateSuccess, nil)
	te, err = getAccountTypeByUID(ctx, 10)
	assert.NoError(t, err)
	assert.Equal(t, constant.AccountTypeUnifiedMargin, te)

	as.EXPECT().
		QueryMemberTag(gomock.Any(),
			gomock.Eq(int64(10)),
			gomock.Eq(user.UnifiedTradingTag)).
		Return(user.UnifiedStateFail, nil)
	as.EXPECT().
		QueryMemberTag(gomock.Any(),
			gomock.Eq(int64(10)),
			gomock.Eq(user.UnifiedMarginTag)).
		Return(user.UnifiedStateFail, nil)
	te, err = getAccountTypeByUID(ctx, 10)
	assert.NoError(t, err)
	assert.Equal(t, constant.AccountTypeNormal, te)

	p.Reset()
	ctrl.Finish()
}

func Test_getUserID(t *testing.T) {
	Convey("test getUserID", t, func() {
		ctx := &types.Ctx{}
		ctx.Request.Header.Set(constant.HeaderAPIKey, "val")
		p := gomonkey.ApplyFuncReturn(openapi.GetMemberID, int64(100), nil).
			ApplyFuncReturn(auth.GetMemberID, int64(100), false, nil).
			ApplyFuncReturn(auth.GetToken, "123")
		defer p.Reset()
		_, d, err := getUserID(ctx)
		So(err, ShouldBeNil)
		So(d, ShouldBeFalse)

		p.ApplyFuncReturn(openapi.GetMemberID, int64(100), nil).
			ApplyFuncReturn(auth.GetMemberID, int64(100), false, nil).
			ApplyFuncReturn(auth.GetToken, "")

		ctx = &types.Ctx{}
		ctx.Request.Header.SetMethod("GET")
		url := &fasthttp.URI{}
		url.SetQueryString("api_key=111")
		ctx.Request.SetURI(url)
		_, d, err = getUserID(ctx)
		So(err, ShouldBeNil)
		So(d, ShouldBeFalse)

		ctx = &types.Ctx{}
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetBodyString(`{"api_key":"1212"}`)
		_, d, err = getUserID(ctx)
		So(err, ShouldBeNil)
		So(d, ShouldBeFalse)

		ctx = &types.Ctx{}
		ctx.Request.Header.SetMethod("xxx")
		ctx.Request.SetBodyString(`{"api_key":"1212"}`)
		_, d, err = getUserID(ctx)
		So(err, ShouldBeNil)
		So(d, ShouldBeFalse)

		ctx = &types.Ctx{}
		ctx.Request.Header.SetMethod("POST")
		ctx.Request.SetBodyString(`{"api_key":"1212"}`)
		_, d, err = getUserID(ctx)
		So(err, ShouldBeNil)
		So(d, ShouldBeFalse)

		p.ApplyFuncReturn(openapi.GetMemberID, int64(100), nil).
			ApplyFuncReturn(auth.GetMemberID, int64(100), false, nil).
			ApplyFuncReturn(auth.GetToken, "123")
		ctx = &types.Ctx{}
		ctx.Request.Header.Set("UserToken", "val")
		_, d, err = getUserID(ctx)
		So(err, ShouldBeNil)
		So(d, ShouldBeFalse)

		p.ApplyFuncReturn(openapi.GetMemberID, int64(100), nil).
			ApplyFuncReturn(auth.GetMemberID, int64(100), true, nil).
			ApplyFuncReturn(auth.GetToken, "123")
		ctx = &types.Ctx{}
		ctx.Request.Header.Set("UserToken", "val")
		_, d, err = getUserID(ctx)
		So(err, ShouldBeNil)
		So(d, ShouldBeTrue)

		p.ApplyFuncReturn(openapi.GetMemberID, int64(100), nil).
			ApplyFuncReturn(auth.GetMemberID, int64(100), false, nil).
			ApplyFuncReturn(auth.GetToken, "")
		ctx = &types.Ctx{}
		ctx.Request.Header.Set("UserToken", "val")
		_, d, err = getUserID(ctx)
		So(err, ShouldNotBeNil)
		So(d, ShouldBeFalse)
	})
}
