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
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestIPFSFS(t *testing.T) {
	ctx := context.Background()
	tc := []struct {
		name     string
		opts     []IPFSOption
		uri      string
		wantURL  string
		wantData string
		wantErr  bool
	}{
		{
			name:     "path resolution - no path",
			opts:     []IPFSOption{},
			uri:      "ipfs://QmTest",
			wantData: "ipfs path content",
		},
		{
			name:     "path resolution - no checksum",
			opts:     []IPFSOption{},
			uri:      "ipfs://QmTest/test.txt",
			wantData: "ipfs path content",
		},
		{
			name:     "path resolution - with valid checksum",
			opts:     []IPFSOption{},
			uri:      fmt.Sprintf("ipfs://QmTest/test.txt?checksum=%s", calculateKeccak256([]byte("ipfs path content"))),
			wantData: "ipfs path content",
		},
		{
			name:     "subdomain resolution - with valid checksum",
			opts:     []IPFSOption{},
			uri:      fmt.Sprintf("ipfs://QmTest/test.txt?checksum=%s", calculateKeccak256([]byte("ipfs subdomain content"))),
			wantData: "ipfs subdomain content",
		},
		{
			name:    "path resolution - with invalid checksum",
			opts:    []IPFSOption{},
			uri:     fmt.Sprintf("ipfs://QmTest/test.txt?checksum=%s", calculateKeccak256([]byte("invalid"))),
			wantErr: true,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					switch req.URL.String() {
					case "https://ipfs-path.io/ipfs/QmTest":
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader("ipfs path content")),
						}, nil
					case "https://ipfs-path.io/ipfs/QmTest/test.txt":
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader("ipfs path content")),
						}, nil
					case "https://QmTest.ipfs-subdomain.io/test.txt":
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(strings.NewReader("ipfs subdomain content")),
						}, nil
					default:
						return &http.Response{
							StatusCode: http.StatusNotFound,
						}, nil
					}
				}),
			}
			opts := append(
				tt.opts,
				WithIPFSHTTPClient(client),
				WithIPFSGateways(
					&IPFSGateway{Scheme: "https", Host: "ipfs-path.io", ResolveFn: IPFSPathResolution},
					&IPFSGateway{Scheme: "https", Host: "ipfs-subdomain.io", ResolveFn: IPFSSubdomainResolution},
				),
			)
			proto := NewIPFSProto(ctx, opts...)
			fs, path, err := ParseURI(proto, tt.uri)
			require.NoError(t, err)
			fs.(*ipfsFS).cfs.rand = false

			file, err := fs.Open(path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer file.Close()

			data, err := io.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, tt.wantData, string(data))
		})
	}
}

func TestIPFSPathResolution(t *testing.T) {
	tests := []struct {
		name    string
		httpFS  *httpFS
		cid     string
		path    string
		wantURL *url.URL
		wantErr bool
	}{
		{
			name: "cid only",
			httpFS: &httpFS{
				baseURI: &url.URL{Scheme: "https", Host: "ipfs.io"},
			},
			cid:  "QmTest",
			path: "",
			wantURL: &url.URL{
				Scheme: "https",
				Host:   "ipfs.io",
				Path:   "/ipfs/QmTest",
			},
		},
		{
			name: "cid with path",
			httpFS: &httpFS{
				baseURI: &url.URL{Scheme: "https", Host: "gateway.pinata.cloud"},
			},
			cid:  "QmTest",
			path: "test.txt",
			wantURL: &url.URL{
				Scheme: "https",
				Host:   "gateway.pinata.cloud",
				Path:   "/ipfs/QmTest/test.txt",
			},
		},
		{
			name: "cid with nested path",
			httpFS: &httpFS{
				baseURI: &url.URL{Scheme: "https", Host: "ipfs.io"},
			},
			cid:  "QmTest",
			path: "path/to/file.json",
			wantURL: &url.URL{
				Scheme: "https",
				Host:   "ipfs.io",
				Path:   "/ipfs/QmTest/path/to/file.json",
			},
		},
		{
			name: "preserve auth info",
			httpFS: &httpFS{
				baseURI: &url.URL{
					Scheme: "https",
					Host:   "ipfs.io",
					User:   url.UserPassword("user", "pass"),
				},
			},
			cid:  "QmTest",
			path: "file.txt",
			wantURL: &url.URL{
				Scheme: "https",
				Host:   "ipfs.io",
				User:   url.UserPassword("user", "pass"),
				Path:   "/ipfs/QmTest/file.txt",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the resolver function for the CID
			resolverFn := IPFSPathResolution(tt.cid)

			// Call the resolver function with the httpFS and path
			url, err := resolverFn(tt.httpFS, tt.path)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL.Scheme, url.Scheme)
			assert.Equal(t, tt.wantURL.Host, url.Host)
			assert.Equal(t, tt.wantURL.Path, url.Path)
			if tt.wantURL.User != nil {
				assert.NotNil(t, url.User)
				assert.Equal(t, tt.wantURL.User.String(), url.User.String())
			}
		})
	}
}

func TestIPFSSubdomainResolution(t *testing.T) {
	tests := []struct {
		name    string
		httpFS  *httpFS
		cid     string
		path    string
		wantURL *url.URL
		wantErr bool
	}{
		{
			name: "cid only",
			httpFS: &httpFS{
				baseURI: &url.URL{Scheme: "https", Host: "dweb.link"},
			},
			cid:  "QmTest",
			path: "",
			wantURL: &url.URL{
				Scheme: "https",
				Host:   "QmTest.dweb.link",
				Path:   "",
			},
		},
		{
			name: "cid with path",
			httpFS: &httpFS{
				baseURI: &url.URL{Scheme: "https", Host: "w3s.link"},
			},
			cid:  "QmTest",
			path: "test.txt",
			wantURL: &url.URL{
				Scheme: "https",
				Host:   "QmTest.w3s.link",
				Path:   "test.txt",
			},
		},
		{
			name: "cid with nested path",
			httpFS: &httpFS{
				baseURI: &url.URL{Scheme: "https", Host: "ipfs.cyou"},
			},
			cid:  "QmTest",
			path: "path/to/file.json",
			wantURL: &url.URL{
				Scheme: "https",
				Host:   "QmTest.ipfs.cyou",
				Path:   "path/to/file.json",
			},
		},
		{
			name: "preserve auth info",
			httpFS: &httpFS{
				baseURI: &url.URL{
					Scheme: "https",
					Host:   "dweb.link",
					User:   url.UserPassword("user", "pass"),
				},
			},
			cid:  "QmTest",
			path: "file.txt",
			wantURL: &url.URL{
				Scheme: "https",
				Host:   "QmTest.dweb.link",
				User:   url.UserPassword("user", "pass"),
				Path:   "file.txt",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the resolver function for the CID
			resolverFn := IPFSSubdomainResolution(tt.cid)

			// Call the resolver function with the httpFS and path
			url, err := resolverFn(tt.httpFS, tt.path)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL.Scheme, url.Scheme)
			assert.Equal(t, tt.wantURL.Host, url.Host)
			assert.Equal(t, tt.wantURL.Path, url.Path)
			if tt.wantURL.User != nil {
				assert.NotNil(t, url.User)
				assert.Equal(t, tt.wantURL.User.String(), url.User.String())
			}
		})
	}
}
