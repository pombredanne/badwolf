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

package planner

import (
	"fmt"
	"reflect"

	"github.com/google/badwolf/bql/semantic"
	"github.com/google/badwolf/bql/table"
	"github.com/google/badwolf/storage"
	"github.com/google/badwolf/triple"
	"github.com/google/badwolf/triple/node"
	"github.com/google/badwolf/triple/predicate"
)

// updateTimeBounds updates the time bounds use for the lookup based on the
// provided graph clause.
func updateTimeBounds(lo *storage.LookupOptions, cls *semantic.GraphClause) *storage.LookupOptions {
	nlo := &storage.LookupOptions{}
	if cls.PLowerBound != nil {
		if lo.LowerAnchor == nil || (lo.LowerAnchor != nil && cls.PLowerBound.After(*lo.LowerAnchor)) {
			lo.LowerAnchor = cls.PLowerBound
		}
	}
	if cls.PUpperBound != nil {
		if lo.UpperAnchor == nil || (lo.UpperAnchor != nil && cls.PUpperBound.Before(*lo.UpperAnchor)) {
			lo.UpperAnchor = cls.PUpperBound
		}
	}
	return nlo
}

// updateTimeBoundsForRow updates the time bounds use for the lookup based on
// the provided graph clause.
func updateTimeBoundsForRow(lo *storage.LookupOptions, cls *semantic.GraphClause, r table.Row) (*storage.LookupOptions, error) {
	lo = updateTimeBounds(lo, cls)
	if cls.PLowerBoundAlias != "" {
		v, ok := r[cls.PLowerBoundAlias]
		if ok && v.T == nil {
			return nil, fmt.Errorf("invalid time anchor value %v for bound %s", v, cls.PLowerBoundAlias)
		}
		if lo.LowerAnchor == nil || (lo.LowerAnchor != nil && v.T.After(*lo.LowerAnchor)) {
			lo.LowerAnchor = v.T
		}
	}
	if cls.PUpperBoundAlias != "" {
		v, ok := r[cls.PUpperBoundAlias]
		if ok && v.T == nil {
			return nil, fmt.Errorf("invalid time anchor value %v for bound %s", v, cls.PUpperBoundAlias)
		}
		if lo.UpperAnchor == nil || (lo.UpperAnchor != nil && v.T.After(*lo.UpperAnchor)) {
			lo.UpperAnchor = v.T
		}
	}
	nlo := updateTimeBounds(lo, cls)
	return nlo, nil
}

// simpleFetch returns a table containing the data specified by the graph
// clause by querying the provided stora. Will return an error if it had poblems
// retrieveing the data.
func simpleFetch(gs []storage.Graph, cls *semantic.GraphClause, lo *storage.LookupOptions) (*table.Table, error) {
	s, p, o := cls.S, cls.P, cls.O
	lo = updateTimeBounds(lo, cls)
	tbl, err := table.New(cls.Bindings())
	if err != nil {
		return nil, err
	}
	if s != nil && p != nil && o != nil {
		// Fully qualified triple.
		t, err := triple.New(s, p, o)
		if err != nil {
			return nil, err
		}
		for _, g := range gs {
			b, err := g.Exist(t)
			if err != nil {
				return nil, err
			}
			if b {
				ts := make(chan *triple.Triple, 1)
				ts <- t
				close(ts)
				if err := addTriples(ts, cls, tbl); err != nil {
					return nil, err
				}
			}
		}
		return tbl, nil
	}
	if s != nil && p != nil && o == nil {
		// SP request.
		for _, g := range gs {
			os, err := g.Objects(s, p, lo)
			if err != nil {
				return nil, err
			}
			var ros []*triple.Object
			for o := range os {
				ros = append(ros, o)
			}
			ts := make(chan *triple.Triple, len(ros))
			for _, o := range ros {
				t, err := triple.New(s, p, o)
				if err != nil {
					return nil, err
				}
				ts <- t
			}
			close(ts)
			if err := addTriples(ts, cls, tbl); err != nil {
				return nil, err
			}
		}
		return tbl, nil
	}
	if s != nil && p == nil && o != nil {
		// SO request.
		for _, g := range gs {
			ps, err := g.PredicatesForSubjectAndObject(s, o, lo)
			if err != nil {
				return nil, err
			}
			var rps []*predicate.Predicate
			for p := range ps {
				rps = append(rps, p)
			}
			ts := make(chan *triple.Triple, len(rps))
			for _, p := range rps {
				t, err := triple.New(s, p, o)
				if err != nil {
					return nil, err
				}
				ts <- t
			}
			close(ts)
			if err := addTriples(ts, cls, tbl); err != nil {
				return nil, err
			}
		}
		return tbl, nil
	}
	if s == nil && p != nil && o != nil {
		// PO request.
		for _, g := range gs {
			ss, err := g.Subjects(p, o, lo)
			if err != nil {
				return nil, err
			}
			var rss []*node.Node
			for s := range ss {
				rss = append(rss, s)
			}
			ts := make(chan *triple.Triple, len(rss))
			for _, s := range rss {
				t, err := triple.New(s, p, o)
				if err != nil {
					return nil, err
				}
				ts <- t
			}
			close(ts)
			if err := addTriples(ts, cls, tbl); err != nil {
				return nil, err
			}
		}
		return tbl, nil
	}
	if s != nil && p == nil && o == nil {
		// S request.
		for _, g := range gs {
			ts, err := g.TriplesForSubject(s, lo)
			if err != nil {
				return nil, err
			}
			if err := addTriples(ts, cls, tbl); err != nil {
				return nil, err
			}
		}
		return tbl, nil
	}
	if s == nil && p != nil && o == nil {
		// P request.
		for _, g := range gs {
			ts, err := g.TriplesForPredicate(p, lo)
			if err != nil {
				return nil, err
			}
			if err := addTriples(ts, cls, tbl); err != nil {
				return nil, err
			}
		}
		return tbl, nil
	}
	if s == nil && p == nil && o != nil {
		// O request.
		for _, g := range gs {
			ts, err := g.TriplesForObject(o, lo)
			if err != nil {
				return nil, err
			}
			if err := addTriples(ts, cls, tbl); err != nil {
				return nil, err
			}
		}
		return tbl, nil
	}
	if s == nil && p == nil && o == nil {
		// Full data request.
		for _, g := range gs {
			ts, err := g.Triples()
			if err != nil {
				return nil, err
			}
			if err := addTriples(ts, cls, tbl); err != nil {
				return nil, err
			}
		}
		return tbl, nil
	}

	return nil, fmt.Errorf("planner.simpleFetch could not recognize request in clause %v", cls)
}

// addTriples add all the retrieved triples from the graphs into the results
// table. The semantic graph clause is also passed to be able to identify what
// bindings to set.
func addTriples(ts storage.Triples, cls *semantic.GraphClause, tbl *table.Table) error {
	for t := range ts {
		r, err := tripleToRow(t, cls)
		if err != nil {
			return err
		}
		if cls.PID != "" {
			// The triples need to be filtered.
			if t.P().ID() != predicate.ID(cls.PID) {
				continue
			}
			if cls.PTemporal && t.P().Type() == predicate.Temporal {
				ta, err := t.P().TimeAnchor()
				if err != nil {
					return fmt.Errorf("failed to retrieve time anchor from time predicate in triple %s with error %v", t, err)
				}
				// Need to check teh bounds of the triple.
				if cls.PLowerBound != nil && cls.PLowerBound.After(*ta) {
					continue
				}
				if cls.PUpperBound != nil && cls.PUpperBound.Before(*ta) {
					continue
				}
			}
		}
		if cls.OID != "" {
			if p, err := t.O().Predicate(); err == nil {
				// The triples need to be filtered.
				if p.ID() != predicate.ID(cls.OID) {
					continue
				}
				if cls.OTemporal && p.Type() == predicate.Temporal {
					ta, err := p.TimeAnchor()
					if err != nil {
						return fmt.Errorf("failed to retrieve time anchor from time predicate in triple %s with error %v", t, err)
					}
					// Need to check teh bounds of the triple.
					if cls.OLowerBound != nil && cls.OLowerBound.After(*ta) {
						continue
					}
					if cls.OUpperBound != nil && cls.OUpperBound.Before(*ta) {
						continue
					}
				}
			}
		}
		if r != nil {
			tbl.AddRow(r)
		}
	}
	return nil
}

// objectToCell returns a cell containing the data boxed in the object.
func objectToCell(o *triple.Object) (*table.Cell, error) {
	c := &table.Cell{}
	if n, err := o.Node(); err == nil {
		c.N = n
		return c, nil
	}
	if p, err := o.Predicate(); err == nil {
		c.P = p
		return c, nil
	}
	if l, err := o.Literal(); err == nil {
		c.L = l
		return c, nil
	}
	return nil, fmt.Errorf("unknown object type in object %q", o)
}

// tripleToRow converts a triple into a row using the binndings specidfied
// on the graph clause.
func tripleToRow(t *triple.Triple, cls *semantic.GraphClause) (table.Row, error) {
	r, s, p, o := make(table.Row), t.S(), t.P(), t.O()

	// Enforce binding validity inside te clause.
	bnd := make(map[string]*table.Cell)
	validBinding := func(k string, v *table.Cell) bool {
		c, ok := bnd[k]
		bnd[k] = v
		if !ok {
			return true
		}
		if reflect.DeepEqual(c, v) {
			return true
		}
		return false
	}

	// Subject related bindings.
	if cls.SBinding != "" {
		c := &table.Cell{N: s}
		r[cls.SBinding] = c
		if !validBinding(cls.SBinding, c) {
			return nil, nil
		}
	}
	if cls.SAlias != "" {
		c := &table.Cell{N: s}
		r[cls.SAlias] = c
		if !validBinding(cls.SAlias, c) {
			return nil, nil
		}
	}
	if cls.STypeAlias != "" {
		c := &table.Cell{S: s.Type().String()}
		r[cls.STypeAlias] = c
		if !validBinding(cls.STypeAlias, c) {
			return nil, nil
		}
	}
	if cls.SIDAlias != "" {
		c := &table.Cell{S: s.ID().String()}
		r[cls.SIDAlias] = c
		if !validBinding(cls.SIDAlias, c) {
			return nil, nil
		}
	}

	// Predicate related bindings.
	if cls.PBinding != "" {
		c := &table.Cell{P: p}
		r[cls.PBinding] = c
		if !validBinding(cls.PBinding, c) {
			return nil, nil
		}
	}
	if cls.PAlias != "" {
		c := &table.Cell{P: p}
		r[cls.PAlias] = c
		if !validBinding(cls.PAlias, c) {
			return nil, nil
		}
	}
	if cls.PIDAlias != "" {
		c := &table.Cell{S: string(p.ID())}
		r[cls.PIDAlias] = c
		if !validBinding(cls.PIDAlias, c) {
			return nil, nil
		}
	}
	if cls.PAnchorBinding != "" {
		if p.Type() != predicate.Temporal {
			return nil, fmt.Errorf("cannot retrieve the time anchor value for non temporal predicate %q in binding %q", p, cls.PAnchorBinding)
		}
		t, err := p.TimeAnchor()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve the time anchor value for predicate %q in binding %q with error %v", p, cls.PAnchorBinding, err)
		}
		c := &table.Cell{T: t}
		r[cls.PAnchorBinding] = c
		if !validBinding(cls.PAnchorBinding, c) {
			return nil, nil
		}
	}

	if cls.PAnchorAlias != "" {
		if p.Type() != predicate.Temporal {
			return nil, fmt.Errorf("cannot retrieve the time anchor value for non temporal predicate %q in binding %q", p, cls.PAnchorAlias)
		}
		t, err := p.TimeAnchor()
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve the time anchor value for predicate %q in binding %q with error %v", p, cls.PAnchorAlias, err)
		}
		c := &table.Cell{T: t}
		r[cls.PAnchorAlias] = c
		if !validBinding(cls.PAnchorAlias, c) {
			return nil, nil
		}
	}

	// Object related bindings.
	if cls.OBinding != "" {
		// Extract the object type.
		c, err := objectToCell(o)
		if err != nil {
			return nil, err
		}
		r[cls.OBinding] = c
		if !validBinding(cls.OBinding, c) {
			return nil, nil
		}
	}
	if cls.OAlias != "" {
		// Extract the object type.
		c, err := objectToCell(o)
		if err != nil {
			return nil, err
		}
		r[cls.OAlias] = c
		if !validBinding(cls.OAlias, c) {
			return nil, nil
		}
	}
	if cls.OTypeAlias != "" {
		n, err := o.Node()
		if err != nil {
			return nil, err
		}
		c := &table.Cell{S: n.Type().String()}
		r[cls.OTypeAlias] = c
		if !validBinding(cls.OTypeAlias, c) {
			return nil, nil
		}
	}
	if cls.OIDAlias != "" {
		n, err := o.Node()
		if err == nil {
			r[cls.OIDAlias] = &table.Cell{S: n.ID().String()}
		} else {
			p, err := o.Predicate()
			if err != nil {
				return nil, err
			}
			c := &table.Cell{S: string(p.ID())}
			r[cls.OIDAlias] = c
			if !validBinding(cls.OIDAlias, c) {
				return nil, nil
			}
		}
	}
	if cls.OAnchorBinding != "" {
		p, err := o.Predicate()
		if err != nil {
			return nil, err
		}
		ts, err := p.TimeAnchor()
		if err != nil {
			return nil, err
		}
		c := &table.Cell{T: ts}
		r[cls.OAnchorBinding] = c
		if !validBinding(cls.OAnchorBinding, c) {
			return nil, nil
		}
	}
	if cls.OAnchorAlias != "" {
		p, err := o.Predicate()
		if err != nil {
			return nil, err
		}
		ts, err := p.TimeAnchor()
		if err != nil {
			return nil, err
		}
		c := &table.Cell{T: ts}
		r[cls.OAnchorAlias] = c
		if !validBinding(cls.OAnchorAlias, c) {
			return nil, nil
		}
	}

	return r, nil
}
