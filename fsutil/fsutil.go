//  Copyright (C) 2021-2023 Chronicle Labs, Inc.
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.

package fsutil

import (
	"io/fs"

	"github.com/chronicleprotocol/suite/pkg/util/errutil"
)

// NewChainFS returns a new FS that combines multiple FS into one.
//
// The FS returned by NewChainFS will use given FS in the order they are
// provided.
func NewChainFS(fs ...fs.FS) fs.FS {
	return &chainFS{fs: fs}
}

type chainFS struct{ fs []fs.FS }

// Open implements the fs.FS interface.
//
// It opens the named file from the first FS that returns a non-nil result.
//
// Otherwise, it returns an error that combines all errors from all FS.
func (c *chainFS) Open(name string) (fs.File, error) {
	var err error
	for _, f := range c.fs {
		fsFile, fsErr := f.Open(name)
		if fsErr != nil {
			err = errutil.Append(err, fsErr)
			continue
		}
		return fsFile, nil
	}
	return nil, err
}

// Glob implements the fs.GlobFS interface.
//
// It returns the names of all files matching pattern or nil if there is no
// matching file.
//
// If any FS returns an error, the error is returned immediately.
func (c *chainFS) Glob(pattern string) (paths []string, err error) {
	for _, f := range c.fs {
		fsPaths, fsErr := fs.Glob(f, pattern)
		if fsErr != nil {
			return nil, fsErr
		}
		paths = append(paths, fsPaths...)
	}
	return paths, nil
}

// Stat implements the fs.StatFS interface.
//
// It returns the FileInfo for the named file from the first FS that returns a
// non-nil result.
//
// Otherwise, it returns an error that combines all errors from all FS.
func (c *chainFS) Stat(name string) (fs.FileInfo, error) {
	var err error
	for _, f := range c.fs {
		fsStat, fsErr := fs.Stat(f, name)
		if fsErr != nil {
			err = errutil.Append(err, fsErr)
			continue
		}
		return fsStat, nil
	}
	return nil, err
}

// ReadFile implements the fs.ReadFileFS interface.
//
// It reads the named file from the first FS that returns a non-nil result.
//
// Otherwise, it returns an error that combines all errors from all FS.
func (c *chainFS) ReadFile(name string) ([]byte, error) {
	var err error
	for _, f := range c.fs {
		fsData, fsErr := fs.ReadFile(f, name)
		if fsErr != nil {
			err = errutil.Append(err, fsErr)
			continue
		}
		return fsData, nil
	}
	return nil, err
}

// ReadDir implements the fs.ReadDirFS interface.
//
// It reads the named directory from the first FS that returns a non-nil result.
//
// Otherwise, it returns an error that combines all errors from all FS.
func (c *chainFS) ReadDir(name string) ([]fs.DirEntry, error) {
	var err error
	for _, f := range c.fs {
		fsDir, fsErr := fs.ReadDir(f, name)
		if fsErr != nil {
			err = errutil.Append(err, fsErr)
			continue
		}
		return fsDir, nil
	}
	return nil, err
}

// Sub implements the fs.SubFS interface.
//
// It returns a new FS that represents the named directory from the first FS
// that returns a non-nil result.
//
// Otherwise, it returns an error that combines all errors from all FS.
func (c *chainFS) Sub(dir string) (fs.FS, error) {
	var err error
	for _, f := range c.fs {
		fsSub, fsErr := fs.Sub(f, dir)
		if fsErr != nil {
			err = errutil.Append(err, fsErr)
			continue
		}
		return fsSub, nil
	}
	return nil, err
}
