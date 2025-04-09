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
	"io"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheFS(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cachefs_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	tc := []struct {
		name     string
		method   string
		file     string
		wantData string
	}{
		{
			name:     "open file",
			method:   "Open",
			file:     "file.txt",
			wantData: "data",
		},
		{
			name:     "read file",
			method:   "ReadFile",
			file:     "file.txt",
			wantData: "data",
		},
		{
			name:     "read nested file",
			method:   "ReadFile",
			file:     "dir/nested.txt",
			wantData: "nested data",
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			testFS := fstest.MapFS{
				"file.txt":       &fstest.MapFile{Data: []byte("data")},
				"dir/nested.txt": &fstest.MapFile{Data: []byte("nested data")},
			}
			cacheFS, err := NewCacheFS(testFS, WithCacheDir(tempDir))
			require.NoError(t, err)
			switch tt.method {
			case "Open":
				// First access - should read from source FS
				f, err := cacheFS.Open(tt.file)
				require.NoError(t, err)
				data, err := io.ReadAll(f)
				require.NoError(t, err)
				f.Close()
				assert.Equal(t, tt.wantData, string(data))

				// Clear content of the file to see that next file read will
				// be from cache
				for k, v := range testFS {
					if k == tt.file {
						v.Data = []byte("")
					}
				}

				// Second access - should read from cache
				f, err = cacheFS.Open(tt.file)
				require.NoError(t, err)
				data, err = io.ReadAll(f)
				require.NoError(t, err)
				f.Close()
				assert.Equal(t, tt.wantData, string(data))

			case "ReadFile":
				// First access - should read from source
				data, err := fs.ReadFile(cacheFS, tt.file)
				require.NoError(t, err)
				assert.Equal(t, tt.wantData, string(data))

				// Clear content of the file to see that next file read will
				// be from cache
				for k, v := range testFS {
					if k == tt.file {
						v.Data = []byte("")
					}
				}

				// Second access - should read from cache
				data, err = fs.ReadFile(cacheFS, tt.file)
				require.NoError(t, err)
				assert.Equal(t, tt.wantData, string(data))
			}
		})
	}
}
