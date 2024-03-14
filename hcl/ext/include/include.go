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
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"

	utilHCL "github.com/chronicleprotocol/suite/pkg/util/hcl"
)

// Include merges the contents of multiple HCL files specified in the "include"
// attribute. It uses glob patterns.
func Include(ctx *hcl.EvalContext, f fs.FS, body hcl.Body, maxDepth int) (hcl.Body, hcl.Diagnostics) {
	// Decode the "include" attribute.
	content, remain, diags := body.PartialContent(&hcl.BodySchema{
		Attributes: []hcl.AttributeSchema{{Name: "include"}},
	})
	attr := content.Attributes["include"]
	if diags.HasErrors() || attr == nil {
		return body, diags
	}
	var includes []string
	if diags = utilHCL.DecodeExpression(ctx, attr.Expr, &includes); diags.HasErrors() {
		return nil, diags
	}

	// Check for too many nested includes.
	if maxDepth < 0 {
		return nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "Too many nested includes",
			Detail:   "Too many nested includes. Possible circular include.",
			Subject:  attr.Expr.Range().Ptr(),
		}}
	}

	// Iterate over the glob patterns.
	var bodies []hcl.Body
	for _, pattern := range includes {
		// Find all files matching the glob pattern.
		paths, err := glob(f, pattern)
		if err != nil {
			return nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Invalid glob pattern",
				Detail:   fmt.Sprintf("Invalid glob pattern %s: %s.", pattern, err),
				Subject:  attr.Expr.Range().Ptr(),
			}}
		}

		// Iterate over the files from the glob pattern.
		for _, path := range paths {
			// Parse the file.
			fileBody, diags := utilHCL.ParseFileFS(f, path, attr.Expr.Range().Ptr())
			if diags.HasErrors() {
				return nil, diags
			}

			// Allow including files only from the same directory or subdirectories.
			sub, err := fs.Sub(f, filepath.Dir(path))
			if err != nil {
				return nil, hcl.Diagnostics{{
					Severity: hcl.DiagError,
					Summary:  "Invalid include dir pattern",
					Detail:   fmt.Sprintf("Invalid include %s: %s.", path, err),
					Subject:  attr.Expr.Range().Ptr(),
				}}
			}

			// Recursively include files.
			body, diags := Include(ctx, sub, fileBody, maxDepth-1)
			if diags.HasErrors() {
				return nil, diags
			}
			bodies = append(bodies, body)
		}
	}

	// Merge the body of the main file with the bodies of the included files.
	return hcl.MergeBodies(append([]hcl.Body{remain}, bodies...)), nil
}

func glob(f fs.FS, pattern string) ([]string, error) {
	if !strings.Contains(pattern, "*") {
		return []string{pattern}, nil
	}
	return fs.Glob(f, pattern)
}
