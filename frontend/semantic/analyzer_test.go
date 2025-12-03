package semantic

import (
	"testing"

	"github.com/seanrogers2657/slang/frontend/ast"
)

func TestAnalyzeLiteral(t *testing.T) {
	t.Run("integer literal", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeLiteral(intLit("42"))
		test.expectType(result, TypeInteger)
		test.expectNoErrors()
	})

	t.Run("string literal", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeLiteral(strLit("hello"))
		test.expectType(result, TypeString)
		test.expectNoErrors()
	})
}

func TestAnalyzeBinaryExpression_Arithmetic(t *testing.T) {
	for _, op := range []string{"+", "-", "*", "/", "%"} {
		t.Run(op, func(t *testing.T) {
			test := newTest(t)
			result := test.analyzer.analyzeBinaryExpression(binExpr(intLit("5"), op, intLit("3")))
			test.expectType(result, TypeInteger)
			test.expectNoErrors()
		})
	}
}

func TestAnalyzeBinaryExpression_Comparison(t *testing.T) {
	for _, op := range []string{"==", "!=", "<", ">", "<=", ">="} {
		t.Run(op, func(t *testing.T) {
			test := newTest(t)
			result := test.analyzer.analyzeBinaryExpression(binExpr(intLit("5"), op, intLit("3")))
			test.expectType(result, TypeInteger)
			test.expectNoErrors()
		})
	}
}

func TestAnalyzeBinaryExpression_TypeError(t *testing.T) {
	tests := []struct {
		name string
		expr *BinaryExprBuilder
	}{
		{"string + int", bin(strLit("test"), "+", intLit("3"))},
		{"int + string", bin(intLit("5"), "+", strLit("test"))},
		{"string - string", bin(strLit("a"), "-", strLit("b"))},
		{"string == string", bin(strLit("a"), "==", strLit("b"))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newTest(t)
			result := test.analyzer.analyzeBinaryExpression(tt.expr.build())
			test.expectType(result, TypeError)
			test.expectErrorContaining("requires integer operands")
		})
	}
}

func TestAnalyzeProgram(t *testing.T) {
	t.Run("valid expression statement", func(t *testing.T) {
		test := newTest(t)
		errs, typedProgram := test.analyzer.Analyze(program(exprStmt(binExpr(intLit("5"), "+", intLit("3")))))

		if len(errs) > 0 {
			t.Errorf("expected no errors, got %d", len(errs))
		}
		if len(typedProgram.Statements) != 1 {
			t.Errorf("expected 1 statement, got %d", len(typedProgram.Statements))
		}
	})

	t.Run("valid print statement", func(t *testing.T) {
		test := newTest(t)
		errs, typedProgram := test.analyzer.Analyze(program(printStmt(intLit("42"))))

		if len(errs) > 0 {
			t.Errorf("expected no errors, got %d", len(errs))
		}
		if len(typedProgram.Statements) != 1 {
			t.Errorf("expected 1 statement, got %d", len(typedProgram.Statements))
		}

		ps, ok := typedProgram.Statements[0].(*TypedPrintStmt)
		if !ok {
			t.Fatal("expected TypedPrintStmt")
		}
		if !ps.Expr.GetType().Equals(TypeInteger) {
			t.Errorf("expected integer type, got %s", ps.Expr.GetType())
		}
	})

	t.Run("type error", func(t *testing.T) {
		test := newTest(t)
		errs, _ := test.analyzer.Analyze(program(exprStmt(binExpr(strLit("hello"), "+", intLit("3")))))

		if len(errs) == 0 {
			t.Error("expected errors, got none")
		}
		if errs[0].Stage != "semantic" {
			t.Errorf("expected semantic stage, got %s", errs[0].Stage)
		}
	})
}

func TestAnalyzeVariableDeclaration(t *testing.T) {
	tests := []struct {
		name         string
		varName      string
		initValue    string
		expectedType Type
	}{
		{"simple integer", "x", "42", TypeInteger},
		{"with underscore", "my_var", "100", TypeInteger},
		{"with digits", "value1", "5", TypeInteger},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newTest(t).withScope()
			result := test.analyzer.analyzeVarDeclStatement(varDecl(tt.varName, false, intLit(tt.initValue)))

			typedVarDecl, ok := result.(*TypedVarDeclStmt)
			if !ok {
				t.Fatal("expected TypedVarDeclStmt")
			}

			test.expectType(typedVarDecl.Initializer, tt.expectedType)
			test.expectNoErrors()

			// Verify variable is in scope
			varInfo, found := test.analyzer.currentScope.lookup(tt.varName)
			if !found {
				t.Errorf("variable %s not found in scope", tt.varName)
			}
			if !varInfo.Type.Equals(tt.expectedType) {
				t.Errorf("variable has wrong type: expected %s, got %s", tt.expectedType, varInfo.Type)
			}
		})
	}
}

func TestAnalyzeIdentifierExpression(t *testing.T) {
	t.Run("valid lookup", func(t *testing.T) {
		test := newTest(t).withScope().declare("myVar", TypeInteger, false)
		result := test.analyzer.analyzeIdentifier(ident("myVar"))
		test.expectType(result, TypeInteger)
		test.expectNoErrors()
	})

	t.Run("undefined variable", func(t *testing.T) {
		test := newTest(t).withScope()
		result := test.analyzer.analyzeIdentifier(ident("undefinedVar"))
		test.expectType(result, TypeError)
		test.expectErrorContaining("undefined variable 'undefinedVar'")
	})
}

func TestAnalyzeDuplicateVariableDeclaration(t *testing.T) {
	test := newTest(t).withScope()
	test.analyzer.analyzeVarDeclStatement(varDecl("x", false, intLit("5")))
	test.analyzer.analyzeVarDeclStatement(varDecl("x", false, intLit("10")))
	test.expectErrorContaining("already declared")
}

func TestAnalyzeVariableInExpression(t *testing.T) {
	t.Run("valid variables", func(t *testing.T) {
		test := newTest(t).withScope().declare("a", TypeInteger, false).declare("b", TypeInteger, false)
		result := test.analyzer.analyzeBinaryExpression(binExpr(ident("a"), "+", ident("b")))
		test.expectType(result, TypeInteger)
		test.expectNoErrors()
	})

	t.Run("undefined variable", func(t *testing.T) {
		test := newTest(t).withScope().declare("a", TypeInteger, false)
		result := test.analyzer.analyzeBinaryExpression(binExpr(ident("a"), "+", ident("b")))
		test.expectType(result, TypeError)
		test.expectErrorContaining("undefined variable")
	})
}

func TestAnalyzeAssignmentStatement(t *testing.T) {
	t.Run("mutable variable", func(t *testing.T) {
		test := newTest(t).withScope().declare("x", TypeInteger, true)
		result := test.analyzer.analyzeAssignStatement(assignStmt("x", intLit("10")))

		typedAssign, ok := result.(*TypedAssignStmt)
		if !ok {
			t.Fatalf("expected TypedAssignStmt, got %T", result)
		}
		if typedAssign.Name != "x" {
			t.Errorf("expected name 'x', got %q", typedAssign.Name)
		}
		if !typedAssign.VarType.Equals(TypeInteger) {
			t.Errorf("expected VarType integer, got %s", typedAssign.VarType)
		}
		test.expectNoErrors()
	})

	t.Run("immutable variable", func(t *testing.T) {
		test := newTest(t).withScope().declare("x", TypeInteger, false)
		test.analyzer.analyzeAssignStatement(assignStmt("x", intLit("10")))
		test.expectErrorContaining("cannot assign to immutable variable")
	})

	t.Run("undefined variable", func(t *testing.T) {
		test := newTest(t).withScope()
		result := test.analyzer.analyzeAssignStatement(assignStmt("undefinedVar", intLit("10")))

		typedAssign, ok := result.(*TypedAssignStmt)
		if !ok {
			t.Fatalf("expected TypedAssignStmt, got %T", result)
		}
		if !typedAssign.VarType.Equals(TypeError) {
			t.Errorf("expected VarType error, got %s", typedAssign.VarType)
		}
		test.expectErrorContaining("undefined variable")
	})

	t.Run("no cascading error from undefined value", func(t *testing.T) {
		test := newTest(t).withScope().declare("x", TypeInteger, true)
		test.analyzer.analyzeAssignStatement(assignStmt("x", ident("undefinedVar")))
		test.expectErrors(1)
		test.expectErrorContaining("undefined variable 'undefinedVar'")
	})
}

// -----------------------------------------------------------------------------
// Helper for deferred BinaryExpr building (preserves type info in test tables)
// -----------------------------------------------------------------------------

type BinaryExprBuilder struct {
	left  interface{}
	op    string
	right interface{}
}

func bin(left interface{}, op string, right interface{}) *BinaryExprBuilder {
	return &BinaryExprBuilder{left: left, op: op, right: right}
}

func (b *BinaryExprBuilder) build() *ast.BinaryExpr {
	return binExpr(b.left.(ast.Expression), b.op, b.right.(ast.Expression))
}
