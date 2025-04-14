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
	"context"
	"errors"
	"fmt"
	"io/fs"
	netURL "net/url"
	"os"
	"path"
	"time"

	"github.com/chronicleprotocol/suite/pkg/util/retry"
)

// NewRetryProto creates a new retry protocol.
//
// The retry protocol will wrap the filesystem returned by a given protocol
// with a retry filesystem.
func NewRetryProto(ctx context.Context, proto Protocol, attempts int, delay time.Duration) Protocol {
	return &retryProto{ctx: ctx, proto: proto, attempts: attempts, delay: delay}
}

type retryProto struct {
	ctx      context.Context
	proto    Protocol
	attempts int
	delay    time.Duration
}

// FileSystem implements the Protocol interface.
func (m *retryProto) FileSystem(uri *netURL.URL) (fs fs.FS, path string, err error) {
	if uri == nil {
		return nil, "", errRetryProtoNilURI
	}
	fs, path, err = m.proto.FileSystem(uri)
	if err != nil {
		return nil, "", errRetryProtoFn(err)
	}
	fs = NewRetryFS(m.ctx, fs, m.attempts, m.delay)
	return
}

type retryFS struct {
	ctx      context.Context
	fs       fs.FS
	attempts int
	delay    time.Duration
}

// NewRetryFS wraps the given FS to add retry functionality.
func NewRetryFS(ctx context.Context, fs fs.FS, attempts int, delay time.Duration) fs.FS {
	return &retryFS{ctx: ctx, fs: fs, attempts: attempts, delay: delay}
}

// Open implements the fs.Open interface.
func (r *retryFS) Open(name string) (f fs.File, err error) {
	return retry.Try2(r.ctx, func(_ context.Context) (fs.File, error, bool) {
		f, err = r.fs.Open(name)
		if err == nil {
			return f, nil, retry.Stop
		}
		if !isRetryable(err) {
			return nil, errRetryFSFn(err), retry.Stop
		}
		return f, errRetryFSFn(err), retry.TryAgain
	}, r.attempts, r.delay)
}

// Glob implements the fs.Glob interface.
func (r *retryFS) Glob(pattern string) ([]string, error) {
	return retry.Try2(r.ctx, func(_ context.Context) (f []string, err error, ok bool) {
		f, err = fs.Glob(r.fs, pattern)
		if err == nil {
			return f, nil, retry.Stop
		}
		if !isRetryable(err) {
			return nil, errRetryFSFn(err), retry.Stop
		}
		return f, errRetryFSFn(err), retry.TryAgain
	}, r.attempts, r.delay)
}

// Stat implements the fs.Stat interface.
func (r *retryFS) Stat(name string) (fs.FileInfo, error) {
	return retry.Try2(r.ctx, func(_ context.Context) (f fs.FileInfo, err error, ok bool) {
		f, err = fs.Stat(r.fs, name)
		if err == nil {
			return f, nil, retry.Stop
		}
		if !isRetryable(err) {
			return nil, errRetryFSFn(err), retry.Stop
		}
		return f, errRetryFSFn(err), retry.TryAgain
	}, r.attempts, r.delay)
}

// ReadFile implements the fs.ReadFile interface.
func (r *retryFS) ReadFile(name string) ([]byte, error) {
	return retry.Try2(r.ctx, func(_ context.Context) (b []byte, err error, ok bool) {
		b, err = fs.ReadFile(r.fs, name)
		if err == nil {
			return b, nil, retry.Stop
		}
		if !isRetryable(err) {
			return nil, errRetryFSFn(err), retry.Stop
		}
		return b, errRetryFSFn(err), retry.TryAgain
	}, r.attempts, r.delay)
}

// ReadDir implements the fs.ReadDir interface.
func (r *retryFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return retry.Try2(r.ctx, func(_ context.Context) (e []fs.DirEntry, err error, ok bool) {
		e, err = fs.ReadDir(r.fs, name)
		if err == nil {
			return e, nil, retry.Stop
		}
		if !isRetryable(err) {
			return nil, errRetryFSFn(err), retry.Stop
		}
		return e, errRetryFSFn(err), retry.TryAgain
	}, r.attempts, r.delay)
}

// Sub implements the fs.Sub interface.
func (r *retryFS) Sub(dir string) (fs.FS, error) {
	return fs.Sub(r.fs, dir)
}

func isRetryable(err error) bool {
	return !errors.Is(err, os.ErrNotExist) && !errors.Is(err, os.ErrPermission) && !errors.Is(err, path.ErrBadPattern) && !isPathError(err)
}

var errRetryProtoNilURI = errors.New("fsutil.retryProto: nil URI")

func errRetryProtoFn(err error) error {
	return fmt.Errorf("fsutil.retryProto: %w", err)
}

func errRetryFSFn(err error) error {
	return fmt.Errorf("fsutil.retryFS: %w", err)
}
