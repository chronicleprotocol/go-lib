// Copyright (C) 2021-2025 Chronicle Labs, Inc.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
