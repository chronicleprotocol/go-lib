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
	"io"
	"io/fs"
	"net/url"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryProto(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		mock     *mockProto
		url      *url.URL
		wantErr  bool
		wantData string
	}{
		{
			name:     "no error",
			mock:     &mockProto{fs: fstest.MapFS{"file.txt": &fstest.MapFile{Data: []byte("content")}}},
			url:      &url.URL{},
			wantData: "content",
		},
		{
			name:    "error on FileSystem",
			mock:    &mockProto{returns: errors.New("some error")},
			url:     &url.URL{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryProto := NewRetryProto(ctx, tt.mock, 3, 10*time.Millisecond)
			retryFS, _, err := retryProto.FileSystem(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			data, err := fs.ReadFile(retryFS, "file.txt")
			require.NoError(t, err)
			require.Equal(t, tt.wantData, string(data))
		})
	}
}

func TestRetryFS(t *testing.T) {
	ctx := context.Background()
	testFS := &failingFS{
		fs: fstest.MapFS{
			"file.txt":     &fstest.MapFile{Data: []byte("data")},
			"dir/sub.txt":  &fstest.MapFile{Data: []byte("subdata")},
			"dir/sub2.txt": &fstest.MapFile{Data: []byte("subdata2")},
		},
	}
	tc := []struct {
		name          string
		method        string
		file          string
		err           error
		errCount      int
		wantErr       bool
		wantResult    any
		wantCallCount int
	}{
		{
			name:          "open - success",
			method:        "Open",
			file:          "file.txt",
			wantResult:    []byte("data"),
			wantCallCount: 1,
		},
		{
			name:          "open - not exist",
			method:        "Open",
			file:          "notexist.txt",
			wantErr:       true,
			wantCallCount: 1,
		},
		{
			name:          "open - permission denied",
			method:        "Open",
			file:          "file.txt",
			err:           os.ErrPermission,
			wantErr:       true,
			wantCallCount: 1,
		},
		{
			name:          "open - path error",
			method:        "Open",
			file:          "/file.txt",
			wantErr:       true,
			wantCallCount: 1,
		},
		{
			name:          "open - retry success",
			method:        "Open",
			file:          "file.txt",
			err:           errors.New("transient"),
			errCount:      2,
			wantResult:    []byte("data"),
			wantCallCount: 3,
		},
		{
			name:          "open - retry fail",
			method:        "Open",
			file:          "file.txt",
			err:           errors.New("permanent"),
			errCount:      3,
			wantErr:       true,
			wantCallCount: 3,
		},
		{
			name:          "glob - success",
			method:        "Glob",
			file:          "dir/*",
			wantResult:    2,
			wantCallCount: 1,
		},
		{
			name:          "glob - bad pattern",
			method:        "Glob",
			file:          "[]",
			wantErr:       true,
			wantCallCount: 1,
		},
		{
			name:          "glob - retry success",
			method:        "Glob",
			file:          "dir/*",
			err:           errors.New("transient"),
			errCount:      2,
			wantResult:    2,
			wantCallCount: 3,
		},
		{
			name:          "glob - retry fail",
			method:        "Glob",
			file:          "dir/*",
			err:           errors.New("permanent"),
			errCount:      3,
			wantErr:       true,
			wantCallCount: 3,
		},
		{
			name:          "stat - success",
			method:        "Stat",
			file:          "file.txt",
			wantResult:    "file.txt",
			wantCallCount: 1,
		},
		{
			name:          "stat - not exist",
			method:        "Stat",
			file:          "notexist.txt",
			wantErr:       true,
			wantCallCount: 1,
		},
		{
			name:          "stat - retry success",
			method:        "Stat",
			file:          "file.txt",
			err:           errors.New("transient"),
			errCount:      2,
			wantResult:    "file.txt",
			wantCallCount: 3,
		},
		{
			name:          "stat - retry fail",
			method:        "Stat",
			file:          "file.txt",
			err:           errors.New("permanent"),
			errCount:      3,
			wantErr:       true,
			wantCallCount: 3,
		},
		{
			name:          "readfile - success",
			method:        "ReadFile",
			file:          "file.txt",
			wantResult:    []byte("data"),
			wantCallCount: 1,
		},
		{
			name:          "readfile - not exist",
			method:        "ReadFile",
			file:          "notexist.txt",
			wantErr:       true,
			wantCallCount: 1,
		},
		{
			name:          "readfile - retry success",
			method:        "ReadFile",
			file:          "file.txt",
			err:           errors.New("transient"),
			errCount:      2,
			wantResult:    []byte("data"),
			wantCallCount: 3,
		},
		{
			name:          "readfile - retry fail",
			method:        "ReadFile",
			file:          "file.txt",
			err:           errors.New("permanent"),
			errCount:      3,
			wantErr:       true,
			wantCallCount: 3,
		},
		{
			name:          "readdir - success",
			method:        "ReadDir",
			file:          "dir",
			wantResult:    2,
			wantCallCount: 1,
		},
		{
			name:          "readdir - not exist",
			method:        "ReadDir",
			file:          "notexist",
			wantErr:       true,
			wantCallCount: 1,
		},
		{
			name:          "readdir - retry success",
			method:        "ReadDir",
			file:          "dir",
			err:           errors.New("transient"),
			errCount:      2,
			wantResult:    2,
			wantCallCount: 3,
		},
		{
			name:          "readdir - retry fail",
			method:        "ReadDir",
			file:          "dir",
			err:           errors.New("permanent"),
			errCount:      3,
			wantErr:       true,
			wantCallCount: 3,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			testFS.err = tt.err
			testFS.callCount = 0
			testFS.errCount = tt.errCount
			if tt.err != nil && tt.errCount == 0 {
				testFS.errCount = 1
			}
			retryFS := NewRetryFS(ctx, testFS, 3, 10*time.Millisecond)
			switch tt.method {
			case "Open":
				f, err := retryFS.Open(tt.file)
				assert.Equal(t, tt.wantCallCount, testFS.callCount)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				defer f.Close()
				if tt.wantResult != nil {
					data, err := io.ReadAll(f)
					require.NoError(t, err)
					require.Equal(t, tt.wantResult.([]byte), data)
				}
			case "Glob":
				f, err := fs.Glob(retryFS, tt.file)
				assert.Equal(t, tt.wantCallCount, testFS.callCount)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Len(t, f, tt.wantResult.(int))
			case "Stat":
				s, err := fs.Stat(retryFS, tt.file)
				assert.Equal(t, tt.wantCallCount, testFS.callCount)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tt.wantResult.(string), s.Name())
			case "ReadFile":
				b, err := fs.ReadFile(retryFS, tt.file)
				assert.Equal(t, tt.wantCallCount, testFS.callCount)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tt.wantResult.([]byte), b)
			case "ReadDir":
				e, err := fs.ReadDir(retryFS, tt.file)
				assert.Equal(t, tt.wantCallCount, testFS.callCount)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Len(t, e, tt.wantResult.(int))
			}
		})
	}
}

type failingFS struct {
	fs        fs.FS
	err       error
	callCount int
	errCount  int
}

func (f *failingFS) Open(name string) (fs.File, error) {
	f.callCount++
	if f.errCount > 0 {
		f.errCount--
		return nil, f.err
	}
	return f.fs.Open(name)
}

func (f *failingFS) Glob(pattern string) ([]string, error) {
	f.callCount++
	if f.errCount > 0 {
		f.errCount--
		return nil, f.err
	}
	return fs.Glob(f.fs, pattern)
}

func (f *failingFS) Stat(name string) (fs.FileInfo, error) {
	f.callCount++
	if f.errCount > 0 {
		f.errCount--
		return nil, f.err
	}
	return fs.Stat(f.fs, name)
}

func (f *failingFS) ReadFile(name string) ([]byte, error) {
	f.callCount++
	if f.errCount > 0 {
		f.errCount--
		return nil, f.err
	}
	return fs.ReadFile(f.fs, name)
}

func (f *failingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	f.callCount++
	if f.errCount > 0 {
		f.errCount--
		return nil, f.err
	}
	return fs.ReadDir(f.fs, name)
}

func (f *failingFS) Sub(dir string) (fs.FS, error) {
	return fs.Sub(f.fs, dir)
}
