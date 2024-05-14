package esi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"time"

	"github.com/fastly/compute-sdk-go/fsthttp"
)

const include = "include"

var (
	closeInclude     = regexp.MustCompile("/>")
	srcAttribute     = regexp.MustCompile(`src="?(.+?)"?( |/>)`)
	altAttribute     = regexp.MustCompile(`alt="?(.+?)"?( |/>)`)
	onErrorAttribute = regexp.MustCompile(`onerror="?(.+?)"?( |/>)`)
)

// safe to pass to any origin.
var headersSafe = []string{
	"Accept",
	"Accept-Language",
}

// safe to pass only to same-origin (same scheme, same host, same port).
var headersUnsafe = []string{
	"Cookie",
	"Authorization",
}

type includeTag struct {
	*baseTag
	silent bool
	alt    string
	src    string
}

func (i *includeTag) loadAttributes(b []byte) error {
	src := srcAttribute.FindSubmatch(b)
	if src == nil {
		return errNotFound
	}

	i.src = string(src[1])

	alt := altAttribute.FindSubmatch(b)
	if alt != nil {
		i.alt = string(alt[1])
	}

	onError := onErrorAttribute.FindSubmatch(b)
	if onError != nil {
		i.silent = string(onError[1]) == "continue"
	}

	return nil
}

func sanitizeURL(u string, reqURL *url.URL) *url.URL {
	parsed, _ := url.Parse(u)

	return reqURL.ResolveReference(parsed)
}

func addHeaders(headers []string, req *fsthttp.Request, rq *fsthttp.Request) {
	for _, h := range headers {
		v := req.Header.Get(h)
		if v != "" {
			rq.Header.Add(h, v)
		}
	}
}

var backends = map[string]struct{}{}

func (i *includeTag) doFetch(ctx context.Context, r *fsthttp.Request, src string) (*fsthttp.Response, error) {
	fetchURL := sanitizeURL(src, r.URL)

	backendName := fmt.Sprintf("this_%s", fetchURL.Host)

	// Don't register backends multiple times
	if _, ok := backends[backendName]; !ok {
		opts := fsthttp.NewBackendOptions()
		opts.HostOverride(fetchURL.Host)
		opts.ConnectTimeout(time.Duration(1) * time.Second)
		opts.FirstByteTimeout(time.Duration(15) * time.Second)
		opts.BetweenBytesTimeout(time.Duration(10) * time.Second)
		opts.UseSSL(true)
		fsthttp.RegisterDynamicBackend(backendName, fetchURL.Host, opts)

		backends[backendName] = struct{}{}
	}

	r.URL = fetchURL
	return r.Send(ctx, backendName)
}

// Input (e.g. include src="https://domain.com/esi-include" alt="https://domain.com/alt-esi-include" />)
// With or without the alt
// With or without a space separator before the closing
// With or without the quotes around the src/alt value.
func (i *includeTag) Process(ctx context.Context, b []byte, req *fsthttp.Request) ([]byte, int) {
	closeIdx := closeInclude.FindIndex(b)

	if closeIdx == nil {
		return nil, len(b)
	}

	i.length = closeIdx[1]
	if e := i.loadAttributes(b[8:i.length]); e != nil {
		return nil, len(b)
	}

	response, err := i.doFetch(ctx, req, i.src)

	if (err != nil || response.StatusCode >= 400) && i.alt != "" {
		response, err = i.doFetch(ctx, req, i.alt)

		if !i.silent && (err != nil || response.StatusCode >= 400) {
			return nil, len(b)
		}
	}

	if response == nil {
		return nil, i.length
	}

	var buf bytes.Buffer

	defer response.Body.Close()
	_, _ = io.Copy(&buf, response.Body)

	b = Parse(ctx, buf.Bytes(), req)

	return b, i.length
}

func (*includeTag) HasClose(b []byte) bool {
	return closeInclude.FindIndex(b) != nil
}

func (*includeTag) GetClosePosition(b []byte) int {
	if idx := closeInclude.FindIndex(b); idx != nil {
		return idx[1]
	}

	return 0
}
