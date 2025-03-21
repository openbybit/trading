package version

import (
	"bgw/pkg/server/metadata"
	"github.com/tj/assert"
	"strconv"
	"testing"
	"time"
)

func TestPassThrough(t *testing.T) {
	resp := NewPassthroughResponse()
	assert.Equal(t, VersionPassthrough, resp.Version())
	resp.SetCode(100)
	assert.Equal(t, int64(0), resp.GetCode())
	resp.SetMessage("mess")
	assert.Equal(t, "", resp.GetMessage())
	resp.SetResult([]byte("mess"))
	assert.Equal(t, "mess", string(resp.GetResult()))
	resp.SetExtInfo([]byte("{\"uid\":123}"))
	assert.Equal(t, []byte(nil), resp.GetExtInfo())
	resp.SetExtMap([]byte("{\"uid\":123}"))
	assert.Equal(t, []byte(nil), resp.GetExtMap())

	r, err := resp.Marshal()
	assert.Equal(t, "mess", string(r))
	assert.NoError(t, err)
}

func TestV1Response(t *testing.T) {
	resp := NewV1Response()
	assert.Equal(t, VersionV1, resp.Version())
	resp.SetCode(100)
	assert.Equal(t, int64(100), resp.GetCode())
	resp.SetMessage("mess")
	assert.Equal(t, "mess", resp.GetMessage())
	resp.SetExtInfo([]byte("{\"uid\":123}"))
	assert.Equal(t, []byte("{\"uid\":123}"), resp.GetExtInfo())
	resp.SetExtMap([]byte("{\"uid\":123}"))
	assert.Equal(t, []byte("{\"uid\":123}"), resp.GetExtMap())
	ti := time.Date(2002, 01, 01, 01, 01, 01, 01, time.UTC)
	resp.SetTime(ti)
	assert.Equal(t, ti, resp.Time)
	token := "12"
	resp.SetToken(&token)
	assert.Equal(t, &token, resp.Token)
	resp.SetLimit(metadata.RateLimitInfo{
		RateLimitStatus:  100,
		RateLimit:        200,
		RateLimitResetMs: 300,
	})
	assert.Equal(t, 200, resp.RateLimit)
	assert.Equal(t, 100, resp.RateLimitStatus)
	assert.Equal(t, 300, resp.RateLimitResetMs)

	resp.SetResult([]byte("mess"))
	assert.Equal(t, "mess", string(resp.GetResult()))
	resp.SetError("xxxx")
	assert.Equal(t, "xxxx", resp.Error)
	r, err := resp.Marshal()
	assert.Equal(t, "{\"ret_code\":100,\"ret_msg\":\"mess\",\"result\":mess,\"ext_code\":\"\",\"ext_info\":{\"uid\":123},\"ext_map\":{\"uid\":123},\"time_now\":\"2002-01-01T01:01:01.000000001Z\",\"token\":\"12\",\"error\":\"xxxx\",\"rate_limit_status\":100,\"rate_limit\":200,\"rate_limit_reset_ms\":300}", string(r))
	assert.NoError(t, err)
}

func TestV2Response(t *testing.T) {
	resp := NewV2Response()
	assert.Equal(t, VersionV2, resp.Version())
	resp.SetCode(100)
	assert.Equal(t, int64(100), resp.GetCode())
	resp.SetMessage("mess")
	assert.Equal(t, "mess", resp.GetMessage())
	resp.SetExtInfo([]byte("{\"uid\":123}"))
	assert.Equal(t, []byte("{\"uid\":123}"), resp.GetExtInfo())
	resp.SetExtMap([]byte("{\"uid\":123}"))
	assert.Equal(t, []byte("{\"uid\":123}"), resp.GetExtMap())
	//resp.se
	resp.SetResult([]byte("mess"))
	assert.Equal(t, "mess", string(resp.GetResult()))

	resp.SetError("xxxx")
	assert.Equal(t, "xxxx", resp.Error)

	r, err := resp.Marshal()
	assert.Equal(t, "{\"retCode\":100,\"retMsg\":\"mess\",\"result\":mess,\"retExtMap\":{\"uid\":123},\"retExtInfo\":{\"uid\":123},\"time\":"+strconv.FormatInt(resp.Time, 10)+",\"error\":\"xxxx\"}", string(r))
	assert.NoError(t, err)
}
