package semantic

import (
	"testing"

	"github.com/seanrogers2657/slang/compiler/ast"
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
			test.expectType(result, TypeBoolean)
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
			test.expectErrorContaining("requires numeric operands")
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

// -----------------------------------------------------------------------------
// Bounds Checking Tests
// -----------------------------------------------------------------------------

func TestAnalyzeTypedVariableDeclaration(t *testing.T) {
	tests := []struct {
		name         string
		typeName     string
		initValue    string
		expectedType Type
	}{
		{"i8 type", "i8", "42", TypeI8},
		{"i16 type", "i16", "1000", TypeI16},
		{"i32 type", "i32", "100000", TypeI32},
		{"i64 type", "i64", "9223372036854775807", TypeI64},
		{"u8 type", "u8", "255", TypeU8},
		{"u16 type", "u16", "65535", TypeU16},
		{"u32 type", "u32", "4294967295", TypeU32},
		{"u64 type", "u64", "18446744073709551615", TypeU64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newTest(t).withScope()
			result := test.analyzer.analyzeVarDeclStatement(typedVarDecl("x", tt.typeName, false, intLit(tt.initValue)))

			typedVarDecl, ok := result.(*TypedVarDeclStmt)
			if !ok {
				t.Fatal("expected TypedVarDeclStmt")
			}

			test.expectNoErrors()

			// Verify variable is in scope with correct type
			varInfo, found := test.analyzer.currentScope.lookup("x")
			if !found {
				t.Error("variable x not found in scope")
			}
			if !varInfo.Type.Equals(tt.expectedType) {
				t.Errorf("variable has wrong type: expected %s, got %s", tt.expectedType, varInfo.Type)
			}
			if !typedVarDecl.DeclaredType.Equals(tt.expectedType) {
				t.Errorf("declared type wrong: expected %s, got %s", tt.expectedType, typedVarDecl.DeclaredType)
			}
		})
	}
}

func TestAnalyzeBoundsChecking(t *testing.T) {
	tests := []struct {
		name      string
		typeName  string
		initValue string
		errorMsg  string
	}{
		// i8 bounds: -128 to 127
		{"i8 overflow positive", "i8", "128", "out of range for i8"},
		{"i8 overflow large", "i8", "200", "out of range for i8"},
		// i16 bounds: -32768 to 32767
		{"i16 overflow", "i16", "32768", "out of range for i16"},
		// i32 bounds: -2147483648 to 2147483647
		{"i32 overflow", "i32", "2147483648", "out of range for i32"},
		// u8 bounds: 0 to 255
		{"u8 overflow", "u8", "256", "out of range for u8"},
		// u16 bounds: 0 to 65535
		{"u16 overflow", "u16", "65536", "out of range for u16"},
		// u32 bounds: 0 to 4294967295
		{"u32 overflow", "u32", "4294967296", "out of range for u32"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newTest(t).withScope()
			test.analyzer.analyzeVarDeclStatement(typedVarDecl("x", tt.typeName, false, intLit(tt.initValue)))
			test.expectErrorContaining(tt.errorMsg)
		})
	}
}

func TestAnalyzeBoundsCheckingNegative(t *testing.T) {
	// Test negative values for unsigned types
	t.Run("negative value for u8", func(t *testing.T) {
		test := newTest(t).withScope()
		// Note: negative literals would be represented as unary minus on a positive literal
		// For now, test that we properly validate against unsigned types
		test.analyzer.analyzeVarDeclStatement(typedVarDecl("x", "u8", false, intLit("-1")))
		test.expectErrorContaining("out of range for u8")
	})
}

// -----------------------------------------------------------------------------
// Type Compatibility Tests
// -----------------------------------------------------------------------------

func TestAnalyzeTypeMismatch(t *testing.T) {
	tests := []struct {
		name      string
		leftType  string
		rightType string
		errorMsg  string
	}{
		{"i32 + i64", "i32", "i64", "requires operands of the same type"},
		{"i8 + i16", "i8", "i16", "requires operands of the same type"},
		{"u8 + u16", "u8", "u16", "requires operands of the same type"},
		{"i32 + u32", "i32", "u32", "requires operands of the same type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newTest(t).withScope()
			// Declare variables with different types
			test.analyzer.analyzeVarDeclStatement(typedVarDecl("a", tt.leftType, false, intLit("10")))
			test.analyzer.analyzeVarDeclStatement(typedVarDecl("b", tt.rightType, false, intLit("20")))
			// Try to use them in a binary expression
			test.analyzer.analyzeBinaryExpression(binExpr(ident("a"), "+", ident("b")))
			test.expectErrorContaining(tt.errorMsg)
		})
	}
}

func TestAnalyzeSameTypeOperations(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		value1   string
		value2   string
	}{
		{"i8 + i8", "i8", "10", "20"},
		{"i16 + i16", "i16", "100", "200"},
		{"i32 + i32", "i32", "1000", "2000"},
		{"i64 + i64", "i64", "10000", "20000"},
		{"u8 + u8", "u8", "10", "20"},
		{"u16 + u16", "u16", "100", "200"},
		{"u32 + u32", "u32", "1000", "2000"},
		{"u64 + u64", "u64", "10000", "20000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newTest(t).withScope()
			test.analyzer.analyzeVarDeclStatement(typedVarDecl("a", tt.typeName, false, intLit(tt.value1)))
			test.analyzer.analyzeVarDeclStatement(typedVarDecl("b", tt.typeName, false, intLit(tt.value2)))
			result := test.analyzer.analyzeBinaryExpression(binExpr(ident("a"), "+", ident("b")))
			test.expectNoErrors()
			expectedType := TypeFromName(tt.typeName)
			test.expectType(result, expectedType)
		})
	}
}

// -----------------------------------------------------------------------------
// Float Type Tests
// -----------------------------------------------------------------------------

func TestAnalyzeFloatLiteral(t *testing.T) {
	t.Run("float literal default type", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeLiteral(floatLit("3.14"))
		test.expectType(result, TypeFloat64)
		test.expectNoErrors()
	})
}

func TestAnalyzeTypedFloatDeclaration(t *testing.T) {
	tests := []struct {
		name         string
		typeName     string
		initValue    string
		expectedType Type
	}{
		{"f32 type", "f32", "3.14", TypeFloat32},
		{"f64 type", "f64", "3.14159", TypeFloat64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newTest(t).withScope()
			result := test.analyzer.analyzeVarDeclStatement(typedVarDecl("x", tt.typeName, false, floatLit(tt.initValue)))

			typedVarDecl, ok := result.(*TypedVarDeclStmt)
			if !ok {
				t.Fatal("expected TypedVarDeclStmt")
			}

			test.expectNoErrors()

			varInfo, found := test.analyzer.currentScope.lookup("x")
			if !found {
				t.Error("variable x not found in scope")
			}
			if !varInfo.Type.Equals(tt.expectedType) {
				t.Errorf("variable has wrong type: expected %s, got %s", tt.expectedType, varInfo.Type)
			}
			_ = typedVarDecl // silence unused warning
		})
	}
}

func TestAnalyzeFloatOperations(t *testing.T) {
	for _, op := range []string{"+", "-", "*", "/"} {
		t.Run("f64 "+op+" f64", func(t *testing.T) {
			test := newTest(t).withScope()
			test.analyzer.analyzeVarDeclStatement(typedVarDecl("a", "f64", false, floatLit("1.5")))
			test.analyzer.analyzeVarDeclStatement(typedVarDecl("b", "f64", false, floatLit("2.5")))
			result := test.analyzer.analyzeBinaryExpression(binExpr(ident("a"), op, ident("b")))
			test.expectNoErrors()
			test.expectType(result, TypeFloat64)
		})
	}
}

func TestAnalyzeFloatIntegerMismatch(t *testing.T) {
	t.Run("f64 + i64", func(t *testing.T) {
		test := newTest(t).withScope()
		test.analyzer.analyzeVarDeclStatement(typedVarDecl("a", "f64", false, floatLit("1.5")))
		test.analyzer.analyzeVarDeclStatement(typedVarDecl("b", "i64", false, intLit("10")))
		test.analyzer.analyzeBinaryExpression(binExpr(ident("a"), "+", ident("b")))
		test.expectErrorContaining("requires operands of the same type")
	})
}

func TestAnalyzeUnknownType(t *testing.T) {
	t.Run("unknown type annotation", func(t *testing.T) {
		test := newTest(t).withScope()
		test.analyzer.analyzeVarDeclStatement(typedVarDecl("x", "unknown_type", false, intLit("42")))
		test.expectErrorContaining("unknown type")
	})
}

// -----------------------------------------------------------------------------
// Boolean Type Tests
// -----------------------------------------------------------------------------

func TestAnalyzeBooleanLiteral(t *testing.T) {
	t.Run("true literal", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeLiteral(boolLit("true"))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("false literal", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeLiteral(boolLit("false"))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})
}

func TestAnalyzeUnaryNot(t *testing.T) {
	t.Run("!true has type bool", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeUnaryExpression(unaryExpr("!", boolLit("true")))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("!false has type bool", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeUnaryExpression(unaryExpr("!", boolLit("false")))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("!!true has type bool", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeUnaryExpression(unaryExpr("!", unaryExpr("!", boolLit("true"))))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("!5 is type error", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeUnaryExpression(unaryExpr("!", intLit("5")))
		test.expectType(result, TypeError)
		test.expectErrorContaining("requires boolean operand")
	})

	t.Run("!variable with boolean type", func(t *testing.T) {
		test := newTest(t).withScope().declare("flag", TypeBoolean, false)
		result := test.analyzer.analyzeUnaryExpression(unaryExpr("!", ident("flag")))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("!variable with integer type", func(t *testing.T) {
		test := newTest(t).withScope().declare("count", TypeInteger, false)
		result := test.analyzer.analyzeUnaryExpression(unaryExpr("!", ident("count")))
		test.expectType(result, TypeError)
		test.expectErrorContaining("requires boolean operand")
	})
}

func TestAnalyzeLogicalAnd(t *testing.T) {
	t.Run("true && false has type bool", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(boolLit("true"), "&&", boolLit("false")))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("true && true has type bool", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(boolLit("true"), "&&", boolLit("true")))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("5 && 3 is type error", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(intLit("5"), "&&", intLit("3")))
		test.expectType(result, TypeError)
		test.expectErrorContaining("requires boolean operands")
	})

	t.Run("true && 5 is type error", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(boolLit("true"), "&&", intLit("5")))
		test.expectType(result, TypeError)
		test.expectErrorContaining("requires boolean operands")
	})

	t.Run("5 && true is type error", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(intLit("5"), "&&", boolLit("true")))
		test.expectType(result, TypeError)
		test.expectErrorContaining("requires boolean operands")
	})
}

func TestAnalyzeLogicalOr(t *testing.T) {
	t.Run("true || false has type bool", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(boolLit("true"), "||", boolLit("false")))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("false || false has type bool", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(boolLit("false"), "||", boolLit("false")))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("5 || 3 is type error", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(intLit("5"), "||", intLit("3")))
		test.expectType(result, TypeError)
		test.expectErrorContaining("requires boolean operands")
	})
}

func TestAnalyzeComparisonReturnsBool(t *testing.T) {
	// This is a breaking change - comparisons now return bool instead of i64
	for _, op := range []string{"==", "!=", "<", ">", "<=", ">="} {
		t.Run(op+" returns bool", func(t *testing.T) {
			test := newTest(t)
			result := test.analyzer.analyzeBinaryExpression(binExpr(intLit("5"), op, intLit("3")))
			test.expectType(result, TypeBoolean)
			test.expectNoErrors()
		})
	}
}

func TestAnalyzeComparisonWithLogical(t *testing.T) {
	t.Run("(5 < 10) && (3 > 1) has type bool", func(t *testing.T) {
		test := newTest(t).withScope()
		// Simulate: (5 < 10) && (3 > 1)
		// First analyze the comparisons
		left := test.analyzer.analyzeBinaryExpression(binExpr(intLit("5"), "<", intLit("10")))
		right := test.analyzer.analyzeBinaryExpression(binExpr(intLit("3"), ">", intLit("1")))

		// Both should be boolean
		test.expectType(left, TypeBoolean)
		test.expectType(right, TypeBoolean)

		// Now create a logical AND of two boolean variables
		test.analyzer.currentScope.declare("a", TypeBoolean, false)
		test.analyzer.currentScope.declare("b", TypeBoolean, false)
		result := test.analyzer.analyzeBinaryExpression(binExpr(ident("a"), "&&", ident("b")))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})
}

func TestAnalyzeBooleanVariableDeclaration(t *testing.T) {
	t.Run("val x = true", func(t *testing.T) {
		test := newTest(t).withScope()
		result := test.analyzer.analyzeVarDeclStatement(varDecl("x", false, boolLit("true")))

		typedVarDecl, ok := result.(*TypedVarDeclStmt)
		if !ok {
			t.Fatal("expected TypedVarDeclStmt")
		}

		test.expectType(typedVarDecl.Initializer, TypeBoolean)
		test.expectNoErrors()

		// Verify variable is in scope with boolean type
		varInfo, found := test.analyzer.currentScope.lookup("x")
		if !found {
			t.Error("variable x not found in scope")
		}
		if !varInfo.Type.Equals(TypeBoolean) {
			t.Errorf("variable has wrong type: expected bool, got %s", varInfo.Type)
		}
	})

	t.Run("val x: bool = false", func(t *testing.T) {
		test := newTest(t).withScope()
		result := test.analyzer.analyzeVarDeclStatement(typedVarDecl("x", "bool", false, boolLit("false")))

		typedVarDecl, ok := result.(*TypedVarDeclStmt)
		if !ok {
			t.Fatal("expected TypedVarDeclStmt")
		}

		test.expectNoErrors()

		varInfo, found := test.analyzer.currentScope.lookup("x")
		if !found {
			t.Error("variable x not found in scope")
		}
		if !varInfo.Type.Equals(TypeBoolean) {
			t.Errorf("variable has wrong type: expected bool, got %s", varInfo.Type)
		}
		_ = typedVarDecl // silence unused warning
	})
}

func TestAnalyzeBooleanArithmeticError(t *testing.T) {
	t.Run("true + false is type error", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(boolLit("true"), "+", boolLit("false")))
		test.expectType(result, TypeError)
		test.expectErrorContaining("requires numeric operands")
	})

	t.Run("true - true is type error", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(boolLit("true"), "-", boolLit("true")))
		test.expectType(result, TypeError)
		test.expectErrorContaining("requires numeric operands")
	})
}

func TestAnalyzeUnknownUnaryOperator(t *testing.T) {
	t.Run("unknown unary operator", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeUnaryExpression(unaryExpr("~", intLit("5")))
		test.expectType(result, TypeError)
		test.expectErrorContaining("unknown operator '~'")
	})
}

func TestAnalyzeGroupingExpression(t *testing.T) {
	t.Run("simple grouping preserves type", func(t *testing.T) {
		test := newTest(t)
		// (42)
		result := test.analyzer.analyzeExpression(groupExpr(intLit("42")))
		test.expectType(result, TypeInteger)
		test.expectNoErrors()
	})

	t.Run("grouped addition returns integer", func(t *testing.T) {
		test := newTest(t)
		// (2 + 3)
		result := test.analyzer.analyzeExpression(groupExpr(binExpr(intLit("2"), "+", intLit("3"))))
		test.expectType(result, TypeInteger)
		test.expectNoErrors()
	})

	t.Run("grouped comparison returns boolean", func(t *testing.T) {
		test := newTest(t)
		// (5 > 3)
		result := test.analyzer.analyzeExpression(groupExpr(binExpr(intLit("5"), ">", intLit("3"))))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("nested grouping preserves type", func(t *testing.T) {
		test := newTest(t)
		// ((42))
		result := test.analyzer.analyzeExpression(groupExpr(groupExpr(intLit("42"))))
		test.expectType(result, TypeInteger)
		test.expectNoErrors()
	})

	t.Run("grouped string literal", func(t *testing.T) {
		test := newTest(t)
		// ("hello")
		result := test.analyzer.analyzeExpression(groupExpr(strLit("hello")))
		test.expectType(result, TypeString)
		test.expectNoErrors()
	})

	t.Run("type error inside grouping propagates", func(t *testing.T) {
		test := newTest(t)
		// ("a" + 5) - type error
		result := test.analyzer.analyzeExpression(groupExpr(binExpr(strLit("a"), "+", intLit("5"))))
		test.expectType(result, TypeError)
		test.expectErrorContaining("requires numeric operands")
	})
}
