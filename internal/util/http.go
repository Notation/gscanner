package util

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
)

var Client *http.Client = &http.Client{
	Timeout: time.Second * 120,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		DisableKeepAlives:     true,
		MaxIdleConns:          0,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   100,
		Proxy:                 http.ProxyFromEnvironment,
	},
}

func Get(url string) (resp *http.Response, err error) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return Client.Do(req)
}

func GetWithTimeout(url string, timeout time.Duration) (resp *http.Response, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return Client.Do(req)
}

func Post(url, contentType string, body io.Reader) (resp *http.Response, err error) {
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return Client.Do(req)
}

func Do(req *http.Request) (*http.Response, error) {
	ctx := context.Background()
	reqWithCtx := req.WithContext(ctx)

	return Client.Do(reqWithCtx)
}

func GetJSON(url string, data interface{}) error {
	resp, err := Get(url)
	if err != nil {
		return fmt.Errorf("get url %s err <%w>", url, err)
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(respBody, data)
	if err != nil {
		log.Printf("GetJSON url <%s> content:\n %s", url, respBody)
		return fmt.Errorf("unmarshal err <%w>", err)
	}
	return nil
}

func DownloadFile(filePath, fileURL string) error {
	out, err := os.Create(filePath)
	if err != nil {
		return errors.Wrap(err, "Create")
	}
	defer out.Close()
	resp, err := Get(fileURL)
	if err != nil {
		return errors.Wrap(err, "Get")
	}
	defer resp.Body.Close()
	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "Copy %d", n)
	}
	return nil
}
