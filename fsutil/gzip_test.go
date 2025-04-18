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
	"compress/gzip"
	"io/fs"
	"net/url"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestGzipProto(t *testing.T) {
	mem := fstest.MapFS{
		"file.txt.gz": &fstest.MapFile{Data: gzipData([]byte("proto test"))},
		"file.txt":    &fstest.MapFile{Data: []byte("raw data")},
	}

	mock := &mockProto{fs: mem}
	uri, _ := url.Parse("file:///")
	gzipProto := NewGzipProto(mock)

	gzipFS, _, err := gzipProto.FileSystem(uri)
	require.NoError(t, err)

	data, err := fs.ReadFile(gzipFS, "file.txt.gz")
	require.NoError(t, err)
	require.Equal(t, "proto test", string(data))

	data, err = fs.ReadFile(gzipFS, "file.txt")
	require.NoError(t, err)
	require.Equal(t, "raw data", string(data))
}

func TestGzipFS(t *testing.T) {
	var (
		testData = []byte("test content")
		gzData   = gzipData(testData)
	)
	tc := []struct {
		name     string
		files    map[string][]byte
		opts     []GzipFSOption
		file     string
		wantData []byte
		wantErr  bool
	}{
		{
			name: "check ext - non gz",
			files: map[string][]byte{
				"file.txt": testData,
			},
			opts:     nil,
			file:     "file.txt",
			wantData: testData,
		},
		{
			name: "check ext - valid gz",
			files: map[string][]byte{
				"file.txt.gz": gzData,
			},
			opts:     nil,
			file:     "file.txt.gz",
			wantData: testData,
		},
		{
			name: "check ext - invalid gz",
			files: map[string][]byte{
				"file.txt.gz": testData,
			},
			opts:    nil,
			file:    "file.txt.gz",
			wantErr: true,
		},
		{
			name: "no ext check - valid gz",
			files: map[string][]byte{
				"file.bin": gzData,
			},
			opts: []GzipFSOption{
				WithGzipCheckExtension(false),
			},
			file:     "file.bin",
			wantData: testData,
		},
		{
			name: "no ext check - invalid gz",
			files: map[string][]byte{
				"file.bin": testData,
			},
			opts: []GzipFSOption{
				WithGzipCheckExtension(false),
			},
			file:    "file.bin",
			wantErr: true,
		},
		{
			name: "custom ext - no gz",
			files: map[string][]byte{
				"file.txt": testData,
			},
			opts: []GzipFSOption{
				WithGzipExtensions("custom"),
			},
			file:     "file.txt",
			wantData: testData,
		},
		{
			name: "custom ext - valid gz",
			files: map[string][]byte{
				"file.custom": gzData,
			},
			opts: []GzipFSOption{
				WithGzipExtensions("custom"),
			},
			file:     "file.custom",
			wantData: testData,
		},
		{
			name: "custom ext - invalid gz",
			files: map[string][]byte{
				"file.custom": testData,
			},
			opts: []GzipFSOption{
				WithGzipExtensions("custom"),
			},
			file:    "file.custom",
			wantErr: true,
		},
		{
			name: "read limit",
			files: map[string][]byte{
				"bigfile.txt.gz": gzipData(bytes.Repeat([]byte("a"), 1024)),
			},
			opts: []GzipFSOption{
				WithGzipReadLimit(10),
			},
			file:    "bigfile.txt.gz",
			wantErr: true,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			testFS := fstest.MapFS{}
			for fname, fdata := range tt.files {
				testFS[fname] = &fstest.MapFile{Data: fdata}
			}
			gzipFS := NewGzipFS(testFS, tt.opts...)
			data, err := fs.ReadFile(gzipFS, tt.file)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantData, data)
		})
	}
}

func gzipData(data []byte) []byte {
	var b bytes.Buffer
	gw := gzip.NewWriter(&b)
	gw.Write(data)
	gw.Close()
	return b.Bytes()
}
