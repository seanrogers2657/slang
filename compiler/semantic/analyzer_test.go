package semantic

import (
	"strings"
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
		{"s8 type", "s8", "42", TypeS8},
		{"s16 type", "s16", "1000", TypeS16},
		{"s32 type", "s32", "100000", TypeS32},
		{"s64 type", "s64", "9223372036854775807", TypeS64},
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
		// s8 bounds: -128 to 127
		{"s8 overflow positive", "s8", "128", "out of range for s8"},
		{"s8 overflow large", "s8", "200", "out of range for s8"},
		// s16 bounds: -32768 to 32767
		{"s16 overflow", "s16", "32768", "out of range for s16"},
		// s32 bounds: -2147483648 to 2147483647
		{"s32 overflow", "s32", "2147483648", "out of range for s32"},
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

func TestAnalyzeUntypedIntegerLiteralOverflow(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		errorMsg string
	}{
		// s64 max is 9223372036854775807
		{"s64 overflow by 1", "9223372036854775808", "out of range for s64"},
		{"large overflow", "99999999999999999999999999999", "out of range for s64"},
		{"s64 min underflow by 1", "-9223372036854775809", "out of range for s64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newTest(t).withScope()
			// Untyped variable declaration - should infer s64 and check bounds
			test.analyzer.analyzeVarDeclStatement(varDecl("x", false, intLit(tt.value)))
			test.expectErrorContaining(tt.errorMsg)
		})
	}
}

func TestAnalyzeUntypedIntegerLiteralValid(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"s64 max", "9223372036854775807"},
		{"s64 min", "-9223372036854775808"},
		{"zero", "0"},
		{"small positive", "42"},
		{"small negative", "-100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test := newTest(t).withScope()
			test.analyzer.analyzeVarDeclStatement(varDecl("x", false, intLit(tt.value)))
			test.expectNoErrors()
		})
	}
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
		{"s32 + s64", "s32", "s64", "requires operands of the same type"},
		{"s8 + s16", "s8", "s16", "requires operands of the same type"},
		{"u8 + u16", "u8", "u16", "requires operands of the same type"},
		{"s32 + u32", "s32", "u32", "requires operands of the same type"},
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
		{"s8 + s8", "s8", "10", "20"},
		{"s16 + s16", "s16", "100", "200"},
		{"s32 + s32", "s32", "1000", "2000"},
		{"s64 + s64", "s64", "10000", "20000"},
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
	t.Run("f64 + s64", func(t *testing.T) {
		test := newTest(t).withScope()
		test.analyzer.analyzeVarDeclStatement(typedVarDecl("a", "f64", false, floatLit("1.5")))
		test.analyzer.analyzeVarDeclStatement(typedVarDecl("b", "s64", false, intLit("10")))
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
	// This is a breaking change - comparisons now return bool instead of s64
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

func TestAnalyzeReturnPathAnalysis(t *testing.T) {
	t.Run("void function without return is valid", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil, exprStmt(intLit("42"))),
		)
		errs, _ := test.analyzer.Analyze(prog)
		for _, err := range errs {
			if strings.Contains(err.Message, "does not return") {
				t.Errorf("unexpected return path error: %s", err.Message)
			}
		}
	})

	t.Run("non-void function with return is valid", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil),
			funcDecl("getVal", "s64", nil, returnStmt(intLit("42"))),
		)
		errs, _ := test.analyzer.Analyze(prog)
		for _, err := range errs {
			if strings.Contains(err.Message, "does not return") {
				t.Errorf("unexpected return path error: %s", err.Message)
			}
		}
	})

	t.Run("non-void function without return is error", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil),
			funcDecl("getVal", "s64", nil, exprStmt(intLit("42"))),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "does not return a value on all code paths") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected error about missing return")
		}
	})

	t.Run("if-else both returning is valid", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil),
			funcDecl("getVal", "s64", nil,
				ifStmtWithElse(
					boolLit("true"),
					[]ast.Statement{returnStmtAST(intLit("1"))},
					[]ast.Statement{returnStmtAST(intLit("2"))},
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		for _, err := range errs {
			if strings.Contains(err.Message, "does not return") {
				t.Errorf("unexpected return path error: %s", err.Message)
			}
		}
	})

	t.Run("if without else is error", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil),
			funcDecl("getVal", "s64", nil,
				ifStmtNoElse(
					boolLit("true"),
					[]ast.Statement{returnStmtAST(intLit("1"))},
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "does not return a value on all code paths") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected error about missing return on else branch")
		}
	})

	t.Run("if-else with only one branch returning is error", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil),
			funcDecl("getVal", "s64", nil,
				ifStmtWithElse(
					boolLit("true"),
					[]ast.Statement{returnStmtAST(intLit("1"))},
					[]ast.Statement{exprStmt(intLit("2"))}, // no return
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "does not return a value on all code paths") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected error about missing return in else branch")
		}
	})
}

func TestAnalyzeMaxFunctionParameters(t *testing.T) {
	t.Run("function with 8 parameters is valid", func(t *testing.T) {
		test := newTest(t)
		params := make([]ast.Parameter, 8)
		for i := 0; i < 8; i++ {
			params[i] = param("p"+string(rune('0'+i)), "s64")
		}
		prog := programWithFuncs(
			funcDecl("main", "void", nil),
			funcDecl("eightParams", "void", params),
		)
		errs, _ := test.analyzer.Analyze(prog)
		for _, err := range errs {
			if strings.Contains(err.Message, "parameters") && strings.Contains(err.Message, "maximum") {
				t.Errorf("unexpected parameter limit error: %s", err.Message)
			}
		}
	})

	t.Run("function with 9 parameters is error", func(t *testing.T) {
		test := newTest(t)
		params := make([]ast.Parameter, 9)
		for i := 0; i < 9; i++ {
			params[i] = param("p"+string(rune('0'+i)), "s64")
		}
		prog := programWithFuncs(
			funcDecl("main", "void", nil),
			funcDecl("nineParams", "void", params),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "9 parameters") && strings.Contains(err.Message, "maximum") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected error about too many parameters")
		}
	})

	t.Run("function call with 9 arguments is error", func(t *testing.T) {
		test := newTest(t)
		// Create a function with 9 params (which will also error)
		// and try to call it with 9 args
		params := make([]ast.Parameter, 9)
		for i := 0; i < 9; i++ {
			params[i] = param("p"+string(rune('0'+i)), "s64")
		}
		args := make([]ast.Expression, 9)
		for i := 0; i < 9; i++ {
			args[i] = intLit("1")
		}
		prog := programWithFuncs(
			funcDecl("main", "void", nil, exprStmt(callExpr("nineParams", args...))),
			funcDecl("nineParams", "void", params),
		)
		errs, _ := test.analyzer.Analyze(prog)
		foundCallError := false
		for _, err := range errs {
			if strings.Contains(err.Message, "9 arguments") && strings.Contains(err.Message, "maximum") {
				foundCallError = true
				break
			}
		}
		if !foundCallError {
			t.Error("expected error about too many arguments in call")
		}
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

func TestAnalyzeWhenStatement(t *testing.T) {
	t.Run("when with else is exhaustive", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				whenExpr(
					whenCase(boolLit("true"), exprStmt(callExpr("exit", intLit("0"))), false),
					whenCase(nil, exprStmt(callExpr("exit", intLit("1"))), true),
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		for _, err := range errs {
			if strings.Contains(err.Message, "not exhaustive") {
				t.Errorf("unexpected exhaustiveness error: %s", err.Message)
			}
		}
	})

	t.Run("when with literal true is exhaustive", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				whenExpr(
					whenCase(binExpr(intLit("5"), ">", intLit("10")), exprStmt(callExpr("exit", intLit("100"))), false),
					whenCase(boolLit("true"), exprStmt(callExpr("exit", intLit("0"))), false),
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		for _, err := range errs {
			if strings.Contains(err.Message, "not exhaustive") {
				t.Errorf("unexpected exhaustiveness error: %s", err.Message)
			}
		}
	})

	t.Run("when without else or true is not exhaustive", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				whenExpr(
					whenCase(binExpr(intLit("5"), ">", intLit("10")), exprStmt(callExpr("exit", intLit("100"))), false),
					whenCase(binExpr(intLit("5"), ">", intLit("5")), exprStmt(callExpr("exit", intLit("50"))), false),
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "not exhaustive") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected exhaustiveness error")
		}
	})

	t.Run("when conditions must be boolean", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				whenExpr(
					whenCase(intLit("42"), exprStmt(callExpr("exit", intLit("0"))), false),
					whenCase(nil, exprStmt(callExpr("exit", intLit("1"))), true),
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "when case condition must be boolean") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected boolean condition error")
		}
	})
}

func TestAnalyzeWhenExpression(t *testing.T) {
	t.Run("when expression without exhaustive branch is error", func(t *testing.T) {
		test := newTest(t)
		test.withScope().declare("x", TypeInteger, false)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				varDecl("y", false, whenExpr(
					whenCase(binExpr(ident("x"), ">", intLit("10")), exprStmt(intLit("42")), false),
				)),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "not exhaustive") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected exhaustiveness error for when expression")
		}
	})

	t.Run("when expression branches must have same type", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				varDecl("x", false, whenExpr(
					whenCase(boolLit("true"), exprStmt(intLit("42")), false),
					whenCase(nil, exprStmt(strLit("hello")), true),
				)),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "different types") || strings.Contains(err.Message, "same type") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected type mismatch error")
		}
	})

	t.Run("when expression with consistent types is valid", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				varDecl("x", false, whenExpr(
					whenCase(boolLit("true"), exprStmt(intLit("42")), false),
					whenCase(nil, exprStmt(intLit("0")), true),
				)),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		for _, err := range errs {
			if strings.Contains(err.Message, "different types") || strings.Contains(err.Message, "same type") {
				t.Errorf("unexpected type error: %s", err.Message)
			}
		}
	})
}

func TestAnalyzeWhileStatement(t *testing.T) {
	t.Run("while with boolean condition is valid", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				varDecl("i", true, intLit("0")),
				whileStmt(
					binExpr(ident("i"), "<", intLit("5")),
					assignStmt("i", binExpr(ident("i"), "+", intLit("1"))),
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		for _, err := range errs {
			if strings.Contains(err.Message, "while-loop condition must be boolean") {
				t.Errorf("unexpected condition error: %s", err.Message)
			}
		}
	})

	t.Run("while condition must be boolean", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				whileStmt(
					intLit("42"),
					exprStmt(callExpr("exit", intLit("0"))),
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "while-loop condition must be boolean") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected boolean condition error")
		}
	})

	t.Run("break inside while is valid", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				whileStmt(
					boolLit("true"),
					breakStmt(),
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		for _, err := range errs {
			if strings.Contains(err.Message, "break") && strings.Contains(err.Message, "outside") {
				t.Errorf("unexpected break error: %s", err.Message)
			}
		}
	})

	t.Run("continue inside while is valid", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				varDecl("i", true, intLit("0")),
				whileStmt(
					binExpr(ident("i"), "<", intLit("5")),
					assignStmt("i", binExpr(ident("i"), "+", intLit("1"))),
					continueStmt(),
				),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		for _, err := range errs {
			if strings.Contains(err.Message, "continue") && strings.Contains(err.Message, "outside") {
				t.Errorf("unexpected continue error: %s", err.Message)
			}
		}
	})

	t.Run("break outside while is error", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				breakStmt(),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "break") && strings.Contains(err.Message, "not inside a loop") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'break outside loop' error")
		}
	})

	t.Run("continue outside while is error", func(t *testing.T) {
		test := newTest(t)
		prog := programWithFuncs(
			funcDecl("main", "void", nil,
				continueStmt(),
			),
		)
		errs, _ := test.analyzer.Analyze(prog)
		found := false
		for _, err := range errs {
			if strings.Contains(err.Message, "continue") && strings.Contains(err.Message, "not inside a loop") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'continue outside loop' error")
		}
	})
}

// =============================================================================
// Nullability Tests
// =============================================================================

func TestAnalyzeNullLiteral(t *testing.T) {
	t.Run("null literal has NothingType", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeLiteral(nullLit())
		test.expectType(result, TypeNothing)
		test.expectNoErrors()
	})
}

func TestAnalyzeNullableTypes(t *testing.T) {
	t.Run("null can be assigned to nullable type", func(t *testing.T) {
		test := newTest(t).withScope()
		result := test.analyzer.analyzeVarDeclStatement(typedVarDecl("x", "s64?", false, nullLit()))

		if result == nil {
			t.Fatal("expected result, got nil")
		}
		test.expectNoErrors()
	})

	t.Run("non-null value can be assigned to nullable type", func(t *testing.T) {
		test := newTest(t).withScope()
		result := test.analyzer.analyzeVarDeclStatement(typedVarDecl("x", "s64?", false, intLit("42")))

		if result == nil {
			t.Fatal("expected result, got nil")
		}
		test.expectNoErrors()
	})

	t.Run("null cannot be assigned to non-nullable type", func(t *testing.T) {
		test := newTest(t).withScope()
		test.analyzer.analyzeVarDeclStatement(typedVarDecl("x", "s64", false, nullLit()))

		test.expectErrorContaining("cannot assign null to non-nullable")
	})

	t.Run("bare null without type annotation is error", func(t *testing.T) {
		test := newTest(t).withScope()
		test.analyzer.analyzeVarDeclStatement(varDecl("x", false, nullLit()))

		test.expectErrorContaining("cannot infer type from null")
	})
}

func TestAnalyzeNullComparison(t *testing.T) {
	t.Run("nullable equals null returns bool", func(t *testing.T) {
		test := newTest(t).withScope().declare("x", NullableType{InnerType: TypeS64}, false)
		result := test.analyzer.analyzeBinaryExpression(binExpr(ident("x"), "==", nullLit()))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("nullable not equals null returns bool", func(t *testing.T) {
		test := newTest(t).withScope().declare("x", NullableType{InnerType: TypeS64}, false)
		result := test.analyzer.analyzeBinaryExpression(binExpr(ident("x"), "!=", nullLit()))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("null equals null returns bool", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeBinaryExpression(binExpr(nullLit(), "==", nullLit()))
		test.expectType(result, TypeBoolean)
		test.expectNoErrors()
	})

	t.Run("non-nullable compared to null is error", func(t *testing.T) {
		test := newTest(t).withScope().declare("x", TypeS64, false)
		test.analyzer.analyzeBinaryExpression(binExpr(ident("x"), "==", nullLit()))
		test.expectErrorContaining("cannot compare non-nullable")
	})
}

func TestNullableTypeHelpers(t *testing.T) {
	t.Run("IsNullable returns true for NullableType", func(t *testing.T) {
		nullableInt := NullableType{InnerType: TypeS64}
		if !IsNullable(nullableInt) {
			t.Error("expected IsNullable to return true")
		}
	})

	t.Run("IsNullable returns false for non-nullable type", func(t *testing.T) {
		if IsNullable(TypeS64) {
			t.Error("expected IsNullable to return false")
		}
	})

	t.Run("MakeNullable wraps type", func(t *testing.T) {
		result := MakeNullable(TypeS64)
		nullable, ok := result.(NullableType)
		if !ok {
			t.Fatal("expected NullableType")
		}
		if !nullable.InnerType.Equals(TypeS64) {
			t.Error("expected inner type to be s64")
		}
	})

	t.Run("MakeNullable does not double-wrap", func(t *testing.T) {
		nullable := NullableType{InnerType: TypeS64}
		result := MakeNullable(nullable)
		if result != nullable {
			t.Error("expected MakeNullable to return same type for already nullable")
		}
	})

	t.Run("UnwrapNullable extracts inner type", func(t *testing.T) {
		nullable := NullableType{InnerType: TypeS64}
		inner, ok := UnwrapNullable(nullable)
		if !ok {
			t.Error("expected ok to be true")
		}
		if !inner.Equals(TypeS64) {
			t.Error("expected inner type to be s64")
		}
	})

	t.Run("UnwrapNullable returns false for non-nullable", func(t *testing.T) {
		_, ok := UnwrapNullable(TypeS64)
		if ok {
			t.Error("expected ok to be false")
		}
	})

	t.Run("NullableSize returns 16 for primitives", func(t *testing.T) {
		size := NullableSize(TypeS64)
		if size != 16 {
			t.Errorf("expected 16, got %d", size)
		}
	})

	t.Run("NullableSize returns 8 for reference types", func(t *testing.T) {
		size := NullableSize(TypeString)
		if size != 8 {
			t.Errorf("expected 8, got %d", size)
		}
	})
}

// -----------------------------------------------------------------------------
// Owned Pointer Type Tests
// -----------------------------------------------------------------------------

func TestOwnedPointerType(t *testing.T) {
	t.Run("OwnedPointerType String() returns correct format", func(t *testing.T) {
		ownType := OwnedPointerType{ElementType: TypeS64}
		if ownType.String() != "*s64" {
			t.Errorf("expected *s64, got %s", ownType.String())
		}
	})

	t.Run("OwnedPointerType Equals works correctly", func(t *testing.T) {
		own1 := OwnedPointerType{ElementType: TypeS64}
		own2 := OwnedPointerType{ElementType: TypeS64}
		own3 := OwnedPointerType{ElementType: TypeS32}

		if !own1.Equals(own2) {
			t.Error("expected Own<s64> to equal Own<s64>")
		}
		if own1.Equals(own3) {
			t.Error("expected Own<s64> to not equal Own<s32>")
		}
		if own1.Equals(TypeS64) {
			t.Error("expected Own<s64> to not equal s64")
		}
	})

	t.Run("OwnedPointerType is not copyable", func(t *testing.T) {
		ownType := OwnedPointerType{ElementType: TypeS64}
		if ownType.IsCopyable() {
			t.Error("expected owned pointer to not be copyable")
		}
	})

	t.Run("IsOwnedPointer returns true for OwnedPointerType", func(t *testing.T) {
		ownType := OwnedPointerType{ElementType: TypeS64}
		if !IsOwnedPointer(ownType) {
			t.Error("expected IsOwnedPointer to return true")
		}
	})

	t.Run("IsOwnedPointer returns false for non-pointer types", func(t *testing.T) {
		if IsOwnedPointer(TypeS64) {
			t.Error("expected IsOwnedPointer to return false for s64")
		}
		if IsOwnedPointer(TypeString) {
			t.Error("expected IsOwnedPointer to return false for string")
		}
	})

	t.Run("UnwrapOwnedPointer extracts element type", func(t *testing.T) {
		ownType := OwnedPointerType{ElementType: TypeS64}
		inner, ok := UnwrapOwnedPointer(ownType)
		if !ok {
			t.Error("expected ok to be true")
		}
		if !inner.Equals(TypeS64) {
			t.Errorf("expected s64, got %s", inner.String())
		}
	})

	t.Run("UnwrapOwnedPointer returns false for non-pointer", func(t *testing.T) {
		_, ok := UnwrapOwnedPointer(TypeS64)
		if ok {
			t.Error("expected ok to be false")
		}
	})

	t.Run("TypeByteSize for OwnedPointerType is 8", func(t *testing.T) {
		ownType := OwnedPointerType{ElementType: TypeS64}
		if TypeByteSize(ownType) != 8 {
			t.Errorf("expected 8 bytes, got %d", TypeByteSize(ownType))
		}
	})
}

func TestNewExprTypeChecking(t *testing.T) {
	t.Run("new integer returns Own<s64>", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeExpression(newExpr(intLit("42")))
		expectedType := OwnedPointerType{ElementType: TypeS64}
		test.expectType(result, expectedType)
		test.expectNoErrors()
	})

	t.Run("new string returns Own<string>", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.analyzeExpression(newExpr(strLit("hello")))
		expectedType := OwnedPointerType{ElementType: TypeString}
		test.expectType(result, expectedType)
		test.expectNoErrors()
	})

	t.Run("new struct literal returns Own<StructType>", func(t *testing.T) {
		test := newTest(t).withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
			StructFieldInfo{Name: "y", Type: TypeS64, Mutable: true, Index: 1},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		result := test.analyzer.analyzeExpression(newExpr(structLiteral("Point", intLit("10"), intLit("20"))))
		expectedType := OwnedPointerType{ElementType: pointType}
		test.expectType(result, expectedType)
		test.expectNoErrors()
	})
}

func TestFieldAccessThroughOwnedPointer(t *testing.T) {
	t.Run("field access on Own<Struct> auto-dereferences", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
			StructFieldInfo{Name: "y", Type: TypeS64, Mutable: true, Index: 1},
		)
		// Declare p as Own<Point>
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		// Access p.x should work and return s64
		result := test.analyzer.analyzeExpression(fieldAccessExpr(ident("p"), "x"))
		test.expectType(result, TypeS64)
		test.expectNoErrors()
	})

	t.Run("field access on Own<Struct> preserves field mutability", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
			StructFieldInfo{Name: "y", Type: TypeS64, Mutable: true, Index: 1},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		result := test.analyzer.analyzeExpression(fieldAccessExpr(ident("p"), "y"))
		typedResult, ok := result.(*TypedFieldAccessExpr)
		if !ok {
			t.Fatal("expected TypedFieldAccessExpr")
		}
		if !typedResult.Mutable {
			t.Error("expected field y to be mutable")
		}
	})

	t.Run("invalid field access on Own<Struct> is error", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
			StructFieldInfo{Name: "y", Type: TypeS64, Mutable: true, Index: 1},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		test.analyzer.analyzeExpression(fieldAccessExpr(ident("p"), "z"))
		test.expectErrorContaining("has no field 'z'")
	})

	t.Run("field access on Own<non-struct> is error", func(t *testing.T) {
		test := newTest(t).withScope()
		ownIntType := OwnedPointerType{ElementType: TypeS64}
		test.declare("p", ownIntType, false)

		test.analyzer.analyzeExpression(fieldAccessExpr(ident("p"), "x"))
		test.expectErrorContaining("cannot access field")
	})
}

func TestNullableOwnedPointerValidation(t *testing.T) {
	t.Run("*Point? is valid nullable owned pointer", func(t *testing.T) {
		test := newTest(t).withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		resolved := test.analyzer.resolveTypeName("*Point?", pos(1, 1))
		nullableOwn, ok := resolved.(NullableType)
		if !ok {
			t.Fatalf("expected NullableType, got %T", resolved)
		}
		ownedInner, ok := nullableOwn.InnerType.(OwnedPointerType)
		if !ok {
			t.Fatalf("expected OwnedPointerType inside, got %T", nullableOwn.InnerType)
		}
		if ownedInner.ElementType.String() != "Point" {
			t.Errorf("expected Point, got %s", ownedInner.ElementType.String())
		}
		test.expectNoErrors()
	})

	t.Run("*Point? with inner nullable is invalid - produces error", func(t *testing.T) {
		test := newTest(t).withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		test.analyzer.resolveTypeName("*Point??", pos(1, 1))
		test.expectErrorContaining("nested nullable types are not allowed")
	})

	t.Run("*s64? with inner nullable is invalid - produces error", func(t *testing.T) {
		test := newTest(t)
		test.analyzer.resolveTypeName("*s64??", pos(1, 1))
		test.expectErrorContaining("nested nullable types are not allowed")
	})
}

func TestOwnedPointerCopy(t *testing.T) {
	t.Run("p.copy() on Own<T> returns Own<T>", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		result := test.analyzer.analyzeExpression(methodCallExpr(ident("p"), "copy"))
		test.expectType(result, ownPointType)
		test.expectNoErrors()
	})

	t.Run("p.copy() with arguments is error", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		test.analyzer.analyzeExpression(methodCallExpr(ident("p"), "copy", intLit("1")))
		test.expectErrorContaining("copy() takes no arguments")
	})

	t.Run("unknown method on Own<T> is error", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		test.analyzer.analyzeExpression(methodCallExpr(ident("p"), "unknown"))
		test.expectErrorContaining("unknown method 'unknown'")
	})
}

// -----------------------------------------------------------------------------
// Ref Pointer Type Tests
// -----------------------------------------------------------------------------

func TestRefPointerType(t *testing.T) {
	t.Run("RefPointerType String() returns correct format", func(t *testing.T) {
		refType := RefPointerType{ElementType: TypeS64}
		if refType.String() != "&s64" {
			t.Errorf("expected &s64, got %s", refType.String())
		}
	})

	t.Run("MutRefPointerType String() returns correct format", func(t *testing.T) {
		refType := MutRefPointerType{ElementType: TypeS64}
		if refType.String() != "&&s64" {
			t.Errorf("expected &&s64, got %s", refType.String())
		}
	})

	t.Run("RefPointerType Equals works correctly", func(t *testing.T) {
		ref1 := RefPointerType{ElementType: TypeS64}
		ref2 := RefPointerType{ElementType: TypeS64}
		ref3 := RefPointerType{ElementType: TypeS32}
		mutRef := MutRefPointerType{ElementType: TypeS64}

		if !ref1.Equals(ref2) {
			t.Error("expected Ref<s64> to equal Ref<s64>")
		}
		if ref1.Equals(ref3) {
			t.Error("expected Ref<s64> to not equal Ref<s32>")
		}
		if ref1.Equals(mutRef) {
			t.Error("expected Ref<s64> to not equal MutRef<s64>")
		}
		if ref1.Equals(TypeS64) {
			t.Error("expected Ref<s64> to not equal s64")
		}
	})

	t.Run("RefPointerType is copyable", func(t *testing.T) {
		refType := RefPointerType{ElementType: TypeS64}
		if !refType.IsCopyable() {
			t.Error("expected reference pointer to be copyable")
		}
	})

	t.Run("MutRefPointerType is copyable", func(t *testing.T) {
		refType := MutRefPointerType{ElementType: TypeS64}
		if !refType.IsCopyable() {
			t.Error("expected mutable reference pointer to be copyable")
		}
	})

	t.Run("IsRefPointer returns true for RefPointerType", func(t *testing.T) {
		refType := RefPointerType{ElementType: TypeS64}
		if !IsRefPointer(refType) {
			t.Error("expected IsRefPointer to return true")
		}
	})

	t.Run("IsMutRefPointer returns true for MutRefPointerType", func(t *testing.T) {
		refType := MutRefPointerType{ElementType: TypeS64}
		if !IsMutRefPointer(refType) {
			t.Error("expected IsMutRefPointer to return true")
		}
	})

	t.Run("IsAnyRefPointer returns true for both ref types", func(t *testing.T) {
		if !IsAnyRefPointer(RefPointerType{ElementType: TypeS64}) {
			t.Error("expected IsAnyRefPointer to return true for Ref")
		}
		if !IsAnyRefPointer(MutRefPointerType{ElementType: TypeS64}) {
			t.Error("expected IsAnyRefPointer to return true for MutRef")
		}
	})

	t.Run("IsRefPointer returns false for non-pointer types", func(t *testing.T) {
		if IsRefPointer(TypeS64) {
			t.Error("expected IsRefPointer to return false for s64")
		}
		if IsRefPointer(OwnedPointerType{ElementType: TypeS64}) {
			t.Error("expected IsRefPointer to return false for Own<s64>")
		}
		if IsRefPointer(MutRefPointerType{ElementType: TypeS64}) {
			t.Error("expected IsRefPointer to return false for MutRef<s64>")
		}
	})

	t.Run("UnwrapRefPointer extracts element type", func(t *testing.T) {
		refType := RefPointerType{ElementType: TypeS64}
		inner, ok := UnwrapRefPointer(refType)
		if !ok {
			t.Error("expected ok to be true")
		}
		if !inner.Equals(TypeS64) {
			t.Errorf("expected s64, got %s", inner.String())
		}
	})

	t.Run("UnwrapMutRefPointer extracts element type", func(t *testing.T) {
		refType := MutRefPointerType{ElementType: TypeS64}
		inner, ok := UnwrapMutRefPointer(refType)
		if !ok {
			t.Error("expected ok to be true")
		}
		if !inner.Equals(TypeS64) {
			t.Errorf("expected s64, got %s", inner.String())
		}
	})

	t.Run("UnwrapRefPointer returns false for non-ref", func(t *testing.T) {
		_, ok := UnwrapRefPointer(TypeS64)
		if ok {
			t.Error("expected ok to be false")
		}
	})

	t.Run("TypeByteSize for RefPointerType is 8", func(t *testing.T) {
		refType := RefPointerType{ElementType: TypeS64}
		if TypeByteSize(refType) != 8 {
			t.Errorf("expected 8 bytes, got %d", TypeByteSize(refType))
		}
	})
}

func TestRefTypeResolution(t *testing.T) {
	t.Run("&s64 resolves correctly", func(t *testing.T) {
		test := newTest(t)
		result := test.analyzer.resolveTypeName("&s64", pos(1, 1))
		expected := RefPointerType{ElementType: TypeS64}
		if !result.Equals(expected) {
			t.Errorf("expected %s, got %s", expected.String(), result.String())
		}
		test.expectNoErrors()
	})

	t.Run("&Point resolves correctly", func(t *testing.T) {
		test := newTest(t).withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		result := test.analyzer.resolveTypeName("&Point", pos(1, 1))
		expected := RefPointerType{ElementType: test.analyzer.TypeRegistry.AllStructs()["Point"]}
		if !result.Equals(expected) {
			t.Errorf("expected %s, got %s", expected.String(), result.String())
		}
		test.expectNoErrors()
	})

	t.Run("&s64?? is invalid - produces error", func(t *testing.T) {
		test := newTest(t)
		test.analyzer.resolveTypeName("&s64??", pos(1, 1))
		test.expectErrorContaining("nested nullable types are not allowed")
	})
}

func TestRefUsageRestrictions(t *testing.T) {
	t.Run("&T allowed as function parameter", func(t *testing.T) {
		test := newTest(t).withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		fn := funcDecl("readPoint", "void", []ast.Parameter{param("p", "&Point")})
		mainFn := funcDecl("main", "void", []ast.Parameter{})
		test.analyzer.Analyze(programWithDecls(fn, mainFn))
		test.expectNoErrors()
	})

	t.Run("&&T creates mutable reference parameter", func(t *testing.T) {
		test := newTest(t).withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		fn := funcDecl("mutatePoint", "void", []ast.Parameter{param("p", "&&Point")})
		mainFn := funcDecl("main", "void", []ast.Parameter{})
		test.analyzer.Analyze(programWithDecls(fn, mainFn))
		test.expectNoErrors()

		// Check that the parameter type is a MutRefPointerType
		fnInfo := test.analyzer.functions["mutatePoint"]
		_, ok := fnInfo.ParamTypes[0].(MutRefPointerType)
		if !ok {
			t.Error("expected parameter to be MutRefPointerType")
		}
	})

	t.Run("var &T is error - use &&T instead", func(t *testing.T) {
		test := newTest(t).withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		fn := funcDecl("badFunc", "void", []ast.Parameter{varParam("p", "&Point")})
		test.analyzer.Analyze(programWithDecls(fn))
		test.expectErrorContaining("use &&T")
	})

	t.Run("var on non-Ref parameter is error", func(t *testing.T) {
		test := newTest(t)
		fn := funcDecl("badFunc", "void", []ast.Parameter{varParam("x", "s64")})
		test.analyzer.Analyze(programWithDecls(fn))
		test.expectErrorContaining("'var' modifier is not supported on parameters")
	})

	t.Run("&T as return type is error", func(t *testing.T) {
		test := newTest(t).withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		fn := funcDecl("badFunc", "&Point", []ast.Parameter{})
		test.analyzer.Analyze(programWithDecls(fn))
		test.expectErrorContaining("references cannot be used as return types")
	})

	t.Run("&T as struct field is error", func(t *testing.T) {
		test := newTest(t).withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		badStruct := structDecl("Container", structField("ptr", "&Point", false))
		test.analyzer.Analyze(programWithDecls(badStruct))
		test.expectErrorContaining("&T cannot be used as a struct field type")
	})

	t.Run("&T as local variable is error", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		stmt := typedVarDecl("p", "&Point", false, intLit("0"))
		test.analyzer.analyzeVarDeclStatement(stmt)
		test.expectErrorContaining("&T cannot be stored in local variables")
	})
}

func TestImplicitOwnToRefConversion(t *testing.T) {
	t.Run("Own<T> auto-borrows to Ref<T> parameter", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]

		// Register a function that takes Ref<Point>
		test.analyzer.functions["readPoint"] = FunctionInfo{
			ParamTypes: []Type{RefPointerType{ElementType: pointType}},
			ReturnType: TypeVoid,
		}

		// Declare p as Own<Point>
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		// Call readPoint(p) - should auto-borrow
		result := test.analyzer.analyzeExpression(callExpr("readPoint", ident("p")))
		test.expectNoErrors()
		if _, isErr := result.GetType().(VoidType); !isErr {
			t.Errorf("expected void return type, got %s", result.GetType().String())
		}
	})

	t.Run("Own<T> auto-borrows to var Ref<T> only if source is mutable", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]

		// Register a function that takes var Ref<Point>
		test.analyzer.functions["mutatePoint"] = FunctionInfo{
			ParamTypes: []Type{MutRefPointerType{ElementType: pointType}},
			ReturnType: TypeVoid,
		}

		// Declare p as var Own<Point> (mutable binding)
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, true) // mutable = true

		// Call mutatePoint(p) - should work because p is mutable
		test.analyzer.analyzeExpression(callExpr("mutatePoint", ident("p")))
		test.expectNoErrors()
	})

	t.Run("val binding can borrow as mutable ref", func(t *testing.T) {
		// With the MutRef refactor, val only controls reassignability, not mutability
		// So val binding CAN be borrowed as MutRef<T>
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]

		// Register a function that takes MutRef<Point>
		test.analyzer.functions["mutatePoint"] = FunctionInfo{
			ParamTypes: []Type{MutRefPointerType{ElementType: pointType}},
			ReturnType: TypeVoid,
		}

		// Declare p as val Own<Point>
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false) // val binding

		// Call mutatePoint(p) - should succeed (val only controls reassignment)
		test.analyzer.analyzeExpression(callExpr("mutatePoint", ident("p")))
		test.expectNoErrors()
	})
}

func TestRefFieldAccess(t *testing.T) {
	t.Run("field access through Ref<T> works", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		refPointType := RefPointerType{ElementType: pointType}
		test.declare("p", refPointType, false)

		result := test.analyzer.analyzeExpression(fieldAccessExpr(ident("p"), "x"))
		test.expectNoErrors()
		test.expectType(result, TypeS64)
	})

	t.Run("Own<T> field through Ref becomes Ref<T>", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Create Container with Own<Point> field
		containerType := StructType{
			Name:   "Container",
			Fields: []StructFieldInfo{{Name: "data", Type: ownPointType, Mutable: false, Index: 0}},
		}
		test.analyzer.TypeRegistry.RegisterStruct("Container", containerType)

		// Declare c as Ref<Container>
		refContainerType := RefPointerType{ElementType: containerType}
		test.declare("c", refContainerType, false)

		// Access c.data - Own<Point> through Ref<Container> should give Ref<Point>
		result := test.analyzer.analyzeExpression(fieldAccessExpr(ident("c"), "data"))
		test.expectNoErrors()

		expectedType := RefPointerType{ElementType: pointType}
		if !result.GetType().Equals(expectedType) {
			t.Errorf("expected %s, got %s", expectedType.String(), result.GetType().String())
		}
	})

	t.Run("assignment through immutable Ref is error", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		refPointType := RefPointerType{ElementType: pointType}
		test.declare("p", refPointType, false)

		// Try to assign through immutable ref
		stmt := &ast.FieldAssignStmt{
			Object:   ident("p"),
			Dot:      pos(1, 2),
			Field:    "x",
			FieldPos: pos(1, 3),
			Equals:   pos(1, 5),
			Value:    intLit("10"),
		}
		test.analyzer.analyzeFieldAssignStatement(stmt)
		test.expectErrorContaining("cannot assign through immutable reference")
	})

	t.Run("assignment through mutable Ref works", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		refPointType := MutRefPointerType{ElementType: pointType}
		test.declare("p", refPointType, false)

		// Assign through mutable ref to mutable field
		stmt := &ast.FieldAssignStmt{
			Object:   ident("p"),
			Dot:      pos(1, 2),
			Field:    "x",
			FieldPos: pos(1, 3),
			Equals:   pos(1, 5),
			Value:    intLit("10"),
		}
		test.analyzer.analyzeFieldAssignStatement(stmt)
		test.expectNoErrors()
	})
}

// -----------------------------------------------------------------------------
// Ownership Tracking Tests
// -----------------------------------------------------------------------------

func TestOwnershipStateEnum(t *testing.T) {
	t.Run("ownership states have correct string representations", func(t *testing.T) {
		if StateOwned.String() != "owned" {
			t.Errorf("expected 'owned', got %q", StateOwned.String())
		}
		if StateMoved.String() != "moved" {
			t.Errorf("expected 'moved', got %q", StateMoved.String())
		}
		if StateBorrowed.String() != "borrowed" {
			t.Errorf("expected 'borrowed', got %q", StateBorrowed.String())
		}
	})
}

func TestIsCopyable(t *testing.T) {
	t.Run("primitives are copyable", func(t *testing.T) {
		copyableTypes := []Type{TypeS8, TypeS16, TypeS32, TypeS64, TypeS128,
			TypeU8, TypeU16, TypeU32, TypeU64, TypeU128,
			TypeFloat32, TypeFloat64, TypeBoolean, TypeString}
		for _, typ := range copyableTypes {
			if !IsCopyable(typ) {
				t.Errorf("expected %s to be copyable", typ)
			}
		}
	})

	t.Run("Own<T> is not copyable", func(t *testing.T) {
		ownType := OwnedPointerType{ElementType: TypeS64}
		if IsCopyable(ownType) {
			t.Errorf("expected Own<s64> to not be copyable")
		}
	})

	t.Run("Ref<T> is copyable", func(t *testing.T) {
		refType := RefPointerType{ElementType: TypeS64}
		if !IsCopyable(refType) {
			t.Errorf("expected Ref<s64> to be copyable")
		}
	})

	t.Run("struct with all copyable fields is copyable", func(t *testing.T) {
		structType := StructType{
			Name: "Point",
			Fields: []StructFieldInfo{
				{Name: "x", Type: TypeS64},
				{Name: "y", Type: TypeS64},
			},
		}
		if !IsCopyable(structType) {
			t.Errorf("expected struct with copyable fields to be copyable")
		}
	})

	t.Run("struct containing Own<T> is not copyable", func(t *testing.T) {
		structType := StructType{
			Name: "Container",
			Fields: []StructFieldInfo{
				{Name: "data", Type: OwnedPointerType{ElementType: TypeS64}},
			},
		}
		if IsCopyable(structType) {
			t.Errorf("expected struct containing Own<T> to not be copyable")
		}
	})

	t.Run("nullable Own<T> is not copyable", func(t *testing.T) {
		nullableOwn := NullableType{InnerType: OwnedPointerType{ElementType: TypeS64}}
		if IsCopyable(nullableOwn) {
			t.Errorf("expected Own<s64>? to not be copyable")
		}
	})

	t.Run("nullable primitive is copyable", func(t *testing.T) {
		nullableInt := NullableType{InnerType: TypeS64}
		if !IsCopyable(nullableInt) {
			t.Errorf("expected s64? to be copyable")
		}
	})
}

func TestContainsOwnedPointer(t *testing.T) {
	t.Run("Own<T> contains owned pointer", func(t *testing.T) {
		if !ContainsOwnedPointer(OwnedPointerType{ElementType: TypeS64}) {
			t.Error("expected Own<s64> to contain owned pointer")
		}
	})

	t.Run("primitive does not contain owned pointer", func(t *testing.T) {
		if ContainsOwnedPointer(TypeS64) {
			t.Error("expected s64 to not contain owned pointer")
		}
	})

	t.Run("struct with Own<T> field contains owned pointer", func(t *testing.T) {
		structType := StructType{
			Name: "Container",
			Fields: []StructFieldInfo{
				{Name: "data", Type: OwnedPointerType{ElementType: TypeS64}},
			},
		}
		if !ContainsOwnedPointer(structType) {
			t.Error("expected struct with Own<T> to contain owned pointer")
		}
	})

	t.Run("nested nullable Own<T> contains owned pointer", func(t *testing.T) {
		nullableOwn := NullableType{InnerType: OwnedPointerType{ElementType: TypeS64}}
		if !ContainsOwnedPointer(nullableOwn) {
			t.Error("expected Own<s64>? to contain owned pointer")
		}
	})
}

func TestOwnershipScope(t *testing.T) {
	t.Run("declare adds variable as owned", func(t *testing.T) {
		scope := newOwnershipScope(nil)
		scope.declare("p", OwnedPointerType{ElementType: TypeS64})

		info, found := scope.lookup("p")
		if !found {
			t.Fatal("expected to find 'p' in ownership scope")
		}
		if info.State != StateOwned {
			t.Errorf("expected state 'owned', got %q", info.State)
		}
	})

	t.Run("markMoved changes state to moved", func(t *testing.T) {
		scope := newOwnershipScope(nil)
		scope.declare("p", OwnedPointerType{ElementType: TypeS64})

		scope.markMoved("p", "q", pos(1, 1))

		info, _ := scope.lookup("p")
		if info.State != StateMoved {
			t.Errorf("expected state 'moved', got %q", info.State)
		}
		if info.MoveInfo.MovedTo != "q" {
			t.Errorf("expected MovedTo 'q', got %q", info.MoveInfo.MovedTo)
		}
	})

	t.Run("lookup finds variable in parent scope", func(t *testing.T) {
		parent := newOwnershipScope(nil)
		parent.declare("p", OwnedPointerType{ElementType: TypeS64})

		child := newOwnershipScope(parent)
		info, found := child.lookup("p")
		if !found {
			t.Fatal("expected to find 'p' in parent scope")
		}
		if info.State != StateOwned {
			t.Errorf("expected state 'owned', got %q", info.State)
		}
	})

	t.Run("markMoved updates variable in parent scope", func(t *testing.T) {
		parent := newOwnershipScope(nil)
		parent.declare("p", OwnedPointerType{ElementType: TypeS64})

		child := newOwnershipScope(parent)
		child.markMoved("p", "q", pos(1, 1))

		// Check in parent
		info, _ := parent.lookup("p")
		if info.State != StateMoved {
			t.Errorf("expected state 'moved' in parent, got %q", info.State)
		}
	})
}

func TestUseAfterMove(t *testing.T) {
	t.Run("use of moved variable produces error", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		// Simulate move: mark p as moved
		test.analyzer.ownershipScope.markMoved("p", "q", pos(1, 1))

		// Now access p - should produce error
		test.analyzer.analyzeExpression(ident("p"))
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("use of non-moved variable is OK", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		// Access p without moving - should be OK
		test.analyzer.analyzeExpression(ident("p"))
		test.expectNoErrors()
	})
}

func TestMoveOnAssignment(t *testing.T) {
	t.Run("assignment of *T variable moves it", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		// Declare q = p - this should move p
		stmt := typedVarDecl("q", "*Point", false, ident("p"))
		test.analyzer.analyzeVarDeclStatement(stmt)

		// p should now be marked as moved
		info, found := test.analyzer.ownershipScope.lookup("p")
		if !found {
			t.Fatal("expected to find 'p' in ownership scope")
		}
		if info.State != StateMoved {
			t.Errorf("expected 'p' to be moved, got %q", info.State)
		}
	})

	t.Run("use after move assignment produces error", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.declare("p", ownPointType, false)

		// Declare q = p - this moves p
		stmt := typedVarDecl("q", "*Point", false, ident("p"))
		test.analyzer.analyzeVarDeclStatement(stmt)
		test.expectNoErrors() // Move itself is fine

		// Now use p - should error
		test.analyzer.analyzeExpression(ident("p"))
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("copyable types are not moved on assignment", func(t *testing.T) {
		test := newTest(t).withScope()
		test.declare("x", TypeS64, false)

		// Declare y = x - this should NOT move x (s64 is copyable)
		stmt := typedVarDecl("y", "s64", false, ident("x"))
		test.analyzer.analyzeVarDeclStatement(stmt)

		// x should still be usable
		test.analyzer.analyzeExpression(ident("x"))
		test.expectNoErrors()
	})
}

func TestBindingMutability(t *testing.T) {
	t.Run("val binding Own<T> can have var fields mutated", func(t *testing.T) {
		// With the MutRef refactor, val only controls reassignability
		// So val binding CAN have var fields mutated
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
			StructFieldInfo{Name: "y", Type: TypeS64, Mutable: true, Index: 1},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		// Declare as val
		test.declare("p", ownPointType, false)

		// p.y = 20 should succeed because y is a var field
		// val only prevents reassigning p itself, not mutating through it
		stmt := fieldAssignStmt(ident("p"), "y", intLit("20"))
		test.analyzer.analyzeFieldAssignStatement(stmt)
		test.expectNoErrors()
	})

	t.Run("var binding Own<T> can have var fields mutated", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
			StructFieldInfo{Name: "y", Type: TypeS64, Mutable: true, Index: 1},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		// Declare as var (mutable=true)
		test.declare("p", ownPointType, true)

		// p.y = 20 should work because p is var and y is var field
		stmt := fieldAssignStmt(ident("p"), "y", intLit("20"))
		test.analyzer.analyzeFieldAssignStatement(stmt)
		test.expectNoErrors()
	})

	t.Run("var binding Own<T> cannot have val fields mutated", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
			StructFieldInfo{Name: "y", Type: TypeS64, Mutable: true, Index: 1},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		// Declare as var (mutable=true)
		test.declare("p", ownPointType, true)

		// p.x = 20 should fail because x is val field
		stmt := fieldAssignStmt(ident("p"), "x", intLit("20"))
		test.analyzer.analyzeFieldAssignStatement(stmt)
		test.expectErrorContaining("cannot assign to immutable field")
	})

	t.Run("nested field access through val binding can mutate var fields", func(t *testing.T) {
		// With the MutRef refactor, val only controls reassignability
		// So nested var fields CAN be mutated through a val binding
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
			StructFieldInfo{Name: "y", Type: TypeS64, Mutable: true, Index: 1},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		rectType := StructType{
			Name: "Rect",
			Fields: []StructFieldInfo{
				{Name: "topLeft", Type: OwnedPointerType{ElementType: pointType}, Mutable: true, Index: 0},
			},
		}
		test.analyzer.TypeRegistry.RegisterStruct("Rect", rectType)
		ownRectType := OwnedPointerType{ElementType: rectType}
		// Declare as val
		test.declare("r", ownRectType, false)

		// r.topLeft.y = 20 should succeed because y is a var field
		// val on r only prevents reassigning r itself
		stmt := fieldAssignStmt(fieldAccessExpr(ident("r"), "topLeft"), "y", intLit("20"))
		test.analyzer.analyzeFieldAssignStatement(stmt)
		test.expectNoErrors()
	})
}

func TestBorrowExclusivity(t *testing.T) {
	t.Run("multiple immutable borrows of same variable is OK", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: false, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		refPointType := RefPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// Register function: readBoth(a: Ref<Point>, b: Ref<Point>)
		test.analyzer.functions["readBoth"] = FunctionInfo{
			ParamTypes: []Type{refPointType, refPointType},
			ReturnType: TypeVoid,
		}

		// Call readBoth(p, p) - multiple immutable borrows should be OK
		call := callExpr("readBoth", ident("p"), ident("p"))
		test.analyzer.analyzeCallExpr(call)
		test.expectNoErrors()
	})

	t.Run("multiple mutable borrows of same variable is ERROR", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		refPointTypeMut := MutRefPointerType{ElementType: pointType}

		// Declare p as var Own<Point> (mutable so we can borrow mutably)
		test.declare("p", ownPointType, true)

		// Register function: mutateBoth(var a: Ref<Point>, var b: Ref<Point>)
		test.analyzer.functions["mutateBoth"] = FunctionInfo{
			ParamTypes: []Type{refPointTypeMut, refPointTypeMut},
			ReturnType: TypeVoid,
		}

		// Call mutateBoth(p, p) - multiple mutable borrows should ERROR
		call := callExpr("mutateBoth", ident("p"), ident("p"))
		test.analyzer.analyzeCallExpr(call)
		test.expectErrorContaining("cannot borrow 'p' as mutable more than once")
	})

	t.Run("mixed mutable and immutable borrow is ERROR", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		refPointType := RefPointerType{ElementType: pointType}
		refPointTypeMut := MutRefPointerType{ElementType: pointType}

		// Declare p as var Own<Point>
		test.declare("p", ownPointType, true)

		// Register function: readAndMutate(a: Ref<Point>, var b: Ref<Point>)
		test.analyzer.functions["readAndMutate"] = FunctionInfo{
			ParamTypes: []Type{refPointType, refPointTypeMut},
			ReturnType: TypeVoid,
		}

		// Call readAndMutate(p, p) - mixed mutable/immutable should ERROR
		call := callExpr("readAndMutate", ident("p"), ident("p"))
		test.analyzer.analyzeCallExpr(call)
		test.expectErrorContaining("cannot borrow 'p' as both mutable and immutable")
	})

	t.Run("borrowing different variables is OK", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		refPointTypeMut := MutRefPointerType{ElementType: pointType}

		// Declare p and q as var Own<Point>
		test.declare("p", ownPointType, true)
		test.declare("q", ownPointType, true)

		// Register function: mutateBoth(var a: Ref<Point>, var b: Ref<Point>)
		test.analyzer.functions["mutateBoth"] = FunctionInfo{
			ParamTypes: []Type{refPointTypeMut, refPointTypeMut},
			ReturnType: TypeVoid,
		}

		// Call mutateBoth(p, q) - different variables should be OK
		call := callExpr("mutateBoth", ident("p"), ident("q"))
		test.analyzer.analyzeCallExpr(call)
		test.expectNoErrors()
	})
}

func TestConditionalMoves(t *testing.T) {
	t.Run("move in then branch invalidates after if (no else)", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// if cond { val q = p }
		// p.x  // ERROR: p was moved
		ifStmt := ifStmtNoElse(
			boolLit("true"),
			[]ast.Statement{
				varDecl("q", false, ident("p")),
			},
		)

		test.analyzer.analyzeStatement(ifStmt)

		// Now try to use p after the if - should error
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("move in else branch invalidates after if", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// if cond { } else { val q = p }
		// p.x  // ERROR: p was moved
		ifStmt := ifStmtWithElse(
			boolLit("true"),
			[]ast.Statement{}, // empty then
			[]ast.Statement{
				varDecl("q", false, ident("p")),
			},
		)

		test.analyzer.analyzeStatement(ifStmt)

		// Now try to use p after the if - should error
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("move in both branches invalidates after if", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// if cond { val q = p } else { val r = p }
		// p.x  // ERROR: p was moved
		ifStmt := ifStmtWithElse(
			boolLit("true"),
			[]ast.Statement{
				varDecl("q", false, ident("p")),
			},
			[]ast.Statement{
				varDecl("r", false, ident("p")),
			},
		)

		test.analyzer.analyzeStatement(ifStmt)

		// Now try to use p after the if - should error
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("no move in any branch keeps variable valid", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// if cond { val x = 1 } else { val y = 2 }
		// p.x  // OK: p was not moved
		ifStmt := ifStmtWithElse(
			boolLit("true"),
			[]ast.Statement{
				varDecl("x", false, intLit("1")),
			},
			[]ast.Statement{
				varDecl("y", false, intLit("2")),
			},
		)

		test.analyzer.analyzeStatement(ifStmt)

		// Now try to use p after the if - should be OK
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectNoErrors()
	})

	t.Run("nested if with move invalidates after outer if", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// if cond1 {
		//   if cond2 { val q = p }
		// }
		// p.x  // ERROR: p was possibly moved
		innerIf := ifStmtNoElse(
			boolLit("true"),
			[]ast.Statement{
				varDecl("q", false, ident("p")),
			},
		)

		outerIf := ifStmtNoElse(
			boolLit("true"),
			[]ast.Statement{innerIf},
		)

		test.analyzer.analyzeStatement(outerIf)

		// Now try to use p after the if - should error
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})
}

func TestShortCircuitOperatorsWithMoves(t *testing.T) {
	// Helper function to create a consume function that takes Own<Point>
	setupConsumeFunction := func(test *analyzerTest) {
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}
		test.analyzer.functions["consume"] = FunctionInfo{
			ParamTypes: []Type{ownPointType},
			ReturnType: TypeBoolean,
		}
	}

	t.Run("move in right side of && is conditional", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		setupConsumeFunction(test)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// true && consume(p) -- consume might not be called if left is false
		expr := binExpr(
			boolLit("true"),
			"&&",
			callExpr("consume", ident("p")),
		)
		test.analyzer.analyzeExpression(expr)

		// Now try to use p after the && - should error because p may have been moved
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("move in right side of || is conditional", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		setupConsumeFunction(test)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// false || consume(p) -- consume might not be called if left is true
		expr := binExpr(
			boolLit("false"),
			"||",
			callExpr("consume", ident("p")),
		)
		test.analyzer.analyzeExpression(expr)

		// Now try to use p after the || - should error because p may have been moved
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("move in left side of && always happens", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		setupConsumeFunction(test)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// consume(p) && true -- consume is always called
		expr := binExpr(
			callExpr("consume", ident("p")),
			"&&",
			boolLit("true"),
		)
		test.analyzer.analyzeExpression(expr)

		// p is definitely moved
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("nested short-circuit with moves", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		setupConsumeFunction(test)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// (true && false) || consume(p) -- nested short-circuit
		inner := binExpr(boolLit("true"), "&&", boolLit("false"))
		outer := binExpr(inner, "||", callExpr("consume", ident("p")))
		test.analyzer.analyzeExpression(outer)

		// p may have been moved
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})
}

func TestConditionalExpressionMoves(t *testing.T) {
	t.Run("if expression move in then branch invalidates after", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// if cond { val q = p; 1 } else { 2 }
		// Use the if as an expression
		ifExpr := &ast.IfStmt{
			IfKeyword: pos(1, 1),
			Condition: boolLit("true"),
			ThenBranch: &ast.BlockStmt{
				LeftBrace:  pos(1, 5),
				Statements: []ast.Statement{
					varDecl("q", false, ident("p")),
					exprStmt(intLit("1")),
				},
				RightBrace: pos(1, 10),
			},
			ElseKeyword: pos(1, 12),
			ElseBranch: &ast.BlockStmt{
				LeftBrace:  pos(1, 17),
				Statements: []ast.Statement{exprStmt(intLit("2"))},
				RightBrace: pos(1, 22),
			},
		}

		test.analyzer.analyzeExpression(ifExpr)

		// Now try to use p after the if expression - should error
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("if expression move in else branch invalidates after", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// if cond { 1 } else { val q = p; 2 }
		ifExpr := &ast.IfStmt{
			IfKeyword: pos(1, 1),
			Condition: boolLit("true"),
			ThenBranch: &ast.BlockStmt{
				LeftBrace:  pos(1, 5),
				Statements: []ast.Statement{exprStmt(intLit("1"))},
				RightBrace: pos(1, 10),
			},
			ElseKeyword: pos(1, 12),
			ElseBranch: &ast.BlockStmt{
				LeftBrace:  pos(1, 17),
				Statements: []ast.Statement{
					varDecl("q", false, ident("p")),
					exprStmt(intLit("2")),
				},
				RightBrace: pos(1, 22),
			},
		}

		test.analyzer.analyzeExpression(ifExpr)

		// Now try to use p after - should error
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("when expression move in case invalidates after", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// when { true -> { val q = p } }
		when := whenExpr(
			whenCase(boolLit("true"), &ast.BlockStmt{
				LeftBrace:  pos(1, 10),
				Statements: []ast.Statement{varDecl("q", false, ident("p"))},
				RightBrace: pos(1, 20),
			}, false),
		)

		test.analyzer.analyzeStatement(when)

		// Now try to use p after - should error
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("when expression move in any case invalidates after", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// when { false -> { }, true -> { val q = p }, else -> { } }
		// p is moved in the second case
		when := whenExpr(
			whenCase(boolLit("false"), &ast.BlockStmt{
				LeftBrace:  pos(1, 10),
				Statements: []ast.Statement{},
				RightBrace: pos(1, 12),
			}, false),
			whenCase(boolLit("true"), &ast.BlockStmt{
				LeftBrace:  pos(1, 20),
				Statements: []ast.Statement{varDecl("q", false, ident("p"))},
				RightBrace: pos(1, 30),
			}, false),
			whenCase(nil, &ast.BlockStmt{
				LeftBrace:  pos(1, 40),
				Statements: []ast.Statement{},
				RightBrace: pos(1, 42),
			}, true),
		)

		test.analyzer.analyzeStatement(when)

		// Now try to use p after - should error because p may have been moved
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})
}

func TestLoopMoveRestrictions(t *testing.T) {
	t.Run("cannot move inside while loop", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// while true { val q = p }  // ERROR: cannot move inside loop
		loop := whileStmt(
			boolLit("true"),
			varDecl("q", false, ident("p")),
		)

		test.analyzer.analyzeStatement(loop)
		test.expectErrorContaining("cannot move 'p' inside a loop")
	})

	t.Run("cannot move inside for loop", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point> and i as s64
		test.declare("p", ownPointType, false)
		test.declare("i", TypeS64, true)

		// for i = 0; i < 10; i = i + 1 { val q = p }  // ERROR: cannot move inside loop
		loop := forStmt(
			assignStmt("i", intLit("0")),
			binExpr(ident("i"), "<", intLit("10")),
			assignStmt("i", binExpr(ident("i"), "+", intLit("1"))),
			varDecl("q", false, ident("p")),
		)

		test.analyzer.analyzeStatement(loop)
		test.expectErrorContaining("cannot move 'p' inside a loop")
	})

	t.Run("cannot move inside nested loop", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// while true {
		//   while true { val q = p }  // ERROR: cannot move inside loop
		// }
		innerLoop := whileStmt(
			boolLit("true"),
			varDecl("q", false, ident("p")),
		)

		outerLoop := whileStmt(
			boolLit("true"),
			innerLoop,
		)

		test.analyzer.analyzeStatement(outerLoop)
		test.expectErrorContaining("cannot move 'p' inside a loop")
	})

	t.Run("move before loop is OK", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// Move before the loop - this is fine
		moveStmt := varDecl("q", false, ident("p"))
		test.analyzer.analyzeStatement(moveStmt)

		// Loop doesn't try to move p
		loop := whileStmt(
			boolLit("true"),
			exprStmt(intLit("1")), // do nothing
		)
		test.analyzer.analyzeStatement(loop)

		test.expectNoErrors()
	})

	t.Run("copyable types can be used inside loops", func(t *testing.T) {
		test := newTest(t).withScope()

		// Declare x as s64 (copyable)
		test.declare("x", TypeS64, false)

		// while true { val y = x }  // OK: s64 is copyable
		loop := whileStmt(
			boolLit("true"),
			varDecl("y", false, ident("x")),
		)

		test.analyzer.analyzeStatement(loop)
		test.expectNoErrors()
	})
}

func TestSelfReferencePrevention(t *testing.T) {
	t.Run("cannot self-assign move-only type: p = p", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as var Own<Point> (mutable so we can reassign)
		test.declare("p", ownPointType, true)

		// p = p  // ERROR: cannot assign to itself
		stmt := assignStmt("p", ident("p"))
		test.analyzer.analyzeStatement(stmt)
		test.expectErrorContaining("cannot assign 'p' to itself")
	})

	t.Run("field access p = p.x is type mismatch not self-reference", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p as var Own<Point>
		test.declare("p", ownPointType, true)

		// p = p.x - this is a type mismatch (s64 vs Own<Point>), not a self-reference error
		// p.x is s64 which is copyable, so no move issues
		stmt := assignStmt("p", fieldAccessExpr(ident("p"), "x"))
		test.analyzer.analyzeStatement(stmt)
		// Should fail for type mismatch, not self-reference
		test.expectErrorContaining("cannot assign s64")
	})

	t.Run("cannot self-referential field assign: container.child = container", func(t *testing.T) {
		// Create a Child struct
		childType := StructType{
			Name: "Child",
			Fields: []StructFieldInfo{
				{Name: "value", Type: TypeS64, Mutable: false, Index: 0},
			},
		}
		ownChildType := OwnedPointerType{ElementType: childType}

		// Create a Container struct with a child field of type Own<Child>?
		containerType := StructType{
			Name: "Container",
			Fields: []StructFieldInfo{
				{Name: "child", Type: NullableType{InnerType: ownChildType}, Mutable: true, Index: 0},
			},
		}
		ownContainerType := OwnedPointerType{ElementType: containerType}

		test := newTest(t).withScope()
		test.analyzer.TypeRegistry.RegisterStruct("Child", childType)
		test.analyzer.TypeRegistry.RegisterStruct("Container", containerType)

		// Declare c as var Own<Container>
		test.declare("c", ownContainerType, true)

		// c.child = c  // ERROR: cannot assign c to a field of itself
		// Note: this test is checking self-reference prevention, not type checking
		// The types don't match (Own<Container> vs Own<Child>?) but self-ref check comes first
		stmt := fieldAssignStmt(ident("c"), "child", ident("c"))
		test.analyzer.analyzeStatement(stmt)
		test.expectErrorContaining("cannot assign 'c' to a field of itself")
	})

	t.Run("self-assign copyable type is OK", func(t *testing.T) {
		test := newTest(t).withScope()

		// Declare x as var s64
		test.declare("x", TypeS64, true)

		// x = x  // OK: s64 is copyable
		stmt := assignStmt("x", ident("x"))
		test.analyzer.analyzeStatement(stmt)
		test.expectNoErrors()
	})

	t.Run("assigning different variables is OK", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Declare p and q as var Own<Point>
		test.declare("p", ownPointType, true)
		test.declare("q", ownPointType, true)

		// p = q  // OK: different variables
		stmt := assignStmt("p", ident("q"))
		test.analyzer.analyzeStatement(stmt)
		test.expectNoErrors()
	})
}

func TestFunctionLevelOwnership(t *testing.T) {
	t.Run("return moves value out of function", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Simulate being in a function with return type Own<Point>
		test.analyzer.currentReturnType = ownPointType

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// return p - this moves p
		retStmt := returnStmt(ident("p"))
		test.analyzer.analyzeStatement(retStmt)

		// Verify p is now moved
		info, found := test.analyzer.ownershipScope.lookup("p")
		if !found {
			t.Fatalf("expected to find 'p' in ownership scope")
		}
		if info.State != StateMoved {
			t.Errorf("expected 'p' to be moved after return, got %s", info.State)
		}
	})

	t.Run("use after return is error", func(t *testing.T) {
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Simulate being in a function with return type Own<Point>
		test.analyzer.currentReturnType = ownPointType

		// Declare p as Own<Point>
		test.declare("p", ownPointType, false)

		// return p - this moves p
		retStmt := returnStmt(ident("p"))
		test.analyzer.analyzeStatement(retStmt)

		// Try to use p after return - should error
		fieldAccess := fieldAccessExpr(ident("p"), "x")
		test.analyzer.analyzeExpression(fieldAccess)
		test.expectErrorContaining("use of moved value 'p'")
	})

	t.Run("returning copyable type does not move", func(t *testing.T) {
		test := newTest(t).withScope()

		// Simulate being in a function with return type s64
		test.analyzer.currentReturnType = TypeS64

		// Declare x as s64 (copyable)
		test.declare("x", TypeS64, false)

		// return x - this copies x, doesn't move it
		retStmt := returnStmt(ident("x"))
		test.analyzer.analyzeStatement(retStmt)

		// Verify x is still owned (not moved)
		info, found := test.analyzer.ownershipScope.lookup("x")
		if found && info.State == StateMoved {
			t.Errorf("expected 'x' to still be owned (not moved), copyable types should copy")
		}
		test.expectNoErrors()
	})

	t.Run("pass-through ownership: take and return Own<T>", func(t *testing.T) {
		// This tests the pattern where a function takes ownership and returns it:
		// transform = (var p: Own<Point>) -> Own<Point> { p.x = 10; p }
		// Note: var prefix needed to mutate fields through the pointer
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Simulate being in a function with return type Own<Point>
		test.analyzer.currentReturnType = ownPointType

		// Declare p as var Own<Point> parameter (mutable binding to allow field mutation)
		test.declare("p", ownPointType, true) // mutable = true

		// Mutate the value: p.x = 10
		mutateStmt := fieldAssignStmt(ident("p"), "x", intLit("10"))
		test.analyzer.analyzeStatement(mutateStmt)

		// return p - passes ownership back to caller
		retStmt := returnStmt(ident("p"))
		test.analyzer.analyzeStatement(retStmt)

		// No errors should occur - this is a valid pass-through ownership pattern
		test.expectNoErrors()

		// Verify p is now moved (returned)
		info, found := test.analyzer.ownershipScope.lookup("p")
		if !found {
			t.Fatalf("expected to find 'p' in ownership scope")
		}
		if info.State != StateMoved {
			t.Errorf("expected 'p' to be moved after return, got %s", info.State)
		}
	})

	t.Run("pass-through ownership without mutation: val binding works", func(t *testing.T) {
		// This tests that immutable binding can still pass ownership through
		// identity = (p: Own<Point>) -> Own<Point> { p }
		test := newTest(t).withScope().withStruct("Point",
			StructFieldInfo{Name: "x", Type: TypeS64, Mutable: true, Index: 0},
		)
		pointType := test.analyzer.TypeRegistry.AllStructs()["Point"]
		ownPointType := OwnedPointerType{ElementType: pointType}

		// Simulate being in a function with return type Own<Point>
		test.analyzer.currentReturnType = ownPointType

		// Declare p as val Own<Point> parameter (immutable binding)
		test.declare("p", ownPointType, false) // mutable = false

		// return p - passes ownership back to caller without mutation
		retStmt := returnStmt(ident("p"))
		test.analyzer.analyzeStatement(retStmt)

		// No errors should occur
		test.expectNoErrors()

		// Verify p is now moved (returned)
		info, found := test.analyzer.ownershipScope.lookup("p")
		if !found {
			t.Fatalf("expected to find 'p' in ownership scope")
		}
		if info.State != StateMoved {
			t.Errorf("expected 'p' to be moved after return, got %s", info.State)
		}
	})
}
