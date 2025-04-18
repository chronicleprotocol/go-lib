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

	"golang.org/x/crypto/sha3"
)

type IPFSOption func(*ipfsFS)

type IPFSGateway struct {
	Scheme    string
	Host      string
	ResolveFn func(cid string) func(f *httpFS, name string) (*netURL.URL, error)
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
func (m *ipfsProto) FileSystem(uri *netURL.URL) (fs fs.FS, path string, err error) {
	if err := validIPFSURI(uri); err != nil {
		return nil, "", err
	}
	fs, err = NewIPFSFS(m.ctx, uri.Host, m.opts...)
	if err != nil {
		return nil, "", errIPFSProtoFn(err)
	}
	path = uriPath(uri, true)
	if path == "" {
		// Empty paths are not allowed by fs.FS.
		path = "."
	}
	return fs, path, nil
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
func NewIPFSFS(ctx context.Context, cid string, opts ...IPFSOption) (fs.FS, error) {
	if cid == "" {
		return nil, errIPFSFSEmptyCID
	}
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
				parseFn: gw.ResolveFn(cid),
			},
			hash:  i.checksumHash,
			param: "checksum",
			mode:  ChecksumFSVerifyAfterOpen,
		})
	}
	i.cfs = cfs
	return i, nil
}

type ipfsFS struct {
	client       *http.Client
	gateways     []*IPFSGateway
	checksumHash func() hash.Hash
	cfs          *chainFS
}

func (h *ipfsFS) Open(name string) (fs.File, error) {
	if err := validPath("open", name); err != nil {
		return nil, errIPFSFSFn(err)
	}
	return h.cfs.Open(name)
}

func IPFSPathResolution(cid string) func(f *httpFS, name string) (*netURL.URL, error) {
	return func(f *httpFS, name string) (*netURL.URL, error) {
		httpPath := "/ipfs/" + cid
		if name != "" && name != "." {
			httpPath += "/" + name
		}
		url := &netURL.URL{
			Scheme: f.baseURI.Scheme,
			User:   f.baseURI.User,
			Host:   f.baseURI.Host,
			Path:   httpPath,
		}
		return url, nil
	}
}

func IPFSSubdomainResolution(cid string) func(f *httpFS, name string) (*netURL.URL, error) {
	return func(f *httpFS, name string) (*netURL.URL, error) {
		if name == "." {
			name = ""
		}
		url := &netURL.URL{
			Scheme: f.baseURI.Scheme,
			User:   f.baseURI.User,
			Host:   fmt.Sprintf("%s.%s", cid, f.baseURI.Host),
			Path:   name,
		}
		return url, nil
	}
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

func validIPFSURI(uri *netURL.URL) error {
	if uri == nil {
		return errIPFSProtoNilURI
	}
	if uri.Scheme != "ipfs" && uri.Scheme != "ipfs+gateway" {
		return errIPFSProtoUnexpectedSchemeFn(uri.Scheme)
	}
	if uri.Opaque != "" {
		return errIPFSProtoOpaqueNotAllowed
	}
	if uri.Host == "" {
		return errIPFSProtoEmptyHost
	}
	if uri.OmitHost {
		return errIPFSProtoOmitHost
	}
	if uri.Fragment != "" || uri.RawFragment != "" {
		return errIPFSProtoFragmentNotAllowed
	}
	return nil
}

var (
	errIPFSProtoNilURI             = errors.New("fsutil.ipfsProto: nil URI")
	errIPFSProtoOpaqueNotAllowed   = errors.New("fsutil.ipfsProto: opaque not allowed")
	errIPFSProtoEmptyHost          = errors.New("fsutil.ipfsProto: empty host")
	errIPFSProtoOmitHost           = errors.New("fsutil.ipfsProto: omit host must be false")
	errIPFSProtoFragmentNotAllowed = errors.New("fsutil.ipfsProto: fragment not allowed")
	errIPFSFSEmptyCID              = fmt.Errorf("fsutil.ipfsFS: empty CID")
)

func errIPFSProtoFn(err error) error {
	return fmt.Errorf("fsutil.ipfsProto: %w", err)
}

func errIPFSFSFn(err error) error {
	return fmt.Errorf("fsutil.ipfsFS: %w", err)
}

func errIPFSProtoUnexpectedSchemeFn(scheme string) error {
	return fmt.Errorf("fsutil.ipfsProto: unexpected scheme: %s", scheme)
}
