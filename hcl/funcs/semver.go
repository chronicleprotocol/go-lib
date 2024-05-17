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
	"errors"
	"fmt"
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"golang.org/x/mod/semver"
)

const (
	semverLess         string = "<"
	semverLessEqual    string = "<="
	semverGreater      string = ">"
	semverGreaterEqual string = ">="
	semverEqual        string = "="
	semverNotEqual     string = "!="
)

func Semver() function.Function {
	spec := function.Spec{
		Description: "Checks if semver matches target",
		Params: []function.Parameter{
			{
				Name:             "version",
				Description:      "Semver version to check.",
				Type:             cty.String,
				AllowNull:        false,
				AllowUnknown:     false,
				AllowMarked:      true,
				AllowDynamicType: true,
			},
			{
				Name:             "target",
				Description:      "Semver target to check against.",
				Type:             cty.String,
				AllowNull:        false,
				AllowUnknown:     false,
				AllowMarked:      true,
				AllowDynamicType: true,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			if len(args) != 2 {
				return cty.NilType, errors.New("expected 2 arguments")
			}
			if !args[0].Type().Equals(cty.String) {
				return cty.NilType, errors.New("version should be a string")
			}
			if !args[1].Type().Equals(cty.String) {
				return cty.NilType, errors.New("target should be a string")
			}
			return cty.Bool, nil
		},
		Impl: func(args []cty.Value, refType cty.Type) (cty.Value, error) {
			if refType != cty.Bool {
				return cty.NilVal, errors.New("invalid arguments")
			}

			version := args[0].AsString()
			target := args[1].AsString()

			if len(version) == 0 {
				return cty.BoolVal(false), errors.New(`invalid version`)
			}
			if len(target) == 0 {
				return cty.BoolVal(false), errors.New(`invalid target`)
			}

			condition := semverEqual
			switch {
			// 2 char conditions
			case strings.HasPrefix(target, semverLessEqual):
				condition = semverLessEqual
			case strings.HasPrefix(target, semverGreaterEqual):
				condition = semverGreaterEqual
			case strings.HasPrefix(target, semverNotEqual):
				condition = semverNotEqual
			// 1 char conditions
			case strings.HasPrefix(target, semverLess):
				condition = semverLess
			case strings.HasPrefix(target, semverGreater):
				condition = semverGreater
			case strings.HasPrefix(target, semverEqual):
				condition = semverEqual
			default:
			}

			// Special case, dev version.
			// The dev version is always higher than any other version.
			if version == "dev" {
				switch condition {
				case semverLess:
					return cty.False, nil
				case semverLessEqual:
					return cty.False, nil
				case semverGreater:
					return cty.True, nil
				case semverGreaterEqual:
					return cty.True, nil
				case semverEqual:
					return cty.False, nil
				case semverNotEqual:
					return cty.True, nil
				}
			}

			target = strings.TrimPrefix(target, condition)   // remove condition from target
			version = "v" + strings.TrimPrefix(version, "v") // add v prefix to version if missing
			target = "v" + strings.TrimPrefix(target, "v")   // add v prefix to target if missing

			if !semver.IsValid(version) {
				return cty.BoolVal(false), fmt.Errorf("invalid version %q", version[1:])
			}
			if !semver.IsValid(target) {
				return cty.BoolVal(false), fmt.Errorf("invalid target %q", target[1:])
			}

			switch semver.Compare(version, target) {
			case -1:
				switch condition {
				case semverLess:
					return cty.True, nil
				case semverLessEqual:
					return cty.True, nil
				default:
					return cty.False, nil
				}
			case 0:
				switch condition {
				case semverLessEqual:
					return cty.True, nil
				case semverGreaterEqual:
					return cty.True, nil
				case semverEqual:
					return cty.True, nil
				default:
					return cty.False, nil
				}
			case 1:
				switch condition {
				case semverGreater:
					return cty.True, nil
				case semverGreaterEqual:
					return cty.True, nil
				default:
					return cty.False, nil
				}
			}

			panic("unreachable")
		},
	}
	return function.New(&spec)
}
