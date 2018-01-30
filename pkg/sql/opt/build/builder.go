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

package build

import (
	"context"
	"fmt"
	"strings"

	"github.com/cockroachdb/cockroach/pkg/sql/opt/xform"
	"github.com/cockroachdb/cockroach/pkg/sql/optbase"
	_ "github.com/cockroachdb/cockroach/pkg/sql/sem/builtins"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/types"
)

type unaryFactoryFunc func(f *xform.Factory, input xform.GroupID) xform.GroupID
type binaryFactoryFunc func(f *xform.Factory, left, right xform.GroupID) xform.GroupID

var comparisonOpMap = [...]binaryFactoryFunc{
	tree.EQ:                (*xform.Factory).ConstructEq,
	tree.LT:                (*xform.Factory).ConstructLt,
	tree.GT:                (*xform.Factory).ConstructGt,
	tree.LE:                (*xform.Factory).ConstructLe,
	tree.GE:                (*xform.Factory).ConstructGe,
	tree.NE:                (*xform.Factory).ConstructNe,
	tree.In:                (*xform.Factory).ConstructIn,
	tree.NotIn:             (*xform.Factory).ConstructNotIn,
	tree.Like:              (*xform.Factory).ConstructLike,
	tree.NotLike:           (*xform.Factory).ConstructNotLike,
	tree.ILike:             (*xform.Factory).ConstructILike,
	tree.NotILike:          (*xform.Factory).ConstructNotILike,
	tree.SimilarTo:         (*xform.Factory).ConstructSimilarTo,
	tree.NotSimilarTo:      (*xform.Factory).ConstructNotSimilarTo,
	tree.RegMatch:          (*xform.Factory).ConstructRegMatch,
	tree.NotRegMatch:       (*xform.Factory).ConstructNotRegMatch,
	tree.RegIMatch:         (*xform.Factory).ConstructRegIMatch,
	tree.NotRegIMatch:      (*xform.Factory).ConstructNotRegIMatch,
	tree.IsDistinctFrom:    (*xform.Factory).ConstructIsDistinctFrom,
	tree.IsNotDistinctFrom: (*xform.Factory).ConstructIsNotDistinctFrom,
	tree.Any:               (*xform.Factory).ConstructAny,
	tree.Some:              (*xform.Factory).ConstructSome,
	tree.All:               (*xform.Factory).ConstructAll,
}

var binaryOpMap = [...]binaryFactoryFunc{
	tree.Bitand:   (*xform.Factory).ConstructBitand,
	tree.Bitor:    (*xform.Factory).ConstructBitor,
	tree.Bitxor:   (*xform.Factory).ConstructBitxor,
	tree.Plus:     (*xform.Factory).ConstructPlus,
	tree.Minus:    (*xform.Factory).ConstructMinus,
	tree.Mult:     (*xform.Factory).ConstructMult,
	tree.Div:      (*xform.Factory).ConstructDiv,
	tree.FloorDiv: (*xform.Factory).ConstructFloorDiv,
	tree.Mod:      (*xform.Factory).ConstructMod,
	tree.Pow:      (*xform.Factory).ConstructPow,
	tree.Concat:   (*xform.Factory).ConstructConcat,
	tree.LShift:   (*xform.Factory).ConstructLShift,
	tree.RShift:   (*xform.Factory).ConstructRShift,
}

var unaryOpMap = [...]unaryFactoryFunc{
	tree.UnaryPlus:       (*xform.Factory).ConstructUnaryPlus,
	tree.UnaryMinus:      (*xform.Factory).ConstructUnaryMinus,
	tree.UnaryComplement: (*xform.Factory).ConstructUnaryComplement,
}

type Builder struct {
	tree.IndexedVarContainer
	factory *xform.Factory
	stmt    tree.Statement
	semaCtx tree.SemaContext
	ctx     context.Context

	// Skip index 0 in order to reserve it to indicate the "unknown" column.
	colMap []columnProps
}

func NewBuilder(ctx context.Context, factory *xform.Factory, stmt tree.Statement) *Builder {
	b := &Builder{factory: factory, stmt: stmt, colMap: make([]columnProps, 1)}

	ivarHelper := tree.MakeIndexedVarHelper(b, 0)
	b.semaCtx.IVarHelper = &ivarHelper
	b.semaCtx.Placeholders = tree.MakePlaceholderInfo()
	b.ctx = ctx

	return b
}

func (b *Builder) Build() (root xform.GroupID, required *xform.PhysicalProps, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	out, _ := b.buildStmt(b.stmt, &scope{builder: b})
	root = out

	// TODO(rytaft): Add physical properties that are required of the root planner
	// group.
	required = &xform.PhysicalProps{}
	return
}

func (b *Builder) buildStmt(stmt tree.Statement, inScope *scope) (out xform.GroupID, outScope *scope) {
	// NB: The case statements are sorted lexicographically.
	switch stmt := stmt.(type) {
	case *tree.ParenSelect:
		return b.buildSelect(stmt.Select, inScope)

	case *tree.Select:
		return b.buildSelect(stmt, inScope)

		// *tree.AlterSequence
		// *tree.AlterTable
		// *tree.AlterUserSetPassword
		// *tree.Backup
		// *tree.BeginTransaction
		// *tree.CancelJob
		// *tree.CancelQuery
		// *tree.CommitTransaction
		// *tree.CopyFrom
		// *tree.CreateDatabase
		// *tree.CreateIndex
		// *tree.CreateSequence
		// *tree.CreateStats
		// *tree.CreateTable
		// *tree.CreateUser
		// *tree.CreateView
		// *tree.Deallocate
		// *tree.Delete
		// *tree.Discard
		// *tree.DropDatabase
		// *tree.DropIndex
		// *tree.DropSequence
		// *tree.DropTable
		// *tree.DropUser
		// *tree.DropView
		// *tree.Execute
		// *tree.Explain
		// *tree.Grant
		// *tree.Import
		// *tree.Insert
		// *tree.PauseJob
		// *tree.Prepare
		// *tree.ReleaseSavepoint
		// *tree.RenameColumn
		// *tree.RenameDatabase
		// *tree.RenameIndex
		// *tree.RenameTable
		// *tree.Restore
		// *tree.ResumeJob
		// *tree.Revoke
		// *tree.RollbackToSavepoint
		// *tree.RollbackTransaction
		// *tree.Savepoint
		// *tree.Scatter
		// *tree.Scrub
		// *tree.SelectClause
		// *tree.SetClusterSetting
		// *tree.SetSessionCharacteristics
		// *tree.SetTransaction
		// *tree.SetVar
		// *tree.SetZoneConfig
		// *tree.ShowBackup
		// *tree.ShowClusterSetting
		// *tree.ShowColumns
		// *tree.ShowConstraints
		// *tree.ShowCreateTable
		// *tree.ShowCreateView
		// *tree.ShowDatabases
		// *tree.ShowFingerprints
		// *tree.ShowGrants
		// *tree.ShowHistogram
		// *tree.ShowIndex
		// *tree.ShowJobs
		// *tree.ShowQueries
		// *tree.ShowRanges
		// *tree.ShowSessions
		// *tree.ShowTableStats
		// *tree.ShowTables
		// *tree.ShowTrace
		// *tree.ShowTransactionStatus
		// *tree.ShowUsers
		// *tree.ShowVar
		// *tree.ShowZoneConfig
		// *tree.Split
		// *tree.TestingRelocate
		// *tree.Truncate
		// *tree.UnionClause
		// *tree.Update
		// *tree.ValuesClause
		// tree.SelectStatement

	default:
		fatalf("unexpected statement: %T", stmt)
		return 0, nil
	}
}

func (b *Builder) buildTable(texpr tree.TableExpr, inScope *scope) (out xform.GroupID, outScope *scope) {
	// NB: The case statements are sorted lexicographically.
	switch source := texpr.(type) {
	case *tree.AliasedTableExpr:
		out, outScope = b.buildTable(source.Expr, inScope)

		// Overwrite output properties with any alias information.
		if source.As.Alias != "" {
			if n := len(source.As.Cols); n > 0 && n != len(outScope.cols) {
				fatalf("rename specified %d columns, but table contains %d", n, len(outScope.cols))
			}

			for i := range outScope.cols {
				outScope.cols[i].table = optbase.TableName(source.As.Alias)
				if i < len(source.As.Cols) {
					outScope.cols[i].name = optbase.ColumnName(source.As.Cols[i])
				}
			}
		}

		return

	case *tree.FuncExpr:
		unimplemented("%T", texpr)

	case *tree.JoinTableExpr:
		switch cond := source.Cond.(type) {
		case *tree.OnJoinCond:
			return b.buildOnJoin(source, cond.Expr, inScope)

		case tree.NaturalJoinCond:
			return b.buildNaturalJoin(source, inScope)

		case *tree.UsingJoinCond:
			return b.buildUsingJoin(source, cond.Cols, inScope)

		default:
			unimplemented("%T", source.Cond)
			return 0, nil
		}

	case *tree.NormalizableTableName:
		tn, err := source.Normalize()
		if err != nil {
			fatalf("%s", err)
		}
		tbl, err := b.factory.Metadata().Catalog().FindTable(b.ctx, tn)
		if err != nil {
			fatalf("%s", err)
		}

		return b.buildScan(tbl, inScope)

	case *tree.ParenTableExpr:
		return b.buildTable(source.Expr, inScope)

	case *tree.StatementSource:
		unimplemented("%T", texpr)

	case *tree.Subquery:
		return b.buildStmt(source.Select, inScope)

	case *tree.TableRef:
		unimplemented("%T", texpr)

	default:
		fatalf("unexpected table expr: %T", texpr)
	}

	return 0, nil
}

func (b *Builder) buildScan(tbl optbase.Table, inScope *scope) (out xform.GroupID, outScope *scope) {
	tblIndex := b.factory.Metadata().AddTable(tbl)

	outScope = inScope.push()
	for i := 0; i < tbl.NumColumns(); i++ {
		col := tbl.Column(i)
		colIndex := b.factory.Metadata().TableColumn(tblIndex, i)
		colProps := columnProps{
			index: colIndex,
			name:  col.ColName(),
			table: tbl.TabName(),
			typ:   col.DatumType(),
		}

		b.colMap = append(b.colMap, colProps)
		outScope.cols = append(outScope.cols, colProps)
	}

	return b.factory.ConstructScan(b.factory.InternPrivate(tblIndex)), outScope
}

func (b *Builder) buildOnJoin(
	join *tree.JoinTableExpr,
	on tree.Expr,
	inScope *scope,
) (out xform.GroupID, outScope *scope) {
	left, leftScope := b.buildTable(join.Left, inScope)
	right, rightScope := b.buildTable(join.Right, inScope)

	// Append columns added by the children, as they are visible to the filter.
	outScope = inScope.push()
	outScope.appendColumns(leftScope)
	outScope.appendColumns(rightScope)

	filter := b.buildScalar(outScope.resolveType(on, types.Bool), outScope)

	return b.constructJoin(join.Join, left, right, filter), outScope
}

func (b *Builder) buildNaturalJoin(join *tree.JoinTableExpr, inScope *scope) (out xform.GroupID, outScope *scope) {
	left, leftScope := b.buildTable(join.Left, inScope)
	right, rightScope := b.buildTable(join.Right, inScope)

	var common tree.NameList
	for _, leftCol := range leftScope.cols {
		for _, rightCol := range rightScope.cols {
			if leftCol.name == rightCol.name && !leftCol.hidden && !rightCol.hidden {
				common = append(common, tree.Name(leftCol.name))
				break
			}
		}
	}

	filter, outScope := b.buildUsingJoinParts(leftScope.cols, rightScope.cols, common, inScope)

	return b.constructJoin(join.Join, left, right, filter), outScope
}

func (b *Builder) buildUsingJoin(
	join *tree.JoinTableExpr,
	names tree.NameList,
	inScope *scope,
) (out xform.GroupID, outScope *scope) {
	left, leftScope := b.buildTable(join.Left, inScope)
	right, rightScope := b.buildTable(join.Right, inScope)

	filter, outScope := b.buildUsingJoinParts(leftScope.cols, rightScope.cols, names, inScope)

	return b.constructJoin(join.Join, left, right, filter), outScope
}

// buildUsingJoinParts builds a set of memo groups that represent the join
// conditions for a USING join or natural join. It finds the columns in the
// left and right relations that match the columns provided in the names
// parameter, and creates equality predicate(s) with those columns. It also
// ensures that only one of the equal columns appear in the output by
// marking the other column as "hidden".
//
// See Builder.buildStmt above for a description of the remaining input and
// return values.
func (b *Builder) buildUsingJoinParts(
	leftCols []columnProps,
	rightCols []columnProps,
	names tree.NameList,
	inScope *scope,
) (out xform.GroupID, outScope *scope) {
	joined := make(map[optbase.ColumnName]*columnProps, len(names))
	conditions := make([]xform.GroupID, 0, len(names))
	outScope = inScope.push()
	for _, name := range names {
		name := optbase.ColumnName(name)

		// For every adjacent pair of tables, add an equality predicate.
		leftCol := findColByName(leftCols, name)
		if leftCol == nil {
			fatalf("unable to resolve name %s", name)
		}

		rightCol := findColByName(rightCols, name)
		if rightCol == nil {
			fatalf("unable to resolve name %s", name)
		}

		outScope.cols = append(outScope.cols, *leftCol)
		joined[name] = &outScope.cols[len(outScope.cols)-1]

		leftVar := b.factory.ConstructVariable(b.factory.InternPrivate(leftCol.index))
		rightVar := b.factory.ConstructVariable(b.factory.InternPrivate(rightCol.index))
		eq := b.factory.ConstructEq(leftVar, rightVar)

		conditions = append(conditions, eq)
	}

	for i, col := range leftCols {
		foundCol, ok := joined[col.name]
		if ok {
			// Hide other columns with the same name.
			if &leftCols[i] == foundCol {
				continue
			}
			col.hidden = true
		}
		outScope.cols = append(outScope.cols, col)
	}

	for _, col := range rightCols {
		_, col.hidden = joined[col.name]
		outScope.cols = append(outScope.cols, col)
	}

	return b.factory.ConstructFilters(b.factory.StoreList(conditions)), outScope
}

// buildScalarProjection takes the output of buildScalar and adds it as new
// columns to the output scope.
func (b *Builder) buildScalarProjection(texpr tree.TypedExpr, inScope, outScope *scope) xform.GroupID {
	// NB: The case statements are sorted lexicographically.
	switch t := texpr.(type) {
	case *columnProps:
		colIndex := xform.ColumnIndex(t.index)
		out := b.factory.ConstructVariable(b.factory.InternPrivate(colIndex))
		outScope.cols = append(outScope.cols, b.colMap[colIndex])
		return out

	case *tree.FuncExpr:
		out, col := b.buildFunction(t, inScope)
		if col != nil {
			// Function was mapped to a column reference, such as in the case
			// of an aggregate.
			outScope.cols = append(outScope.cols, b.colMap[col.index])
		}
		return out

	case *tree.ParenExpr:
		return b.buildScalarProjection(t.TypedInnerExpr(), inScope, outScope)

	case *tree.UnresolvedName:
		vn, err := t.NormalizeVarName()
		if err != nil {
			panic(err)
		}
		return b.buildScalarProjection(vn, inScope, outScope)

	default:
		out := b.buildScalar(texpr, inScope)
		b.synthesizeColumn(outScope, "", texpr.ResolvedType())
		return out
	}
}

// buildScalar converts a tree.TypedExpr to an Expr tree.
func (b *Builder) buildScalar(scalar tree.TypedExpr, inScope *scope) xform.GroupID {
	switch t := scalar.(type) {
	case *columnProps:
		return b.factory.ConstructVariable(b.factory.InternPrivate(t.index))

	case *tree.AllColumnsSelector:
		fatalf("unexpected unresolved scalar expr: %T", scalar)

	case *tree.AndExpr:
		return b.factory.ConstructAnd(b.buildScalar(t.TypedLeft(), inScope), b.buildScalar(t.TypedRight(), inScope))

	case *tree.Array:
		unimplemented("%T", scalar)

	case *tree.ArrayFlatten:
		unimplemented("%T", scalar)

	case *tree.BinaryExpr:
		return binaryOpMap[t.Operator](b.factory,
			b.buildScalar(t.TypedLeft(), inScope),
			b.buildScalar(t.TypedRight(), inScope))

	case *tree.CaseExpr:
		unimplemented("%T", scalar)

	case *tree.CastExpr:
		unimplemented("%T", scalar)

	case *tree.CoalesceExpr:
		unimplemented("%T", scalar)

	case *tree.CollateExpr:
		unimplemented("%T", scalar)

	case *tree.ColumnItem:
		fatalf("unexpected unresolved scalar expr: %T", scalar)

	case *tree.ComparisonExpr:
		// TODO(peter): handle t.SubOperator.
		return comparisonOpMap[t.Operator](b.factory,
			b.buildScalar(t.TypedLeft(), inScope),
			b.buildScalar(t.TypedRight(), inScope))

	case tree.DefaultVal:
		unimplemented("%T", scalar)

	case *tree.FuncExpr:
		out, _ := b.buildFunction(t, inScope)
		return out

	case *tree.IfExpr:
		unimplemented("%T", scalar)

	case *tree.IndirectionExpr:
		unimplemented("%T", scalar)

	case *tree.IsOfTypeExpr:
		unimplemented("%T", scalar)

	case *tree.NotExpr:
		return b.factory.ConstructNot(b.buildScalar(t.TypedInnerExpr(), inScope))

	case *tree.NullIfExpr:
		unimplemented("%T", scalar)

	case *tree.OrExpr:
		return b.factory.ConstructOr(b.buildScalar(t.TypedLeft(), inScope), b.buildScalar(t.TypedRight(), inScope))

	case *tree.ParenExpr:
		return b.buildScalar(t.TypedInnerExpr(), inScope)

	case *tree.Placeholder:
		return b.factory.ConstructPlaceholder(b.factory.InternPrivate(t))

	case *tree.RangeCond:
		unimplemented("%T", scalar)

	case *subquery:
		if t.exists {
			return b.factory.ConstructExists(t.out)
		}

		v := b.factory.ConstructVariable(b.factory.InternPrivate(t.cols[0].index))
		return b.factory.ConstructSubquery(t.out, v)

	case *tree.Tuple:
		list := make([]xform.GroupID, len(t.Exprs))
		for i := range t.Exprs {
			list[i] = b.buildScalar(t.Exprs[i].(tree.TypedExpr), inScope)
		}
		return b.factory.ConstructTuple(b.factory.StoreList(list))

	case *tree.UnaryExpr:
		return unaryOpMap[t.Operator](b.factory, b.buildScalar(t.TypedInnerExpr(), inScope))

	case tree.UnqualifiedStar:
		fatalf("unexpected unresolved scalar expr: %T", scalar)

	case *tree.UnresolvedName:
		fatalf("unexpected unresolved scalar expr: %T", scalar)

		// NB: this is the exception to the sorting of the case statements. The
		// tree.Datum case needs to occur after *tree.Placeholder which implements
		// Datum.
	case tree.Datum:
		// *DArray
		// *DBool
		// *DBytes
		// *DCollatedString
		// *DDate
		// *DDecimal
		// *DFloat
		// *DIPAddr
		// *DInt
		// *DInterval
		// *DJSON
		// *DOid
		// *DOidWrapper
		// *DString
		// *DTable
		// *DTime
		// *DTimestamp
		// *DTimestampTZ
		// *DTuple
		// *DUuid
		// dNull
		return b.factory.ConstructConst(b.factory.InternPrivate(t))
	}

	fatalf("unexpected scalar expr: %T", scalar)
	return 0
}

func (b *Builder) buildFunction(f *tree.FuncExpr, inScope *scope) (out xform.GroupID, col *columnProps) {
	def, err := f.Func.Resolve(b.semaCtx.SearchPath)
	if err != nil {
		fatalf("%v", err)
	}

	isAgg := isAggregate(def)
	if isAgg {
		// Look for any column references contained within the aggregate.
		inScope.startAggFunc()
	}

	argList := make([]xform.GroupID, 0, len(f.Exprs))
	for _, pexpr := range f.Exprs {
		var arg xform.GroupID
		if _, ok := pexpr.(tree.UnqualifiedStar); ok {
			arg = b.factory.ConstructConst(b.factory.InternPrivate(tree.NewDInt(1)))
		} else {
			arg = b.buildScalar(pexpr.(tree.TypedExpr), inScope)
		}

		argList = append(argList, arg)
	}

	out = b.factory.ConstructFunction(b.factory.StoreList(argList), b.factory.InternPrivate(def))

	if isAgg {
		refScope := inScope.endAggFunc(out)

		// If the aggregate already exists as a column, use that. Otherwise
		// create a new column and add it the list of aggregates that need to
		// be computed by the groupby expression.
		// TODO(andy): turns out this doesn't really do anything because the
		//             list passed to ConstructFunction isn't interned.
		col = refScope.findAggregate(out)
		if col == nil {
			col = b.synthesizeColumn(refScope, "", f.ResolvedType())

			// Add the aggregate to the list of aggregates that need to be computed by
			// the groupby expression.
			refScope.groupby.aggs = append(refScope.groupby.aggs, out)
		}

		// Replace the function call with a reference to the column.
		out = b.factory.ConstructVariable(b.factory.InternPrivate(col.index))
	}

	return
}

func (b *Builder) buildSelect(stmt *tree.Select, inScope *scope) (out xform.GroupID, outScope *scope) {
	// NB: The case statements are sorted lexicographically.
	switch t := stmt.Select.(type) {
	case *tree.ParenSelect:
		return b.buildSelect(t.Select, inScope)

	case *tree.SelectClause:
		return b.buildSelectClause(stmt, inScope)

	case *tree.UnionClause:
		out, outScope = b.buildUnion(t, inScope)

	case *tree.ValuesClause:
		return b.buildValuesClause(t, inScope)

	default:
		fatalf("unexpected select statement: %T", stmt.Select)
	}

	// TODO(rytaft): Build Order By
	// TODO(peter): stmt.Limit
	return
}

func (b *Builder) buildValuesClause(values *tree.ValuesClause, inScope *scope) (out xform.GroupID, outScope *scope) {
	var numCols int
	if len(values.Tuples) > 0 {
		numCols = len(values.Tuples[0].Exprs)
	}

	// Synthesize right number of null-typed columns.
	outScope = inScope.push()
	for i := 0; i < numCols; i++ {
		b.synthesizeColumn(outScope, "", types.Null)
	}

	rows := make([]xform.GroupID, 0, len(values.Tuples))

	for _, tuple := range values.Tuples {
		if numCols != len(tuple.Exprs) {
			panic(fmt.Errorf(
				"VALUES lists must all be the same length, expected %d columns, found %d",
				numCols, len(tuple.Exprs)))
		}

		elems := make([]xform.GroupID, numCols)

		for i, expr := range tuple.Exprs {
			texpr := inScope.resolveType(expr, types.Any)
			typ := texpr.ResolvedType()
			elems[i] = b.buildScalar(texpr, inScope)

			// Verify that types of each tuple match one another.
			if outScope.cols[i].typ == types.Null {
				outScope.cols[i].typ = typ
			} else if typ != types.Null && !typ.Equivalent(outScope.cols[i].typ) {
				panic(fmt.Errorf("VALUES list type mismatch, %s for %s", typ, outScope.cols[i].typ))
			}
		}

		rows = append(rows, b.factory.ConstructTuple(b.factory.StoreList(elems)))
	}

	out = b.factory.ConstructValues(b.factory.StoreList(rows), b.factory.InternPrivate(makeColSet(outScope.cols)))
	return
}

// Pass the entire Select statement rather than just the select clause in
// order to handle ORDER BY scoping rules. ORDER BY can sort results using
// columns from the FROM/GROUP BY clause and/or from the projection list.
func (b *Builder) buildSelectClause(stmt *tree.Select, inScope *scope) (out xform.GroupID, outScope *scope) {
	sel := stmt.Select.(*tree.SelectClause)

	var fromScope *scope
	out, fromScope = b.buildFrom(sel.From, sel.Where, inScope)

	// The "from" columns are visible to any grouping expressions.
	groupings, groupingsScope := b.buildGroupingList(sel.GroupBy, fromScope)

	// Set the grouping scope so that any aggregates will be added to the set
	// of grouping columns.
	if groupings == nil {
		// Even though there is no groupby clause, create a grouping scope
		// anyway, since one or more aggregate functions in the projection list
		// triggers an implicit groupby.
		groupingsScope = fromScope.push()
		fromScope.groupby.groupingsScope = groupingsScope
	} else {
		// Add aggregate columns directly to the existing groupings scope.
		groupingsScope.groupby.groupingsScope = groupingsScope
	}

	// Any "grouping" columns are visible to both the "having" and "projection"
	// expressions. The build has the side effect of extracting aggregation
	// columns.
	var having xform.GroupID
	if sel.Having != nil {
		having = b.buildScalar(groupingsScope.resolveType(sel.Having.Expr, types.Bool), groupingsScope)
	}

	// If the projection is empty or a simple pass-through, then
	// buildProjectionList will return nil values.
	var projections []xform.GroupID
	var projectionsScope *scope
	if groupings == nil {
		projections, projectionsScope = b.buildProjectionList(sel.Exprs, fromScope)
	} else {
		projections, projectionsScope = b.buildProjectionList(sel.Exprs, groupingsScope)
	}

	// Wrap with groupby operator if groupings or aggregates exist.
	if groupings != nil || len(groupingsScope.groupby.aggs) > 0 {
		// Any aggregate columns that were discovered would have been appended
		// to the end of the grouping scope.
		aggCols := groupingsScope.cols[len(groupingsScope.cols)-len(groupingsScope.groupby.aggs):]
		aggList := b.constructProjectionList(groupingsScope.groupby.aggs, aggCols)

		var groupingCols []columnProps
		if groupings != nil {
			groupingCols = groupingsScope.cols
		}

		groupingList := b.constructProjectionList(groupings, groupingCols)
		out = b.factory.ConstructGroupBy(out, groupingList, aggList)

		// Wrap with having filter if it exists.
		if having != 0 {
			out = b.factory.ConstructSelect(out, having)
		}

		outScope = groupingsScope
	} else {
		// No aggregates, so current output scope is the "from" scope.
		outScope = fromScope
	}

	if stmt.OrderBy != nil {
    // TODO(rytaft): Build Order By. Order By relies on the existence of
    // ordering physical properties.
	}

	// Wrap with project operator if it exists.
	if projections != nil {
		out = b.factory.ConstructProject(out, b.constructProjectionList(projections, projectionsScope.cols))
		outScope = projectionsScope
	}

	// Wrap with distinct operator if it exists.
	out = b.buildDistinct(out, sel.Distinct, outScope.cols, outScope)
	return
}

func (b *Builder) buildFrom(from *tree.From, where *tree.Where, inScope *scope) (out xform.GroupID, outScope *scope) {
	var left, right xform.GroupID

	for _, table := range from.Tables {
		var rightScope *scope
		right, rightScope = b.buildTable(table, inScope)

		if left == 0 {
			left = right
			outScope = rightScope
			continue
		}

		outScope.appendColumns(rightScope)

		left = b.factory.ConstructInnerJoin(left, right, b.factory.ConstructTrue())
	}

	if left == 0 {
		// TODO(peter): This should be a table with 1 row and 0 columns to match
		// current cockroach behavior.
		rows := []xform.GroupID{b.factory.ConstructTuple(b.factory.StoreList(nil))}
		out = b.factory.ConstructValues(b.factory.StoreList(rows), b.factory.InternPrivate(&xform.ColSet{}))
		outScope = inScope
	} else {
		out = left
	}

	if where != nil {
		// All "from" columns are visible to the filter expression.
		texpr := outScope.resolveType(where.Expr, types.Bool)
		filter := b.buildScalar(texpr, outScope)
		out = b.factory.ConstructSelect(out, filter)
	}

	return
}

func (b *Builder) buildGroupingList(groupBy tree.GroupBy, inScope *scope) (groupings []xform.GroupID, outScope *scope) {
	if groupBy == nil {
		return
	}

	outScope = inScope.push()

	groupings = make([]xform.GroupID, 0, len(groupBy))
	for _, e := range groupBy {
		scalar := b.buildScalarProjection(inScope.resolveType(e, types.Any), inScope, outScope)
		groupings = append(groupings, scalar)
	}

	return
}

func (b *Builder) buildProjectionList(
	selects tree.SelectExprs,
	inScope *scope,
) (projections []xform.GroupID, outScope *scope) {
	if len(selects) == 0 {
		return nil, nil
	}

	outScope = inScope.push()
	projections = make([]xform.GroupID, 0, len(selects))
	for _, e := range selects {
		end := len(outScope.cols)
		subset := b.buildProjection(e.Expr, inScope, outScope)
		projections = append(projections, subset...)

		// Update the name of the column if there is an alias defined.
		if e.As != "" {
			for i := range outScope.cols[end:] {
				outScope.cols[i].name = optbase.ColumnName(e.As)
			}
		}
	}

	// Don't add an unnecessary "pass through" project expression.
	if len(inScope.groupby.groupingsScope.groupby.aggs) > 0 {
		// If aggregates will be projected, check against them instead.
		inScope = inScope.groupby.groupingsScope
	}

	if len(outScope.cols) == len(inScope.cols) {
		matches := true
		for i := range inScope.cols {
			if inScope.cols[i].index != outScope.cols[i].index {
				matches = false
				break
			}
		}

		if matches {
			return nil, nil
		}
	}

	return
}

func (b *Builder) buildProjection(projection tree.Expr, inScope, outScope *scope) (projections []xform.GroupID) {
	// We only have to handle "*" and "<name>.*" in the switch below. Other names
	// will be handled by scope.resolve().
	//
	// NB: The case statements are sorted lexicographically.
	switch t := projection.(type) {
	case *tree.AllColumnsSelector:
		tableName := optbase.TableName(t.TableName.Table())
		for _, col := range inScope.cols {
			if col.table == tableName && !col.hidden {
				v := b.factory.ConstructVariable(b.factory.InternPrivate(col.index))
				projections = append(projections, v)
				outScope.cols = append(outScope.cols, col)
			}
		}
		if len(projections) == 0 {
			fatalf("unknown table %s", t)
		}
		return projections

	case tree.UnqualifiedStar:
		for _, col := range inScope.cols {
			if !col.hidden {
				v := b.factory.ConstructVariable(b.factory.InternPrivate(col.index))
				projections = append(projections, v)
				outScope.cols = append(outScope.cols, col)
			}
		}
		if len(projections) == 0 {
			fatalf("failed to expand *")
		}
		return

	case *tree.UnresolvedName:
		vn, err := t.NormalizeVarName()
		if err != nil {
			panic(err)
		}
		return b.buildProjection(vn, inScope, outScope)

	default:
		texpr := inScope.resolveType(projection, types.Any)
		return []xform.GroupID{b.buildScalarProjection(texpr, inScope, outScope)}
	}
}

// buildDistinct builds a set of memo groups that represent a DISTINCT
// expression.
//
// in        contains the memo group ID of the input expression.
// distinct  is true if this is a DISTINCT expression. If distinct is false,
//           we just return `in`.
// byCols    is the set of columns in the DISTINCT expression. Since
//           DISTINCT is equivalent to GROUP BY without any aggregations,
//           byCols are essentially the grouping columns.
// inScope   contains the name bindings that are visible for this DISTINCT
//           expression (e.g., passed in from an enclosing statement).
//
// The return value corresponds to the top-level memo group ID for this
// DISTINCT expression.
func (b *Builder) buildDistinct(in xform.GroupID, distinct bool, byCols []columnProps, inScope *scope) xform.GroupID {
	if !distinct {
		return in
	}

	// Distinct is equivalent to group by without any aggregations.
	groupings := make([]xform.GroupID, 0, len(byCols))
	for i := range byCols {
		v := b.factory.ConstructVariable(b.factory.InternPrivate(byCols[i].index))
		groupings = append(groupings, v)
	}

	groupingList := b.constructProjectionList(groupings, byCols)
	aggList := b.constructProjectionList(nil, nil)
	return b.factory.ConstructGroupBy(in, groupingList, aggList)
}

func (b *Builder) buildAppendingProject(
	in xform.GroupID,
	inScope *scope,
	projections []xform.GroupID,
	projectionsScope *scope,
) (out xform.GroupID, outScope *scope) {
	if projections == nil {
		return in, inScope
	}

	outScope = projectionsScope.push()

	combined := make([]xform.GroupID, 0, len(inScope.cols)+len(projectionsScope.cols))
	for i := range inScope.cols {
		col := &inScope.cols[i]
		outScope.cols = append(outScope.cols, *col)
		combined = append(combined, b.factory.ConstructVariable(b.factory.InternPrivate(col.index)))
	}

	for i := range projectionsScope.cols {
		col := &projectionsScope.cols[i]

		// Only append projection columns that aren't already present.
		if findColByIndex(outScope.cols, col.index) == nil {
			outScope.cols = append(outScope.cols, *col)
			combined = append(combined, b.factory.ConstructVariable(b.factory.InternPrivate(col.index)))
		}
	}

	if len(outScope.cols) == len(inScope.cols) {
		// All projection columns were already present, so no need to construct
		// the projection expression.
		return in, inScope
	}

	out = b.factory.ConstructProject(in, b.constructProjectionList(combined, outScope.cols))
	return
}

func (b *Builder) buildUnion(clause *tree.UnionClause, inScope *scope) (out xform.GroupID, outScope *scope) {
	left, leftScope := b.buildSelect(clause.Left, inScope)
	right, rightScope := b.buildSelect(clause.Right, inScope)

	// Build map from left columns to right columns.
	colMap := make(xform.ColMap)
	for i := range leftScope.cols {
		colMap[leftScope.cols[i].index] = rightScope.cols[i].index
	}

	switch clause.Type {
	case tree.UnionOp:
		out = b.factory.ConstructUnion(left, right, b.factory.InternPrivate(&colMap))
	case tree.IntersectOp:
		out = b.factory.ConstructIntersect(left, right)
	case tree.ExceptOp:
		out = b.factory.ConstructExcept(left, right)
	}

	outScope = leftScope
	return
}

func (b *Builder) synthesizeColumn(scope *scope, label string, typ types.T) *columnProps {
	if label == "" {
		label = fmt.Sprintf("column%d", len(scope.cols)+1)
	}

	colIndex := b.factory.Metadata().AddColumn(label)
	col := columnProps{typ: typ, index: colIndex}
	b.colMap = append(b.colMap, col)
	scope.cols = append(scope.cols, col)
	return &scope.cols[len(scope.cols)-1]
}

func (b *Builder) constructJoin(joinType string, left, right, filter xform.GroupID) xform.GroupID {
	switch joinType {
	case "JOIN", "INNER JOIN", "CROSS JOIN":
		return b.factory.ConstructInnerJoin(left, right, filter)
	case "LEFT JOIN":
		return b.factory.ConstructLeftJoin(left, right, filter)
	case "RIGHT JOIN":
		return b.factory.ConstructRightJoin(left, right, filter)
	case "FULL JOIN":
		return b.factory.ConstructFullJoin(left, right, filter)
	default:
		unimplemented("unsupported JOIN type %s", joinType)
		return 0
	}
}

func (b *Builder) constructProjectionList(items []xform.GroupID, cols []columnProps) xform.GroupID {
	return b.factory.ConstructProjections(b.factory.StoreList(items), b.factory.InternPrivate(makeColSet(cols)))
}

func (b *Builder) IndexedVarEval(idx int, ctx *tree.EvalContext) (tree.Datum, error) {
	unimplemented("queryState.IndexedVarEval")
	return nil, fmt.Errorf("unimplemented")
}

func (b *Builder) IndexedVarResolvedType(idx int) types.T {
	return b.colMap[xform.ColumnIndex(idx)].typ
}

func (b *Builder) IndexedVarNodeFormatter(idx int) tree.NodeFormatter {
	unimplemented("queryState.IndexedVarNodeFormatter")
	return nil
}

func isAggregate(def *tree.FunctionDefinition) bool {
	return strings.EqualFold(def.Name, "count") ||
		strings.EqualFold(def.Name, "count_rows") ||
		strings.EqualFold(def.Name, "min") ||
		strings.EqualFold(def.Name, "max") ||
		strings.EqualFold(def.Name, "sum") ||
		strings.EqualFold(def.Name, "avg")
}

func makeColSet(cols []columnProps) *xform.ColSet {
	// Create column index list parameter to the ProjectionList op.
	var colSet xform.ColSet
	for i := range cols {
		colSet.Add(int(cols[i].index))
	}
	return &colSet
}

func findColByName(cols []columnProps, name optbase.ColumnName) *columnProps {
	for i := range cols {
		col := &cols[i]
		if col.name == name {
			return col
		}
	}

	return nil
}

func findColByIndex(cols []columnProps, colIndex xform.ColumnIndex) *columnProps {
	for i := range cols {
		col := &cols[i]
		if col.index == colIndex {
			return col
		}
	}

	return nil
}
