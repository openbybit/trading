// 1. download sha256, check current file if changed, if no change,then over.
// 2. download tar.gz file, check if tampering occurred
// 3. decompress tar.gz
// 4. delete old mmdb file, rename new mmdb file to old mmdb
// 5. delete tar.gz

package geoipdb

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"code.bydev.io/fbu/gateway/gway.git/gcore/filesystem"
	"github.com/valyala/fasttemplate"
)

type downloader interface {
	// Do return update filename, update tar name, update error
	Do(id string) (string, string, error)
}

type download struct {
	license   string
	storePath string
	timeout   time.Duration
	tpl       *fasttemplate.Template
}

func newDownloader(license, storePath string, timeout time.Duration) downloader {
	return &download{
		license:   license,
		storePath: storePath,
		timeout:   timeout,
		tpl:       fasttemplate.New(template, "{{", "}}"),
	}
}

// Do
// The return value will be true if there are new changes, false otherwise.
func (d *download) Do(id string) (string, string, error) {
	u := d.tpl.ExecuteString(map[string]interface{}{
		"id":      id,
		"license": d.license,
	})
	downloadSha256 := func(url string) (string, string, error) {
		sha256sum, err := d.httpGet(url + ".sha256")
		if err != nil {
			return "", "", err
		}
		// 9c724d50fef54c8d159a36250b0ed7a700e1187c4ebf5765f79015c175a9dea4  GeoIP2-City_20221018.tar.gz
		meta := strings.TrimSpace(string(sha256sum))
		fileNameSum := strings.Split(meta, "  ")
		if len(fileNameSum) != 2 {
			return "", "", invalidSha256
		}
		return fileNameSum[0], fileNameSum[1], nil
	}
	// download sha256
	sha256Sum, tarName, err := downloadSha256(u)
	if err != nil {
		return "", "", err
	}
	// check current file if changed
	content, err := os.ReadFile(d.storePath + "/" + tarName)
	if err == nil {
		// file content no update
		if sum := fmt.Sprintf("%x", sha256.Sum256(content)); sum == sha256Sum {
			return "", "", nil
		}
	} else {
		if !os.IsNotExist(err) {
			return "", "", err
		}
	}

	var data []byte
	// download zip file
	for i := 0; i < 3; i++ {
		data, err = d.httpGet(u)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if sum := fmt.Sprintf("%x", sha256.Sum256(data)); sum == sha256Sum {
			break
		}
	}
	if len(data) == 0 {
		return "", "", fmt.Errorf("data: %s get error", id)
	}
	err = os.WriteFile(d.storePath+"/"+tarName, data, 0644)
	if err != nil {
		return "", "", err
	}
	filename, err := d.decompress(tarName)
	if err != nil {
		return "", "", fmt.Errorf("decompress %s error: %w", tarName, err)
	}

	return filename, tarName, nil
}

// depress
func (d *download) decompress(tarFile string) (string, error) {
	file, err := os.OpenFile(d.getFilePath(tarFile), os.O_RDONLY, 0644)
	if err != nil {
		return "", err
	}
	gr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	tr := tar.NewReader(gr)
	var (
		prefix   string
		filename string
	)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return "", err
		}
		if strings.HasSuffix(hdr.Name, "/") {
			prefix = hdr.Name
			continue
		}
		if hdr.Name == prefix+cityDataBase {
			filename = cityLiteDataBase + ".tmp"
			if err = filesystem.WriteFileByIO(d.getFilePath(filename), tr); err != nil {
				return "", err
			}
		}
		if hdr.Name == prefix+countryDataBase {
			filename = countryLiteDataBase + ".tmp"
			if err = filesystem.WriteFileByIO(d.getFilePath(countryLiteDataBase+".tmp"), tr); err != nil {
				return "", err
			}
		}
	}
	return filename, nil
}

func (d *download) getFilePath(fileName string) string {
	return d.storePath + "/" + fileName
}

func (d *download) httpGet(url string) ([]byte, error) {
	timeout := 200 * time.Second
	if d.timeout > time.Second*200 {
		timeout = d.timeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("geo NewRequestWithContext url: %s, error:%w", url, err)
	}

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("geo Do Request url: %s, error:%w", url, err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("geo Request ReadAll url: %s, error:%w", url, err)
	}
	_ = resp.Body.Close()
	return data, nil
}

const squidProxy = "SQUID_PROXY"

var (
	httpOnce sync.Once
	client   *http.Client
)

func httpClient() *http.Client {
	httpOnce.Do(func() {
		proxy := os.Getenv(squidProxy)
		if proxy == "" {
			client = http.DefaultClient
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
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(p),
			},
		}
	})
	return client
}
