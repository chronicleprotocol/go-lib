// Copyright (C) 2021-2023 Chronicle Labs, Inc.
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
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockFS struct {
	files map[string]*mockFile
	dir   map[string]*mockFile
}

func newMockFS() *mockFS {
	return &mockFS{
		files: make(map[string]*mockFile),
		dir:   make(map[string]*mockFile),
	}
}

func (m *mockFS) Open(name string) (fs.File, error) {
	if file, ok := m.files[name]; ok {
		return file, nil
	}
	if file, ok := m.dir[name]; ok {
		return file, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFS) Glob(pattern string) ([]string, error) {
	var matches []string
	for name := range m.files {
		if ok, _ := path.Match(pattern, name); ok {
			matches = append(matches, name)
		}
	}
	for name := range m.dir {
		if ok, _ := path.Match(pattern, name); ok {
			matches = append(matches, name)
		}
	}
	return matches, nil
}

func (m *mockFS) ReadDir(name string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	for k, _ := range m.dir {
		if k == name {
			entries = append(entries, &mockDirEntry{name: path.Base(k)})
		}
	}
	for k, _ := range m.files {
		if path.Dir(k) == name {
			entries = append(entries, &mockDirEntry{name: path.Base(k)})
		}
	}
	if len(entries) == 0 {
		return nil, os.ErrNotExist
	}
	return entries, nil
}

func (m *mockFS) Sub(name string) (fs.FS, error) {
	if _, ok := m.dir[name]; ok {
		f := make(map[string]*mockFile)
		for k, v := range m.files {
			if path.Dir(k) == name {
				f[strings.TrimPrefix(k, name+"/")] = v
			}
		}
		return &mockFS{files: f}, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockFS) addFile(name string, content []byte) {
	dir := path.Dir(name)
	if _, ok := m.dir[dir]; dir != "." && !ok {
		panic("directory doesn't exist")
	}
	if strings.Count(name, "/") > 1 {
		panic("nested directories are not supported")
	}
	m.files[name] = newMockFile(name, false, content)
}

func (m *mockFS) addDir(name string) {
	if strings.Count(name, "/") > 0 {
		panic("nested directories are not supported")
	}
	m.dir[name] = newMockFile(name, true, nil)
}

type mockFile struct {
	name    string
	dir     bool
	content *bytes.Buffer
}

func newMockFile(name string, dir bool, content []byte) *mockFile {
	return &mockFile{
		name:    name,
		dir:     dir,
		content: bytes.NewBuffer(content),
	}
}

func (m *mockFile) Stat() (fs.FileInfo, error) {
	return &mockFileInfo{name: m.name, dir: m.dir}, nil
}

func (m *mockFile) Read(b []byte) (n int, err error) {
	return m.content.Read(b)
}

func (m *mockFile) Close() error {
	return nil
}

type mockFileInfo struct {
	name string
	dir  bool
}

func (m *mockFileInfo) Name() string {
	return m.name
}

func (m *mockFileInfo) Size() int64 {
	return 0
}

func (m *mockFileInfo) Mode() fs.FileMode {
	return 0
}

func (m *mockFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (m *mockFileInfo) IsDir() bool {
	return m.dir
}

func (m *mockFileInfo) Sys() any {
	return nil
}

type mockDirEntry struct {
	name string
}

func (m *mockDirEntry) Name() string {
	return m.name
}

func (m *mockDirEntry) IsDir() bool {
	return true
}

func (m *mockDirEntry) Type() fs.FileMode {
	return 0
}

func (m *mockDirEntry) Info() (fs.FileInfo, error) {
	return &mockFileInfo{name: m.name, dir: true}, nil
}

func TestChainFS_Open(t *testing.T) {
	mockFs1 := newMockFS()
	mockFs2 := newMockFS()
	mockFs1.addFile("file1.txt", []byte("file1 content"))
	mockFs2.addFile("file2.txt", []byte("file2 content"))
	mockFs1.addFile("common.txt", []byte("common1 content"))
	mockFs2.addFile("common.txt", []byte("common2 content"))
	cfs := NewChainFS(mockFs1, mockFs2)

	t.Run("fetch first file", func(t *testing.T) {
		file, err := cfs.Open("file1.txt")
		assert.NotNil(t, file)
		assert.Nil(t, err)
	})

	t.Run("fetch second file", func(t *testing.T) {
		file, err := cfs.Open("file2.txt")
		assert.NotNil(t, file)
		assert.Nil(t, err)
	})

	t.Run("fetch common file", func(t *testing.T) {
		file, err := cfs.Open("common.txt")
		content, _ := io.ReadAll(file)
		assert.Equal(t, content, []byte("common1 content"))
		assert.NotNil(t, file)
		assert.Nil(t, err)
	})

	t.Run("non existing file", func(t *testing.T) {
		_, err := cfs.Open("non-existing.txt")
		assert.NotNil(t, err)
	})
}

func TestChainFS_Glob(t *testing.T) {
	mockFs1 := newMockFS()
	mockFs2 := newMockFS()
	mockFs1.addFile("file1.txt", []byte("file1 content"))
	mockFs2.addFile("file2.txt", []byte("file2 content"))
	mockFs1.addFile("common.txt", []byte("common1 content"))
	mockFs2.addFile("common.txt", []byte("common2 content"))
	cfs := NewChainFS(mockFs1, mockFs2)

	t.Run("pattern matches files in both FS", func(t *testing.T) {
		paths, err := fs.Glob(cfs, "*.txt")
		assert.Nil(t, err)
		assert.Contains(t, paths, "file1.txt")
		assert.Contains(t, paths, "file2.txt")
		assert.Contains(t, paths, "common.txt")
	})

	t.Run("pattern matches files in only one FS", func(t *testing.T) {
		paths, err := fs.Glob(cfs, "file1.txt")
		assert.Nil(t, err)
		assert.Contains(t, paths, "file1.txt")
	})

	t.Run("pattern doesn't match any files", func(t *testing.T) {
		paths, err := fs.Glob(cfs, "non-existing.txt")
		assert.Nil(t, err)
		assert.Empty(t, paths)
	})
}

func TestChainFS_Stat(t *testing.T) {
	mockFs1 := newMockFS()
	mockFs2 := newMockFS()
	mockFs1.addFile("file1.txt", []byte("file1 content"))
	mockFs2.addFile("file2.txt", []byte("file2 content"))
	mockFs1.addFile("common.txt", []byte("common1 content"))
	mockFs2.addFile("common.txt", []byte("common2 content"))
	cfs := NewChainFS(mockFs1, mockFs2)

	t.Run("file exists in both FS", func(t *testing.T) {
		_, err := fs.Stat(cfs, "common.txt")
		assert.Nil(t, err)
	})

	t.Run("file exists in only one FS", func(t *testing.T) {
		_, err := fs.Stat(cfs, "file1.txt")
		assert.Nil(t, err)
	})

	t.Run("file doesn't exist in any FS", func(t *testing.T) {
		_, err := fs.Stat(cfs, "non-existing.txt")
		assert.NotNil(t, err)
	})
}

func TestChainFS_ReadFile(t *testing.T) {
	mockFs1 := newMockFS()
	mockFs2 := newMockFS()
	mockFs1.addFile("file1.txt", []byte("file1 content"))
	mockFs2.addFile("file2.txt", []byte("file2 content"))
	mockFs1.addFile("common.txt", []byte("common1 content"))
	mockFs2.addFile("common.txt", []byte("common2 content"))
	cfs := NewChainFS(mockFs1, mockFs2)

	t.Run("file exists in both FS", func(t *testing.T) {
		content, err := fs.ReadFile(cfs, "common.txt")
		assert.Nil(t, err)
		assert.Equal(t, content, []byte("common1 content"))
	})

	t.Run("file exists in only one FS", func(t *testing.T) {
		content, err := fs.ReadFile(cfs, "file1.txt")
		assert.Nil(t, err)
		assert.Equal(t, content, []byte("file1 content"))
	})

	t.Run("file doesn't exist in any FS", func(t *testing.T) {
		_, err := fs.ReadFile(cfs, "non-existing.txt")
		assert.NotNil(t, err)
	})
}

func TestChainFS_ReadDir(t *testing.T) {
	mockFs1 := newMockFS()
	mockFs2 := newMockFS()
	mockFs1.addDir("dir1")
	mockFs2.addDir("dir2")
	mockFs1.addDir("common")
	mockFs2.addDir("common")
	cfs := NewChainFS(mockFs1, mockFs2)

	t.Run("directory exists in both FS", func(t *testing.T) {
		_, err := fs.ReadDir(cfs, "common")
		assert.Nil(t, err)
	})

	t.Run("directory exists in only one FS", func(t *testing.T) {
		_, err := fs.ReadDir(cfs, "dir1")
		assert.Nil(t, err)
	})

	t.Run("directory doesn't exist in any FS", func(t *testing.T) {
		_, err := fs.ReadDir(cfs, "non-existing")
		assert.NotNil(t, err)
	})
}

func TestChainFS_Sub(t *testing.T) {
	mockFs1 := newMockFS()
	mockFs2 := newMockFS()
	mockFs1.addDir("dir1")
	mockFs2.addDir("dir2")
	mockFs1.addDir("common")
	mockFs2.addDir("common")
	mockFs1.addFile("dir1/file.txt", []byte("file content"))
	mockFs2.addFile("dir2/file.txt", []byte("file content"))
	mockFs1.addFile("common/file.txt", []byte("file content"))
	mockFs2.addFile("common/file.txt", []byte("file content"))
	cfs := NewChainFS(mockFs1, mockFs2)

	t.Run("directory exists in both FS", func(t *testing.T) {
		s, err := fs.Sub(cfs, "common")
		assert.Nil(t, err)

		_, err = fs.Stat(s, "file.txt")
		assert.Nil(t, err)
	})

	t.Run("directory exists in only one FS", func(t *testing.T) {
		s, err := fs.Sub(cfs, "dir1")
		assert.Nil(t, err)

		_, err = fs.Stat(s, "file.txt")
		assert.Nil(t, err)
	})

	t.Run("directory doesn't exist in any FS", func(t *testing.T) {
		_, err := fs.Sub(cfs, "non-existing")
		assert.NotNil(t, err)
	})
}
