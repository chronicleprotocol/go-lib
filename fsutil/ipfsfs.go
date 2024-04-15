package fsutil

import (
	"fmt"
	"io/fs"
	"math/rand"
)

//TODO: Add cid validation
//TODO: Get data directly from an IPFS node (use remote node or internally started node) - configurable

func NewIpfsGatewayFS(httpFS fs.FS, gateways ...string) fs.FS {
	if len(gateways) == 0 {
		gateways = ipfsGateways
	}
	return &ipfsGatewayFS{
		fs:       httpFS,
		gateways: gateways,
	}
}

type ipfsGatewayFS struct {
	fs fs.FS

	gateways []string
}

func (f *ipfsGatewayFS) ReadFile(name string) ([]byte, error) {
	gateways := f.gateways[:] //nolint:gocritic
	rand.Shuffle(len(gateways), func(i, j int) {
		gateways[i], gateways[j] = gateways[j], gateways[i]
	})
	for _, gateway := range gateways {
		content, err := fs.ReadFile(f.fs, fmt.Sprintf(gateway, name))
		if err == nil {
			return content, nil
		}
	}
	return nil, fmt.Errorf("all IPFS gateways failed")
}

func (f *ipfsGatewayFS) Open(name string) (fs.File, error) {
	return open(f, name)
}

var ipfsGateways = []string{
	"https://ipfs.io/ipfs/%s",
	"https://gateway.pinata.cloud/ipfs/%s",
}
