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
	"fmt"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
)

// MakeToFunc returns a function that converts its argument to the given type
// using the cty/convert package.
//
// It should work just like the "to*" functions in Terraform.
func MakeToFunc(wantTyp cty.Type) function.Function {
	return function.New(&function.Spec{
		Description: fmt.Sprintf("Converts the given value to %s type.", wantTyp.FriendlyName()),
		Params: []function.Parameter{
			{
				Name:             "value",
				Description:      "The value to convert.",
				Type:             cty.DynamicPseudoType,
				AllowNull:        true,
				AllowUnknown:     false,
				AllowMarked:      true,
				AllowDynamicType: true,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			valTyp := args[0].Type()
			if valTyp.Equals(wantTyp) {
				return wantTyp, nil
			}
			if convert.GetConversionUnsafe(args[0].Type(), wantTyp) == nil {
				return cty.NilType, function.NewArgErrorf(
					0,
					fmt.Sprintf(
						"cannot convert %s to %s: ",
						valTyp.FriendlyNameForConstraint(),
						wantTyp.FriendlyNameForConstraint(),
					),
					convert.MismatchMessage(valTyp, wantTyp),
				)
			}
			return wantTyp, nil
		},
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			val, err := convert.Convert(args[0], retType)
			if err != nil {
				return cty.NilVal, function.NewArgErrorf(
					0,
					"cannot convert %s to %s: %s",
					args[0].Type().FriendlyName(),
					retType.FriendlyName(),
					err,
				)
			}
			return val, nil
		},
	})
}
