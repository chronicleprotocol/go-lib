package fsutil

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtoFS(t *testing.T) {
	ctx := context.Background()

	httpFS := NewHTTPFS(ctx, &http.Client{Timeout: 5 * time.Second})

	gatewayFn, err := RandIpfsGatewayFn()
	require.NoError(t, err)
	renameHTTPFS := NewRenameFS(httpFS, gatewayFn)
	retryRenameHTTPFS := NewRetryFS(ctx, renameHTTPFS, 9, time.Second)

	retryHTTPFS := NewRetryFS(ctx, httpFS, 9, time.Second)
	protoFS, err := NewProtoFS(
		ProtoFSItem{Scheme: "ipfs", FS: retryRenameHTTPFS},
		ProtoFSItem{Scheme: "ipfs+gateway", FS: retryRenameHTTPFS},
		ProtoFSItem{Scheme: "https", FS: retryHTTPFS},
	)
	require.NoError(t, err)
	assert.NotNil(t, protoFS)

	t.Run("fetch file from IPFS", func(t *testing.T) {
		t.Skip()
		file, err := protoFS.Open("ipfs://bafybeihkoviema7g3gxyt6la7vd5ho32ictqbilu3wnlo3rs7ewhnp7lly")
		assert.NoError(t, err)
		assert.NotNil(t, file)
	})
	t.Run("fetch file from IPFS GW", func(t *testing.T) {
		t.Skip()
		file, err := protoFS.Open("ipfs+gateway://bafybeihkoviema7g3gxyt6la7vd5ho32ictqbilu3wnlo3rs7ewhnp7lly")
		assert.NoError(t, err)
		assert.NotNil(t, file)
	})
	t.Run("fetch file from HTTP", func(t *testing.T) {
		t.Skip()
		file, err := protoFS.Open("https://raw.githubusercontent.com/chronicleprotocol/charts/main/README.md")
		assert.NoError(t, err)
		assert.NotNil(t, file)
	})
}
