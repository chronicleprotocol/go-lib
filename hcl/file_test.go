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

package hcl

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
)

func TestParseFiles(t *testing.T) {
	tests := []struct {
		name          string
		paths         []string
		expectedBody  hcl.Body
		expectedError string
	}{
		{
			name: "valid configurations",
			paths: []string{
				"./testdata/valid1.hcl",
				"./testdata/valid2.hcl",
			},
		},
		{
			name: "invalid configurations",
			paths: []string{
				"./testdata/valid1.hcl",
				"./testdata/invalid.hcl",
			},
			expectedError: "invalid.hcl", // Invalid file must be reported.
		},
		{
			name: "non-existent file",
			paths: []string{
				"./testdata/valid1.hcl",
				"./testdata/non-existent.hcl",
			},
			expectedError: "Cannot read file ./testdata/non-existent.hcl", // Non-existent file must be reported.
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, diags := ParseFiles(tt.paths, nil)
			if len(tt.expectedError) > 0 {
				assert.NotNil(t, diags)
				assert.True(t, diags.HasErrors())
				assert.Contains(t, diags.Error(), tt.expectedError)

			} else {
				assert.Nil(t, diags)
				assert.False(t, diags.HasErrors())
				assert.NotNil(t, body)
			}
		})
	}
}
