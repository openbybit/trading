package openapi

import (
	"bgw/pkg/server/metadata"
	ropenapi "bgw/pkg/service/openapi"
	"bgw/pkg/test"
	"bgw/pkg/test/mock"
	"errors"
	"git.bybit.com/svc/stub/pkg/pb/api/user"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/golang/mock/gomock"
	"github.com/tj/assert"
	"github.com/valyala/fasthttp"
	"testing"
)

func TestQueryParse(t *testing.T) {
	query := `a=b&c=d`
	p := queryParse([]byte(query))
	t.Log(p)

	query = `cursor=eyJtaW5JRCI6MzY3Njk3LCJtYXhJRCI6MzY3Njk4fQ==&limit=2`
	p = queryParse([]byte(query))
	t.Log(p)

	query = `cursor=&limit=2`
	p = queryParse([]byte(query))
	t.Log(p)
}

func TestGetMemberID(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	id, err := GetMemberID(rctx, "")
	assert.Equal(t, int64(0), id)
	assert.EqualError(t, err, "only support v3: Request parameter error.")

	p := gomonkey.ApplyFuncReturn(metadata.MDFromContext, nil)
	id, err = GetMemberID(&fasthttp.RequestCtx{}, "1212")
	assert.Equal(t, int64(0), id)
	assert.EqualError(t, err, "Internal System Error.")
	p.Reset()
}

func TestGetMemberID1(t *testing.T) {
	p := gomonkey.ApplyFuncReturn(metadata.MDFromContext, nil)
	id, err := GetMemberID(&fasthttp.RequestCtx{}, "1212")
	assert.Equal(t, int64(0), id)
	assert.EqualError(t, err, "Internal System Error.")
	p.Reset()
}

func TestGetMemberID3(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	ctrl := gomock.NewController(t)
	oi := mock.NewMockOpenAPIServiceIface(ctrl)
	p := gomonkey.ApplyFuncReturn(ropenapi.GetOpenapiService, oi, errors.New("xxx2"))

	id, err := GetMemberID(rctx, "1212")
	assert.Equal(t, int64(0), id)
	assert.EqualError(t, err, "xxx2")

	p.Reset()
	ctrl.Finish()
}

func TestGetMemberID4(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	ctrl := gomock.NewController(t)
	oi := mock.NewMockOpenAPIServiceIface(ctrl)
	p := gomonkey.ApplyFuncReturn(ropenapi.GetOpenapiService, oi, nil)
	oi.EXPECT().GetAPIKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("xxx3"))
	id, err := GetMemberID(rctx, "1212")
	assert.Equal(t, int64(0), id)
	assert.EqualError(t, err, "xxx3")

	p.Reset()
	ctrl.Finish()
}

func TestGetMemberID5(t *testing.T) {
	rctx, _ := test.NewReqCtx()
	ctrl := gomock.NewController(t)
	oi := mock.NewMockOpenAPIServiceIface(ctrl)
	p := gomonkey.ApplyFuncReturn(ropenapi.GetOpenapiService, oi, nil)
	oi.EXPECT().GetAPIKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(&user.MemberLogin{
		MemberId: 0,
	}, nil)
	id, err := GetMemberID(rctx, "1212")
	assert.Equal(t, int64(0), id)
	assert.EqualError(t, err, "invalid member id: API key is invalid.")

	p.Reset()
	ctrl.Finish()
}

func TestGetMemberID6(t *testing.T) {
	rctx, md := test.NewReqCtx()
	ctrl := gomock.NewController(t)
	oi := mock.NewMockOpenAPIServiceIface(ctrl)
	p := gomonkey.ApplyFuncReturn(ropenapi.GetOpenapiService, oi, nil)
	oi.EXPECT().GetAPIKey(gomock.Any(), gomock.Any(), gomock.Any()).Return(&user.MemberLogin{
		MemberId: 10000,
	}, nil)
	id, err := GetMemberID(rctx, "1212")
	assert.Equal(t, int64(10000), id)
	assert.Equal(t, int64(10000), md.UID)
	assert.NoError(t, err)
	p.Reset()
	ctrl.Finish()
}
