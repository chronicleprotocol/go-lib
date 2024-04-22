package fsutil

import (
	"net/url"
	"time"

	"golang.org/x/exp/rand"

	"github.com/chronicleprotocol/suite/pkg/util/sliceutil"
)

// see: https://specs.ipfs.tech/http-gateways/path-gateway/#http-api
//TODO: Add cid validation
//TODO: Get data directly from an IPFS node (use remote node or internally started node) - configurable

func RandIpfsGatewayFn(gateways ...string) (func(url.URL) string, error) {
	gwList, err := makeGatewaysFromStrings(gateways...)
	if err != nil {
		return nil, err
	}
	n := len(gwList)
	rand.Seed(uint64(time.Now().UnixNano()))
	return func(u url.URL) string {
		gw := gwList[rand.Intn(n)]

		if u.Path != "" {
			u.Path += u.Host + "/" + u.Path
		} else {
			u.Path += u.Host
		}
		u.Path = "/ipfs/" + u.Path

		if gw.Scheme != "" {
			u.Scheme = gw.Scheme
		} else {
			u.Scheme = "https"
		}

		u.Host = gw.Host

		return u.String()
	}, nil
}

func makeGatewaysFromStrings(gateways ...string) ([]*url.URL, error) {
	if len(gateways) == 0 {
		return ipfsGateways, nil
	}
	return sliceutil.MapErr(gateways, url.Parse)
}

var ipfsGateways = []*url.URL{
	{Host: "storry.tv"},
	{Host: "ipfs.io"},
	{Host: "dweb.link"},
	{Host: "cloudflare-ipfs.com"},
	{Host: "cf-ipfs.com"},
	{Host: "gateway.pinata.cloud"},
	{Host: "hardbin.com"},
	{Host: "ipfs.runfission.com"},
	{Host: "ipfs.eth.aragon.network"},
	{Host: "nftstorage.link"},
	{Host: "4everland.io"},
	{Host: "w3s.link"},
	{Host: "trustless-gateway.link"},
}
