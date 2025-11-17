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
