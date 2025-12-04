package codegen

import (
	"strings"
	"testing"
)

func TestIntOperation_Arithmetic(t *testing.T) {
	tests := []struct {
		op       string
		signed   bool
		expected []string
	}{
		{"+", true, []string{"add x2, x0, x1"}},
		{"-", true, []string{"sub x2, x0, x1"}},
		{"*", true, []string{"mul x2, x0, x1"}},
		{"/", true, []string{"sdiv x2, x0, x1"}},
		{"/", false, []string{"udiv x2, x0, x1"}},
		{"%", true, []string{"sdiv x3, x0, x1", "msub x2, x3, x1, x0"}},
		{"%", false, []string{"udiv x3, x0, x1", "msub x2, x3, x1, x0"}},
	}

	for _, tt := range tests {
		name := tt.op
		if !tt.signed {
			name += "_unsigned"
		}
		t.Run(name, func(t *testing.T) {
			output, err := IntOperation(tt.op, tt.signed)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected %q in output, got:\n%s", exp, output)
				}
			}
		})
	}
}

func TestIntOperation_Comparison_Signed(t *testing.T) {
	tests := []struct {
		op       string
		expected []string
	}{
		{"==", []string{"cmp x0, x1", "cset x2, eq"}},
		{"!=", []string{"cmp x0, x1", "cset x2, ne"}},
		{"<", []string{"cmp x0, x1", "cset x2, lt"}},
		{">", []string{"cmp x0, x1", "cset x2, gt"}},
		{"<=", []string{"cmp x0, x1", "cset x2, le"}},
		{">=", []string{"cmp x0, x1", "cset x2, ge"}},
	}

	for _, tt := range tests {
		t.Run(tt.op+"_signed", func(t *testing.T) {
			output, err := IntOperation(tt.op, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected %q in output, got:\n%s", exp, output)
				}
			}
		})
	}
}

func TestIntOperation_Comparison_Unsigned(t *testing.T) {
	tests := []struct {
		op       string
		expected []string
	}{
		{"==", []string{"cmp x0, x1", "cset x2, eq"}},
		{"!=", []string{"cmp x0, x1", "cset x2, ne"}},
		{"<", []string{"cmp x0, x1", "cset x2, lo"}},  // lo = lower (unsigned)
		{">", []string{"cmp x0, x1", "cset x2, hi"}},  // hi = higher (unsigned)
		{"<=", []string{"cmp x0, x1", "cset x2, ls"}}, // ls = lower or same
		{">=", []string{"cmp x0, x1", "cset x2, hs"}}, // hs = higher or same
	}

	for _, tt := range tests {
		t.Run(tt.op+"_unsigned", func(t *testing.T) {
			output, err := IntOperation(tt.op, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected %q in output, got:\n%s", exp, output)
				}
			}
		})
	}
}

func TestIntOperation_Unsupported(t *testing.T) {
	unsupported := []string{"&", "|", "^", "<<", ">>", "&&", "||"}

	for _, op := range unsupported {
		t.Run(op, func(t *testing.T) {
			_, err := IntOperation(op, true)
			if err == nil {
				t.Errorf("expected error for unsupported operation %q", op)
			}
			if !strings.Contains(err.Error(), "unsupported") {
				t.Errorf("error should mention 'unsupported', got: %v", err)
			}
		})
	}
}

func TestFloatOperation_Arithmetic(t *testing.T) {
	tests := []struct {
		op       string
		expected string
	}{
		{"+", "fadd d0, d1, d0"},
		{"-", "fsub d0, d1, d0"},
		{"*", "fmul d0, d1, d0"},
		{"/", "fdiv d0, d1, d0"},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			output, err := FloatOperation(tt.op)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output, got:\n%s", tt.expected, output)
			}
		})
	}
}

func TestFloatOperation_Comparison(t *testing.T) {
	tests := []struct {
		op       string
		expected []string
	}{
		{"==", []string{"fcmp d1, d0", "cset x2, eq"}},
		{"!=", []string{"fcmp d1, d0", "cset x2, ne"}},
		{"<", []string{"fcmp d1, d0", "cset x2, mi"}},
		{">", []string{"fcmp d1, d0", "cset x2, gt"}},
		{"<=", []string{"fcmp d1, d0", "cset x2, ls"}},
		{">=", []string{"fcmp d1, d0", "cset x2, ge"}},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			output, err := FloatOperation(tt.op)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected %q in output, got:\n%s", exp, output)
				}
			}
		})
	}
}

func TestFloatOperation_Unsupported(t *testing.T) {
	unsupported := []string{"%", "&", "|", "^"}

	for _, op := range unsupported {
		t.Run(op, func(t *testing.T) {
			_, err := FloatOperation(op)
			if err == nil {
				t.Errorf("expected error for unsupported float operation %q", op)
			}
			if !strings.Contains(err.Error(), "unsupported") {
				t.Errorf("error should mention 'unsupported', got: %v", err)
			}
		})
	}
}
