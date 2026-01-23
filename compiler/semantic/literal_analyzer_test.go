package semantic

import (
	"testing"

	"github.com/seanrogers2657/slang/compiler/ast"
)

func TestFieldContainer_GetFields(t *testing.T) {
	t.Run("StructType implements FieldContainer", func(t *testing.T) {
		st := StructType{
			Name: "Point",
			Fields: []StructFieldInfo{
				{Name: "x", Type: TypeS64, Index: 0},
				{Name: "y", Type: TypeS64, Index: 1},
			},
		}

		var container FieldContainer = st
		fields := container.GetFields()

		if len(fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(fields))
		}
		if fields[0].Name != "x" {
			t.Errorf("expected first field 'x', got %q", fields[0].Name)
		}
		if fields[1].Name != "y" {
			t.Errorf("expected second field 'y', got %q", fields[1].Name)
		}
	})

	t.Run("ClassType implements FieldContainer", func(t *testing.T) {
		ct := ClassType{
			Name: "Counter",
			Fields: []StructFieldInfo{
				{Name: "count", Type: TypeS64, Index: 0},
			},
		}

		var container FieldContainer = ct
		fields := container.GetFields()

		if len(fields) != 1 {
			t.Errorf("expected 1 field, got %d", len(fields))
		}
		if fields[0].Name != "count" {
			t.Errorf("expected field 'count', got %q", fields[0].Name)
		}
	})
}

func TestGetTypeKindName(t *testing.T) {
	tests := []struct {
		name      string
		container FieldContainer
		want      string
	}{
		{
			name:      "StructType returns struct",
			container: StructType{Name: "Point"},
			want:      "struct",
		},
		{
			name:      "ClassType returns class",
			container: ClassType{Name: "Counter"},
			want:      "class",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTypeKindName(tt.container)
			if got != tt.want {
				t.Errorf("getTypeKindName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAnalyzeNamedLiteral(t *testing.T) {
	// Create a simple struct type for testing
	pointType := StructType{
		Name: "Point",
		Fields: []StructFieldInfo{
			{Name: "x", Type: TypeS64, Index: 0, Mutable: false},
			{Name: "y", Type: TypeS64, Index: 1, Mutable: false},
		},
	}

	t.Run("valid named literal", func(t *testing.T) {
		test := newTest(t)

		namedArgs := []ast.NamedArgument{
			{
				Name:    "x",
				NamePos: pos(1, 1),
				Value:   intLit("10"),
			},
			{
				Name:    "y",
				NamePos: pos(1, 10),
				Value:   intLit("20"),
			},
		}

		result := test.analyzer.analyzeNamedLiteral(
			pointType,
			"Point",
			namedArgs,
			pos(1, 1),
			pos(1, 20),
		)

		if !result.valid {
			t.Error("expected valid result")
		}
		if len(result.args) != 2 {
			t.Errorf("expected 2 args, got %d", len(result.args))
		}
		test.expectNoErrors()
	})

	t.Run("wrong argument count", func(t *testing.T) {
		test := newTest(t)

		namedArgs := []ast.NamedArgument{
			{
				Name:    "x",
				NamePos: pos(1, 1),
				Value:   intLit("10"),
			},
		}

		test.analyzer.analyzeNamedLiteral(
			pointType,
			"Point",
			namedArgs,
			pos(1, 1),
			pos(1, 20),
		)

		test.expectErrors(1)
	})

	t.Run("unknown field", func(t *testing.T) {
		test := newTest(t)

		namedArgs := []ast.NamedArgument{
			{
				Name:    "x",
				NamePos: pos(1, 1),
				Value:   intLit("10"),
			},
			{
				Name:    "z", // unknown field
				NamePos: pos(1, 10),
				Value:   intLit("20"),
			},
		}

		test.analyzer.analyzeNamedLiteral(
			pointType,
			"Point",
			namedArgs,
			pos(1, 1),
			pos(1, 20),
		)

		test.expectErrorContaining("no field 'z'")
	})

	t.Run("duplicate field", func(t *testing.T) {
		test := newTest(t)

		namedArgs := []ast.NamedArgument{
			{
				Name:    "x",
				NamePos: pos(1, 1),
				Value:   intLit("10"),
			},
			{
				Name:    "x", // duplicate
				NamePos: pos(1, 10),
				Value:   intLit("20"),
			},
		}

		test.analyzer.analyzeNamedLiteral(
			pointType,
			"Point",
			namedArgs,
			pos(1, 1),
			pos(1, 20),
		)

		test.expectErrorContaining("specified multiple times")
	})
}

func TestAnalyzePositionalLiteral(t *testing.T) {
	pointType := StructType{
		Name: "Point",
		Fields: []StructFieldInfo{
			{Name: "x", Type: TypeS64, Index: 0, Mutable: false},
			{Name: "y", Type: TypeS64, Index: 1, Mutable: false},
		},
	}

	t.Run("valid positional literal", func(t *testing.T) {
		test := newTest(t)

		args := []ast.Expression{
			intLit("10"),
			intLit("20"),
		}

		result := test.analyzer.analyzePositionalLiteral(
			pointType,
			"Point",
			args,
			pos(1, 1),
			pos(1, 20),
		)

		if len(result) != 2 {
			t.Errorf("expected 2 args, got %d", len(result))
		}
		test.expectNoErrors()
	})

	t.Run("wrong argument count", func(t *testing.T) {
		test := newTest(t)

		args := []ast.Expression{
			intLit("10"),
		}

		test.analyzer.analyzePositionalLiteral(
			pointType,
			"Point",
			args,
			pos(1, 1),
			pos(1, 20),
		)

		test.expectErrors(1)
	})

	t.Run("type mismatch", func(t *testing.T) {
		test := newTest(t)

		args := []ast.Expression{
			strLit("hello"), // string instead of int
			intLit("20"),
		}

		test.analyzer.analyzePositionalLiteral(
			pointType,
			"Point",
			args,
			pos(1, 1),
			pos(1, 20),
		)

		test.expectErrors(1)
	})
}
