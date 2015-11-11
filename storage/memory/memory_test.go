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

package memory

import (
	"testing"
	"time"

	"github.com/google/badwolf/storage"
	"github.com/google/badwolf/triple"
	"github.com/google/badwolf/triple/literal"
	"github.com/google/badwolf/triple/predicate"
)

func TestMemoryStore(t *testing.T) {
	s := NewStore()
	// Create a new graph.
	if _, err := s.NewGraph("test"); err != nil {
		t.Errorf("memoryStore.NewGraph: should never fail to crate a graph; %s", err)
	}
	// Get an existing graph.
	if _, err := s.Graph("test"); err != nil {
		t.Errorf("memoryStore.Graph: should never fail to get an existing graph; %s", err)
	}
	// Delete an existing graph.
	if err := s.DeleteGraph("test"); err != nil {
		t.Errorf("memoryStore.DeleteGraph: should never fail to delete an existing graph; %s", err)
	}
	// Get a non existing graph.
	if _, err := s.Graph("test"); err == nil {
		t.Errorf("memoryStore.Graph: should never succeed to get a non existing graph; %s", err)
	}
	// Delete an existing graph.
	if err := s.DeleteGraph("test"); err == nil {
		t.Errorf("memoryStore.DeleteGraph: should never succed to delete a non existing graph; %s", err)
	}
}

func TestDefaultLookupChecker(t *testing.T) {
	dlu := storage.DefaultLookup
	c := newChecker(dlu)
	ip, err := predicate.NewImmutable("foo")
	if err != nil {
		t.Fatal(err)
	}
	tp, err := predicate.NewTemporal("bar", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !c.CheckAndUpdate(ip) {
		t.Errorf("Immutable predicates should always validate with default lookup %v", dlu)
	}
	if !c.CheckAndUpdate(tp) {
		t.Errorf("Temporal predicates should always validate with default lookup %v", dlu)
	}
}

func TestLimitedItemsLookupChecker(t *testing.T) {
	blu := &storage.LookupOptions{MaxElements: 1}
	c := newChecker(blu)
	ip, err := predicate.NewImmutable("foo")
	if err != nil {
		t.Fatal(err)
	}
	if !c.CheckAndUpdate(ip) {
		t.Errorf("The first predicate should always succeeed on bounded lookup %v", blu)
	}
	for i := 0; i < 10; i++ {
		if c.CheckAndUpdate(ip) {
			t.Errorf("Bounded lookup %v should never succeed after being exahausted", blu)
		}
	}
}

func TestTemporalBoundedLookupChecker(t *testing.T) {
	lpa, err := predicate.Parse("\"foo\"@[2013-07-19T13:12:04.669618843-07:00]")
	if err != nil {
		t.Fatalf("Failed to parse fixture predicate with error %v", err)
	}
	mpa, err := predicate.Parse("\"foo\"@[2014-07-19T13:12:04.669618843-07:00]")
	if err != nil {
		t.Fatalf("Failed to parse fixture predicate with error %v", err)
	}
	upa, err := predicate.Parse("\"foo\"@[2015-07-19T13:12:04.669618843-07:00]")
	if err != nil {
		t.Fatalf("Failed to parse fixture predicate with error %v", err)
	}
	// Check lower bound
	lb, _ := lpa.TimeAnchor()
	blu := &storage.LookupOptions{LowerAnchor: lb}
	clu := newChecker(blu)
	if !clu.CheckAndUpdate(mpa) {
		t.Errorf("Failed to reject invalid predicate %v by checker %v", mpa, clu)
	}
	lb, _ = mpa.TimeAnchor()
	blu = &storage.LookupOptions{LowerAnchor: lb}
	clu = newChecker(blu)
	if clu.CheckAndUpdate(lpa) {
		t.Errorf("Failed to reject invalid predicate %v by checker %v", mpa, clu)
	}
	// Check upper bound.
	ub, _ := upa.TimeAnchor()
	buu := &storage.LookupOptions{UpperAnchor: ub}
	cuu := newChecker(buu)
	if !cuu.CheckAndUpdate(mpa) {
		t.Errorf("Failed to reject invalid predicate %v by checker %v", mpa, cuu)
	}
	ub, _ = mpa.TimeAnchor()
	buu = &storage.LookupOptions{UpperAnchor: ub}
	cuu = newChecker(buu)
	if cuu.CheckAndUpdate(upa) {
		t.Errorf("Failed to reject invalid predicate %v by checker %v", mpa, cuu)
	}
}

func getTestTriples(t *testing.T) []*triple.Triple {
	ts := []*triple.Triple{}
	ss := []string{
		"/u<john>\t\"knows\"@[]\t/u<mary>",
		"/u<john>\t\"knows\"@[]\t/u<peter>",
		"/u<john>\t\"knows\"@[]\t/u<alice>",
		"/u<mary>\t\"knows\"@[]\t/u<andrew>",
		"/u<mary>\t\"knows\"@[]\t/u<kim>",
		"/u<mary>\t\"knows\"@[]\t/u<alice>",
	}
	for _, s := range ss {
		trpl, err := triple.ParseTriple(s, literal.DefaultBuilder())
		if err != nil {
			t.Errorf("triple.Parse failed to parse valid triple %s with error %v", s, err)
			continue
		}
		ts = append(ts, trpl)
	}
	return ts
}

func TestAddRemoveTriples(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	if err := g.RemoveTriples(ts); err != nil {
		t.Errorf("g.RemoveTriples(_) failed failed to remove test triples with error %v", err)
	}
}

func TestObjects(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	os, err := g.Objects(ts[0].S(), ts[0].P(), storage.DefaultLookup)
	if err != nil {
		t.Errorf("g.Objects(%s, %s) failed with error %v", ts[0].S(), ts[0].P(), err)
	}
	cnt := 0
	for o := range os {
		cnt++
		n, _ := o.Node()
		ty, id := n.Type().String(), n.ID().String()
		if ty != "/u" || (id != "mary" && id != "peter" && id != "alice") {
			t.Errorf("g.Objects(%s, %s) failed to return a valid object; returned %s instead", ts[0].S(), ts[0].P(), n)
		}
	}
	if cnt != 3 {
		t.Errorf("g.Objects(%s, %s) failed to retrieve 3 objects, got %d instead", ts[0].S(), ts[0].P(), cnt)
	}
}

func TestSubjects(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	ss, err := g.Subjects(ts[0].P(), ts[0].O(), storage.DefaultLookup)
	if err != nil {
		t.Errorf("g.Subjects(%s, %s) failed with error %v", ts[0].P(), ts[0].O(), err)
	}
	cnt := 0
	for s := range ss {
		cnt++
		ty, id := s.Type().String(), s.ID().String()
		if ty != "/u" || id != "john" {
			t.Errorf("g.Subjects(%s, %s) failed to return a valid subject; returned %s instead", ts[0].P(), ts[0].O(), s)
		}
	}
	if cnt != 1 {
		t.Errorf("g.Objects(%s, %s) failed to retrieve 1 objects, got %d instead", ts[0].S(), ts[0].P(), cnt)
	}
}

func TestPredicatesForSubjectAndObject(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	ps, err := g.PredicatesForSubjectAndObject(ts[0].S(), ts[0].O(), storage.DefaultLookup)
	if err != nil {
		t.Errorf("g.PredicatesForSubjectAndObject(%s, %s) failed with error %v", ts[0].S(), ts[0].O(), err)
	}
	cnt := 0
	for p := range ps {
		cnt++
		if p.Type() != predicate.Immutable || p.ID() != "knows" {
			t.Errorf("g.PredicatesForSubjectAndObject(%s, %s) failed to return a valid subject; returned %s instead", ts[0].S(), ts[0].O(), p)
		}
	}
	if cnt != 1 {
		t.Errorf("g.PredicatesForSubjectAndObject(%s, %s) failed to retrieve 1 predicate, got %d instead", ts[0].S(), ts[0].O(), cnt)
	}
}

func TestPredicatesForSubject(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	ps, err := g.PredicatesForSubject(ts[0].S(), storage.DefaultLookup)
	if err != nil {
		t.Errorf("g.PredicatesForSubject(%s) failed with error %v", ts[0].S(), err)
	}
	cnt := 0
	for p := range ps {
		cnt++
		if p.Type() != predicate.Immutable || p.ID() != "knows" {
			t.Errorf("g.PredicatesForSubject(%s) failed to return a valid predicate; returned %s instead", ts[0].S(), p)
		}
	}
	if cnt != 3 {
		t.Errorf("g.PredicatesForSubjectAndObject(%s) failed to retrieve 3 predicates, got %d instead", ts[0].S(), cnt)
	}
}

func TestPredicatesForObject(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	ps, err := g.PredicatesForObject(ts[0].O(), storage.DefaultLookup)
	if err != nil {
		t.Errorf("g.PredicatesForObject(%s) failed with error %v", ts[0].O(), err)
	}
	cnt := 0
	for p := range ps {
		cnt++
		if p.Type() != predicate.Immutable || p.ID() != "knows" {
			t.Errorf("g.PredicatesForObject(%s) failed to return a valid predicate; returned %s instead", ts[0].O(), p)
		}
	}
	if cnt != 1 {
		t.Errorf("g.PredicatesForObject(%s) failed to retrieve 1 predicate, got %d instead", ts[0].O(), cnt)
	}
}

func TestTriplesForSubject(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	trpls, err := g.TriplesForSubject(ts[0].S(), storage.DefaultLookup)
	if err != nil {
		t.Errorf("g.TriplesForSubject(%s) failed with error %v", ts[0].S(), err)
	}
	cnt := 0
	for _ = range trpls {
		cnt++
	}
	if cnt != 3 {
		t.Errorf("g.triplesForSubject(%s) failed to retrieve 3 predicates, got %d instead", ts[0].S(), cnt)
	}
}

func TestTriplesForPredicate(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	trpls, err := g.TriplesForPredicate(ts[0].P(), storage.DefaultLookup)
	if err != nil {
		t.Errorf("g.TriplesForPredicate(%s) failed with error %v", ts[0].S(), err)
	}
	cnt := 0
	for _ = range trpls {
		cnt++
	}
	if cnt != 6 {
		t.Errorf("g.triplesForPredicate(%s) failed to retrieve 3 predicates, got %d instead", ts[0].P(), cnt)
	}
}

func TestTriplesForObject(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	trpls, err := g.TriplesForObject(ts[0].O(), storage.DefaultLookup)
	if err != nil {
		t.Errorf("g.TriplesForObject(%s) failed with error %v", ts[0].O(), err)
	}
	cnt := 0
	for _ = range trpls {
		cnt++
	}
	if cnt != 1 {
		t.Errorf("g.TriplesForObject(%s) failed to retrieve 1 predicates, got %d instead", ts[0].O(), cnt)
	}
}

func TestTriplesForSubjectAndPredicate(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	trpls, err := g.TriplesForSubjectAndPredicate(ts[0].S(), ts[0].P(), storage.DefaultLookup)
	if err != nil {
		t.Errorf("g.TriplesForSubjectAndPredicate(%s, %s) failed with error %v", ts[0].S(), ts[0].P(), err)
	}
	cnt := 0
	for _ = range trpls {
		cnt++
	}
	if cnt != 3 {
		t.Errorf("g.TriplesForSubjectAndPredicate(%s, %s) failed to retrieve 3 predicates, got %d instead", ts[0].S(), ts[0].P(), cnt)
	}
}

func TestTriplesForPredicateAndObject(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	trpls, err := g.TriplesForPredicateAndObject(ts[0].P(), ts[0].O(), storage.DefaultLookup)
	if err != nil {
		t.Errorf("g.TriplesForPredicateAndObject(%s, %s) failed with error %v", ts[0].P(), ts[0].O(), err)
	}
	cnt := 0
	for _ = range trpls {
		cnt++
	}
	if cnt != 1 {
		t.Errorf("g.TriplesForPredicateAndObject(%s, %s) failed to retrieve 1 predicates, got %d instead", ts[0].P(), ts[0].O(), cnt)
	}
}

func TestExists(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	for _, trpl := range ts {
		b, err := g.Exist(trpl)
		if err != nil {
			t.Errorf("g.Exist should have not failed for triple %s with error %s", trpl, err)
		}
		if !b {
			t.Errorf("g.Exist should have not failed for triple %s", trpl)
		}
	}
}

func TestTriples(t *testing.T) {
	ts := getTestTriples(t)
	g, _ := NewStore().NewGraph("test")
	if err := g.AddTriples(ts); err != nil {
		t.Errorf("g.AddTriples(_) failed failed to add test triples with error %v", err)
	}
	trpls, err := g.Triples()
	if err != nil {
		t.Fatal(err)
	}
	cnt := 0
	for _ = range trpls {
		cnt++
	}
	if cnt != 6 {
		t.Errorf("g.TriplesForPredicateAndObject(%s, %s) failed to retrieve 1 predicates, got %d instead", ts[0].P(), ts[0].O(), cnt)
	}
}
