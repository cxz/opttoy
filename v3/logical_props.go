package v3

import (
	"bytes"
	"fmt"

	"github.com/cockroachdb/cockroach/pkg/sql/parser"
)

type columnRef struct {
	props *logicalProps
	index int
}

// queryState holds per-query state such as the tables referenced by the query
// and the mapping from table name to the column index for those tables columns
// within the query.
type queryState struct {
	catalog map[string]*table
	tables  map[string]bitmapIndex
	// query index to properties and column index.
	columns []columnRef
	data    []interface{}
}

func (s *queryState) addData(d interface{}) int32 {
	s.data = append(s.data, d)
	return int32(len(s.data))
}

func (s *queryState) getData(idx int32) interface{} {
	if idx == 0 {
		return nil
	}
	return s.data[idx-1]
}

type columnProps struct {
	name    string
	tables  []string
	index   bitmapIndex
	notNull bool // TODO(peter): unimplemented
	// TODO(peter): value constraints.
}

func (c columnProps) hasColumn(tableName, colName string) bool {
	if colName != c.name {
		return false
	}
	if tableName == "" {
		return true
	}
	return c.hasTable(tableName)
}

func (c columnProps) hasTable(tableName string) bool {
	for _, t := range c.tables {
		if t == tableName {
			return true
		}
	}
	return false
}

func (c columnProps) resolvedName(tableName string) *parser.ColumnItem {
	if tableName == "" {
		if len(c.tables) > 0 {
			tableName = c.tables[0]
		}
	}
	return &parser.ColumnItem{
		TableName: parser.TableName{
			TableName:               parser.Name(tableName),
			DBNameOriginallyOmitted: true,
		},
		ColumnName: parser.Name(c.name),
	}
}

func (c columnProps) newVariableExpr(tableName string, table *logicalProps) *expr {
	e := &expr{
		op:        variableOp,
		dataIndex: table.state.addData(c.resolvedName(tableName)),
		props:     table,
	}
	e.inputVars.set(c.index)
	e.updateProperties()
	return e
}

// TODO(peter): determine representation of functional dependencies and column
// constraints. For every column, we need to track nullability. This can be
// done using a bitmap.
//
// We also need to identify keys which are a set of columns and thus can also
// be represented by a bitmap. A "strong key" is one in which each column is
// NOT NULL. A "weak key" is one on which at least one column can be NULL.
//
// A functional dependency implies an associated function. For a key, the
// function is a database lookup. Given a key, we can determine the other
// columns in the table. Functional dependencies also arise in projections. For
// a given set of columns, we need to maintain an expression (the projection)
// which computes another column. If the function is invertible, we can also
// have the inverse dependency.
//
// Note that only projectOp contains projections. So we can hold a pointer to
// the projection expression. [An additional scalar property is
// invertibility. "a = b + 1" is invertible, "a = lower(b)" is not].
type logicalProps struct {
	name    string
	columns []columnProps
	keys    []tableKey // TODO(peter): unimplemented
	state   *queryState
}

func (t *logicalProps) String() string {
	var buf bytes.Buffer
	for i, col := range t.columns {
		if i > 0 {
			buf.WriteString(" ")
		}
		if tables := col.tables; len(tables) > 1 {
			buf.WriteString("{")
			for j, table := range tables {
				if j > 0 {
					buf.WriteString(",")
				}
				buf.WriteString(table)
			}
			buf.WriteString("}")
		} else if len(tables) == 1 {
			buf.WriteString(tables[0])
		} else {
			buf.WriteString(t.name)
		}
		buf.WriteString(".")
		buf.WriteString(col.name)
		buf.WriteString(":")
		fmt.Fprintf(&buf, "%d", col.index)
	}
	return buf.String()
}

func (t *logicalProps) newColumnExpr(name string) *expr {
	for _, col := range t.columns {
		if col.name == name {
			return col.newVariableExpr(t.name, t)
		}
	}
	return nil
}

func concatLogicalProperties(left, right *logicalProps) *logicalProps {
	t := &logicalProps{
		columns: make([]columnProps, len(left.columns)+len(right.columns)),
		state:   left.state,
	}
	copy(t.columns[:], left.columns)
	copy(t.columns[len(left.columns):], right.columns)
	return t
}