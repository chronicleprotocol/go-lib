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

// Package fsutil provides various implementations of the fs.FS interface,
// primarily for working with remote files.
//
// To support URI schemes, the package defines the Protocol interface, which
// takes a URI and returns an appropriate fs.FS implementation and the path to the
// file within the filesystem.
//
// To support multiple URI schemes, the package provides the "Mux" protocol,
// which delegates the URI to the appropriate protocol based on the scheme.
//
// Example:
//
//	mux := NewMux(map[string]ProtoFunc{
//		"file":  func(uri *url.URL) (Protocol, error) { return NewFileProto(), nil },
//		"http":  func(uri *url.URL) (Protocol, error) { return NewHTTPProto(context.Background()), nil },
//		"https": func(uri *url.URL) (Protocol, error) { return NewHTTPProto(context.Background()), nil },
//	})
//
//	// Get the appropriate filesystem and path for the URI.
//	fs, path, err := mux.FileSystem(errutil.Must(http.Parse("http://example.com/file")))
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	b, err := fs.ReadFile(path)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Println(string(b))
package fsutil
