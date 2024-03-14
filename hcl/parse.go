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
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ParseSource parses a single HCL source file.
func ParseSource(filename string, src []byte) (hcl.Body, hcl.Diagnostics) {
	file, diags := hclsyntax.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, diags
	}
	return file.Body, diags
}

// ParseSources into separate bodies and merge them into one.
//
// The keys of the map are used as the filename for the source.
func ParseSources(srcs map[string][]byte) (hcl.Body, hcl.Diagnostics) {
	var (
		bodies []hcl.Body
		diags  hcl.Diagnostics
	)
	for k, v := range srcs {
		srcBody, srcDiags := ParseSource(k, v)
		diags = diags.Extend(srcDiags)
		if srcDiags.HasErrors() {
			continue
		}
		bodies = append(bodies, srcBody)
	}
	if diags.HasErrors() {
		return nil, diags
	}
	return hcl.MergeBodies(bodies), diags
}

// ParseFile parses a single HCL file from the filesystem.
func ParseFile(path string, subject *hcl.Range) (hcl.Body, hcl.Diagnostics) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "Failed to read configuration",
			Detail:   fmt.Sprintf("Cannot read file %s: %s.", path, err),
			Subject:  subject,
		}}
	}
	return ParseSource(filepath.Base(path), src)
}

// ParseFileFS parses a single HCL file using the given fs.FS.
func ParseFileFS(f fs.FS, path string, subject *hcl.Range) (hcl.Body, hcl.Diagnostics) {
	src, err := fs.ReadFile(f, path)
	if err != nil {
		return nil, hcl.Diagnostics{{
			Severity: hcl.DiagError,
			Summary:  "Failed to read configuration",
			Detail:   fmt.Sprintf("Cannot read file %s: %s.", path, err),
			Subject:  subject,
		}}
	}
	return ParseSource(filepath.Base(path), src)
}
