// TODO(peter): transformations
//
// - Distinct/group-by elimination. If the grouping columns are a key from the
//   input, we don't need to perform the grouping.
//
// - Group-by pull-up. Pull group-by above a join.
//
// - Group-by push-down. Push group-by below a join.

package v3

import (
	"fmt"

	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
)

func unimplemented(format string, args ...interface{}) {
	panic("unimplemented: " + fmt.Sprintf(format, args...))
}

func fatalf(format string, args ...interface{}) {
	panic(fmt.Sprintf(format, args...))
}

type planner struct {
	catalog map[string]*table
}

func newPlanner() *planner {
	return &planner{
		catalog: make(map[string]*table),
	}
}

func (p *planner) exec(stmt tree.Statement) string {
	switch stmt := stmt.(type) {
	case *tree.CreateTable:
		tab := createTable(p.catalog, stmt)
		return tab.String()
	default:
		unimplemented("%T", stmt)
	}
	return ""
}

func (p *planner) prep(stmt tree.Statement) *expr {
	state := &queryState{
		catalog: p.catalog,
		tables:  make(map[string]bitmapIndex),
	}
	e := build(stmt, &scope{
		props: &logicalProps{},
		state: state,
	})
	initKeys(e, state)
	return e
}
