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
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainFS(t *testing.T) {
	testFS1 := fstest.MapFS{
		"file1.txt":    &fstest.MapFile{Data: []byte("data1")},
		"shared.txt":   &fstest.MapFile{Data: []byte("fs1")},
		"dir/sub1.txt": &fstest.MapFile{Data: []byte("subdata1")},
	}
	testFS2 := fstest.MapFS{
		"file2.txt":    &fstest.MapFile{Data: []byte("data2")},
		"shared.txt":   &fstest.MapFile{Data: []byte("fs2")},
		"dir/sub2.txt": &fstest.MapFile{Data: []byte("subdata2")},
	}
	tc := []struct {
		name      string
		method    string
		fs        []fs.FS
		file      string
		randOrder bool
		wantErr   bool
		wantData  string
		wantLen   int
	}{
		{
			name:     "open - file in first fs",
			method:   "Open",
			fs:       []fs.FS{testFS1, testFS2},
			file:     "file1.txt",
			wantData: "data1",
		},
		{
			name:     "open - file in second fs",
			method:   "Open",
			fs:       []fs.FS{testFS1, testFS2},
			file:     "file2.txt",
			wantData: "data2",
		},
		{
			name:     "open - shared file (first fs wins)",
			method:   "Open",
			fs:       []fs.FS{testFS1, testFS2},
			file:     "shared.txt",
			wantData: "fs1",
		},
		{
			name:    "open - file not found",
			method:  "Open",
			fs:      []fs.FS{testFS1, testFS2},
			file:    "notexist.txt",
			wantErr: true,
		},
		{
			name:     "readfile - file in first fs",
			method:   "ReadFile",
			fs:       []fs.FS{testFS1, testFS2},
			file:     "file1.txt",
			wantData: "data1",
		},
		{
			name:     "readfile - file in second fs",
			method:   "ReadFile",
			fs:       []fs.FS{testFS1, testFS2},
			file:     "file2.txt",
			wantData: "data2",
		},
		{
			name:    "readfile - not exist",
			method:  "ReadFile",
			fs:      []fs.FS{testFS1, testFS2},
			file:    "notexist.txt",
			wantErr: true,
		},
		{
			name:    "readdir - success",
			method:  "ReadDir",
			fs:      []fs.FS{testFS1, testFS2},
			file:    "dir",
			wantLen: 2,
		},
		{
			name:    "readdir - not exist",
			method:  "ReadDir",
			fs:      []fs.FS{testFS1, testFS2},
			file:    "notexist",
			wantErr: true,
		},
		{
			name:    "glob - matches across filesystems",
			method:  "Glob",
			fs:      []fs.FS{testFS1, testFS2},
			file:    "*.txt",
			wantLen: 3,
		},
		{
			name:    "glob - no matches",
			method:  "Glob",
			fs:      []fs.FS{testFS1, testFS2},
			file:    "nomatch*",
			wantLen: 0,
		},
		{
			name:    "glob - bad pattern",
			method:  "Glob",
			fs:      []fs.FS{testFS1, testFS2},
			file:    "[",
			wantErr: true,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			opts := []ChainFSOption{WithChainFilesystems(tt.fs...)}
			if tt.randOrder {
				opts = append(opts, WithChainRandOrder())
			}
			chainFS := NewChainFS(opts...)
			switch tt.method {
			case "Open":
				f, err := chainFS.Open(tt.file)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				defer f.Close()
				data, err := io.ReadAll(f)
				require.NoError(t, err)
				assert.Equal(t, tt.wantData, string(data))
			case "ReadFile":
				data, err := fs.ReadFile(chainFS, tt.file)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.wantData, string(data))
			case "ReadDir":
				entries, err := fs.ReadDir(chainFS, tt.file)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Len(t, entries, tt.wantLen)
			case "Glob":
				matches, err := fs.Glob(chainFS, tt.file)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Len(t, matches, tt.wantLen)
			}
		})
	}
}
