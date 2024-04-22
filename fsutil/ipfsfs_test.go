package fsutil

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chronicleprotocol/suite/pkg/util/sliceutil"
)

func TestRandIpfsGatewayName(t *testing.T) {
	tcs := []struct {
		arg            string
		expectedSuffix string
	}{{
		arg:            "ipfs://bafybeihkoviema7g3gxyt6la7vd5ho32ictqbilu3wnlo3rs7ewhnp7lly",
		expectedSuffix: "/ipfs/bafybeihkoviema7g3gxyt6la7vd5ho32ictqbilu3wnlo3rs7ewhnp7lly",
	}, {
		arg:            "ipfs://bafybeihkoviema7g3gxyt6la7vd5ho32ictqbilu3wnlo3rs7ewhnp7lly?xxx=111&yyy=222#zzz",
		expectedSuffix: "/ipfs/bafybeihkoviema7g3gxyt6la7vd5ho32ictqbilu3wnlo3rs7ewhnp7lly?xxx=111&yyy=222#zzz",
	}, { // for backward compatibility
		arg:            "ipfs+gateway://bafybeihkoviema7g3gxyt6la7vd5ho32ictqbilu3wnlo3rs7ewhnp7lly",
		expectedSuffix: "/ipfs/bafybeihkoviema7g3gxyt6la7vd5ho32ictqbilu3wnlo3rs7ewhnp7lly",
	}}

	{ // test default gateways
		fn, err := RandIpfsGatewayFn()
		require.NoError(t, err)

		gateways := sliceutil.Map(ipfsGateways, func(u *url.URL) string {
			if u.Scheme == "" {
				u.Scheme = "https"
			}
			return u.String()
		})
		for i, tc := range tcs {
			require.NotEmpty(t, tc.expectedSuffix, "expectedSuffix is empty")
			var u *url.URL
			t.Run(fmt.Sprintf("has suffix %d", i), func(t *testing.T) {
				u, err = url.Parse(tc.arg)
				require.NoError(t, err)
				s := fn(*u)
				assert.Truef(t, strings.HasSuffix(s, tc.expectedSuffix), "expected suffix %q\nin %q", tc.expectedSuffix, s)
			})
			for j := 0; j < 100; j++ {
				t.Run(fmt.Sprintf("has prefix in default list %d %d", i, j), func(t *testing.T) {
					s := fn(*u)
					assert.Truef(t, stringListContainsPrefix(gateways, s), "expected prefix in %q", s)
				})
			}
		}
	}

	{ // test custom gateways
		gateways := []string{"https://example.com", "https://example.org"}
		fn, err := RandIpfsGatewayFn(gateways...)
		require.NoError(t, err)

		for i, tc := range tcs {
			u, err := url.Parse(tc.arg)
			require.NoError(t, err)
			for j := 0; j < 100; j++ {
				t.Run(fmt.Sprintf("has prefix in local list %d %d", i, j), func(t *testing.T) {
					s := fn(*u)
					assert.Truef(t, stringListContainsPrefix(gateways, s), "expected prefix in %q", s)
				})
			}
		}
	}
}

func stringListContainsPrefix(list []string, name string) bool {
	for _, s := range list {
		if strings.HasPrefix(name, s) {
			return true
		}
	}
	return false
}
