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

package secrets

import (
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	utilHCL "github.com/chronicleprotocol/go-lib/hcl"
)

const (
	keystoreFile = "./testdata/keystore.json"
	keystorePass = "helloworld"
)

func TestVariables(t *testing.T) {
	tt := []struct {
		filename        string
		skipDecrypt     bool
		expectedErr     string
		expectedSecrets map[string]cty.Value
	}{
		{
			filename: "./testdata/valid.hcl",
			expectedSecrets: map[string]cty.Value{
				"foo": cty.StringVal("hello world\n"),
			},
		},
		{
			filename:    "./testdata/valid-skip-decrypt.hcl",
			skipDecrypt: true,
			expectedSecrets: map[string]cty.Value{
				"foo": cty.StringVal("<encrypted>"),
			},
		},
		{
			filename:    "./testdata/wrong-key.hcl",
			expectedErr: "Secret can not be decrypted",
		},
		{
			filename:    "./testdata/missing-own-key.hcl",
			expectedErr: "Secret has no value for own public key",
		},
		{
			filename:    "./testdata/wrong-types.hcl",
			expectedErr: "Secret value is not a map",
		},
		{
			filename:    "./testdata/wrong-hex-value.hcl",
			expectedErr: "Secret is not hex encoded",
		},
		{
			filename:    "./testdata/missing-eth-block.hcl",
			expectedErr: "Wrong number of etherum blocks",
		},
		{
			filename:    "./testdata/duplicated-value.hcl",
			expectedErr: "Duplicate secret value",
		},
	}

	prev := os.Getenv(skipDecryptEnv)
	os.Setenv(skipDecryptEnv, "true")
	t.Cleanup(func() {
		os.Setenv(skipDecryptEnv, prev)
	})
	for _, tc := range tt {
		t.Run(tc.filename, func(t *testing.T) {
			os.Setenv(skipDecryptEnv, strconv.FormatBool(tc.skipDecrypt))

			body, diags := utilHCL.ParseFile(tc.filename, nil)
			require.False(t, diags.HasErrors(), diags.Error())

			hclCtx := &hcl.EvalContext{}
			body, diags = DecryptSecrets(hclCtx, body)
			if tc.expectedErr != "" {
				require.True(t, diags.HasErrors())
				require.Contains(t, diags.Error(), tc.expectedErr)
			} else {
				require.False(t, diags.HasErrors(), errors.Join(diags.Errs()...))
				object := hclCtx.Variables[varName]
				vars := make(map[string]cty.Value)
				if !object.IsNull() {
					vars = object.AsValueMap()
				}
				for k, v := range tc.expectedSecrets {
					assert.True(t, vars[k].RawEquals(v), "%s: expected %s to equal %s", k, vars[k].GoString(), v.GoString())
				}

				// "secrets" block should be removed from the body.
				secretBlocks, _, diags := body.PartialContent(&hcl.BodySchema{
					Blocks: []hcl.BlockHeaderSchema{{Type: secretsBlockName}},
				})
				require.False(t, diags.HasErrors(), diags.Error())
				require.Empty(t, secretBlocks.Blocks)
			}
		})
	}
}
