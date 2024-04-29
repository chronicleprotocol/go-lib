package funcs

import (
	"fmt"
	"unicode"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"golang.org/x/mod/semver"
)

const (
	semverLess         byte = '<'
	semverEqual        byte = '='
	semverGreaterEqual byte = '~'
	semverGreater      byte = '>'
)

var (
	semverCompChars = []byte{
		semverLess,
		semverEqual,
		semverGreaterEqual,
		semverGreater,
	}
)

func Semver() function.Function {
	spec := function.Spec{
		Description: "Checks if semver matches target",
		Params: []function.Parameter{
			{
				Name:         "version",
				Description:  "semver to check",
				Type:         cty.String,
				AllowNull:    false,
				AllowUnknown: false,
				// TODO: not sure about marked values, maybe they should be allowed
				AllowMarked:      false,
				AllowDynamicType: true,
			},
			{
				Name:         "target",
				Description:  "conditional target to check",
				Type:         cty.String,
				AllowNull:    false,
				AllowUnknown: false,
				// TODO: not sure about marked values, maybe they should be allowed
				AllowMarked:      false,
				AllowDynamicType: true,
			},
		},
		Type: func(args []cty.Value) (cty.Type, error) {
			if len(args) != 2 {
				return cty.NilType, fmt.Errorf("expected 2 arguments")
			}
			if !args[0].Type().Equals(cty.String) {
				return cty.NilType, fmt.Errorf("version should be a string")
			}
			if !args[1].Type().Equals(cty.String) {
				return cty.NilType, fmt.Errorf("target should be a string")
			}
			return cty.Bool, nil
		},
		Impl: func(args []cty.Value, refType cty.Type) (cty.Value, error) {
			if refType != cty.Bool {
				return cty.NilVal, fmt.Errorf("invalid arguments")
			}

			ver := args[0].AsString()
			target := args[1].AsString()

			if len(target) == 0 {
				return cty.BoolVal(false), fmt.Errorf(`invalid target ""`)
			}

			cond := semverEqual
			if !unicode.IsDigit(rune(target[0])) {
				found := false
				for _, c := range semverCompChars {
					if c == target[0] {
						cond = c
						found = true
						target = target[1:]
						break
					}
				}
				if !found {
					return cty.BoolVal(false), fmt.Errorf("invalid target %q: unknown condition", target)
				}
			}

			ver = "v" + ver
			target = "v" + target

			if !semver.IsValid(ver) {
				return cty.BoolVal(false), fmt.Errorf("invalid version %q", ver)
			}
			if !semver.IsValid(target) {
				return cty.BoolVal(false), fmt.Errorf("invalid target %q", target)
			}

			switch semver.Compare(ver, target) {
			case -1:
				switch cond {
				case semverLess:
					return cty.BoolVal(true), nil
				default:
					return cty.BoolVal(false), nil
				}
			case 0:
				switch cond {
				case semverEqual:
					return cty.BoolVal(true), nil
				case semverGreaterEqual:
					return cty.BoolVal(true), nil
				default:
					return cty.BoolVal(false), nil
				}
			case 1:
				switch cond {
				case semverGreater:
					return cty.BoolVal(true), nil
				case semverGreaterEqual:
					return cty.BoolVal(true), nil
				default:
					return cty.BoolVal(false), nil
				}
			}
			panic("should be unreachable")
		},
	}
	return function.New(&spec)
}
