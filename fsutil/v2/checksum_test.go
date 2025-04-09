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

	"github.com/defiweb/go-eth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

func TestChecksumFS(t *testing.T) {
	testFS := fstest.MapFS{
		"file.txt": &fstest.MapFile{Data: []byte("data")},
	}
	tc := []struct {
		name      string
		method    string
		fs        fs.FS
		file      string
		randOrder bool
		wantErr   bool
		wantData  string
	}{
		{
			name:     "open - without checksum",
			method:   "Open",
			fs:       testFS,
			file:     "file.txt",
			wantData: "data",
		},
		{
			name:     "open - with checksum",
			method:   "Open",
			fs:       testFS,
			file:     "file.txt?checksum=" + calculateKeccak256([]byte("data")).String(),
			wantData: "data",
		},
		{
			name:    "open - with checksum mismatch",
			method:  "Open",
			fs:      testFS,
			file:    "file.txt?checksum=" + calculateKeccak256([]byte("data2")).String(),
			wantErr: true,
		},
		{
			name:     "readFile - without checksum",
			method:   "ReadFile",
			fs:       testFS,
			file:     "file.txt",
			wantData: "data",
		},
		{
			name:     "readFile - with checksum",
			method:   "ReadFile",
			fs:       testFS,
			file:     "file.txt?checksum=" + calculateKeccak256([]byte("data")).String(),
			wantData: "data",
		},
		{
			name:    "readFile - with checksum mismatch",
			method:  "ReadFile",
			fs:      testFS,
			file:    "file.txt?checksum=" + calculateKeccak256([]byte("data2")).String(),
			wantErr: true,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			checksumFS, err := NewChecksumFS(tt.fs)
			require.NoError(t, err)
			switch tt.method {
			case "Open":
				f, err := checksumFS.Open(tt.file)
				require.NoError(t, err)
				defer f.Close()
				data, err := io.ReadAll(f)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.wantData, string(data))
			case "ReadFile":
				data, err := fs.ReadFile(checksumFS, tt.file)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, tt.wantData, string(data))
			}
		})
	}
}

func calculateKeccak256(data []byte) types.Hash {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	return types.Hash(h.Sum(nil))
}
