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
	"context"
	"errors"
	"fmt"
	"hash"
	"io/fs"
	"net/http"
	netURL "net/url"
	"strings"

	"golang.org/x/crypto/sha3"
)

type IPFSOption func(*ipfsFS)

type IPFSGateway struct {
	Scheme    string
	Host      string
	ResolveFn func(fs *httpFS, name string) (*netURL.URL, error)
}

// WithIPFSHTTPClient sets the HTTP client used to perform HTTP requests.
func WithIPFSHTTPClient(client *http.Client) IPFSOption {
	return func(c *ipfsFS) {
		c.client = client
	}
}

// WithIPFSGateways sets the IPFS gateways used to resolve IPFS paths.
func WithIPFSGateways(gateways ...*IPFSGateway) IPFSOption {
	return func(c *ipfsFS) {
		c.gateways = gateways
	}
}

// WithIPFSChecksumHash sets the hash function used to compute the checksum.
func WithIPFSChecksumHash(hash func() hash.Hash) IPFSOption {
	return func(c *ipfsFS) {
		c.checksumHash = hash
	}
}

// NewIPFSProto creates a new IPFS protocol.
//
// The IPFS protocol is used to create an IPFS file system.
func NewIPFSProto(ctx context.Context, opts ...IPFSOption) Protocol {
	return &ipfsProto{ctx: ctx, opts: opts}
}

type ipfsProto struct {
	ctx  context.Context
	opts []IPFSOption
}

// FileSystem implements the Protocol interface.
func (m *ipfsProto) FileSystem(url *netURL.URL) (fs fs.FS, path string, err error) {
	if url == nil {
		return nil, "", errIPFSNilURI
	}
	if url.Scheme != "ipfs" && url.Scheme != "ipfs+gateway" {
		return nil, "", errIPFSUnexpectedSchemeFn(url.Scheme)
	}
	if url.Opaque != "" {
		return nil, "", errIPFSInvalidURIFn("opaque not allowed")
	}
	return NewIPFSFS(m.ctx, m.opts...), fmt.Sprintf("%s/%s", url.Host, uriPath(url, true)), nil
}

// NewIPFSFS creates a new IPFS filesystem.
//
// The IPFS filesystem uses IPFS gateways to resolve IPFS paths. To verify
// the integrity of the file contents and ensure that returned data is valid,
// an optional checksum hash can be provided as a "checksum" parameter in the URL.
//
// It is important to provide a checksum, as there is no guarantee that
// the data returned from IPFS gateways is valid. A misconfigured or malicious
// gateway could return a different or corrupted file.
func NewIPFSFS(ctx context.Context, opts ...IPFSOption) fs.FS {
	i := &ipfsFS{}
	for _, opt := range opts {
		opt(i)
	}
	if i.client == nil {
		i.client = http.DefaultClient
	}
	if len(i.gateways) == 0 {
		i.gateways = ipfsGateways
	}
	if i.checksumHash == nil {
		i.checksumHash = sha3.NewLegacyKeccak256
	}
	cfs := &chainFS{rand: true}
	for _, gw := range i.gateways {
		cfs.fs = append(cfs.fs, &checksumFS{
			fs: &httpFS{
				ctx:     ctx,
				client:  i.client,
				baseURI: &netURL.URL{Scheme: gw.Scheme, Host: gw.Host},
				parseFn: gw.ResolveFn,
			},
			hash:  i.checksumHash,
			param: "checksum",
			mode:  ChecksumFSVerifyAfterOpen,
		})
	}
	i.cfs = cfs
	return i
}

type ipfsFS struct {
	client       *http.Client
	gateways     []*IPFSGateway
	checksumHash func() hash.Hash
	cfs          *chainFS
}

func (h *ipfsFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, errIPFSInvalidPathFn(name)
	}
	return h.cfs.Open(name)
}

func IPFSPathResolution(f *httpFS, name string) (*netURL.URL, error) {
	var cid string
	var path string
	parts := strings.SplitN(name, "/", 2)
	switch len(parts) {
	case 1:
		cid = parts[0]
	default:
		cid = parts[0]
		path = parts[1]
	}
	url := &netURL.URL{
		Scheme: f.baseURI.Scheme,
		User:   f.baseURI.User,
		Host:   f.baseURI.Host,
		Path:   fmt.Sprintf("/ipfs/%s/%s", cid, path),
	}
	return url, nil
}

func IPFSSubdomainResolution(f *httpFS, name string) (*netURL.URL, error) {
	var cid string
	var path string
	parts := strings.SplitN(name, "/", 2)
	switch len(parts) {
	case 1:
		cid = parts[0]
	default:
		cid = parts[0]
		path = parts[1]
	}
	url := &netURL.URL{
		Scheme: f.baseURI.Scheme,
		User:   f.baseURI.User,
		Host:   fmt.Sprintf("%s.%s", cid, f.baseURI.Host),
		Path:   path,
	}
	return url, nil
}

var ipfsGateways = []*IPFSGateway{
	{Scheme: "https", Host: "ipfs.io", ResolveFn: IPFSPathResolution},
	{Scheme: "https", Host: "gateway.pinata.cloud", ResolveFn: IPFSPathResolution},
	{Scheme: "https", Host: "trustless-gateway.link", ResolveFn: IPFSPathResolution},
	{Scheme: "https", Host: "dweb.link", ResolveFn: IPFSSubdomainResolution},
	{Scheme: "https", Host: "storry.tv", ResolveFn: IPFSPathResolution},
	{Scheme: "https", Host: "w3s.link", ResolveFn: IPFSPathResolution},
	{Scheme: "https", Host: "4everland.io", ResolveFn: IPFSPathResolution},
	{Scheme: "https", Host: "flk-ipfs.xyz", ResolveFn: IPFSPathResolution},
	{Scheme: "https", Host: "ipfs.cyou", ResolveFn: IPFSPathResolution},
	{Scheme: "https", Host: "nftstorage.link", ResolveFn: IPFSPathResolution},
}

var (
	errIPFSNilURI           = errors.New("fsutil.ipfs: nil URI")
	errIPFSInvalidURI       = errors.New("fsutil.ipfs: invalid URI")
	errIPFSInvalidPath      = errors.New("fsutil.ipfs: invalid path")
	errIPFSUnexpectedScheme = errors.New("fsutil.ipfs: unexpected scheme")
)

func errIPFSInvalidPathFn(path string) error {
	return fmt.Errorf("%w: %w", errIPFSInvalidPath, errInvalidPathFn(path))
}

func errIPFSUnexpectedSchemeFn(scheme string) error {
	return fmt.Errorf("%w: %s", errIPFSUnexpectedScheme, scheme)
}

func errIPFSInvalidURIFn(msg string) error {
	return fmt.Errorf("%w: %s", errIPFSInvalidURI, msg)
}
