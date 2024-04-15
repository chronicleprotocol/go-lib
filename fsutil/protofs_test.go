package fsutil

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProtoFS(t *testing.T) {
	ctx := context.Background()
	r := NewRetryFS(ctx, NewHTTPFS(ctx, &http.Client{Timeout: 5 * time.Second}), 3, time.Second)
	fs := NewProtoFS(
		ProtoFSItem{
			Scheme: "ipfs+gateway",
			FS:     NewIpfsGatewayFS(r),
			NameFn: func(u url.URL) string {
				return u.Host + "/" + u.Path
			},
		},
		ProtoFSItem{Scheme: "https", FS: r},
	)
	assert.NotNil(t, fs)

	t.Run("fetch file from IPFS", func(t *testing.T) {
		file, err := fs.Open("ipfs+gateway://bafybeihkoviema7g3gxyt6la7vd5ho32ictqbilu3wnlo3rs7ewhnp7lly")
		assert.NoError(t, err)
		assert.NotNil(t, file)
	})

	t.Run("fetch file from HTTP", func(t *testing.T) {
		file, err := fs.Open("https://raw.githubusercontent.com/chronicleprotocol/charts/main/README.md")
		assert.NoError(t, err)
		assert.NotNil(t, file)
	})
}
