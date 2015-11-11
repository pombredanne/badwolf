// Copyright 2015 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package table

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/badwolf/triple/literal"
	"github.com/google/badwolf/triple/node"
	"github.com/google/badwolf/triple/predicate"
)

func TestNew(t *testing.T) {
	testTable := []struct {
		bs  []string
		err bool
	}{
		{[]string{}, false},
		{[]string{"?foo"}, false},
		{[]string{"?foo", "?bar"}, false},
		{[]string{"?foo", "?bar"}, false},
		{[]string{"?foo", "?bar", "?foo", "?bar"}, true},
	}
	for _, entry := range testTable {
		if _, err := New(entry.bs); (err == nil) == entry.err {
			t.Errorf("table.Name failed; want %v for %v ", entry.err, entry.bs)
		}
	}
}

func TestCellString(t *testing.T) {
	now := time.Now()
	n := node.NewBlankNode()
	p, err := predicate.NewImmutable("foo")
	if err != nil {
		t.Fatalf("failed to create predicate with error %v", err)
	}
	l, err := literal.DefaultBuilder().Parse(`"true"^^type:bool`)
	if err != nil {
		t.Fatalf("failed to create literal with error %v", err)
	}
	testTable := []struct {
		c    *Cell
		want string
	}{
		{c: &Cell{S: "foo"}, want: `foo`},
		{c: &Cell{N: n}, want: n.String()},
		{c: &Cell{P: p}, want: p.String()},
		{c: &Cell{L: l}, want: l.String()},
		{c: &Cell{T: &now}, want: now.Format(time.RFC3339Nano)},
	}
	for _, entry := range testTable {
		if got := entry.c.String(); got != entry.want {
			t.Errorf("Cell.String failed to return the right string; got %q, want %q", got, entry.want)
		}
	}
}

func TestRowToTextLine(t *testing.T) {
	r, b := make(Row), &bytes.Buffer{}
	r["?foo"] = &Cell{S: "foo"}
	r["?bar"] = &Cell{S: "bar"}
	err := r.ToTextLine(b, []string{"?foo", "?bar"}, "")
	if err != nil {
		t.Errorf("row.ToTextLine failed to serialize the row with error %v", err)
	}
	if got, want := b.String(), "foo\tbar"; got != want {
		t.Errorf("row.ToTextLine failed to serialize the row; got %q, want %q", got, want)
	}
}

func TestTableManipulation(t *testing.T) {
	newRow := func() Row {
		r := make(Row)
		r["?foo"] = &Cell{S: "foo"}
		r["?bar"] = &Cell{S: "bar"}
		return r
	}
	tbl, err := New([]string{"?foo", "?bar"})
	if err != nil {
		t.Fatal(errors.New("tbl.New failed to crate a new valid table"))
	}
	for i := 0; i < 10; i++ {
		tbl.AddRow(newRow())
	}
	if got, want := tbl.NumRows(), 10; got != want {
		t.Errorf("tbl.Number: got %d,  wanted %d instead", got, want)
	}
	c := newRow()
	for _, r := range tbl.Rows() {
		if !reflect.DeepEqual(c, r) {
			t.Errorf("tbl contains inconsitent row %v, want %v", r, c)
		}
	}
	for i := 0; i < 10; i++ {
		if r, ok := tbl.Row(i); !ok || !reflect.DeepEqual(c, r) {
			t.Errorf("tbl contains inconsitent row %v, want %v", r, c)
		}
	}
	if got, want := tbl.Bindings(), []string{"?foo", "?bar"}; !reflect.DeepEqual(got, want) {
		t.Errorf("tbl.Bindings() return inconsistent bindings; got %v, want %v", got, want)
	}
}

func TestBindingExtensions(t *testing.T) {
	testBindings := []string{"?foo", "?bar"}
	tbl, err := New(testBindings)
	if err != nil {
		t.Fatal(errors.New("tbl.New failed to crate a new valid table"))
	}
	for _, b := range testBindings {
		if !tbl.HasBinding(b) {
			t.Errorf("tbl.HasBinding(%q) returned false for an existing binding", b)
		}
	}
	newBindings := []string{"?new", "?biding"}
	for _, b := range newBindings {
		if tbl.HasBinding(b) {
			t.Errorf("tbl.HasBinding(%q) returned true for a non existing binding", b)
		}
	}
	mixedBindings := append(testBindings, testBindings...)
	mixedBindings = append(mixedBindings, newBindings...)
	tbl.AddBindings(mixedBindings)
	for _, b := range tbl.Bindings() {
		if !tbl.HasBinding(b) {
			t.Errorf("tbl.HasBinding(%q) returned false for an existing binding", b)
		}
	}
	if got, want := len(tbl.Bindings()), 4; got != want {
		t.Errorf("tbl.Bindings() returned the wrong number of bindings; got %d, want %d", got, want)
	}
}

func TestTableToText(t *testing.T) {
	newRow := func() Row {
		r := make(Row)
		r["?foo"] = &Cell{S: "foo"}
		r["?bar"] = &Cell{S: "bar"}
		return r
	}
	tbl, err := New([]string{"?foo", "?bar"})
	if err != nil {
		t.Fatal(errors.New("tbl.New failed to crate a new valid table"))
	}
	for i := 0; i < 3; i++ {
		tbl.AddRow(newRow())
	}
	want := "?foo, ?bar\nfoo, bar\nfoo, bar\nfoo, bar\n"
	if got, err := tbl.ToText(", "); err != nil || got.String() != want {
		t.Errorf("tbl.ToText failed to rerialize the text;\nGot:\n%s\nWant:\n%s", got, want)
	}
}

func TestEqualBindings(t *testing.T) {
	testTable := []struct {
		b1   map[string]bool
		b2   map[string]bool
		want bool
	}{
		{
			b1:   map[string]bool{},
			b2:   map[string]bool{},
			want: true,
		},
		{
			b1: map[string]bool{},
			b2: map[string]bool{
				"?foo": true,
				"?bar": true,
			},
			want: false,
		},
		{
			b1: map[string]bool{
				"?foo": true,
				"?bar": true,
			},
			b2:   map[string]bool{},
			want: false,
		},
		{
			b1: map[string]bool{
				"?foo": true,
				"?bar": true,
			},
			b2: map[string]bool{
				"?foo":   true,
				"?bar":   true,
				"?other": true,
			},
			want: false,
		},
	}
	for _, entry := range testTable {
		if got, want := equalBindings(entry.b1, entry.b2), entry.want; got != want {
			t.Errorf("equalBidings returned %v instead of %v for values %v, %v", got, want, entry.b1, entry.b2)
		}
	}
}

func testTable(t *testing.T) *Table {
	newRow := func() Row {
		r := make(Row)
		r["?foo"] = &Cell{S: "foo"}
		r["?bar"] = &Cell{S: "bar"}
		return r
	}
	tbl, err := New([]string{"?foo", "?bar"})
	if err != nil {
		t.Fatal(errors.New("tbl.New failed to crate a new valid table"))
	}
	for i := 0; i < 3; i++ {
		tbl.AddRow(newRow())
	}
	return tbl
}

func TestAppendTable(t *testing.T) {
	newEmpty := func() *Table {
		empty, err := New([]string{})
		if err != nil {
			t.Fatal(err)
		}
		return empty
	}
	newNonEmpty := func(twice bool) *Table {
		tbl := testTable(t)
		if twice {
			tbl.data = append(tbl.data, tbl.data...)
		}
		return tbl
	}
	testTable := []struct {
		t    *Table
		t2   *Table
		want *Table
	}{
		{
			t:    newEmpty(),
			t2:   newNonEmpty(false),
			want: newNonEmpty(false),
		},
		{
			t:    newNonEmpty(false),
			t2:   newNonEmpty(false),
			want: newNonEmpty(true),
		},
	}
	for _, entry := range testTable {
		if err := entry.t.AppendTable(entry.t2); err != nil {
			t.Errorf("Failed to append %s to %s with error %v", entry.t2, entry.t, err)
		}
		if got, want := len(entry.t.Bindings()), len(entry.want.Bindings()); got != want {
			t.Errorf("Append returned the wrong number of bindings; got %d, want %d", got, want)
		}
		if got, want := len(entry.t.Rows()), len(entry.want.Rows()); got != want {
			t.Errorf("Append returned the wrong number of rows; got %d, want %d", got, want)
		}
	}
}

func TestDisjoingBinding(t *testing.T) {
	testTable := []struct {
		b1   map[string]bool
		b2   map[string]bool
		want bool
	}{
		{
			b1:   map[string]bool{},
			b2:   map[string]bool{},
			want: true,
		},
		{
			b1: map[string]bool{},
			b2: map[string]bool{
				"?foo": true,
				"?bar": true,
			},
			want: true,
		},
		{
			b1: map[string]bool{
				"?foo": true,
				"?bar": true,
			},
			b2:   map[string]bool{},
			want: true,
		},
		{
			b1: map[string]bool{
				"?foo": true,
				"?bar": true,
			},
			b2: map[string]bool{
				"?foo":   true,
				"?bar":   true,
				"?other": true,
			},
			want: false,
		},
	}
	for _, entry := range testTable {
		if got, want := disjointBinding(entry.b1, entry.b2), entry.want; got != want {
			t.Errorf("equalBidings returned %v instead of %v for values %v, %v", got, want, entry.b1, entry.b2)
		}
	}
}

func testDotTable(t *testing.T, bindings []string, size int) *Table {
	newRow := func(n int) Row {
		r := make(Row)
		for _, b := range bindings {
			r[b] = &Cell{S: fmt.Sprintf("%s_%d", b, n)}
			r[b] = &Cell{S: fmt.Sprintf("%s_%d", b, n)}
		}
		return r
	}
	tbl, err := New(bindings)
	if err != nil {
		t.Fatal(errors.New("tbl.New failed to crate a new valid table"))
	}
	for i := 0; i < size; i++ {
		tbl.AddRow(newRow(i))
	}
	return tbl
}

func TestDotProduct(t *testing.T) {
	testTable := []struct {
		t    *Table
		t2   *Table
		want *Table
	}{
		{
			t:    testDotTable(t, []string{"?foo"}, 3),
			t2:   testDotTable(t, []string{"?bar"}, 3),
			want: testDotTable(t, []string{"?foo", "?bar"}, 9),
		},
		{
			t:    testDotTable(t, []string{"?foo"}, 3),
			t2:   testDotTable(t, []string{"?bar", "?other"}, 6),
			want: testDotTable(t, []string{"?foo", "?bar", "?other"}, 18),
		},
	}
	for _, entry := range testTable {
		if err := entry.t.DotProduct(entry.t2); err != nil {
			t.Errorf("Failed to dot product %s to %s with error %v", entry.t2, entry.t, err)
		}
		if got, want := len(entry.t.Bindings()), len(entry.want.Bindings()); got != want {
			t.Errorf("Append returned the wrong number of bindings; got %d, want %d", got, want)
		}
		if got, want := len(entry.t.Rows()), len(entry.want.Rows()); got != want {
			t.Errorf("Append returned the wrong number of rows; got %d, want %d", got, want)
		}
	}
}

func TestDotProductContent(t *testing.T) {
	t1, t2 := testDotTable(t, []string{"?foo"}, 3), testDotTable(t, []string{"?bar"}, 3)
	if err := t1.DotProduct(t2); err != nil {
		t.Errorf("Failed to dot product %s to %s with error %v", t2, t1, err)
	}
	if len(t1.Rows()) != 9 {
		t.Errorf("DotProduct returned the wrong number of rows (%d)", len(t1.Rows()))
	}
	if len(t1.Bindings()) != 2 {
		t.Errorf("DotProduct returned the wrong number of bindings (%d)", len(t1.Bindings()))
	}
	fn := func(idx int) *Cell {
		return &Cell{S: fmt.Sprintf("?foo_%d", idx/3)}
	}
	bn := func(idx int) *Cell {
		return &Cell{S: fmt.Sprintf("?bar_%d", idx%3)}
	}
	for idx, r := range t1.Rows() {
		if gf, wf, gb, wb := r["?foo"], fn(idx), r["?bar"], bn(idx); !reflect.DeepEqual(gf, wf) || !reflect.DeepEqual(gb, wb) {
			t.Errorf("DotProduct returned the wrong row %v on position %d; %v %v %v %v", r, idx, gf, wf, gb, wb)
		}
	}
}

func TestDeleteRow(t *testing.T) {
	testTable := []struct {
		t   *Table
		idx int
		out bool
	}{
		{
			t:   testDotTable(t, []string{"?foo"}, 3),
			idx: -1,
			out: false,
		},
		{
			t:   testDotTable(t, []string{"?foo"}, 3),
			idx: 0,
			out: true,
		},
		{
			t:   testDotTable(t, []string{"?foo"}, 3),
			idx: 1,
			out: true,
		},
		{
			t:   testDotTable(t, []string{"?foo"}, 3),
			idx: 2,
			out: true,
		},
		{
			t:   testDotTable(t, []string{"?foo"}, 3),
			idx: 3,
			out: false,
		},
	}
	for _, entry := range testTable {
		if err := entry.t.DeleteRow(entry.idx); (err != nil) == entry.out {
			t.Errorf("Failed to delete row %d with error %v", entry.idx, err)
		}
		if entry.out && len(entry.t.Rows()) != 2 {
			t.Errorf("Failed successfully delete row %d ending with %d rows", entry.idx, len(entry.t.Rows()))
		}
	}
}

func TestTruncate(t *testing.T) {
	tbl := testDotTable(t, []string{"?foo"}, 3)

	if got, want := len(tbl.Rows()), 3; got != want {
		t.Errorf("Failed to create a table with %d rows instead of %v", got, want)
	}
	tbl.Truncate()
	if got, want := len(tbl.Rows()), 0; got != want {
		t.Errorf("Failed to create a table with %d rows instead of %v", got, want)
	}
}
