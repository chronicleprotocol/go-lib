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
	"errors"
	"fmt"
	"io/fs"
	netURL "net/url"
	"os"
)

type FileOption func(*fileProto)

// WithFileWorkingDir sets the working directory for the file protocol.
func WithFileWorkingDir(wd string) FileOption {
	return func(f *fileProto) {
		f.wd = wd
	}
}

// NewFileProto creates a new file protocol that uses the local filesystem.
// The URI scheme must be "file" and the host must be empty or "localhost".
// The working directory is set to "/" by default.
func NewFileProto(opts ...FileOption) Protocol {
	f := &fileProto{}
	for _, opt := range opts {
		opt(f)
	}
	if f.wd == "" {
		f.wd = "/"
	}
	return f
}

type fileProto struct {
	wd string
}

// FileSystem implements the Protocol interface.
func (m *fileProto) FileSystem(url *netURL.URL) (fs fs.FS, path string, err error) {
	if url == nil {
		return nil, "", errFileNilURI
	}
	if url.Scheme != "file" {
		return nil, "", errFileUnexpectedSchemeFn(url.Scheme)
	}
	if url.Host != "" && url.Host != "localhost" {
		return nil, "", errFileUnexpectedHostFn(url.Host)
	}
	return os.DirFS(m.wd), uriPath(url, true), nil
}

var errFileNilURI = errors.New("fsutil.fileProto: nil URI")

func errFileUnexpectedSchemeFn(scheme string) error {
	return fmt.Errorf("fsutil.fileProto: unexpected scheme: %s", scheme)
}

func errFileUnexpectedHostFn(host string) error {
	return fmt.Errorf("fsutil.fileProto: unexpected host: %s, must be empty or 'localhost'", host)
}
