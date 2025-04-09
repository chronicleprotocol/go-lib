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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	m := []string{"a", "b", "c"}
	n := Copy(m)
	assert.Equal(t, m, n)
	assert.NotSame(t, &m, &n)
}

func TestContains(t *testing.T) {
	m := []string{"a", "b", "c"}
	assert.True(t, Contains(m, "a"))
	assert.False(t, Contains(m, "d"))
}

func TestContainsAll(t *testing.T) {
	m := []string{"a", "b", "c"}
	assert.True(t, ContainsAll(m, []string{"a", "b"}))
	assert.False(t, ContainsAll(m, []string{"a", "d"}))
}

func TestMap(t *testing.T) {
	m := []string{"a", "b", "c"}
	n := Map(m, strings.ToUpper)
	assert.Equal(t, []string{"A", "B", "C"}, n)
	assert.NotSame(t, &m, &n)
}

func TestFilter(t *testing.T) {
	m := []string{"a", "b", "c"}
	n := Filter(m, func(s string) bool { return s != "c" })
	assert.Equal(t, []string{"a", "b"}, n)
	assert.NotSame(t, &m, &n)
}

func TestIsUnique(t *testing.T) {
	assert.True(t, IsUnique([]string{"a", "b", "c"}))
	assert.False(t, IsUnique([]string{"a", "b", "a"}))
}

func TestIntersect(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, Intersect([]string{"a", "b", "c"}, []string{"a", "b"}))
	assert.Equal(t, []string{"a", "b"}, Intersect([]string{"a", "b"}, []string{"a", "b", "c"}))
	assert.Equal(t, []string{"a", "b"}, Intersect([]string{"a", "b", "c"}, []string{"a", "b", "c"}, []string{"a", "b"}))
	assert.Equal(t, []string{}, Intersect([]string{"a", "b", "c"}, []string{"d", "e", "f"}))
	assert.Equal(t, []string{}, Intersect([]string{"d", "e", "f"}, []string{"a", "b", "c"}))
	assert.Equal(t, []string{}, Intersect([]string{"a", "b", "c"}, []string{}))
	assert.Equal(t, []string{}, Intersect([]string{}, []string{"a", "b", "c"}))
	assert.Equal(t, []string{}, Intersect([]string{}, []string{}))
}

func TestUnique(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, Unique([]string{"a", "b", "c"}))
	assert.Equal(t, []string{"a", "b", "c"}, Unique([]string{"a", "b", "a", "c", "b"}))
	assert.Equal(t, []string{"a", "b", "c"}, Unique([]string{"a", "b", "c", "a", "b", "c"}))
	assert.Equal(t, []string{"a", "b"}, Unique([]string{"a", "b"}, []string{"b"}))
	assert.Equal(t, []string{}, Unique([]string{}))
}

func TestOnce(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, Once([]string{"a", "b", "c"}))
	assert.Equal(t, []string{"c"}, Once([]string{"a", "b", "a", "c", "b"}))
	assert.Equal(t, []string{}, Once([]string{"a", "b", "c", "a", "b", "c"}))
	assert.Equal(t, []string{"a"}, Once([]string{"a", "b"}, []string{"b"}))
	assert.Equal(t, []string{}, Once([]string{}))
}

func TestIndexOf(t *testing.T) {
	assert.Equal(t, 1, IndexOf([]string{"a", "b", "c"}, "b"))
	assert.Equal(t, -1, IndexOf([]string{"a", "b", "c"}, "d"))
}

func TestAppendUnique(t *testing.T) {
	assert.Equal(t, []string{"a", "b", "c"}, AppendUnique([]string{"a", "b"}, "c"))
	assert.Equal(t, []string{"a", "b", "c"}, AppendUnique([]string{"a", "b", "c"}, "c"))
	assert.Equal(t, []string{"a", "b", "c", "d"}, AppendUnique([]string{"a", "b", "c"}, "d"))
	assert.Equal(t, []string{"a", "b", "c", "d"}, AppendUnique([]string{"a", "b", "c"}, "c", "d"))
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, AppendUnique([]string{"a", "b", "c"}, "d", "e"))
}

func TestAppendUniqueSort(t *testing.T) {
	assert.Equal(t, []int{5}, AppendUniqueSort([]int{}, 5))
	assert.Equal(t, []int{1, 2, 3, 4, 5}, AppendUniqueSort([]int{1, 3, 5}, 2, 4))
	assert.Equal(t, []int{1, 2, 3, 4, 5}, AppendUniqueSort([]int{1, 3, 5}, 2, 3, 4))
	assert.Equal(t, []int{1, 2, 3}, AppendUniqueSort([]int{1, 2, 3}, 1, 2, 3))
	assert.Equal(t, []string{"a", "b", "c", "d"}, AppendUniqueSort([]string{"a", "c"}, "b", "d", "c"))
	assert.Equal(t, []float64{1.1, 2.2, 3.3, 4.4}, AppendUniqueSort([]float64{1.1, 3.3}, 2.2, 4.4))
}

func TestAppendUniqueSortFunc(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	compareByAge := func(a, b Person) int {
		return a.Age - b.Age
	}

	alice := Person{Name: "Alice", Age: 30}
	bob := Person{Name: "Bob", Age: 25}
	charlie := Person{Name: "Charlie", Age: 35}

	assert.Equal(t, []Person{alice}, AppendUniqueSortFunc([]Person{}, compareByAge, alice))
	assert.Equal(t, []Person{bob, alice, charlie}, AppendUniqueSortFunc([]Person{bob, charlie}, compareByAge, alice))
	assert.Equal(t, []Person{bob, alice}, AppendUniqueSortFunc([]Person{bob, alice}, compareByAge, alice))

	// Test with reverse sorted integers
	reverseCompare := func(a, b int) int {
		return b - a
	}

	assert.Equal(t, []int{5, 3, 1}, AppendUniqueSortFunc([]int{5, 1}, reverseCompare, 3))
	assert.Equal(t, []int{5, 3, 1}, AppendUniqueSortFunc([]int{5, 3, 1}, reverseCompare, 3))
}

func TestMapErr(t *testing.T) {
	// Test successful mapping
	m := []int{1, 2, 3}
	successFn := func(i int) (string, error) {
		return string(rune(i + '0')), nil
	}
	result, err := MapErr(m, successFn)
	assert.NoError(t, err)
	assert.Equal(t, []string{"1", "2", "3"}, result)

	// Test error case
	errFn := func(i int) (string, error) {
		if i == 2 {
			return "", assert.AnError
		}
		return string(rune(i + '0')), nil
	}
	result, err = MapErr(m, errFn)
	assert.Error(t, err)
	assert.Equal(t, assert.AnError, err)
	assert.Nil(t, result)
}
