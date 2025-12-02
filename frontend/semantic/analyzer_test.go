package semantic

import (
	"testing"

	"github.com/seanrogers2657/slang/frontend/ast"
)

func TestAnalyzeLiteral(t *testing.T) {
	tests := []struct {
		name     string
		literal  *ast.LiteralExpr
		expected Type
	}{
		{
			name: "integer literal",
			literal: &ast.LiteralExpr{
				Kind:     ast.LiteralTypeNumber,
				Value:    "42",
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 2},
			},
			expected: TypeInteger,
		},
		{
			name: "string literal",
			literal: &ast.LiteralExpr{
				Kind:     ast.LiteralTypeString,
				Value:    "hello",
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 7},
			},
			expected: TypeString,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer("test.sl")
			result := analyzer.analyzeLiteral(tt.literal)

			if !result.GetType().Equals(tt.expected) {
				t.Errorf("expected type %s, got %s", tt.expected.String(), result.GetType().String())
			}

			if len(analyzer.errors) > 0 {
				t.Errorf("expected no errors, got %d errors", len(analyzer.errors))
			}
		})
	}
}

func TestAnalyzeBinaryExpression_Arithmetic(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		expected Type
	}{
		{"addition", "+", TypeInteger},
		{"subtraction", "-", TypeInteger},
		{"multiplication", "*", TypeInteger},
		{"division", "/", TypeInteger},
		{"modulo", "%", TypeInteger},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer("test.sl")

			expr := &ast.BinaryExpr{
				Left: &ast.LiteralExpr{
					Kind:     ast.LiteralTypeNumber,
					Value:    "5",
					StartPos: ast.Position{Line: 1, Column: 1},
					EndPos:   ast.Position{Line: 1, Column: 1},
				},
				Op: tt.op,
				Right: &ast.LiteralExpr{
					Kind:     ast.LiteralTypeNumber,
					Value:    "3",
					StartPos: ast.Position{Line: 1, Column: 5},
					EndPos:   ast.Position{Line: 1, Column: 5},
				},
				LeftPos:  ast.Position{Line: 1, Column: 1},
				OpPos:    ast.Position{Line: 1, Column: 3},
				RightPos: ast.Position{Line: 1, Column: 5},
			}

			result := analyzer.analyzeBinaryExpression(expr)

			if !result.GetType().Equals(tt.expected) {
				t.Errorf("expected type %s, got %s", tt.expected.String(), result.GetType().String())
			}

			if len(analyzer.errors) > 0 {
				t.Errorf("expected no errors, got %d errors: %v", len(analyzer.errors), analyzer.errors[0].Message)
			}
		})
	}
}

func TestAnalyzeBinaryExpression_Comparison(t *testing.T) {
	tests := []struct {
		name     string
		op       string
		expected Type
	}{
		{"equal", "==", TypeInteger},
		{"not equal", "!=", TypeInteger},
		{"less than", "<", TypeInteger},
		{"greater than", ">", TypeInteger},
		{"less or equal", "<=", TypeInteger},
		{"greater or equal", ">=", TypeInteger},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer("test.sl")

			expr := &ast.BinaryExpr{
				Left: &ast.LiteralExpr{
					Kind:     ast.LiteralTypeNumber,
					Value:    "5",
					StartPos: ast.Position{Line: 1, Column: 1},
					EndPos:   ast.Position{Line: 1, Column: 1},
				},
				Op: tt.op,
				Right: &ast.LiteralExpr{
					Kind:     ast.LiteralTypeNumber,
					Value:    "3",
					StartPos: ast.Position{Line: 1, Column: 5},
					EndPos:   ast.Position{Line: 1, Column: 5},
				},
				LeftPos:  ast.Position{Line: 1, Column: 1},
				OpPos:    ast.Position{Line: 1, Column: 3},
				RightPos: ast.Position{Line: 1, Column: 5},
			}

			result := analyzer.analyzeBinaryExpression(expr)

			if !result.GetType().Equals(tt.expected) {
				t.Errorf("expected type %s, got %s", tt.expected.String(), result.GetType().String())
			}

			if len(analyzer.errors) > 0 {
				t.Errorf("expected no errors, got %d errors: %v", len(analyzer.errors), analyzer.errors[0].Message)
			}
		})
	}
}

func TestAnalyzeBinaryExpression_TypeError(t *testing.T) {
	tests := []struct {
		name          string
		op            string
		leftType      ast.LiteralType
		rightType     ast.LiteralType
		expectedError string
	}{
		{
			name:          "string + integer",
			op:            "+",
			leftType:      ast.LiteralTypeString,
			rightType:     ast.LiteralTypeNumber,
			expectedError: "requires integer operands",
		},
		{
			name:          "integer + string",
			op:            "+",
			leftType:      ast.LiteralTypeNumber,
			rightType:     ast.LiteralTypeString,
			expectedError: "requires integer operands",
		},
		{
			name:          "string - string",
			op:            "-",
			leftType:      ast.LiteralTypeString,
			rightType:     ast.LiteralTypeString,
			expectedError: "requires integer operands",
		},
		{
			name:          "string == string comparison not supported yet",
			op:            "==",
			leftType:      ast.LiteralTypeString,
			rightType:     ast.LiteralTypeString,
			expectedError: "requires integer operands",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer("test.sl")

			expr := &ast.BinaryExpr{
				Left: &ast.LiteralExpr{
					Kind:     tt.leftType,
					Value:    "test",
					StartPos: ast.Position{Line: 1, Column: 1},
					EndPos:   ast.Position{Line: 1, Column: 1},
				},
				Op: tt.op,
				Right: &ast.LiteralExpr{
					Kind:     tt.rightType,
					Value:    "test",
					StartPos: ast.Position{Line: 1, Column: 5},
					EndPos:   ast.Position{Line: 1, Column: 5},
				},
				LeftPos:  ast.Position{Line: 1, Column: 1},
				OpPos:    ast.Position{Line: 1, Column: 3},
				RightPos: ast.Position{Line: 1, Column: 5},
			}

			result := analyzer.analyzeBinaryExpression(expr)

			// Should return error type
			if !result.GetType().Equals(TypeError) {
				t.Errorf("expected error type, got %s", result.GetType().String())
			}

			// Should have at least one error
			if len(analyzer.errors) == 0 {
				t.Error("expected at least one error, got none")
			}
		})
	}
}

func TestAnalyzeProgram(t *testing.T) {
	t.Run("valid program with expression statement", func(t *testing.T) {
		analyzer := NewAnalyzer("test.sl")

		program := &ast.Program{
			Statements: []ast.Statement{
				&ast.ExprStmt{
					Expr: &ast.BinaryExpr{
						Left: &ast.LiteralExpr{
							Kind:     ast.LiteralTypeNumber,
							Value:    "5",
							StartPos: ast.Position{Line: 1, Column: 1},
							EndPos:   ast.Position{Line: 1, Column: 1},
						},
						Op: "+",
						Right: &ast.LiteralExpr{
							Kind:     ast.LiteralTypeNumber,
							Value:    "3",
							StartPos: ast.Position{Line: 1, Column: 5},
							EndPos:   ast.Position{Line: 1, Column: 5},
						},
						LeftPos:  ast.Position{Line: 1, Column: 1},
						OpPos:    ast.Position{Line: 1, Column: 3},
						RightPos: ast.Position{Line: 1, Column: 5},
					},
				},
			},
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 5},
		}

		errs, typedProgram := analyzer.Analyze(program)

		if len(errs) > 0 {
			t.Errorf("expected no errors, got %d errors", len(errs))
		}

		if len(typedProgram.Statements) != 1 {
			t.Errorf("expected 1 statement, got %d", len(typedProgram.Statements))
		}
	})

	t.Run("valid program with print statement", func(t *testing.T) {
		analyzer := NewAnalyzer("test.sl")

		program := &ast.Program{
			Statements: []ast.Statement{
				&ast.PrintStmt{
					Keyword: ast.Position{Line: 1, Column: 1},
					Expr: &ast.LiteralExpr{
						Kind:     ast.LiteralTypeNumber,
						Value:    "42",
						StartPos: ast.Position{Line: 1, Column: 7},
						EndPos:   ast.Position{Line: 1, Column: 8},
					},
				},
			},
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 8},
		}

		errs, typedProgram := analyzer.Analyze(program)

		if len(errs) > 0 {
			t.Errorf("expected no errors, got %d errors", len(errs))
		}

		if len(typedProgram.Statements) != 1 {
			t.Errorf("expected 1 statement, got %d", len(typedProgram.Statements))
		}

		// Check that it's a typed print statement
		printStmt, ok := typedProgram.Statements[0].(*TypedPrintStmt)
		if !ok {
			t.Error("expected TypedPrintStmt")
		}

		if !printStmt.Expr.GetType().Equals(TypeInteger) {
			t.Errorf("expected integer type, got %s", printStmt.Expr.GetType().String())
		}
	})

	t.Run("program with type error", func(t *testing.T) {
		analyzer := NewAnalyzer("test.sl")

		program := &ast.Program{
			Statements: []ast.Statement{
				&ast.ExprStmt{
					Expr: &ast.BinaryExpr{
						Left: &ast.LiteralExpr{
							Kind:     ast.LiteralTypeString,
							Value:    "hello",
							StartPos: ast.Position{Line: 1, Column: 1},
							EndPos:   ast.Position{Line: 1, Column: 7},
						},
						Op: "+",
						Right: &ast.LiteralExpr{
							Kind:     ast.LiteralTypeNumber,
							Value:    "3",
							StartPos: ast.Position{Line: 1, Column: 11},
							EndPos:   ast.Position{Line: 1, Column: 11},
						},
						LeftPos:  ast.Position{Line: 1, Column: 1},
						OpPos:    ast.Position{Line: 1, Column: 9},
						RightPos: ast.Position{Line: 1, Column: 11},
					},
				},
			},
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 11},
		}

		errs, _ := analyzer.Analyze(program)

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
		{
			name:         "simple integer variable",
			varName:      "x",
			initValue:    "42",
			expectedType: TypeInteger,
		},
		{
			name:         "variable with underscore",
			varName:      "my_var",
			initValue:    "100",
			expectedType: TypeInteger,
		},
		{
			name:         "variable with digits",
			varName:      "value1",
			initValue:    "5",
			expectedType: TypeInteger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer := NewAnalyzer("test.sl")
			// Create a function scope for the variable
			analyzer.enterScope()

			varDecl := &ast.VarDeclStmt{
				ValKeyword: ast.Position{Line: 1, Column: 1},
				Name:       tt.varName,
				NamePos:    ast.Position{Line: 1, Column: 5},
				Equals:     ast.Position{Line: 1, Column: 7},
				Initializer: &ast.LiteralExpr{
					Kind:     ast.LiteralTypeNumber,
					Value:    tt.initValue,
					StartPos: ast.Position{Line: 1, Column: 9},
					EndPos:   ast.Position{Line: 1, Column: 9 + len(tt.initValue)},
				},
			}

			result := analyzer.analyzeVarDeclStatement(varDecl)

			// Cast to TypedVarDeclStmt to check the initializer type
			typedVarDecl, ok := result.(*TypedVarDeclStmt)
			if !ok {
				t.Fatal("expected TypedVarDeclStmt")
			}

			if !typedVarDecl.Initializer.GetType().Equals(tt.expectedType) {
				t.Errorf("expected initializer type %s, got %s", tt.expectedType.String(), typedVarDecl.Initializer.GetType().String())
			}

			if len(analyzer.errors) > 0 {
				t.Errorf("expected no errors, got %d errors: %v", len(analyzer.errors), analyzer.errors[0].Message)
			}

			// Verify variable is in scope with correct type
			varType, found := analyzer.currentScope.lookup(tt.varName)
			if !found {
				t.Errorf("variable %s not found in scope", tt.varName)
			}
			if !varType.Equals(tt.expectedType) {
				t.Errorf("variable %s has wrong type in scope: expected %s, got %s", tt.varName, tt.expectedType.String(), varType.String())
			}
		})
	}
}

func TestAnalyzeIdentifierExpression(t *testing.T) {
	t.Run("valid identifier lookup", func(t *testing.T) {
		analyzer := NewAnalyzer("test.sl")
		analyzer.enterScope()

		// First declare a variable
		analyzer.currentScope.declare("myVar", TypeInteger)

		identifier := &ast.IdentifierExpr{
			Name:     "myVar",
			StartPos: ast.Position{Line: 2, Column: 1},
			EndPos:   ast.Position{Line: 2, Column: 5},
		}

		result := analyzer.analyzeIdentifier(identifier)

		if !result.GetType().Equals(TypeInteger) {
			t.Errorf("expected type integer, got %s", result.GetType().String())
		}

		if len(analyzer.errors) > 0 {
			t.Errorf("expected no errors, got %d errors", len(analyzer.errors))
		}
	})

	t.Run("undefined variable error", func(t *testing.T) {
		analyzer := NewAnalyzer("test.sl")
		analyzer.enterScope()

		identifier := &ast.IdentifierExpr{
			Name:     "undefinedVar",
			StartPos: ast.Position{Line: 1, Column: 1},
			EndPos:   ast.Position{Line: 1, Column: 12},
		}

		result := analyzer.analyzeIdentifier(identifier)

		if !result.GetType().Equals(TypeError) {
			t.Errorf("expected error type, got %s", result.GetType().String())
		}

		if len(analyzer.errors) == 0 {
			t.Error("expected an error for undefined variable, got none")
		}

		if len(analyzer.errors) > 0 && analyzer.errors[0].Message != "undefined variable 'undefinedVar'" {
			t.Errorf("unexpected error message: %s", analyzer.errors[0].Message)
		}
	})
}

func TestAnalyzeDuplicateVariableDeclaration(t *testing.T) {
	t.Run("duplicate variable in same scope", func(t *testing.T) {
		analyzer := NewAnalyzer("test.sl")
		analyzer.enterScope()

		// First declaration
		varDecl1 := &ast.VarDeclStmt{
			ValKeyword: ast.Position{Line: 1, Column: 1},
			Name:       "x",
			NamePos:    ast.Position{Line: 1, Column: 5},
			Equals:     ast.Position{Line: 1, Column: 7},
			Initializer: &ast.LiteralExpr{
				Kind:     ast.LiteralTypeNumber,
				Value:    "5",
				StartPos: ast.Position{Line: 1, Column: 9},
				EndPos:   ast.Position{Line: 1, Column: 9},
			},
		}
		analyzer.analyzeVarDeclStatement(varDecl1)

		// Second declaration with same name
		varDecl2 := &ast.VarDeclStmt{
			ValKeyword: ast.Position{Line: 2, Column: 1},
			Name:       "x",
			NamePos:    ast.Position{Line: 2, Column: 5},
			Equals:     ast.Position{Line: 2, Column: 7},
			Initializer: &ast.LiteralExpr{
				Kind:     ast.LiteralTypeNumber,
				Value:    "10",
				StartPos: ast.Position{Line: 2, Column: 9},
				EndPos:   ast.Position{Line: 2, Column: 10},
			},
		}
		analyzer.analyzeVarDeclStatement(varDecl2)

		if len(analyzer.errors) == 0 {
			t.Error("expected an error for duplicate variable declaration, got none")
		}

		foundDuplicateError := false
		for _, err := range analyzer.errors {
			if err.Message == "variable 'x' is already declared in this scope" {
				foundDuplicateError = true
				break
			}
		}

		if !foundDuplicateError {
			t.Errorf("expected duplicate variable error message, got: %v", analyzer.errors)
		}
	})
}

func TestAnalyzeVariableInExpression(t *testing.T) {
	t.Run("variable in binary expression", func(t *testing.T) {
		analyzer := NewAnalyzer("test.sl")
		analyzer.enterScope()

		// Declare variables
		analyzer.currentScope.declare("a", TypeInteger)
		analyzer.currentScope.declare("b", TypeInteger)

		expr := &ast.BinaryExpr{
			Left: &ast.IdentifierExpr{
				Name:     "a",
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
			Op: "+",
			Right: &ast.IdentifierExpr{
				Name:     "b",
				StartPos: ast.Position{Line: 1, Column: 5},
				EndPos:   ast.Position{Line: 1, Column: 5},
			},
			LeftPos:  ast.Position{Line: 1, Column: 1},
			OpPos:    ast.Position{Line: 1, Column: 3},
			RightPos: ast.Position{Line: 1, Column: 5},
		}

		result := analyzer.analyzeBinaryExpression(expr)

		if !result.GetType().Equals(TypeInteger) {
			t.Errorf("expected type integer, got %s", result.GetType().String())
		}

		if len(analyzer.errors) > 0 {
			t.Errorf("expected no errors, got %d errors: %v", len(analyzer.errors), analyzer.errors[0].Message)
		}
	})

	t.Run("undefined variable in expression produces error", func(t *testing.T) {
		analyzer := NewAnalyzer("test.sl")
		analyzer.enterScope()

		// Only declare 'a', not 'b'
		analyzer.currentScope.declare("a", TypeInteger)

		expr := &ast.BinaryExpr{
			Left: &ast.IdentifierExpr{
				Name:     "a",
				StartPos: ast.Position{Line: 1, Column: 1},
				EndPos:   ast.Position{Line: 1, Column: 1},
			},
			Op: "+",
			Right: &ast.IdentifierExpr{
				Name:     "b",
				StartPos: ast.Position{Line: 1, Column: 5},
				EndPos:   ast.Position{Line: 1, Column: 5},
			},
			LeftPos:  ast.Position{Line: 1, Column: 1},
			OpPos:    ast.Position{Line: 1, Column: 3},
			RightPos: ast.Position{Line: 1, Column: 5},
		}

		result := analyzer.analyzeBinaryExpression(expr)

		// The result should be an error type because right operand is undefined
		if !result.GetType().Equals(TypeError) {
			t.Errorf("expected error type, got %s", result.GetType().String())
		}

		if len(analyzer.errors) == 0 {
			t.Error("expected an error for undefined variable, got none")
		}
	})
}
