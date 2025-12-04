package codegen

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/frontend/ast"
	"github.com/seanrogers2657/slang/frontend/semantic"
)

// Helper to create a minimal TypedProgram with a main function
func makeTypedProgram(stmts []semantic.TypedStatement) *semantic.TypedProgram {
	return &semantic.TypedProgram{
		Declarations: []semantic.TypedDeclaration{
			&semantic.TypedFunctionDecl{
				Name:       "main",
				Parameters: []semantic.TypedParameter{},
				ReturnType: semantic.TypeVoid,
				Body: &semantic.TypedBlockStmt{
					Statements: stmts,
				},
			},
		},
	}
}

// Helper to create a TypedProgram with a custom function
func makeTypedProgramWithFunc(fn *semantic.TypedFunctionDecl) *semantic.TypedProgram {
	return &semantic.TypedProgram{
		Declarations: []semantic.TypedDeclaration{fn},
	}
}

func TestNewTypedCodeGenerator(t *testing.T) {
	program := makeTypedProgram(nil)
	sourceLines := []string{"fn main() { }"}

	gen := NewTypedCodeGenerator(program, sourceLines)

	if gen == nil {
		t.Fatal("expected non-nil generator")
	}
	if gen.program != program {
		t.Error("program not set correctly")
	}
	if len(gen.sourceLines) != 1 {
		t.Error("sourceLines not set correctly")
	}
}

func TestTypedCodeGenerator_Generate_EmptyMain(t *testing.T) {
	program := makeTypedProgram([]semantic.TypedStatement{})
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedParts := []string{
		".global _start",
		"_start:",
		"bl _main",
		"_main:",
		"stp x29, x30, [sp, #-16]!",
		"mov x29, sp",
		"mov x0, #0", // default return for void main
		"ldp x29, x30, [sp], #16",
		"ret",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("expected output to contain %q\nFull output:\n%s", part, output)
		}
	}
}

func TestTypedCodeGenerator_IntegerLiteral(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"small positive", "42", "mov x2, #42"},
		{"zero", "0", "mov x2, #0"},
		{"large number", "1000", "mov x2, #1000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := &semantic.TypedExprStmt{
				Expr: &semantic.TypedLiteralExpr{
					Type:    semantic.TypeI64,
					LitType: ast.LiteralTypeInteger,
					Value:   tt.value,
				},
			}
			program := makeTypedProgram([]semantic.TypedStatement{stmt})
			gen := NewTypedCodeGenerator(program, nil)

			output, err := gen.Generate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected output to contain %q\nFull output:\n%s", tt.expected, output)
			}
		})
	}
}

func TestTypedCodeGenerator_VarDecl(t *testing.T) {
	tests := []struct {
		name     string
		varName  string
		initVal  string
		expected []string
	}{
		{
			name:    "simple integer variable",
			varName: "x",
			initVal: "42",
			expected: []string{
				"mov x2, #42",
				"str x2, [x29, #-16]",
			},
		},
		{
			name:    "another variable",
			varName: "count",
			initVal: "100",
			expected: []string{
				"mov x2, #100",
				"str x2, [x29, #-16]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := &semantic.TypedVarDeclStmt{
				Name:         tt.varName,
				DeclaredType: semantic.TypeI64,
				Initializer: &semantic.TypedLiteralExpr{
					Type:    semantic.TypeI64,
					LitType: ast.LiteralTypeInteger,
					Value:   tt.initVal,
				},
			}
			program := makeTypedProgram([]semantic.TypedStatement{stmt})
			gen := NewTypedCodeGenerator(program, nil)

			output, err := gen.Generate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected output to contain %q\nFull output:\n%s", exp, output)
				}
			}
		})
	}
}

func TestTypedCodeGenerator_VarDeclWithExpression(t *testing.T) {
	// val sum = 10 + 20
	stmt := &semantic.TypedVarDeclStmt{
		Name:         "sum",
		DeclaredType: semantic.TypeI64,
		Initializer: &semantic.TypedBinaryExpr{
			Type: semantic.TypeI64,
			Left: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeI64,
				LitType: ast.LiteralTypeInteger,
				Value:   "10",
			},
			Op: "+",
			Right: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeI64,
				LitType: ast.LiteralTypeInteger,
				Value:   "20",
			},
		},
	}
	program := makeTypedProgram([]semantic.TypedStatement{stmt})
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"mov x0, #10",
		"mov x1, #20",
		"add x2, x0, x1",
		"str x2, [x29, #-16]",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain %q\nFull output:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_BinaryExpr_AllIntOperations(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		expected []string
	}{
		{"addition", "+", []string{"add x2, x0, x1"}},
		{"subtraction", "-", []string{"sub x2, x0, x1"}},
		{"multiplication", "*", []string{"mul x2, x0, x1"}},
		{"division", "/", []string{"sdiv x2, x0, x1"}},
		{"modulo", "%", []string{"sdiv x3, x0, x1", "msub x2, x3, x1, x0"}},
		{"equal", "==", []string{"cmp x0, x1", "cset x2, eq"}},
		{"not equal", "!=", []string{"cmp x0, x1", "cset x2, ne"}},
		{"less than", "<", []string{"cmp x0, x1", "cset x2, lt"}},
		{"greater than", ">", []string{"cmp x0, x1", "cset x2, gt"}},
		{"less or equal", "<=", []string{"cmp x0, x1", "cset x2, le"}},
		{"greater or equal", ">=", []string{"cmp x0, x1", "cset x2, ge"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := &semantic.TypedExprStmt{
				Expr: &semantic.TypedBinaryExpr{
					Type: semantic.TypeI64,
					Left: &semantic.TypedLiteralExpr{
						Type:    semantic.TypeI64,
						LitType: ast.LiteralTypeInteger,
						Value:   "5",
					},
					Op: tt.op,
					Right: &semantic.TypedLiteralExpr{
						Type:    semantic.TypeI64,
						LitType: ast.LiteralTypeInteger,
						Value:   "3",
					},
				},
			}
			program := makeTypedProgram([]semantic.TypedStatement{stmt})
			gen := NewTypedCodeGenerator(program, nil)

			output, err := gen.Generate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected output to contain %q for op %q\nFull output:\n%s", exp, tt.op, output)
				}
			}
		})
	}
}

func TestTypedCodeGenerator_VariableReference(t *testing.T) {
	// val x = 42
	// x (reference)
	stmts := []semantic.TypedStatement{
		&semantic.TypedVarDeclStmt{
			Name:         "x",
			DeclaredType: semantic.TypeI64,
			Initializer: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeI64,
				LitType: ast.LiteralTypeInteger,
				Value:   "42",
			},
		},
		&semantic.TypedExprStmt{
			Expr: &semantic.TypedIdentifierExpr{
				Type: semantic.TypeI64,
				Name: "x",
			},
		},
	}
	program := makeTypedProgram(stmts)
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"str x2, [x29, #-16]", // store var
		"ldr x2, [x29, #-16]", // load var reference
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain %q\nFull output:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_Assignment(t *testing.T) {
	// var x = 10
	// x = 20
	stmts := []semantic.TypedStatement{
		&semantic.TypedVarDeclStmt{
			Name:         "x",
			Mutable:      true,
			DeclaredType: semantic.TypeI64,
			Initializer: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeI64,
				LitType: ast.LiteralTypeInteger,
				Value:   "10",
			},
		},
		&semantic.TypedAssignStmt{
			Name:    "x",
			VarType: semantic.TypeI64,
			Value: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeI64,
				LitType: ast.LiteralTypeInteger,
				Value:   "20",
			},
		},
	}
	program := makeTypedProgram(stmts)
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have two stores to the same location
	count := strings.Count(output, "str x2, [x29, #-16]")
	if count != 2 {
		t.Errorf("expected 2 stores to x29, #-16, got %d\nFull output:\n%s", count, output)
	}
}

func TestTypedCodeGenerator_ReturnStatement(t *testing.T) {
	fn := &semantic.TypedFunctionDecl{
		Name:       "getAnswer",
		Parameters: []semantic.TypedParameter{},
		ReturnType: semantic.TypeI64,
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedReturnStmt{
					Value: &semantic.TypedLiteralExpr{
						Type:    semantic.TypeI64,
						LitType: ast.LiteralTypeInteger,
						Value:   "42",
					},
				},
			},
		},
	}

	program := &semantic.TypedProgram{
		Declarations: []semantic.TypedDeclaration{
			fn,
			&semantic.TypedFunctionDecl{
				Name:       "main",
				Parameters: []semantic.TypedParameter{},
				ReturnType: semantic.TypeVoid,
				Body:       &semantic.TypedBlockStmt{Statements: []semantic.TypedStatement{}},
			},
		},
	}
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"_getAnswer:",
		"mov x2, #42",
		"mov x0, x2", // move result to return register
		"ret",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain %q\nFull output:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_ExitBuiltin(t *testing.T) {
	stmt := &semantic.TypedExprStmt{
		Expr: &semantic.TypedCallExpr{
			Type: semantic.TypeVoid,
			Name: "exit",
			Arguments: []semantic.TypedExpression{
				&semantic.TypedLiteralExpr{
					Type:    semantic.TypeI64,
					LitType: ast.LiteralTypeInteger,
					Value:   "42",
				},
			},
		},
	}
	program := makeTypedProgram([]semantic.TypedStatement{stmt})
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"mov x2, #42",
		"mov x0, x2",
		"mov x16, #1", // exit syscall
		"svc #0",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain %q\nFull output:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_FunctionWithParameters(t *testing.T) {
	// fn add(a: i64, b: i64): i64 { return a + b }
	fn := &semantic.TypedFunctionDecl{
		Name: "add",
		Parameters: []semantic.TypedParameter{
			{Name: "a", Type: semantic.TypeI64},
			{Name: "b", Type: semantic.TypeI64},
		},
		ReturnType: semantic.TypeI64,
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedReturnStmt{
					Value: &semantic.TypedBinaryExpr{
						Type: semantic.TypeI64,
						Left: &semantic.TypedIdentifierExpr{
							Type: semantic.TypeI64,
							Name: "a",
						},
						Op: "+",
						Right: &semantic.TypedIdentifierExpr{
							Type: semantic.TypeI64,
							Name: "b",
						},
					},
				},
			},
		},
	}

	program := &semantic.TypedProgram{
		Declarations: []semantic.TypedDeclaration{
			fn,
			&semantic.TypedFunctionDecl{
				Name:       "main",
				Parameters: []semantic.TypedParameter{},
				ReturnType: semantic.TypeVoid,
				Body:       &semantic.TypedBlockStmt{Statements: []semantic.TypedStatement{}},
			},
		},
	}
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"_add:",
		"str x0, [x29, #-16]", // store param a
		"str x1, [x29, #-32]", // store param b
		"ldr x0, [x29, #-16]", // load a
		"ldr x1, [x29, #-32]", // load b
		"add x2, x0, x1",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain %q\nFull output:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_FunctionCall(t *testing.T) {
	// fn double(x: i64): i64 { return x * 2 }
	// fn main() { double(21) }
	doubleFn := &semantic.TypedFunctionDecl{
		Name: "double",
		Parameters: []semantic.TypedParameter{
			{Name: "x", Type: semantic.TypeI64},
		},
		ReturnType: semantic.TypeI64,
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedReturnStmt{
					Value: &semantic.TypedBinaryExpr{
						Type: semantic.TypeI64,
						Left: &semantic.TypedIdentifierExpr{
							Type: semantic.TypeI64,
							Name: "x",
						},
						Op: "*",
						Right: &semantic.TypedLiteralExpr{
							Type:    semantic.TypeI64,
							LitType: ast.LiteralTypeInteger,
							Value:   "2",
						},
					},
				},
			},
		},
	}

	mainFn := &semantic.TypedFunctionDecl{
		Name:       "main",
		Parameters: []semantic.TypedParameter{},
		ReturnType: semantic.TypeVoid,
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedExprStmt{
					Expr: &semantic.TypedCallExpr{
						Type: semantic.TypeI64,
						Name: "double",
						Arguments: []semantic.TypedExpression{
							&semantic.TypedLiteralExpr{
								Type:    semantic.TypeI64,
								LitType: ast.LiteralTypeInteger,
								Value:   "21",
							},
						},
					},
				},
			},
		},
	}

	program := &semantic.TypedProgram{
		Declarations: []semantic.TypedDeclaration{doubleFn, mainFn},
	}
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"_double:",
		"_main:",
		"mov x2, #21",
		"mov x0, x2", // move arg to x0
		"bl _double", // call double
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain %q\nFull output:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_NestedBinaryExpr(t *testing.T) {
	// (2 + 3) * 4
	stmt := &semantic.TypedExprStmt{
		Expr: &semantic.TypedBinaryExpr{
			Type: semantic.TypeI64,
			Left: &semantic.TypedBinaryExpr{
				Type: semantic.TypeI64,
				Left: &semantic.TypedLiteralExpr{
					Type:    semantic.TypeI64,
					LitType: ast.LiteralTypeInteger,
					Value:   "2",
				},
				Op: "+",
				Right: &semantic.TypedLiteralExpr{
					Type:    semantic.TypeI64,
					LitType: ast.LiteralTypeInteger,
					Value:   "3",
				},
			},
			Op: "*",
			Right: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeI64,
				LitType: ast.LiteralTypeInteger,
				Value:   "4",
			},
		},
	}
	program := makeTypedProgram([]semantic.TypedStatement{stmt})
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should compute 2+3 first, then multiply by 4
	expected := []string{
		"add x2, x0, x1", // 2 + 3
		"mul x2, x0, x1", // result * 4
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected output to contain %q\nFull output:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenContext_DeclareVariable(t *testing.T) {
	ctx := NewBaseContext(nil)

	// First variable at offset 16
	offset1 := ctx.DeclareVariable("x", semantic.TypeI64)
	if offset1 != 16 {
		t.Errorf("expected first offset to be 16, got %d", offset1)
	}

	// Second variable at offset 32
	offset2 := ctx.DeclareVariable("y", semantic.TypeI64)
	if offset2 != 32 {
		t.Errorf("expected second offset to be 32, got %d", offset2)
	}

	// Verify lookup
	slot, ok := ctx.GetVariable("x")
	if !ok {
		t.Error("expected to find variable x")
	}
	if slot.Offset != 16 {
		t.Errorf("expected x offset to be 16, got %d", slot.Offset)
	}
}

func TestTypedCodeGenContext_GetVariableNotFound(t *testing.T) {
	ctx := NewBaseContext(nil)

	_, ok := ctx.GetVariable("nonexistent")
	if ok {
		t.Error("expected variable lookup to fail for nonexistent variable")
	}
}

func TestTypedCodeGenContext_SourceLineComment(t *testing.T) {
	sourceLines := []string{
		"fn main() {",
		"    val x = 42",
		"}",
	}
	ctx := NewBaseContext(sourceLines)

	// Line 2 (1-indexed)
	comment := ctx.GetSourceLineComment(ast.Position{Line: 2})
	if !strings.Contains(comment, "val x = 42") {
		t.Errorf("expected comment to contain source line, got %q", comment)
	}

	// Out of bounds
	comment = ctx.GetSourceLineComment(ast.Position{Line: 100})
	if comment != "" {
		t.Errorf("expected empty comment for out of bounds line, got %q", comment)
	}

	// Line 0 (invalid)
	comment = ctx.GetSourceLineComment(ast.Position{Line: 0})
	if comment != "" {
		t.Errorf("expected empty comment for line 0, got %q", comment)
	}
}

func TestTypedCodeGenerator_NoFunctions(t *testing.T) {
	program := &semantic.TypedProgram{
		Declarations: []semantic.TypedDeclaration{},
	}
	gen := NewTypedCodeGenerator(program, nil)

	_, err := gen.Generate()
	if err == nil {
		t.Error("expected error for program with no functions")
	}
	// The typed generator rejects legacy programs (no declarations)
	if !strings.Contains(err.Error(), "legacy") && !strings.Contains(err.Error(), "no functions") {
		t.Errorf("expected error about legacy programs or no functions, got: %v", err)
	}
}

func TestTypedCodeGenerator_UndefinedVariable(t *testing.T) {
	// Reference undefined variable
	stmt := &semantic.TypedExprStmt{
		Expr: &semantic.TypedIdentifierExpr{
			Type: semantic.TypeI64,
			Name: "undefined_var",
		},
	}
	program := makeTypedProgram([]semantic.TypedStatement{stmt})
	gen := NewTypedCodeGenerator(program, nil)

	_, err := gen.Generate()
	if err == nil {
		t.Error("expected error for undefined variable reference")
	}
	if !strings.Contains(err.Error(), "undefined variable") {
		t.Errorf("expected 'undefined variable' error, got: %v", err)
	}
}

func TestTypedCodeGenerator_UndefinedVariableInAssignment(t *testing.T) {
	// Assign to undefined variable
	stmt := &semantic.TypedAssignStmt{
		Name:    "undefined_var",
		VarType: semantic.TypeI64,
		Value: &semantic.TypedLiteralExpr{
			Type:    semantic.TypeI64,
			LitType: ast.LiteralTypeInteger,
			Value:   "10",
		},
	}
	program := makeTypedProgram([]semantic.TypedStatement{stmt})
	gen := NewTypedCodeGenerator(program, nil)

	_, err := gen.Generate()
	if err == nil {
		t.Error("expected error for assignment to undefined variable")
	}
	if !strings.Contains(err.Error(), "undefined variable") {
		t.Errorf("expected 'undefined variable' error, got: %v", err)
	}
}

func TestTypedCodeGenerator_FloatLiteral(t *testing.T) {
	stmt := &semantic.TypedExprStmt{
		Expr: &semantic.TypedLiteralExpr{
			Type:    semantic.TypeFloat64,
			LitType: ast.LiteralTypeFloat,
			Value:   "3.14159",
		},
	}
	program := makeTypedProgram([]semantic.TypedStatement{stmt})
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		".data",
		".double 3.14159",
		"adrp x8",
		"ldr d0",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_FloatBinaryExpr(t *testing.T) {
	stmt := &semantic.TypedExprStmt{
		Expr: &semantic.TypedBinaryExpr{
			Left: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeFloat64,
				LitType: ast.LiteralTypeFloat,
				Value:   "1.5",
			},
			Op: "+",
			Right: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeFloat64,
				LitType: ast.LiteralTypeFloat,
				Value:   "2.5",
			},
			Type: semantic.TypeFloat64,
		},
	}
	program := makeTypedProgram([]semantic.TypedStatement{stmt})
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"fmov d1, d0", // save left to d1
		"fadd d0, d1, d0",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_PrintIntegerLiteral(t *testing.T) {
	stmt := &semantic.TypedExprStmt{
		Expr: &semantic.TypedCallExpr{
			Name: "print",
			Type: semantic.TypeVoid,
			Arguments: []semantic.TypedExpression{
				&semantic.TypedLiteralExpr{
					Type:    semantic.TypeI64,
					LitType: ast.LiteralTypeInteger,
					Value:   "42",
				},
			},
		},
	}
	program := makeTypedProgram([]semantic.TypedStatement{stmt})
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"buffer: .space 32",
		"newline: .byte 10",
		"int_to_string",
		"mov x2, #42",
		"bl int_to_string",
		"mov x16, #4", // write syscall
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_PrintStringLiteral(t *testing.T) {
	stmt := &semantic.TypedExprStmt{
		Expr: &semantic.TypedCallExpr{
			Name: "print",
			Type: semantic.TypeVoid,
			Arguments: []semantic.TypedExpression{
				&semantic.TypedLiteralExpr{
					Type:    semantic.TypeString,
					LitType: ast.LiteralTypeString,
					Value:   "hello world",
				},
			},
		},
	}
	program := makeTypedProgram([]semantic.TypedStatement{stmt})
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"str_0:",
		"hello world",
		"adrp x1",
		"mov x2, #11", // length
		"mov x0, #1",  // stdout
		"mov x16, #4", // write syscall
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_IntegerLiteralTypes(t *testing.T) {
	tests := []struct {
		name     string
		typ      semantic.Type
		expected string
	}{
		{"i8", semantic.TypeI8, "sxtb x2, w2"},
		{"i16", semantic.TypeI16, "sxth x2, w2"},
		{"i32", semantic.TypeI32, "sxtw x2, w2"},
		{"u8", semantic.TypeU8, "and x2, x2, #0xFF"},
		{"u16", semantic.TypeU16, "and x2, x2, #0xFFFF"},
		{"u32", semantic.TypeU32, "mov w2, w2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt := &semantic.TypedExprStmt{
				Expr: &semantic.TypedLiteralExpr{
					Type:    tt.typ,
					LitType: ast.LiteralTypeInteger,
					Value:   "42",
				},
			}
			program := makeTypedProgram([]semantic.TypedStatement{stmt})
			gen := NewTypedCodeGenerator(program, nil)

			output, err := gen.Generate()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output for %s type, got:\n%s", tt.expected, tt.name, output)
			}
		})
	}
}

func TestTypedCodeGenerator_FloatVariable(t *testing.T) {
	stmts := []semantic.TypedStatement{
		&semantic.TypedVarDeclStmt{
			Name:         "f",
			DeclaredType: semantic.TypeFloat64,
			Initializer: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeFloat64,
				LitType: ast.LiteralTypeFloat,
				Value:   "1.5",
			},
		},
		&semantic.TypedExprStmt{
			Expr: &semantic.TypedIdentifierExpr{
				Type: semantic.TypeFloat64,
				Name: "f",
			},
		},
	}
	program := makeTypedProgram(stmts)
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"str d0, [x29, #-16]", // store float variable
		"ldr d0, [x29, #-16]", // load float variable
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestTypedCodeGenerator_FloatAssignment(t *testing.T) {
	stmts := []semantic.TypedStatement{
		&semantic.TypedVarDeclStmt{
			Name:         "f",
			DeclaredType: semantic.TypeFloat64,
			Initializer: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeFloat64,
				LitType: ast.LiteralTypeFloat,
				Value:   "1.0",
			},
		},
		&semantic.TypedAssignStmt{
			Name:    "f",
			VarType: semantic.TypeFloat64,
			Value: &semantic.TypedLiteralExpr{
				Type:    semantic.TypeFloat64,
				LitType: ast.LiteralTypeFloat,
				Value:   "2.0",
			},
		},
	}
	program := makeTypedProgram(stmts)
	gen := NewTypedCodeGenerator(program, nil)

	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have two float stores
	count := strings.Count(output, "str d0, [x29")
	if count != 2 {
		t.Errorf("expected 2 float stores, got %d in:\n%s", count, output)
	}
}

func TestTypedCodeGenerator_FloatReturn(t *testing.T) {
	fn := &semantic.TypedFunctionDecl{
		Name:       "getFloat",
		ReturnType: semantic.TypeFloat64,
		Parameters: []semantic.TypedParameter{},
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedReturnStmt{
					Value: &semantic.TypedLiteralExpr{
						Type:    semantic.TypeFloat64,
						LitType: ast.LiteralTypeFloat,
						Value:   "3.14",
					},
				},
			},
		},
	}

	program := &semantic.TypedProgram{
		Declarations: []semantic.TypedDeclaration{
			fn,
			&semantic.TypedFunctionDecl{
				Name:       "main",
				ReturnType: semantic.TypeVoid,
				Parameters: []semantic.TypedParameter{},
				Body: &semantic.TypedBlockStmt{
					Statements: []semantic.TypedStatement{},
				},
			},
		},
	}

	gen := NewTypedCodeGenerator(program, nil)
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Float return uses fmov to transfer d0 to x0
	if !strings.Contains(output, "fmov x0, d0") {
		t.Errorf("expected 'fmov x0, d0' for float return, got:\n%s", output)
	}
}

func TestTypedCodeGenerator_OperandToReg_CallExpr(t *testing.T) {
	// Test that call expressions work as operands in binary expressions
	stmts := []semantic.TypedStatement{
		&semantic.TypedExprStmt{
			Expr: &semantic.TypedBinaryExpr{
				Left: &semantic.TypedCallExpr{
					Name: "foo",
					Type: semantic.TypeI64,
					Arguments: []semantic.TypedExpression{
						&semantic.TypedLiteralExpr{
							Type:    semantic.TypeI64,
							LitType: ast.LiteralTypeInteger,
							Value:   "1",
						},
					},
				},
				Op: "+",
				Right: &semantic.TypedLiteralExpr{
					Type:    semantic.TypeI64,
					LitType: ast.LiteralTypeInteger,
					Value:   "2",
				},
				Type: semantic.TypeI64,
			},
		},
	}

	fn := &semantic.TypedFunctionDecl{
		Name:       "foo",
		ReturnType: semantic.TypeI64,
		Parameters: []semantic.TypedParameter{
			{Name: "x", Type: semantic.TypeI64},
		},
		Body: &semantic.TypedBlockStmt{
			Statements: []semantic.TypedStatement{
				&semantic.TypedReturnStmt{
					Value: &semantic.TypedIdentifierExpr{
						Type: semantic.TypeI64,
						Name: "x",
					},
				},
			},
		},
	}

	mainFn := &semantic.TypedFunctionDecl{
		Name:       "main",
		ReturnType: semantic.TypeVoid,
		Parameters: []semantic.TypedParameter{},
		Body: &semantic.TypedBlockStmt{
			Statements: stmts,
		},
	}

	program := &semantic.TypedProgram{
		Declarations: []semantic.TypedDeclaration{fn, mainFn},
	}

	gen := NewTypedCodeGenerator(program, nil)
	output, err := gen.Generate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"bl _foo",
		"add x2, x0, x1",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}
