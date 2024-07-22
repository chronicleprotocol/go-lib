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

func (f *retryFS) ReadFile(name string) ([]byte, error) {
	for retry := f.retryCount; retry > 0; retry-- {
		content, err := fs.ReadFile(f.fs, name)
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
	return nil, fmt.Errorf("exceeded %d retries for %s", f.retryCount, name)
}

func (f *retryFS) Open(name string) (fs.File, error) {
	return open(f, name)
}
