package fsutil

import (
	"fmt"
	"io/fs"
	"net/url"
)

type ProtoFSItem struct {
	Scheme string
	FS     fs.FS
	NameFn func(url.URL) string
}

func NewProtoFS(items ...ProtoFSItem) fs.FS {
	f := make(protoFS, len(items))
	for _, item := range items {
		f[item.Scheme] = protoFSItem{
			fs:   item.FS,
			name: item.NameFn,
		}
	}
	return &f
}

type protoFSItem struct {
	fs   fs.FS
	name func(url.URL) string
}

type protoFS map[string]protoFSItem

func (f *protoFS) ReadFile(name string) ([]byte, error) {
	u, err := url.Parse(name)
	if err != nil {
		return nil, err
	}
	fsItem, ok := (*f)[u.Scheme]
	if !ok {
		return nil, fmt.Errorf("unsupported protocol: %s", u.Scheme)
	}
	if fsItem.name != nil {
		return fs.ReadFile(fsItem.fs, fsItem.name(*u))
	}
	return fs.ReadFile(fsItem.fs, u.String())
}

func (f *protoFS) Open(name string) (fs.File, error) {
	return open(f, name)
}
