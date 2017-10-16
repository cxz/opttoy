package v3

import (
	"bytes"
	"fmt"
)

func init() {
	operatorTab[projectOp] = operatorInfo{
		name: "projectOp",

		format: func(e *expr, buf *bytes.Buffer, level int) {
			indent := spaces[:2*level]
			fmt.Fprintf(buf, "%s%v (%s)", indent, e.op, e.table)
			e.formatVars(buf)
			buf.WriteString("\n")
			formatExprs(buf, "projections", e.aux1(), level)
			formatExprs(buf, "filters", e.filters(), level)
			formatExprs(buf, "inputs", e.inputs(), level)
		},

		updateProperties: func(expr *expr) {
			expr.inputVars = 0
			for _, filter := range expr.filters() {
				expr.inputVars |= filter.inputVars
			}
			expr.outputVars = 0
			for _, project := range expr.aux1() {
				expr.inputVars |= project.inputVars
				expr.outputVars |= project.outputVars
			}
		},
	}
}