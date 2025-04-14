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
	netURL "net/url"
)

// ProtoFunc is a function that creates a Protocol from a URL.
type ProtoFunc func(*netURL.URL) (Protocol, error)

// NewMux creates a new protocol multiplexer that routes URIs to registered protocols
// based on their scheme.
func NewMux(ps map[string]ProtoFunc) Protocol {
	return &mux{ps: ps}
}

type mux struct {
	ps map[string]ProtoFunc
}

// FileSystem implements the Protocol interface.
func (m *mux) FileSystem(uri *netURL.URL) (fs.FS, string, error) {
	if uri == nil {
		return nil, "", errMuxNilURI
	}
	if uri.Scheme == "" {
		uri.Scheme = "file"
	}
	if f, ok := m.ps[uri.Scheme]; ok {
		p, err := f(uri)
		if err != nil {
			return nil, "", err
		}
		return p.FileSystem(uri)
	}
	return nil, "", errMuxUnknownSchemeFn(uri.Scheme)
}

var (
	errMuxNilURI        = fmt.Errorf("fsutil.mux: nil URI")
	errMuxUnknownScheme = fmt.Errorf("fsutil.mux: unknown scheme")
)

func errMuxUnknownSchemeFn(scheme string) error {
	return fmt.Errorf("%w: %s", errMuxUnknownScheme, scheme)
}
