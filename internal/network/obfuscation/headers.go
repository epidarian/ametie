package obfuscation

import (
	"math/rand"
	"net/http"
	"time"
)

// BrowserHeaders contains realistic browser headers
var BrowserHeaders = []map[string]string{
	{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
		"Accept-Encoding": "gzip, deflate, br",
		"DNT":             "1",
		"Connection":      "keep-alive",
		"Upgrade-Insecure-Requests": "1",
	},
	{
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.5",
		"Accept-Encoding": "gzip, deflate, br",
		"Connection":      "keep-alive",
	},
	{
		"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
		"Accept-Encoding": "gzip, deflate",
		"Connection":      "keep-alive",
	},
}

// ApplyBrowserHeaders applies realistic browser headers to a request
func ApplyBrowserHeaders(req *http.Request) {
	rand.Seed(time.Now().UnixNano())
	headers := BrowserHeaders[rand.Intn(len(BrowserHeaders))]

	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

// AddRandomHeaders adds random but realistic headers
func AddRandomHeaders(req *http.Request) {
	rand.Seed(time.Now().UnixNano())

	// Random referer
	referers := []string{
		"https://www.google.com/",
		"https://www.bing.com/",
		"https://duckduckgo.com/",
		"",
	}
	if rand.Float32() > 0.3 {
		req.Header.Set("Referer", referers[rand.Intn(len(referers))])
	}

	// Random cache control
	if rand.Float32() > 0.5 {
		req.Header.Set("Cache-Control", "no-cache")
	}
}

// RotateHeaderNames rotates custom header names based on time
func RotateHeaderNames() map[string]string {
	dayOfYear := time.Now().YearDay()
	rotation := dayOfYear % 7

	variants := []map[string]string{
		{"request_id": "X-Request-Id", "client_time": "X-Client-Time", "node_id": "X-Node-Id"},
		{"request_id": "X-Req-Id", "client_time": "X-Time", "node_id": "X-Node"},
		{"request_id": "X-RID", "client_time": "X-CT", "node_id": "X-NID"},
		{"request_id": "X-Request-ID", "client_time": "X-ClientTime", "node_id": "X-NodeID"},
		{"request_id": "X-Rq-Id", "client_time": "X-C-Time", "node_id": "X-Node-Identifier"},
		{"request_id": "X-ReqID", "client_time": "X-Client-Timestamp", "node_id": "X-NodeId"},
		{"request_id": "X-RequestID", "client_time": "X-CTime", "node_id": "X-NID"},
	}

	return variants[rotation]
}

