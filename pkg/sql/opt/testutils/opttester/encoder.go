// Copyright 2018 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package opttester

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroach/pkg/sql/opt"
	"github.com/cockroachdb/cockroach/pkg/sql/opt/cat"
	"github.com/cockroachdb/cockroach/pkg/sql/opt/memo"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
)

const (
	maxCols = 1600
)

// An encoder encodes a query plan into a format that can be understood
// by a neural network.
type encoder struct {
	// columns stores the encoding of each column in the query plan, indexed
	// by the optimizer's unique column index.
	columns map[opt.ColumnID]uint32

	md *opt.Metadata
}

// encode converts the given query plan into a comma-separated string
// of integers by walking the query tree in a post-order traversal, and
// printing the integers that represent the operator and arguments
// at each node.
func (e *encoder) encode(expr opt.Expr) string {
	e.columns = make(map[opt.ColumnID]uint32)
	e.md = expr.(memo.RelExpr).Memo().Metadata()
	out := e.encodeImpl(expr, nil)
	return strings.Join(out, ",")
}

func (e *encoder) encodeImpl(expr, parent opt.Expr) []string {
	out := make([]string, 0, expr.ChildCount()+1)
	for i := 0; i < expr.ChildCount(); i++ {
		out = append(out, e.encodeImpl(expr.Child(i), expr)...)
	}

	switch expr.Op() {
	case opt.ConstOp:
		colID := columnFromComparison(expr, parent)
		frac := constToFractionOfDomain(expr.Private().(tree.Datum), colID)
		out = append(out, fmt.Sprintf("%f", float64(e.columns[colID])+frac))

	case opt.ScanOp:
		scan := expr.(*memo.ScanExpr)
		tableID := scan.Table
		table := e.md.Table(tableID)

		// Save the column information.
		for i, n := 0, table.ColumnCount(); i < n; i++ {
			colID := tableID.ColumnID(i)
			e.columns[colID] = uint32(table.ID()*maxCols) + uint32(i+1)
		}

		// Add a fraction for the index.
		frac := float64(scan.Index) / float64(table.IndexCount())
		out = append(out, fmt.Sprintf("%f", float64(table.ID()*maxCols)+frac))

	case opt.VariableOp:
		variable := expr.(*memo.VariableExpr)
		colID := variable.Col
		if _, ok := e.columns[colID]; !ok {
			tableID := e.md.ColumnMeta(colID).Table
			if tableID == 0 {
				// Column was synthesized.
				e.columns[colID] = uint32(opt.NumOperators) + uint32(colID)
			} else {
				table := e.md.Table(tableID)
				ord := cat.FindTableColumnByName(table, tree.Name(e.md.ColumnMeta(colID).Alias))
				e.columns[colID] = uint32(table.ID()*maxCols) + uint32(ord+1)
			}
		}
		out = append(out, fmt.Sprintf("%d", e.columns[colID]))
	}

	out = append(out, fmt.Sprintf("%d", expr.Op()))
	return out
}

func columnFromComparison(expr, parent opt.Expr) opt.ColumnID {
	var colID opt.ColumnID
	if parent == nil || !opt.IsComparisonOp(parent) {
		return colID
	}

	for i, n := 0, parent.ChildCount(); i < n; i++ {
		child := parent.Child(i)
		if expr == child {
			continue
		}
		if child.Op() == opt.VariableOp {
			variableColID := child.(*memo.VariableExpr).Col
			if colID == 0 || colID == variableColID {
				colID = variableColID
			} else {
				// There is more than one variable, so reset the column ID.
				colID = 0
				break
			}
		}
	}
	return colID
}

func constToFractionOfDomain(datum tree.Datum, id opt.ColumnID) float64 {
	// HACK - for now just return 0.5.
	return 0.5
}

type encodedPlan struct {
	cost    memo.Cost
	encoded string
}

// encodedPlans is a struct used for sorting encoded plans by increasing cost.
// It implements sort.Interface.
type encodedPlans struct {
	plans []encodedPlan
}

// Len is part of sort.Interface.
func (ep encodedPlans) Len() int {
	return len(ep.plans)
}

// Less is part of sort.Interface.
func (ep encodedPlans) Less(i, j int) bool {
	if ep.plans[i].cost == ep.plans[j].cost {
		return ep.plans[i].encoded < ep.plans[j].encoded
	}
	return ep.plans[i].cost < ep.plans[j].cost
}

// Swap is part of sort.Interface.
func (ep encodedPlans) Swap(i, j int) {
	ep.plans[i], ep.plans[j] = ep.plans[j], ep.plans[i]
}

func (ep encodedPlans) String() string {
	var buf bytes.Buffer
	for i := range ep.plans {
		// Remove duplicates.
		if i > 0 && ep.plans[i] != ep.plans[i-1] {
			//			fmt.Fprintf(&buf, "%s\n", ep.plans[i].encoded)
			fmt.Fprintf(&buf, "%f %s\n", ep.plans[i].cost, ep.plans[i].encoded)
		}
	}
	return buf.String()
}
