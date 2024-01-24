//  Copyright (C) 2021-2023 Chronicle Labs, Inc.
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.

package include

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	utilHCL "github.com/chronicleprotocol/suite/pkg/util/hcl"
)

func TestVariables(t *testing.T) {
	tests := []struct {
		filename    string
		asserts     func(t *testing.T, body hcl.Body)
		expectedErr string
	}{
		{
			filename: "./testdata/include.hcl",
			asserts: func(t *testing.T, body hcl.Body) {
				attrs, diags := body.JustAttributes()
				require.False(t, diags.HasErrors(), diags.Error())
				assert.NotNil(t, attrs["foo"])
				assert.NotNil(t, attrs["bar"])
			},
		},
		{
			filename: "./testdata/relative-dir.hcl",
			asserts: func(t *testing.T, body hcl.Body) {
				attrs, diags := body.JustAttributes()
				require.False(t, diags.HasErrors(), diags.Error())
				assert.NotNil(t, attrs["foo"])
			},
		},
		{
			filename:    "./testdata/self-include.hcl",
			expectedErr: "self-include.hcl:1,11-3,2: Too many nested includes;",
		},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			body, diags := utilHCL.ParseFile(tt.filename, nil)
			require.False(t, diags.HasErrors(), diags.Error())

			hclCtx := &hcl.EvalContext{}
			body, diags = Include(hclCtx, body, "./testdata", 2)
			if tt.expectedErr != "" {
				require.True(t, diags.HasErrors(), diags.Error())
				assert.Contains(t, diags.Error(), tt.expectedErr)
				return
			} else {
				require.False(t, diags.HasErrors(), diags.Error())
				tt.asserts(t, body)

				// The "include" attribute should be removed from the body.
				emptySchema := &hcl.BodySchema{
					Attributes: []hcl.AttributeSchema{{Name: "foo"}, {Name: "bar"}},
				}
				_, diags = body.Content(emptySchema)
				require.False(t, diags.HasErrors(), diags.Error())
			}
		})
	}
}
