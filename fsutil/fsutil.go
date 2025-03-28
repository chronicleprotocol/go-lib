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
	"bytes"
	"errors"
	"io/fs"
	"time"
)

//TODO: Add content hash verification with a provided hash `func (content []byte,hash types.Hash) (bool)`

type fileInfo struct {
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	isDir   bool
	sys     interface{}
}

func (i *fileInfo) Name() string       { return i.name }
func (i *fileInfo) Size() int64        { return i.size }
func (i *fileInfo) Mode() fs.FileMode  { return i.mode }
func (i *fileInfo) ModTime() time.Time { return i.modTime }
func (i *fileInfo) IsDir() bool        { return i.isDir }
func (i *fileInfo) Sys() interface{}   { return i.sys }

type file struct {
	reader *bytes.Reader
	info   fs.FileInfo
}

func (f *file) Stat() (fs.FileInfo, error)       { return f.info, nil }
func (f *file) Read(p []byte) (n int, err error) { return f.reader.Read(p) }
func (f *file) Close() error                     { return nil }
func (f *file) ReadDir(n int) ([]fs.DirEntry, error) {
	return nil, errors.New("directories not supported")
}

// open
// Contrary to fs package, the downloading FS is mainly based on the ReadFile method.
// The Open function is not used in the downloading FS but is supported and is calling ReadFile method.
func open(f fs.FS, name string) (fs.File, error) {
	content, err := fs.ReadFile(f, name)
	if err != nil {
		return nil, err
	}
	return &file{
		reader: bytes.NewReader(content),
		info: &fileInfo{
			name:    name,
			size:    int64(len(content)),
			mode:    0,
			modTime: time.Now(),
			isDir:   false,
			sys:     nil,
		},
	}, nil
}
