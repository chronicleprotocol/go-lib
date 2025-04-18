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
	"math/rand/v2"
	netURL "net/url"
	"strings"

	"github.com/chronicleprotocol/suite/pkg/util/errutil"
	"github.com/chronicleprotocol/suite/pkg/util/sliceutil"
)

type ChainFSOption func(*chainFS)

// WithChainFilesystems sets the file systems to chain.
func WithChainFilesystems(fs ...fs.FS) ChainFSOption {
	return func(c *chainFS) {
		c.fs = append(c.fs, fs...)
	}
}

// WithChainRandOrder sets the file systems to chain in random order.
func WithChainRandOrder() ChainFSOption {
	return func(c *chainFS) {
		c.rand = true
	}
}

// NewChainProto creates a new chain protocol.
func NewChainProto(opts ...ChainFSOption) Protocol {
	return &chainProto{opts: opts}
}

type chainProto struct {
	opts []ChainFSOption
}

// FileSystem implements the Protocol interface.
func (c *chainProto) FileSystem(url *netURL.URL) (fs fs.FS, path string, err error) {
	if url == nil {
		return nil, "", errChainProtoNilURI
	}
	return NewChainFS(c.opts...), uriPath(url, true), nil
}

// NewChainFS creates a new chain filesystem.
//
// The chain filesystem chains multiple file systems together. It will try to
// open a file in the first file system. If it fails, it will try the next one,
// and so on. If all file systems fail, it will return an error.
func NewChainFS(opts ...ChainFSOption) fs.FS {
	f := &chainFS{}
	for _, opt := range opts {
		opt(f)
	}
	return f
}

type chainFS struct {
	fs   []fs.FS
	rand bool
}

// Open implements the fs.Open interface.
func (c *chainFS) Open(name string) (fs.File, error) {
	if err := validPath("open", name); err != nil {
		return nil, errChainFSFn(err)
	}
	var err error
	for i := range c.iter() {
		f, fErr := c.fs[i].Open(name)
		if fErr == nil {
			return f, nil
		}
		err = errutil.Append(err, fErr)
	}
	return nil, errChainFSFn(err)
}

// Glob implements the fs.Glob interface.
func (c *chainFS) Glob(pattern string) (s []string, err error) {
	if err := validPattern("glob", pattern); err != nil {
		return nil, errChainFSFn(err)
	}
	for i := range c.iter() {
		f, fErr := fs.Glob(c.fs[i], pattern)
		if fErr != nil {
			return nil, errChainFSFn(fErr)
		}
		s = sliceutil.AppendUniqueSort(s, f...)
	}
	return s, nil
}

// Stat implements the fs.Stat interface.
func (c *chainFS) Stat(name string) (fs.FileInfo, error) {
	if err := validPath("stat", name); err != nil {
		return nil, errChainFSFn(err)
	}
	var err error
	for i := range c.iter() {
		f, fErr := fs.Stat(c.fs[i], name)
		if fErr == nil {
			return f, nil
		}
		err = errutil.Append(err, fErr)
	}
	return nil, errChainFSFn(err)
}

// ReadFile implements the fs.ReadFile interface.
func (c *chainFS) ReadFile(name string) ([]byte, error) {
	if err := validPath("readFile", name); err != nil {
		return nil, errChainFSFn(err)
	}
	var err error
	for i := range c.iter() {
		f, fErr := fs.ReadFile(c.fs[i], name)
		if fErr == nil {
			return f, nil
		}
		err = errutil.Append(err, fErr)
	}
	return nil, errChainFSFn(err)
}

// ReadDir implements the fs.ReadDir interface.
func (c *chainFS) ReadDir(name string) (s []fs.DirEntry, err error) {
	if err := validPath("readDir", name); err != nil {
		return nil, errChainFSFn(err)
	}
	for i := range c.iter() {
		f, fErr := fs.ReadDir(c.fs[i], name)
		if fErr != nil {
			err = errutil.Append(err, fErr)
			continue
		}
		s = sliceutil.AppendUniqueSortFunc(s, func(a, b fs.DirEntry) int {
			return strings.Compare(a.Name(), b.Name())
		}, f...)
	}
	if err != nil {
		return nil, errChainFSFn(err)
	}
	return s, nil
}

// Sub implements the fs.Sub interface.
func (c *chainFS) Sub(name string) (fs.FS, error) {
	if err := validPath("sib", name); err != nil {
		return nil, errChainFSFn(err)
	}
	var err error
	for i := range c.iter() {
		f, fErr := fs.Sub(c.fs[i], name)
		if fErr == nil {
			return f, nil
		}
		err = errutil.Append(err, fErr)
	}
	return nil, errChainFSFn(err)
}

func (c *chainFS) iter() []int {
	if c.rand {
		return rand.Perm(len(c.fs))
	}
	i := make([]int, len(c.fs))
	for n := range c.fs {
		i[n] = n
	}
	return i
}

var errChainProtoNilURI = fmt.Errorf("fsutil.chainProto: nil URI")

func errChainFSFn(err error) error {
	return fmt.Errorf("fsutil.chainFS: %w", err)
}
