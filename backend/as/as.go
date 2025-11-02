package as

import (
	"fmt"
	"strings"

	"github.com/seanrogers2657/slang/frontend/parser"
)

type AsGenerator interface {
	Generate() (string, error)
}

func NewAsGenerator(
	root *parser.Expr,
) AsGenerator {
	return &asGenerator{
		root: root,
	}
}

type asGenerator struct {
	root *parser.Expr
}

func (c *asGenerator) Generate() (string, error) {
	return GenerateExpr(c.root)
}

func GenerateExpr(expr *parser.Expr) (string, error) {
	builder := strings.Builder{}
	builder.WriteString(".global _start\n")
	builder.WriteString(".align 4\n")

	builder.WriteString("_start:\n")
	builder.WriteString(fmt.Sprintf("    mov x0, #%s\n", expr.Left.Value))
	builder.WriteString(fmt.Sprintf("    mov x1, #%s\n", expr.Right.Value))

	switch expr.Op {
	case "+":
		builder.WriteString("    add x2, x0, x1\n")
	case "-":
		builder.WriteString("    sub x2, x0, x1\n")
	case "*":
		builder.WriteString("    mul x2, x0, x1\n")
	case "/":
		builder.WriteString("    sdiv x2, x0, x1\n")
	case "%":
		// Modulo: x2 = x0 - (x0 / x1) * x1
		builder.WriteString("    sdiv x3, x0, x1\n")
		builder.WriteString("    msub x2, x3, x1, x0\n")
	case "==":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, eq\n")
	case "!=":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, ne\n")
	case "<":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, lt\n")
	case ">":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, gt\n")
	case "<=":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, le\n")
	case ">=":
		builder.WriteString("    cmp x0, x1\n")
		builder.WriteString("    cset x2, ge\n")
	default:
		return "", fmt.Errorf("unsupported operation %s when generating code", expr.Op)
	}

	builder.WriteString("    mov x0, #1\n")
	builder.WriteString("    mov x16, #0\n")
	builder.WriteString("    svc #0\n")

	return builder.String(), nil
}
