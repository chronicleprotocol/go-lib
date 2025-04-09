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
	"io"
	"io/fs"
	"net/url"
	"time"
)

// Protocol defines a file system protocol. It provides a file system instance
// and a path within that file system for a given URI.
//
// The returned file system always points to the highest possible level
// directory.
type Protocol interface {
	FileSystem(uri *url.URL) (fs fs.FS, path string, err error)
}

// fileInfo implements the fs.FileInfo interface.
type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
	sys     any
}

func (i *fileInfo) Name() string       { return i.name }
func (i *fileInfo) Size() int64        { return i.size }
func (i *fileInfo) Mode() fs.FileMode  { return i.mode }
func (i *fileInfo) ModTime() time.Time { return i.modTime }
func (i *fileInfo) IsDir() bool        { return i.isDir }
func (i *fileInfo) Sys() any           { return i.sys }

// file implements the fs.File interface.
type file struct {
	reader io.ReadCloser
	info   fs.FileInfo
}

func (f *file) Stat() (fs.FileInfo, error)           { return f.info, nil }
func (f *file) Read(p []byte) (n int, err error)     { return f.reader.Read(p) }
func (f *file) Close() error                         { return f.reader.Close() }
func (f *file) ReadDir(_ int) ([]fs.DirEntry, error) { return nil, errReadDirUnsupported }

var errReadDirUnsupported = errors.New("fsutil.file: ReadDir not supported")

func isPathError(err error) bool {
	var e *fs.PathError
	return errors.As(err, &e)
}
