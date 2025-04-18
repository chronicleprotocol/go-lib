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
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"strings"
)

// ParseURI is a helper function that parses a URI for a given protocol and returns
// the appropriate filesystem and path.
//
// As a special case, if the URI does not contain a scheme, it is assumed to be
// a file URI and is prefixed with "file:///".
func ParseURI(p Protocol, uri string) (fs.FS, string, error) {
	if !strings.Contains(uri, "://") {
		uri = "file:///" + uri
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, "", errParseURIFn(err)
	}
	return p.FileSystem(u)
}

// uriSplit splits a URL into a base URL (scheme, host, etc.) and a path
// component. The base URL has all path, query, and fragment information
// removed.
func uriSplit(uri *url.URL) (base *url.URL, path string) {
	path = uriPath(uri, true)
	base = uriCopy(uri)
	base.Path = ""
	base.RawPath = ""
	base.ForceQuery = false
	base.RawQuery = ""
	base.Fragment = ""
	base.RawFragment = ""
	return base, path
}

// uriCopy creates a deep copy of the given URL.
// Returns nil if the input URL is nil.
func uriCopy(uri *url.URL) *url.URL {
	if uri == nil {
		return nil
	}
	c := *uri
	return &c
}

// uriPath extracts the path component from a URL, optionally including query
// and fragment parts.
func uriPath(uri *url.URL, inclQueryAndFragment bool) string {
	w := strings.Builder{}
	p := uri.EscapedPath()
	if len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	w.WriteString(p)
	switch w.String() {
	case "":
		w.WriteString(".")
	case "/":
		w.Reset()
		w.WriteString(".")
	}
	if inclQueryAndFragment {
		if uri.ForceQuery || uri.RawQuery != "" {
			w.WriteByte('?')
			w.WriteString(uri.RawQuery)
		}
		if uri.Fragment != "" {
			w.WriteByte('#')
			w.WriteString(uri.EscapedFragment())
		}
	}
	return path.Clean(w.String())
}

func errParseURIFn(err error) error {
	return fmt.Errorf("fsutil.ParseURI: %w", err)
}
