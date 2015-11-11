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

package semantic

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/badwolf/bql/lexer"
	"github.com/google/badwolf/triple"
	"github.com/google/badwolf/triple/literal"
	"github.com/google/badwolf/triple/node"
	"github.com/google/badwolf/triple/predicate"
)

func TestDataAccumulatorHook(t *testing.T) {
	st := &Statement{}
	ces := []ConsumedElement{
		NewConsumedSymbol("FOO"),
		NewConsumedToken(&lexer.Token{
			Type: lexer.ItemNode,
			Text: "/_<s>",
		}),
		NewConsumedSymbol("FOO"),
		NewConsumedToken(&lexer.Token{
			Type: lexer.ItemPredicate,
			Text: `"p"@[]`,
		}),
		NewConsumedSymbol("FOO"),
		NewConsumedToken(&lexer.Token{
			Type: lexer.ItemNode,
			Text: "/_<o>",
		}),
		NewConsumedSymbol("FOO"),
		NewConsumedToken(&lexer.Token{
			Type: lexer.ItemNode,
			Text: "/_<s>",
		}),
		NewConsumedSymbol("FOO"),
		NewConsumedToken(&lexer.Token{
			Type: lexer.ItemPredicate,
			Text: `"p"@[]`,
		}),
		NewConsumedSymbol("FOO"),
		NewConsumedToken(&lexer.Token{
			Type: lexer.ItemNode,
			Text: "/_<o>",
		}),
	}
	var (
		hook ElementHook
		err  error
	)
	hook = dataAccumulator(literal.DefaultBuilder())
	for _, ce := range ces {
		hook, err = hook(st, ce)
		if err != nil {
			t.Errorf("semantic.DataAccumulator hook should have never failed for %v with error %v", ce, err)
		}
	}
	data := st.Data()
	if len(data) != 2 {
		t.Errorf("semantic.DataAccumulator hook should have produced 2 triples; instead produced %v", st.Data())
	}
	for _, trpl := range data {
		if got, want := trpl.S().String(), "/_<s>"; got != want {
			t.Errorf("semantic.DataAccumulator hook failed to parse subject correctly; got %v, want %v", got, want)
		}
		if got, want := trpl.P().String(), `"p"@[]`; got != want {
			t.Errorf("semantic.DataAccumulator hook failed to parse prdicate correctly; got %v, want %v", got, want)
		}
		if got, want := trpl.O().String(), "/_<o>"; got != want {
			t.Errorf("semantic.DataAccumulator hook failed to parse object correctly; got %v, want %v", got, want)
		}
	}
}

func TestSemanticAcceptInsertDelete(t *testing.T) {
	st := &Statement{}
	ces := []ConsumedElement{
		NewConsumedSymbol("FOO"),
		NewConsumedToken(&lexer.Token{
			Type: lexer.ItemBinding,
			Text: "?foo",
		}),
		NewConsumedToken(&lexer.Token{
			Type: lexer.ItemComma,
			Text: ",",
		}),
		NewConsumedSymbol("FOO"),
		NewConsumedToken(&lexer.Token{
			Type: lexer.ItemBinding,
			Text: "?bar",
		}),
	}
	var (
		hook ElementHook
		err  error
	)
	hook = graphAccumulator()
	for _, ce := range ces {
		hook, err = hook(st, ce)
		if err != nil {
			t.Errorf("semantic.GraphAccumulator hook should have never failed for %v with error %v", ce, err)
		}
	}
	data := st.Graphs()
	if len(data) != 2 {
		t.Errorf("semantic.GraphAccumulator hook should have produced 2 graph bindings; instead produced %v", st.Graphs())
	}
	for _, g := range data {
		if g != "?foo" && g != "?bar" {
			t.Errorf("semantic.GraphAccumulator hook failed to provied either ?foo or ?bar; got %v instead", g)
		}
	}
}

func TestTypeBindingClauseHook(t *testing.T) {
	f := TypeBindingClauseHook(Insert)
	st := &Statement{}
	f(st, Symbol("FOO"))
	if got, want := st.Type(), Insert; got != want {
		t.Errorf("semantic.TypeBidingHook failed to set the right type; got %s, want %s", got, want)
	}
}

func TestWhereInitClauseHook(t *testing.T) {
	f := whereInitWorkingClause()
	st := &Statement{}
	f(st, Symbol("FOO"))
	if st.WorkingClause() == nil {
		t.Errorf("semantic.WhereInitWorkingClause should have returned a valid working clause for statement %v", st)
	}
}

func TestWhereWorkingClauseHook(t *testing.T) {
	f := whereNextWorkingClause()
	st := &Statement{}
	st.ResetWorkingGraphClause()
	f(st, Symbol("FOO"))
	f(st, Symbol("FOO"))
	if len(st.GraphPatternClauses()) != 2 {
		t.Errorf("semantic.whereNextWorkingClause should have returned two clauses for statement %v", st)
	}
}

type testTable struct {
	valid bool
	id    string
	ces   []ConsumedElement
	want  *GraphClause
}

func runTabulatedClauseHookTest(t *testing.T, testName string, f ElementHook, table []testTable) {
	st := &Statement{}
	st.ResetWorkingGraphClause()
	failed := false
	for _, entry := range table {
		for _, ce := range entry.ces {
			if _, err := f(st, ce); err != nil {
				if entry.valid {
					t.Errorf("%s case %q should have never failed with error: %v", testName, entry.id, err)
				} else {
					failed = true
				}
			}
		}
		if entry.valid {
			if got, want := st.WorkingClause(), entry.want; !reflect.DeepEqual(got, want) {
				t.Errorf("%s case %q should have populated all subject fields; got %+v, want %+v", testName, entry.id, got, want)
			}
		} else {
			if !failed {
				t.Errorf("%s failed to reject invalid case %q", testName, entry.id)
			}
		}
		st.ResetWorkingGraphClause()
	}
}

func TestWhereSubjectClauseHook(t *testing.T) {
	st := &Statement{}
	f := whereSubjectClause()
	st.ResetWorkingGraphClause()
	n, err := node.Parse("/_<foo>")
	if err != nil {
		t.Fatalf("node.Parse failed with error %v", err)
	}
	runTabulatedClauseHookTest(t, "semantic.whereSubjectClause", f, []testTable{
		{
			valid: true,
			id:    "node_example",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemNode,
					Text: "/_<foo>",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemType,
					Text: "type",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				S:          n,
				SAlias:     "?bar",
				STypeAlias: "?bar2",
				SIDAlias:   "?bar3",
			},
		},
		{
			valid: true,
			id:    "binding_example",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?foo",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemType,
					Text: "type",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				SBinding:   "?foo",
				SAlias:     "?bar",
				STypeAlias: "?bar2",
				SIDAlias:   "?bar3",
			},
		},
	})
}

func TestWherePredicatClauseHook(t *testing.T) {
	st := &Statement{}
	f := wherePredicateClause()
	st.ResetWorkingGraphClause()
	p, err := predicate.Parse(`"foo"@[2015-07-19T13:12:04.669618843-07:00]`)
	if err != nil {
		t.Fatalf("predicate.Parse failed with error %v", err)
	}
	tlb, err := time.Parse(time.RFC3339Nano, `2015-07-19T13:12:04.669618843-07:00`)
	if err != nil {
		t.Fatalf("time.Parse failed to parse valid lower time bound with error %v", err)
	}
	tub, err := time.Parse(time.RFC3339Nano, `2016-07-19T13:12:04.669618843-07:00`)
	if err != nil {
		t.Fatalf("time.Parse failed to parse valid upper time bound with error %v", err)
	}
	runTabulatedClauseHookTest(t, "semantic.wherePredicateClause", f, []testTable{
		{
			valid: true,
			id:    "valid predicate",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemPredicate,
					Text: `"foo"@[2015-07-19T13:12:04.669618843-07:00]`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAt,
					Text: "at",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				P:            p,
				PAlias:       "?bar",
				PIDAlias:     "?bar2",
				PAnchorAlias: "?bar3",
				PTemporal:    true,
			},
		},
		{
			valid: true,
			id:    "valid predicate with binding",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemPredicate,
					Text: `"foo"@[?foo]`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAt,
					Text: "at",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				PID:            "foo",
				PAnchorBinding: "?foo",
				PAlias:         "?bar",
				PIDAlias:       "?bar2",
				PAnchorAlias:   "?bar3",
				PTemporal:      true,
			},
		},
		{
			valid: true,
			id:    "valid bound with bindings",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemPredicateBound,
					Text: `"foo"@[?fooLower,?fooUpper]`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAt,
					Text: "at",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				PID:              "foo",
				PLowerBoundAlias: "?fooLower",
				PUpperBoundAlias: "?fooUpper",
				PAlias:           "?bar",
				PIDAlias:         "?bar2",
				PAnchorAlias:     "?bar3",
				PTemporal:        true,
			},
		},
		{
			valid: true,
			id:    "valid bound with dates",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemPredicateBound,
					Text: `"foo"@[2015-07-19T13:12:04.669618843-07:00,2016-07-19T13:12:04.669618843-07:00]`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAt,
					Text: "at",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				PID:          "foo",
				PLowerBound:  &tlb,
				PUpperBound:  &tub,
				PAlias:       "?bar",
				PIDAlias:     "?bar2",
				PAnchorAlias: "?bar3",
				PTemporal:    true,
			},
		},
		{
			valid: false,
			id:    "invalid bound with dates",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemPredicateBound,
					Text: `"foo"@[2016-07-19T13:12:04.669618843-07:00,2015-07-19T13:12:04.669618843-07:00]`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAt,
					Text: "at",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{},
		},
	})
}

func TestWhereObjectClauseHook(t *testing.T) {
	st := &Statement{}
	f := whereObjectClause()
	st.ResetWorkingGraphClause()
	node, err := node.Parse("/_<foo>")
	if err != nil {
		t.Fatalf("node.Parse failed with error %v", err)
	}
	n := triple.NewNodeObject(node)
	pred, err := predicate.Parse(`"foo"@[2015-07-19T13:12:04.669618843-07:00]`)
	if err != nil {
		t.Fatalf("predicate.Parse failed with error %v", err)
	}
	p := triple.NewPredicateObject(pred)
	tlb, err := time.Parse(time.RFC3339Nano, `2015-07-19T13:12:04.669618843-07:00`)
	if err != nil {
		t.Fatalf("time.Parse failed to parse valid lower time bound with error %v", err)
	}
	tub, err := time.Parse(time.RFC3339Nano, `2016-07-19T13:12:04.669618843-07:00`)
	if err != nil {
		t.Fatalf("time.Parse failed to parse valid upper time bound with error %v", err)
	}
	l, err := triple.ParseObject(`"1"^^type:int64`, literal.DefaultBuilder())
	if err != nil {
		t.Fatalf("literal.Parse should have never fail to pars %s with error %v", `"1"^^type:int64`, err)
	}

	runTabulatedClauseHookTest(t, "semantic.whereObjectClause", f, []testTable{
		{
			valid: true,
			id:    "node_example",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemNode,
					Text: "/_<foo>",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemType,
					Text: "type",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				O:          n,
				OAlias:     "?bar",
				OTypeAlias: "?bar2",
				OIDAlias:   "?bar3",
			},
		},
		{
			valid: true,
			id:    "binding_example",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?foo",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemType,
					Text: "type",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				OBinding:   "?foo",
				OAlias:     "?bar",
				OTypeAlias: "?bar2",
				OIDAlias:   "?bar3",
			},
		},
		{
			valid: true,
			id:    "valid predicate",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemPredicate,
					Text: `"foo"@[2015-07-19T13:12:04.669618843-07:00]`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAt,
					Text: "at",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				O:            p,
				OAlias:       "?bar",
				OIDAlias:     "?bar2",
				OAnchorAlias: "?bar3",
				OTemporal:    true,
			},
		},
		{
			valid: true,
			id:    "valid predicate with binding",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemPredicate,
					Text: `"foo"@[?foo]`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAt,
					Text: "at",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				OID:            "foo",
				OAnchorBinding: "?foo",
				OAlias:         "?bar",
				OIDAlias:       "?bar2",
				OAnchorAlias:   "?bar3",
				OTemporal:      true,
			},
		},
		{
			valid: true,
			id:    "valid bound with bindings",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemPredicateBound,
					Text: `"foo"@[?fooLower,?fooUpper]`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAt,
					Text: "at",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				OID:              "foo",
				OLowerBoundAlias: "?fooLower",
				OUpperBoundAlias: "?fooUpper",
				OAlias:           "?bar",
				OIDAlias:         "?bar2",
				OAnchorAlias:     "?bar3",
				OTemporal:        true,
			},
		},
		{
			valid: true,
			id:    "valid bound with dates",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemPredicateBound,
					Text: `"foo"@[2015-07-19T13:12:04.669618843-07:00,2016-07-19T13:12:04.669618843-07:00]`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAt,
					Text: "at",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				OID:          "foo",
				OLowerBound:  &tlb,
				OUpperBound:  &tub,
				OAlias:       "?bar",
				OIDAlias:     "?bar2",
				OAnchorAlias: "?bar3",
				OTemporal:    true,
			},
		},
		{
			valid: false,
			id:    "invalid bound with dates",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemPredicateBound,
					Text: `"foo"@[2016-07-19T13:12:04.669618843-07:00,2015-07-19T13:12:04.669618843-07:00]`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemID,
					Text: "id",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar2",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAt,
					Text: "at",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar3",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{},
		},
		{
			valid: true,
			id:    "literal with alias",
			ces: []ConsumedElement{
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemLiteral,
					Text: `"1"^^type:int64`,
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemAs,
					Text: "as",
				}),
				NewConsumedSymbol("FOO"),
				NewConsumedToken(&lexer.Token{
					Type: lexer.ItemBinding,
					Text: "?bar",
				}),
				NewConsumedSymbol("FOO"),
			},
			want: &GraphClause{
				O:      l,
				OAlias: "?bar"},
		},
	})
}
