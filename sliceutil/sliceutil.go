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

package sliceutil

import (
	"cmp"
	"slices"
)

// Copy returns a copy of the slice.
func Copy[T any](s []T) []T {
	newSlice := make([]T, len(s))
	copy(newSlice, s)
	return newSlice
}

// Contains returns true if s slice contains e element.
func Contains[T comparable](s []T, e T) bool {
	for _, x := range s {
		if x == e {
			return true
		}
	}
	return false
}

// ContainsAll returns true if s slice contains all elements in e slice.
func ContainsAll[T comparable](s []T, e []T) bool {
	for _, x := range e {
		if !Contains(s, x) {
			return false
		}
	}
	return true
}

// Map returns a new slice with the results of applying the function f to each
// element of the original slice.
func Map[T, U any](s []T, f func(T) U) []U {
	out := make([]U, len(s))
	for i, x := range s {
		out[i] = f(x)
	}
	return out
}

// MapErr returns a new slice with the results of applying the function f to each
// element of the original slice and fails when applying the function to eny of the elements fails.
func MapErr[T, U any](s []T, f func(T) (U, error)) (r []U, err error) {
	out := make([]U, len(s))
	for i, x := range s {
		out[i], err = f(x)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Filter returns a new slice with the elements of the original slice that
// satisfy the predicate f.
func Filter[T any](s []T, f func(T) bool) []T {
	out := make([]T, 0, len(s))
	for _, x := range s {
		if f(x) {
			out = append(out, x)
		}
	}
	return out
}

// IsUnique returns true if all elements in the slice are unique.
func IsUnique[T comparable](s []T) bool {
	seen := make(map[T]bool)
	for _, x := range s {
		if seen[x] {
			return false
		}
		seen[x] = true
	}
	return true
}

// Intersect returns a new slice with the elements that are present in all
// slices.
func Intersect[T comparable](slices ...[]T) []T {
	if len(slices) == 0 {
		return nil
	}

	// Find the smallest slice.
	min := slices[0]
	for _, s := range slices {
		if len(s) < len(min) {
			min = s
		}
	}

	// Iterate over the smallest slice and check if the element is present in
	// all other slices.
	out := make([]T, 0, len(min))
	for _, x := range min {
		found := true
		for _, s := range slices {
			if !Contains(s, x) {
				found = false
				break
			}
		}
		if found {
			out = append(out, x)
		}
	}
	return out
}

// Unique returns a new slice with the unique elements of all slices.
// The order of the elements is preserved.
func Unique[T comparable](slices ...[]T) []T {
	if len(slices) == 0 {
		return nil
	}
	s := make(map[T]struct{})
	o := make([]T, 0)
	for _, x := range slices {
		for _, x := range x {
			if _, ok := s[x]; !ok {
				o = append(o, x)
			}
			s[x] = struct{}{}
		}
	}
	return o
}

// Once returns a new slice with the elements that are present only once in all
// slices.
func Once[T comparable](slices ...[]T) []T {
	c := make(map[T]int)
	o := make([]T, 0)
	for _, s := range slices {
		for _, s := range s {
			c[s]++
		}
	}
	for _, s := range slices {
		for _, s := range s {
			if c[s] == 1 {
				o = append(o, s)
			}
		}
	}
	return o
}

// IndexOf returns the index of the first occurrence of e in s, or -1 if e is
// not present in s.
func IndexOf[T comparable](s []T, e T) int {
	for i, x := range s {
		if x == e {
			return i
		}
	}
	return -1
}

// AppendUnique appends e to s if e is not already present in s.
func AppendUnique[T comparable](s []T, ee ...T) []T {
	for _, e := range ee {
		if Contains(s, e) {
			continue
		}
		s = append(s, e)
	}
	return s
}

// AppendUniqueSort appends e to s if e is not already present in s and
// sorts the slice.
//
// Note, that s slice must be sorted before calling this function.
func AppendUniqueSort[T cmp.Ordered](s []T, vs ...T) []T {
	for _, v := range vs {
		idx, has := slices.BinarySearch(s, v)
		if !has {
			s = slices.Insert(s, idx, v)
		}
	}
	return s
}

// AppendUniqueSortFunc appends e to s if e is not already present in s and
// sorts the slice.
//
// Note, that s slice must be sorted before calling this function.
func AppendUniqueSortFunc[T any](s []T, cmp func(T, T) int, vs ...T) []T {
	for _, v := range vs {
		idx, has := slices.BinarySearchFunc(s, v, cmp)
		if !has {
			s = slices.Insert(s, idx, v)
		}
	}
	return s
}
