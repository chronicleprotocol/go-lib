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
	"fmt"
	"hash"
	"io"
	"io/fs"
	netURL "net/url"
	"strings"

	"github.com/defiweb/go-eth/types"
	"golang.org/x/crypto/sha3"
)

type ChecksumFSVerifyMode int

const (
	// ChecksumFSVerifyAfterRead verifies the checksum after reading the file
	// contents.
	ChecksumFSVerifyAfterRead ChecksumFSVerifyMode = iota

	// ChecksumFSVerifyAfterOpen verifies the checksum immediately after
	// opening the file.
	ChecksumFSVerifyAfterOpen
)

type ChecksumFSOption func(*checksumFS)

// WithChecksumParamName sets the name of the URL query parameter that contains
// the checksum value. The default parameter name is "checksum".
func WithChecksumParamName(name string) ChecksumFSOption {
	return func(c *checksumFS) {
		c.param = name
	}
}

// WithChecksumHash sets the hash function used to compute the checksum. The
// default hash function is LegacyKeccak256.
func WithChecksumHash(hash func() hash.Hash) ChecksumFSOption {
	return func(c *checksumFS) {
		c.hash = hash
	}
}

// NewChecksumProto creates a new checksum protocol.
func NewChecksumProto(proto Protocol, opts ...ChecksumFSOption) Protocol {
	return &checksumProto{proto: proto, opts: opts}
}

// WithChecksumVerifyMode sets the mode of the checksum verification. The
// default mode is ChecksumFSVerifyAfterRead.
func WithChecksumVerifyMode(mode ChecksumFSVerifyMode) ChecksumFSOption {
	return func(c *checksumFS) {
		c.mode = mode
	}
}

type checksumProto struct {
	proto Protocol
	opts  []ChecksumFSOption
}

// FileSystem implements the Protocol interface.
func (c *checksumProto) FileSystem(url *netURL.URL) (fs fs.FS, path string, err error) {
	fs, path, err = c.proto.FileSystem(url)
	if err != nil {
		return nil, "", errChecksumProtoFn(err)
	}
	fs, err = NewChecksumFS(fs, c.opts...)
	if err != nil {
		return nil, "", errChecksumProtoFn(err)
	}
	return
}

// NewChecksumFS creates a new checksum file system.
//
// The file system wraps an existing file system, computes the checksum of the
// file contents, and compares it with the provided checksum. The checksum must
// be provided in the file name as a query parameter, e.g.,
// "file?checksum=0x1234...".
//
// If the checksum does not match, the file system returns an error when
// reading the file.
func NewChecksumFS(fs fs.FS, opts ...ChecksumFSOption) (fs.FS, error) {
	c := &checksumFS{fs: fs}
	for _, opt := range opts {
		opt(c)
	}
	if c.param == "" {
		c.param = "checksum"
	}
	if c.hash == nil {
		c.hash = sha3.NewLegacyKeccak256
	}
	if c.mode < 0 || c.mode > ChecksumFSVerifyAfterOpen {
		return nil, errChecksumFSUnsupportedMode
	}
	return c, nil
}

type checksumFS struct {
	fs    fs.FS
	hash  func() hash.Hash
	param string
	mode  ChecksumFSVerifyMode
}

func (c *checksumFS) Open(n string) (fs.File, error) {
	n, h := c.checksumParam(n)
	f, err := c.fs.Open(n)
	if err != nil {
		return nil, errChecksumFSFn(err)
	}
	if h == types.ZeroHash {
		return f, nil
	}
	switch c.mode {
	case ChecksumFSVerifyAfterRead:
		return checksumFile{file: f, checksum: h, hash: c.hash()}, nil
	case ChecksumFSVerifyAfterOpen:
		stat, err := f.Stat()
		if err != nil {
			return nil, errChecksumFSFn(err)
		}
		cfile := checksumFile{file: f, checksum: h, hash: c.hash()}
		data, err := io.ReadAll(cfile)
		if err != nil {
			return nil, errChecksumFSFn(err)
		}
		return &file{
			reader: io.NopCloser(bytes.NewReader(data)),
			info:   stat,
		}, nil
	default:
		return nil, errChecksumFSUnsupportedMode
	}
}

// Glob implements the fs.FS interface.
func (c *checksumFS) Glob(pattern string) ([]string, error) {
	return fs.Glob(c, pattern)
}

// Stat implements the fs.FS interface.
func (c *checksumFS) Stat(name string) (fs.FileInfo, error) {
	return fs.Stat(c, name)
}

// ReadFile implements the fs.ReadFileFS interface.
func (c *checksumFS) ReadFile(name string) ([]byte, error) {
	f, err := c.Open(name)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(f)
}

// ReadDir implements the fs.ReadDirFS interface.
func (c *checksumFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return fs.ReadDir(c, name)
}

// checksumParam extracts the checksum value from the file name and returns the
// file name without the checksum parameter.
func (c *checksumFS) checksumParam(name string) (string, types.Hash) {
	q := strings.Index(name, "?")
	if q == -1 {
		return name, types.ZeroHash
	}
	v, err := netURL.ParseQuery(name[q+1:])
	if err != nil {
		return name, types.ZeroHash
	}
	h, err := types.HashFromHex(v.Get(c.param), types.PadNone)
	if err != nil {
		return name, types.ZeroHash
	}
	v.Del(c.param)
	if len(v) == 0 {
		return name[:q], h
	}
	return name[:q] + "?" + v.Encode(), h
}

// checksumFile computes the checksum of the file contents and
// compares it with the known checksum. The checksum is computed on the fly
// while reading the file contents and is compared with the known checksum when
// the read operation is complete.
type checksumFile struct {
	file     fs.File
	hash     hash.Hash
	checksum types.Hash
}

// Stat implements the fs.File interface.
func (c checksumFile) Stat() (fs.FileInfo, error) {
	return c.file.Stat()
}

// Read implements the fs.File interface.
func (c checksumFile) Read(b []byte) (int, error) {
	n, err := c.file.Read(b)
	if errors.Is(err, io.EOF) {
		if c.checksum != c.calcChecksum() {
			return 0, errChecksumFSMismatch
		}
		return 0, io.EOF
	}
	if err != nil {
		return 0, err
	}
	c.hash.Write(b[:n])
	return n, nil
}

// Close implements the fs.File interface.
func (c checksumFile) Close() error {
	return c.file.Close()
}

func (c checksumFile) calcChecksum() types.Hash {
	return types.Hash(c.hash.Sum(nil))
}

var (
	errChecksumFSUnsupportedMode = errors.New("fsutil.checksumFS: unsupported verify mode")
	errChecksumFSMismatch        = errors.New("fsutil.checksumFS: checksum mismatch")
)

func errChecksumProtoFn(err error) error {
	return fmt.Errorf("fsutil.checksumProto: %w", err)
}

func errChecksumFSFn(err error) error {
	return fmt.Errorf("fsutil.checksumFS: %w", err)
}
