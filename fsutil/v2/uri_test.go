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
	"net/url"
	"testing"
)

func TestUriCopy(t *testing.T) {
	tests := []struct {
		name string
		uri  *url.URL
		want *url.URL
	}{
		{
			name: "nil url",
			uri:  nil,
			want: nil,
		},
		{
			name: "empty url",
			uri:  &url.URL{},
			want: &url.URL{},
		},
		{
			name: "full url",
			uri: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/resource",
				RawQuery: "param=value",
				Fragment: "fragment",
			},
			want: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/resource",
				RawQuery: "param=value",
				Fragment: "fragment",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uriCopy(tt.uri)

			// Check for nil case
			if tt.want == nil && got != nil {
				t.Errorf("uriCopy() = %v, want %v", got, tt.want)
				return
			}
			if tt.want != nil && got == nil {
				t.Errorf("uriCopy() = %v, want %v", got, tt.want)
				return
			}
			if tt.want == nil && got == nil {
				return
			}

			// Compare non-nil URLs
			if got.String() != tt.want.String() {
				t.Errorf("uriCopy() = %v, want %v", got, tt.want)
			}

			// Ensure it's a deep copy
			if tt.uri != nil {
				tt.uri.Scheme = "changed"
				if got.Scheme == "changed" {
					t.Errorf("uriCopy() did not create a deep copy")
				}
			}
		})
	}
}

func TestUriPath(t *testing.T) {
	tests := []struct {
		name                 string
		uri                  *url.URL
		inclQueryAndFragment bool
		want                 string
	}{
		{
			name: "path only",
			uri: &url.URL{
				Path: "/path/to/resource",
			},
			inclQueryAndFragment: false,
			want:                 "path/to/resource",
		},
		{
			name: "path with query but not included",
			uri: &url.URL{
				Path:     "/path/to/resource",
				RawQuery: "param=value",
			},
			inclQueryAndFragment: false,
			want:                 "path/to/resource",
		},
		{
			name: "path with query included",
			uri: &url.URL{
				Path:     "/path/to/resource",
				RawQuery: "param=value",
			},
			inclQueryAndFragment: true,
			want:                 "path/to/resource?param=value",
		},
		{
			name: "path with fragment included",
			uri: &url.URL{
				Path:     "/path/to/resource",
				Fragment: "section",
			},
			inclQueryAndFragment: true,
			want:                 "path/to/resource#section",
		},
		{
			name: "path with query and fragment included",
			uri: &url.URL{
				Path:     "/path/to/resource",
				RawQuery: "param=value",
				Fragment: "section",
			},
			inclQueryAndFragment: true,
			want:                 "path/to/resource?param=value#section",
		},
		{
			name: "empty path",
			uri: &url.URL{
				Path: "",
			},
			inclQueryAndFragment: true,
			want:                 "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uriPath(tt.uri, tt.inclQueryAndFragment)
			if got != tt.want {
				t.Errorf("uriPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUriSplit(t *testing.T) {
	tests := []struct {
		name     string
		uri      *url.URL
		wantBase string
		wantPath string
	}{
		{
			name: "simple url",
			uri: &url.URL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "/path/to/resource",
			},
			wantBase: "https://example.com",
			wantPath: "path/to/resource",
		},
		{
			name: "url with query and fragment",
			uri: &url.URL{
				Scheme:   "https",
				Host:     "example.com",
				Path:     "/path/to/resource",
				RawQuery: "param=value",
				Fragment: "section",
			},
			wantBase: "https://example.com",
			wantPath: "path/to/resource?param=value#section",
		},
		{
			name: "file url",
			uri: &url.URL{
				Scheme: "file",
				Path:   "/path/to/file.txt",
			},
			wantBase: "file:",
			wantPath: "path/to/file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBase, gotPath := uriSplit(tt.uri)
			if gotBase.String() != tt.wantBase {
				t.Errorf("uriSplit() base = %v, want %v", gotBase.String(), tt.wantBase)
			}
			if gotPath != tt.wantPath {
				t.Errorf("uriSplit() path = %v, want %v", gotPath, tt.wantPath)
			}
		})
	}
}
