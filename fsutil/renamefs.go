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
	"io/fs"
	"net/url"
)

func NewRenameFS(fs fs.FS, rename func(url.URL) string) fs.FS {
	if rename == nil {
		return fs
	}
	return &renameFS{fs: fs, rename: rename}
}

// renameFS is a fs.FS that renames the URL before passing it to the underlying fs.FS
type renameFS struct {
	fs     fs.FS
	rename func(url.URL) string
}

func (f *renameFS) ReadFile(name string) ([]byte, error) {
	if f.rename == nil {
		return fs.ReadFile(f.fs, name)
	}
	u, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(f.fs, f.rename(*u))
}

func (f *renameFS) Open(name string) (fs.File, error) {
	return open(f, name)
}
