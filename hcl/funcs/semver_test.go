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

package funcs

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestSemver(t *testing.T) {
	tt := []struct {
		Name      string
		Version   cty.Value
		Target    cty.Value
		Expected  cty.Value
		ExpectErr string
	}{
		{
			Name:      "empty version",
			Version:   cty.StringVal(""),
			Target:    cty.StringVal("1.2.3"),
			ExpectErr: "invalid version",
		},
		{
			Name:      "empty target",
			Version:   cty.StringVal("1.2.3"),
			Target:    cty.StringVal(""),
			ExpectErr: "invalid target",
		},
		{
			Name:      "invalid version",
			Version:   cty.StringVal("hello"),
			Target:    cty.StringVal("1.2.3"),
			ExpectErr: "invalid version",
		},
		{
			Name:      "invalid target",
			Version:   cty.StringVal("1.2.3"),
			Target:    cty.StringVal("1.2.hello"),
			ExpectErr: "invalid target",
		},
		{
			Name:      "invalid target condition",
			Version:   cty.StringVal("1.2.3"),
			Target:    cty.StringVal(")1.2.3"),
			ExpectErr: "unknown condition",
		},
		{
			Name:     "ok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal("1.2.3"),
			Expected: cty.True,
		},
		{
			Name:     "nok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal("1.2.4"),
			Expected: cty.False,
		},
		{
			Name:     "= ok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal("=1.2.3"),
			Expected: cty.True,
		},
		{
			Name:     "= nok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal("=1.2.4"),
			Expected: cty.False,
		},
		{
			Name:     "< ok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal("<1.2.4"),
			Expected: cty.True,
		},
		{
			Name:     "< nok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal("<1.2.3"),
			Expected: cty.False,
		},
		{
			Name:     "> ok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal(">1.2.2"),
			Expected: cty.True,
		},
		{
			Name:     "> nok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal(">1.2.3"),
			Expected: cty.False,
		},
		{
			Name:     "~ greater ok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal("~1.2.2"),
			Expected: cty.True,
		},
		{
			Name:     "~ equal ok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal("~1.2.3"),
			Expected: cty.True,
		},
		{
			Name:     "~ nok",
			Version:  cty.StringVal("1.2.3"),
			Target:   cty.StringVal("~1.2.4"),
			Expected: cty.False,
		},
	}
	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			fn := Semver()
			args := []cty.Value{tc.Version, tc.Target}
			got, err := fn.Call(args)
			if tc.ExpectErr == "" {
				require.NoError(t, err)
				require.Equal(t, tc.Expected, got, "unexpected semver result")
			} else {
				require.ErrorContains(t, err, tc.ExpectErr)
			}
		})
	}
}
