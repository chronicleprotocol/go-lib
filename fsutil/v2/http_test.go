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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPProto(t *testing.T) {
	ctx := context.Background()
	tc := []struct {
		name     string
		opts     []HTTPFSOption
		uri      string
		wantPath string
		wantErr  bool
	}{
		{
			name:    "nil URL",
			uri:     "",
			wantErr: true,
		},
		{
			name:    "unexpected scheme",
			uri:     "file://localhost",
			wantErr: true,
		},
		{
			name:    "empty host",
			uri:     "http://",
			wantErr: true,
		},
		{
			name:     "valid URL",
			uri:      "http://localhost",
			wantPath: "",
		},
		{
			name:     "path",
			uri:      "http://localhost/test",
			wantPath: "test",
		},
		{
			name:     "query",
			uri:      "http://localhost/test?query",
			wantPath: "test?query",
		},
		{
			name:    "fragment",
			uri:     "http://localhost/test#fragment",
			wantErr: true,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			httpProto := NewHTTPProto(ctx, tt.opts...)
			var u *url.URL
			if tt.uri != "" {
				var err error
				u, err = url.Parse(tt.uri)
				require.NoError(t, err)
			}
			httpFS, path, err := httpProto.FileSystem(u)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, httpFS)
			assert.Equal(t, tt.wantPath, path)
		})
	}
}

func TestHTTPFS(t *testing.T) {
	ctx := context.Background()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/test.txt", "/dir/test2.txt":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("test content"))
		case "/query.txt":
			if r.URL.RawQuery != "query" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("test content"))
		case "/unauthorized.txt":
			w.WriteHeader(http.StatusUnauthorized)
		case "/paymentrequired.txt":
			w.WriteHeader(http.StatusPaymentRequired)
		case "/forbidden.txt":
			w.WriteHeader(http.StatusForbidden)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	tc := []struct {
		name              string
		baseURL           string
		file              string
		wantErr           bool
		wantNotExistsErr  bool
		wantPermissionErr bool
	}{
		{
			name:    "valid request",
			baseURL: "http://localhost",
			file:    "test.txt",
		},
		{
			name:    "valid request - subdir",
			baseURL: "http://localhost",
			file:    "dir/test2.txt",
		},
		{
			name:    "valid request - basedir",
			baseURL: "http://localhost/dir",
			file:    "test2.txt",
		},
		{
			name:    "valid request - basedir with trailing slash",
			baseURL: "http://localhost/dir/",
			file:    "test2.txt",
		},
		{
			name:    "query",
			baseURL: "http://localhost/",
			file:    "query.txt?query",
		},
		{
			name:    "invalid path",
			baseURL: "http://localhost/dir",
			file:    "../test.txt",
			wantErr: true,
		},
		{
			name:             "file not found",
			baseURL:          "http://localhost",
			file:             "notfound.txt",
			wantErr:          true,
			wantNotExistsErr: true,
		},
		{
			name:              "unauthorized",
			baseURL:           "http://localhost",
			file:              "unauthorized.txt",
			wantErr:           true,
			wantPermissionErr: true,
		},
		{
			name:              "payment required",
			baseURL:           "http://localhost",
			file:              "paymentrequired.txt",
			wantErr:           true,
			wantPermissionErr: true,
		},
		{
			name:              "forbidden",
			baseURL:           "http://localhost",
			file:              "forbidden.txt",
			wantErr:           true,
			wantPermissionErr: true,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(handler)
			defer server.Close()

			baseURL, err := url.Parse(tt.baseURL)
			require.NoError(t, err)

			baseURL.Host = server.Listener.Addr().String()

			httpFS, err := NewHTTPFS(ctx, baseURL)
			require.NoError(t, err)

			file, err := httpFS.Open(tt.file)
			if tt.wantNotExistsErr {
				assert.True(t, errors.Is(err, os.ErrNotExist))
			}
			if tt.wantPermissionErr {
				assert.True(t, errors.Is(err, os.ErrPermission))
			}
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			content, err := io.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, "test content", string(content))
		})
	}
}
