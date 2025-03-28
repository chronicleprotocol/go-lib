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
	"fmt"
	"io/fs"
	"time"
)

//TODO: RandomNameFS could be a ChainFS with static shuffled slice of FSes but it would always use same ordering.

func NewRetryFS(ctx context.Context, fs fs.FS, retryCount int, retryDelay time.Duration) fs.FS {
	return &retryFS{
		ctx:        ctx,
		fs:         fs,
		retryCount: retryCount,
		retryDelay: retryDelay,
	}
}

type retryFS struct {
	ctx context.Context
	fs  fs.FS

	retryCount int
	retryDelay time.Duration
}

func (f *retryFS) ReadFile(name string) (content []byte, err error) {
	for retry := f.retryCount; retry > 0; retry-- {
		content, err = fs.ReadFile(f.fs, name)
		if err != nil {
			select {
			case <-f.ctx.Done():
				return nil, f.ctx.Err()
			case <-time.After(f.retryDelay):
				continue
			}
		}
		return content, nil
	}
	return nil, fmt.Errorf("exceeded %d retries for %s: %w", f.retryCount, name, err)
}

func (f *retryFS) Open(name string) (fs.File, error) {
	return open(f, name)
}
