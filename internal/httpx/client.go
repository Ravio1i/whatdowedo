// Package httpx holds a shared HTTP client and small JSON helpers.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const UserAgent = "whatdowedo/1.0 (+https://github.com/ravio1i/whatdowedo)"

// Client is shared by all providers so connection pooling works across them.
var Client = &http.Client{Timeout: 20 * time.Second}

// GetJSON performs a GET and decodes the JSON body into out.
func GetJSON(ctx context.Context, url string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return doJSON(req, headers, out)
}

// PostJSON performs a POST with a JSON body and decodes the JSON response into out.
func PostJSON(ctx context.Context, url string, headers map[string]string, body any, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return doJSON(req, headers, out)
}

// PostForm performs a POST with a raw text body (used by Overpass) and decodes JSON.
func PostRaw(ctx context.Context, url, contentType string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	return doJSON(req, nil, out)
}

func doJSON(req *http.Request, headers map[string]string, out any) error {
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		snippet := string(data)
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return fmt.Errorf("%s: HTTP %d: %s", req.URL.Host, resp.StatusCode, snippet)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}
