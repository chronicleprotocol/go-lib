package fsutil

import (
	"io/fs"
	"net/url"
)

func NewRenameFS(fs fs.FS, rename func(url.URL) string) fs.FS {
	if rename == nil {
		return fs
	}
	return &renameFS{fs: fs, rename: rename}
}

// renameFS is a fs.FS that renames the URL before passing it to the underlying fs.FS
type renameFS struct {
	fs     fs.FS
	rename func(url.URL) string
}

func (f *renameFS) ReadFile(name string) ([]byte, error) {
	if f.rename == nil {
		return fs.ReadFile(f.fs, name)
	}
	u, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	return fs.ReadFile(f.fs, f.rename(*u))
}

func (f *renameFS) Open(name string) (fs.File, error) {
	return open(f, name)
}
