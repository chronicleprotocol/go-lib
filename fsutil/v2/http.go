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
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	netURL "net/url"
	"time"
)

type HTTPFSOption func(*httpFS)

// WithHTTPClient sets the HTTP client used to perform HTTP requests.
func WithHTTPClient(client *http.Client) HTTPFSOption {
	return func(f *httpFS) {
		f.client = client
	}
}

// NewHTTPProto creates a new HTTP protocol.

// The HTTP protocol is used to create an HTTP file system.
func NewHTTPProto(ctx context.Context, opts ...HTTPFSOption) Protocol {
	return &httpProto{ctx: ctx, opts: opts}
}

type httpProto struct {
	ctx  context.Context
	opts []HTTPFSOption
}

// FileSystem implements the Protocol interface.
func (m *httpProto) FileSystem(uri *netURL.URL) (fs fs.FS, path string, err error) {
	if err := validURI(uri); err != nil {
		return nil, "", errHTTPProtoFn(err)
	}
	var base *netURL.URL
	base, path = uriSplit(uri)
	fs, err = NewHTTPFS(m.ctx, base, m.opts...)
	if err != nil {
		return nil, "", errHTTPProtoFn(err)
	}
	return fs, path, nil
}

// NewHTTPFS creates a new HTTP file system.
func NewHTTPFS(ctx context.Context, baseURI *netURL.URL, opts ...HTTPFSOption) (fs.FS, error) {
	if err := validURI(baseURI); err != nil {
		return nil, errHTTPFSFn(err)
	}
	fs := &httpFS{ctx: ctx}
	for _, opt := range opts {
		opt(fs)
	}
	if fs.client == nil {
		fs.client = http.DefaultClient
	}
	fs.baseURI = baseURI
	return fs, nil
}

type httpFS struct {
	ctx     context.Context
	client  *http.Client
	baseURI *netURL.URL

	// parseFn allows to define a custom name parsing function.
	parseFn func(fs *httpFS, name string) (*netURL.URL, error)
}

// Open implements the fs.FS interface.
func (f *httpFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, errHTTPFSInvalidPathFn(name, nil)
	}
	url, err := f.parse(name)
	if err != nil {
		return nil, errHTTPFSInvalidPathFn(name, err)
	}
	req, err := http.NewRequestWithContext(f.ctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, errHTTPFSRequestErrorFn(url, err)
	}
	res, err := f.client.Do(req)
	if err != nil {
		return nil, errHTTPFSRequestErrorFn(url, err)
	}
	if res.StatusCode != http.StatusOK {
		// Use fs package errors when possible to increase compatibility.
		switch res.StatusCode {
		case http.StatusNotFound:
			return nil, errHTTPFSRequestErrorFn(url, fs.ErrNotExist)
		case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden:
			return nil, errHTTPFSRequestErrorFn(url, fs.ErrPermission)
		}
		return nil, errHTTPFSRequestErrorCodeFn(url, res.StatusCode)
	}
	return &file{
		reader: res.Body,
		info: &fileInfo{
			name:    name,
			size:    res.ContentLength,
			mode:    0,
			modTime: lastModTime(res.Header),
			isDir:   false,
		},
	}, nil
}

func (f *httpFS) parse(name string) (*netURL.URL, error) {
	if f.parseFn != nil {
		return f.parseFn(f, name)
	}
	pathURI, err := netURL.Parse(name)
	if err != nil {
		return nil, err
	}
	uri := f.baseURI.JoinPath(uriPath(pathURI, false))
	uri.ForceQuery = pathURI.ForceQuery
	uri.RawQuery = pathURI.RawQuery
	return uri, nil
}

func lastModTime(headers http.Header) time.Time {
	if t, err := time.Parse(time.RFC1123, headers.Get("Last-Modified")); err == nil {
		return t
	}
	return time.Now()
}

func validURI(uri *netURL.URL) error {
	if uri == nil {
		return errors.New("nil URI")
	}
	if uri.Scheme != "http" && uri.Scheme != "https" {
		return fmt.Errorf("unknown scheme: %s", uri.Scheme)
	}
	if uri.Opaque != "" {
		return errors.New("non-nil opaque")
	}
	if uri.Host == "" {
		return errors.New("empty host")
	}
	if uri.OmitHost {
		return errors.New("omit host must be false")
	}
	if uri.Fragment != "" || uri.RawFragment != "" {
		return errors.New("fragment not allowed")
	}
	return nil
}

func errHTTPProtoFn(err error) error {
	return fmt.Errorf("fsutil.httpProto: %w", err)
}

func errHTTPFSFn(err error) error {
	return fmt.Errorf("fsutil.httpFS: %w", err)
}

func errHTTPFSInvalidPathFn(path string, err error) error {
	if err == nil {
		return fmt.Errorf("fsutil.httpFS: invalid path: %w", errInvalidPathFn(path))
	}
	return fmt.Errorf("fsutil.httpFS: invalid path: %w: %w", errInvalidPathFn(path), err)
}

func errHTTPFSRequestErrorFn(url *netURL.URL, err error) error {
	return fmt.Errorf("fsutil.httpFS: %s: %w", url.String(), err)
}

func errHTTPFSRequestErrorCodeFn(url *netURL.URL, code int) error {
	return fmt.Errorf("fsutil.httpFS: %s: unexpected status code: %d %s", url.String(), code, http.StatusText(code))
}
