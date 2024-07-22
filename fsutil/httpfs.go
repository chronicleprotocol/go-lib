package fsutil

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"

	"github.com/chronicleprotocol/suite/pkg/util/errutil"
)

func NewHTTPFS(ctx context.Context, httpClient httpClient) fs.FS {
	return &httpFS{ctx: ctx, httpClient: httpClient}
}

type httpFS struct {
	ctx        context.Context
	httpClient httpClient
}

func (f *httpFS) ReadFile(name string) (content []byte, err error) {
	httpRequest, err := http.NewRequestWithContext(f.ctx, http.MethodGet, name, nil)
	if err != nil {
		return nil, err
	}
	res, err := f.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content: %w", err)
	}
	defer func() { err = errutil.Append(err, res.Body.Close()) }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status not OK: %s %d %s", res.Proto, res.StatusCode, res.Status)
	}
	return io.ReadAll(res.Body)
}

func (f *httpFS) Open(name string) (fs.File, error) {
	return open(f, name)
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
