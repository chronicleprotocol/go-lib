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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	netURL "net/url"
	"os"
	"path"
)

type CacheFSOption func(*cacheFS)

// WithCacheDir sets the cache directory.
func WithCacheDir(dir string) CacheFSOption {
	return func(c *cacheFS) {
		c.dir = dir
	}
}

// NewCacheProto creates a new cache protocol.
//
// The cache protocol will wrap the filesystem returned by a given protocol
// with a cache filesystem.
func NewCacheProto(proto Protocol, opts ...CacheFSOption) Protocol {
	return &cacheProto{proto: proto, opts: opts}
}

type cacheProto struct {
	proto Protocol
	opts  []CacheFSOption
}

// FileSystem implements the Protocol interface.
func (c *cacheProto) FileSystem(url *netURL.URL) (fs fs.FS, path string, err error) {
	if url == nil {
		return nil, "", errCacheProtoNilURI
	}
	fs, path, err = c.proto.FileSystem(url)
	if err != nil {
		return nil, "", errCacheProtoFn(err)
	}
	fs, err = NewCacheFS(fs, c.opts...)
	if err != nil {
		return nil, "", errCacheProtoFn(err)
	}
	return
}

// NewCacheFS creates a new cache filesystem.
//
// The cache filesystem caches the contents of the files in the cache directory.
// If the file is not found in the cache, it will be read from the underlying
// file system and cached.
func NewCacheFS(fs fs.FS, opts ...CacheFSOption) (fs.FS, error) {
	c := &cacheFS{fs: fs}
	for _, opt := range opts {
		opt(c)
	}
	if c.dir == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return nil, errCacheFSFn(err)
		}
		dir = path.Join(dir, "suite")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errCacheFSFn(err)
		}
		c.dir = dir
	}
	return c, nil
}

type cacheFS struct {
	fs  fs.FS
	dir string
}

// Open implements the fs.Open interface.
func (c *cacheFS) Open(name string) (fs.File, error) {
	if err := validPath("open", name); err != nil {
		return nil, errCacheFSFn(err)
	}
	if f, err := c.cacheOpen(name); err == nil {
		return f, nil
	}
	f, err := c.fs.Open(name)
	if err != nil {
		return nil, errCacheFSFn(err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, errCacheFSFn(err)
	}
	if err := f.Close(); err != nil {
		return nil, errCacheFSFn(err)
	}
	if err := c.cacheWrite(name, b); err != nil {
		return nil, errCacheFSFn(err)
	}
	f, err = c.cacheOpen(name)
	if err != nil {
		return nil, errCacheFSFn(err)
	}
	return f, nil
}

// Glob implements the fs.Glob interface.
func (c *cacheFS) Glob(pattern string) ([]string, error) {
	if err := validPattern("glob", pattern); err != nil {
		return nil, errCacheFSFn(err)
	}
	return fs.Glob(c.fs, pattern)
}

// Stat implements the fs.Stat interface.
func (c *cacheFS) Stat(name string) (fs.FileInfo, error) {
	if err := validPath("stat", name); err != nil {
		return nil, errCacheFSFn(err)
	}
	return fs.Stat(c.fs, name)
}

// ReadFile implements the fs.ReadFile interface.
func (c *cacheFS) ReadFile(name string) ([]byte, error) {
	if err := validPath("readFile", name); err != nil {
		return nil, errCacheFSFn(err)
	}
	if f, err := c.cacheOpen(name); err == nil {
		b, err := io.ReadAll(f)
		if err != nil {
			return nil, errCacheFSFn(err)
		}
		return b, nil
	}
	b, err := fs.ReadFile(c.fs, name)
	if err != nil {
		return nil, errCacheFSFn(err)
	}
	if err := c.cacheWrite(name, b); err != nil {
		return nil, errCacheFSFn(err)
	}
	b, err = c.cacheRead(name)
	if err != nil {
		return nil, errCacheFSFn(err)
	}
	return b, nil
}

// ReadDir implements the fs.ReadDir interface.
func (c *cacheFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if err := validPath("readDir", name); err != nil {
		return nil, errCacheFSFn(err)
	}
	return fs.ReadDir(c.fs, name)
}

// Sub implements the fs.Sub interface.
func (c *cacheFS) Sub(name string) (fs.FS, error) {
	if err := validPath("sub", name); err != nil {
		return nil, errCacheFSFn(err)
	}
	return fs.Sub(c.fs, name)
}

// cacheOpen opens a file in the cache directory.
func (c *cacheFS) cacheOpen(name string) (fs.File, error) {
	f, err := os.Open(c.cachePath(name))
	if err != nil {
		return nil, err
	}
	return f, nil
}

// cacheRead reads a file from the cache directory.
func (c *cacheFS) cacheRead(name string) ([]byte, error) {
	return os.ReadFile(c.cachePath(name))
}

func (c *cacheFS) cacheWrite(name string, content []byte) error {
	f, err := os.Create(c.cachePath(name))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(content)
	return err
}

func (c *cacheFS) cachePath(name string) string {
	hash := sha1.New()
	hash.Write([]byte(name))
	return path.Join(c.dir, hex.EncodeToString(hash.Sum(nil)))
}

var errCacheProtoNilURI = fmt.Errorf("fsutil.cacheProto: nil URI")

func errCacheProtoFn(err error) error {
	return fmt.Errorf("fsutil.cacheProto: %w", err)
}

func errCacheFSFn(err error) error {
	return fmt.Errorf("fsutil.cacheFS: %w", err)
}
