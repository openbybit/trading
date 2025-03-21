package galert

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	httpClient     *http.Client
	httpClientOnce sync.Once
)

func getHttpClient() *http.Client {
	httpClientOnce.Do(func() {
		httpClient = &http.Client{
			Timeout: 3 * time.Second, // 设置发送超时时间
		}

		const envKeySquidProxy = "SQUID_PROXY"
		proxy := os.Getenv(envKeySquidProxy)
		if proxy == "" {
			return
		}
		if !strings.Contains(proxy, "://") {
			proxy = "http://" + proxy
		}

		p, err := url.Parse(proxy)
		if err != nil {
			log.Println("httpClient url.Parse err:" + err.Error() + ",proxy" + proxy)
			return
		}
		httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(p)}
	})
	return httpClient
}

func sendPost(ctx context.Context, url string, data []byte) error {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	rsp, err := getHttpClient().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = rsp.Body.Close() }()
	type Result struct {
		StatusCode    int
		StatusMessage string
		Code          int `json:"code"`
	}

	rspBody, _ := io.ReadAll(rsp.Body)
	result := &Result{}
	if err := json.Unmarshal(rspBody, result); err != nil || result.StatusCode != 0 || result.Code != 0 {
		log.Println("alert sendPost error:", string(rspBody))
	}
	return nil
}
