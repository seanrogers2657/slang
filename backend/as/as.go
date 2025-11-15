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
	program *parser.Program,
) AsGenerator {
	return &asGenerator{
		program: program,
	}
}

type asGenerator struct {
	program *parser.Program
}

func (c *asGenerator) Generate() (string, error) {
	return GenerateProgram(c.program)
}

func GenerateProgram(program *parser.Program) (string, error) {
	builder := strings.Builder{}

	// Check if we need a .data section for strings
	hasStrings := false
	stringIndex := 0
	stringMap := make(map[*parser.Literal]string) // Map literals to their label names

	for _, stmt := range program.Statements {
		if stmt.Left.Type == parser.LiteralTypeString {
			hasStrings = true
			if _, exists := stringMap[stmt.Left]; !exists {
				stringMap[stmt.Left] = fmt.Sprintf("str_%d", stringIndex)
				stringIndex++
			}
		}
		if stmt.Right.Type == parser.LiteralTypeString {
			hasStrings = true
			if _, exists := stringMap[stmt.Right]; !exists {
				stringMap[stmt.Right] = fmt.Sprintf("str_%d", stringIndex)
				stringIndex++
			}
		}
	}

	// Write .data section if needed
	if hasStrings {
		builder.WriteString(".data\n")
		builder.WriteString(".align 3\n")

		// Define all unique string literals
		for literal, label := range stringMap {
			builder.WriteString(fmt.Sprintf("%s:\n", label))
			builder.WriteString(fmt.Sprintf("    .asciz %q\n", literal.Value))
		}

		builder.WriteString("\n.text\n")
	}

	builder.WriteString(".global _start\n")
	builder.WriteString(".align 4\n")
	builder.WriteString("_start:\n")

	// Generate code for each statement
	for _, stmt := range program.Statements {
		code, err := GenerateExprInline(stmt, stringMap)
		if err != nil {
			return "", err
		}
		builder.WriteString(code)
	}

	// Exit syscall (only at the end)
	builder.WriteString("    mov x0, #1\n")
	builder.WriteString("    mov x16, #0\n")
	builder.WriteString("    svc #0\n")

	return builder.String(), nil
}

func GenerateExprInline(expr *parser.Expr, stringMap map[*parser.Literal]string) (string, error) {
	builder := strings.Builder{}

	// Load left operand
	if expr.Left.Type == parser.LiteralTypeString {
		label := stringMap[expr.Left]
		builder.WriteString(fmt.Sprintf("    adr x0, %s\n", label))
	} else {
		builder.WriteString(fmt.Sprintf("    mov x0, #%s\n", expr.Left.Value))
	}

	// Load right operand
	if expr.Right.Type == parser.LiteralTypeString {
		label := stringMap[expr.Right]
		builder.WriteString(fmt.Sprintf("    adr x1, %s\n", label))
	} else {
		builder.WriteString(fmt.Sprintf("    mov x1, #%s\n", expr.Right.Value))
	}

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

	return builder.String(), nil
}

func GenerateExpr(expr *parser.Expr) (string, error) {
	builder := strings.Builder{}

	// Check if we need a .data section for strings
	hasStrings := expr.Left.Type == parser.LiteralTypeString || expr.Right.Type == parser.LiteralTypeString

	if hasStrings {
		builder.WriteString(".data\n")
		builder.WriteString(".align 3\n")

		// Define string literals in .data section
		if expr.Left.Type == parser.LiteralTypeString {
			builder.WriteString("str_left:\n")
			// Escape the string for assembly
			builder.WriteString(fmt.Sprintf("    .asciz %q\n", expr.Left.Value))
		}

		if expr.Right.Type == parser.LiteralTypeString {
			builder.WriteString("str_right:\n")
			builder.WriteString(fmt.Sprintf("    .asciz %q\n", expr.Right.Value))
		}

		builder.WriteString("\n.text\n")
	}

	builder.WriteString(".global _start\n")
	builder.WriteString(".align 4\n")

	builder.WriteString("_start:\n")

	// Load left operand
	if expr.Left.Type == parser.LiteralTypeString {
		builder.WriteString("    adr x0, str_left\n")
	} else {
		builder.WriteString(fmt.Sprintf("    mov x0, #%s\n", expr.Left.Value))
	}

	// Load right operand
	if expr.Right.Type == parser.LiteralTypeString {
		builder.WriteString("    adr x1, str_right\n")
	} else {
		builder.WriteString(fmt.Sprintf("    mov x1, #%s\n", expr.Right.Value))
	}

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
