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
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	netURL "net/url"
	"strings"

	"github.com/chronicleprotocol/go-lib/errutil"
)

const (
	defaultGzipExt       = "gz"
	defaultGzipReadLimit = 1024 * 1024 * 128 // 128MiB
)

type GzipFSOption func(*gzipFS)

// WithGzipReadLimit sets the maximum size of the decompressed data.
// If the decompressed data exceeds the limit, the io.ErrUnexpectedEOF error
// will be returned. The default limit is 128MiB.
func WithGzipReadLimit(limit int64) GzipFSOption {
	return func(c *gzipFS) {
		c.readLimit = limit
	}
}

// WithGzipCheckExtension enables or disables checking the file extension
// to determine whether to decompress the file. If enabled, only files with
// the specified extensions will be decompressed.
func WithGzipCheckExtension(check bool) GzipFSOption {
	return func(c *gzipFS) {
		c.checkExt = check
	}
}

// WithGzipExtensions sets the list of file extensions that will be decompressed.
// The default extension is "gz".
// Ignored if WithGzipCheckExtension is set to false.
func WithGzipExtensions(exts ...string) GzipFSOption {
	return func(c *gzipFS) {
		c.exts = exts
	}
}

// NewGzipProto creates a new gzip protocol.
//
// The gzip protocol will wrap the filesystem returned by a given protocol
// with a gzip filesystem.
func NewGzipProto(proto Protocol, opts ...GzipFSOption) Protocol {
	return &gzipProto{proto: proto, opts: opts}
}

type gzipProto struct {
	proto Protocol
	opts  []GzipFSOption
}

// FileSystem implements the Protocol interface.
func (m *gzipProto) FileSystem(uri *netURL.URL) (fs fs.FS, path string, err error) {
	if uri == nil {
		return nil, "", errGzipProtoNilURI
	}
	fs, path, err = m.proto.FileSystem(uri)
	if err != nil {
		return nil, "", errGzipProtoFn(err)
	}
	fs = NewGzipFS(fs, m.opts...)
	return
}

// NewGzipFS creates a new gzip filesystem.
//
// The gzip filesystem will wrap the given filesystem and add gzip
// decompression functionality.
func NewGzipFS(fs fs.FS, opts ...GzipFSOption) fs.FS {
	c := &gzipFS{
		fs:        fs,
		readLimit: defaultGzipReadLimit,
		checkExt:  true,
		exts:      []string{defaultGzipExt},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type gzipFS struct {
	fs        fs.FS
	readLimit int64
	checkExt  bool
	exts      []string
}

// Open implements the fs.FS interface.
func (c *gzipFS) Open(name string) (fs.File, error) {
	if err := validPath("open", name); err != nil {
		return nil, errGzipFSFn(err)
	}
	f, err := c.fs.Open(name)
	if err != nil {
		return nil, errGzipFSFn(err)
	}
	if !c.shouldDecompress(name) {
		return c.fs.Open(name)
	}
	return newGzipFile(f, c.readLimit)
}

// Glob implements the fs.GlobFS interface.
func (c *gzipFS) Glob(pattern string) ([]string, error) {
	if err := validPattern("glob", pattern); err != nil {
		return nil, errGzipFSFn(err)
	}
	return fs.Glob(c.fs, pattern)
}

// Stat implements the fs.StatFS interface.
func (c *gzipFS) Stat(name string) (fs.FileInfo, error) {
	if err := validPath("stat", name); err != nil {
		return nil, errGzipFSFn(err)
	}
	return fs.Stat(c.fs, name)
}

// ReadFile implements the fs.ReadFileFS interface.
func (c *gzipFS) ReadFile(name string) ([]byte, error) {
	if err := validPath("readFile", name); err != nil {
		return nil, errGzipFSFn(err)
	}
	if !c.shouldDecompress(name) {
		b, err := fs.ReadFile(c.fs, name)
		if err != nil {
			return nil, errGzipFSFn(err)
		}
		return b, nil
	}
	f, err := c.Open(name)
	if err != nil {
		return nil, errGzipFSFn(err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, errGzipFSFn(err)
	}
	if err := f.Close(); err != nil {
		return nil, errGzipFSFn(err)
	}
	return b, nil
}

// ReadDir implements the fs.ReadDirFS interface.
func (c *gzipFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if err := validPath("readDir", name); err != nil {
		return nil, errGzipFSFn(err)
	}
	return fs.ReadDir(c.fs, name)
}

func (c *gzipFS) shouldDecompress(name string) bool {
	if !c.checkExt {
		return true
	}
	for _, ext := range c.exts {
		if strings.HasSuffix(name, "."+ext) {
			return true
		}
	}
	return false
}

type gzipFile struct {
	f   fs.File
	g   io.ReadCloser
	n   int64 // bytes remaining
	err error
}

func newGzipFile(f fs.File, n int64) (*gzipFile, error) {
	g, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	return &gzipFile{f: f, g: g, n: n}, nil
}

// Stat implements the fs.File interface.
func (c *gzipFile) Stat() (fs.FileInfo, error) {
	return c.f.Stat()
}

// Read implements the fs.File interface.
func (c *gzipFile) Read(p []byte) (n int, err error) {
	if c.err != nil {
		return 0, c.err
	}
	if c.n <= 0 {
		if _, err := c.g.Read(make([]byte, 1)); errors.Is(err, io.EOF) {
			c.err = io.EOF
			return 0, c.err
		}
		c.err = io.ErrUnexpectedEOF
		return 0, c.err
	}
	if int64(len(p)) > c.n {
		p = p[0:c.n]
	}
	n, err = c.g.Read(p)
	if err != nil {
		c.err = err
	}
	c.n -= int64(n)
	return n, err
}

// Close implements the fs.File interface.
func (c *gzipFile) Close() error {
	var err error
	if cErr := c.g.Close(); cErr != nil {
		err = errutil.Append(err, cErr)
	}
	if cErr := c.f.Close(); cErr != nil {
		err = errutil.Append(err, cErr)
	}
	if err != nil {
		return err
	}
	return nil
}

var errGzipProtoNilURI = errors.New("fsutil.gzipProto: nil URI")

func errGzipProtoFn(err error) error {
	return fmt.Errorf("fsutil.gzipProto: %w", err)
}

func errGzipFSFn(err error) error {
	return fmt.Errorf("fsutil.gzipFS: %w", err)
}
