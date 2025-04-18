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
	"io/fs"
	"net/url"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileProto(t *testing.T) {
	_, testFilePath, _, _ := runtime.Caller(0)
	tc := []struct {
		name     string
		opts     []FileOption
		uri      string
		wantErr  bool
		wantData []byte
	}{
		{
			name:    "nil URL",
			uri:     "",
			wantErr: true,
		},
		{
			name:    "unexpected scheme",
			uri:     "http://localhost",
			wantErr: true,
		},
		{
			name:    "unexpected host",
			uri:     "file://example.com",
			wantErr: true,
		},
		{
			name: "empty host",
			opts: []FileOption{
				WithFileWorkingDir(path.Join(path.Dir(testFilePath), "testdata")),
			},
			uri:      "file:///",
			wantData: []byte("test content"),
		},
		{
			name: "localhost host",
			opts: []FileOption{
				WithFileWorkingDir(path.Join(path.Dir(testFilePath), "testdata")),
			},
			uri:      "file://localhost/",
			wantData: []byte("test content"),
		},
		{
			name: "read file - with path",
			opts: []FileOption{
				WithFileWorkingDir(path.Dir(testFilePath)),
			},
			uri:      "file:///testdata",
			wantData: []byte("test content"),
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFileProto(tt.opts...)
			var u *url.URL
			if tt.uri != "" {
				var err error
				u, err = url.Parse(tt.uri)
				require.NoError(t, err)
			}
			ffs, fsPath, err := p.FileSystem(u)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.wantData != nil {
				data, err := fs.ReadFile(ffs, path.Join(fsPath, "test.txt"))
				require.NoError(t, err)
				require.Equal(t, tt.wantData, data)
			} else {
				_, ok := ffs.(fs.ReadDirFS)
				require.True(t, ok)
			}
		})
	}
}
