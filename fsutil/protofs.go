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
	"fmt"
	"io/fs"
	"net/url"
)

type ProtoFSItem struct {
	Scheme string
	FS     fs.FS
}

func NewProtoFS(items ...ProtoFSItem) (fs.FS, error) {
	f := make(protoFS, len(items))
	for _, item := range items {
		if item.FS == nil {
			return nil, fmt.Errorf("fs is nil for scheme: %s", item.Scheme)
		}
		f[item.Scheme] = item.FS
	}
	return &f, nil
}

// protoFS is a map of protocol to fs.FS
type protoFS map[string]fs.FS

func (f *protoFS) ReadFile(name string) ([]byte, error) {
	u, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	fsItem, ok := (*f)[u.Scheme]
	if !ok {
		return nil, fmt.Errorf("unsupported protocol: %s", u.Scheme)
	}
	return fs.ReadFile(fsItem, name)
}

func (f *protoFS) Open(name string) (fs.File, error) {
	return open(f, name)
}
