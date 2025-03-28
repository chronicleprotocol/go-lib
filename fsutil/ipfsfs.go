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
	rand.Seed(uint64(time.Now().UnixNano())) //nolint:gosec // disable G115
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
	return sliceutil.MapErr(gateways, func(s string) (*url.URL, error) {
		u, err := url.Parse(s)
		if err != nil {
			return nil, err
		}
		if u.Host == "" {
			u = &url.URL{Host: s}
		}
		return u, nil
	})
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
