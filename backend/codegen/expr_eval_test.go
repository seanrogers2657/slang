package codegen

import (
	"strings"
	"testing"
)

func TestEmitBinaryExprSetup_BothSimple(t *testing.T) {
	eval := &BinaryExprEvaluator{
		LeftIsComplex:  false,
		RightIsComplex: false,
		GenerateLeftToReg: func(reg string) (string, error) {
			return "    mov " + reg + ", #10\n", nil
		},
		GenerateRightToReg: func(reg string) (string, error) {
			return "    mov " + reg + ", #20\n", nil
		},
	}

	output, err := EmitBinaryExprSetup(eval)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"mov x0, #10",
		"mov x1, #20",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}

	// Should not have stack operations for simple case
	if strings.Contains(output, "[sp") {
		t.Errorf("simple case should not use stack, got:\n%s", output)
	}
}

func TestEmitBinaryExprSetup_LeftComplex(t *testing.T) {
	eval := &BinaryExprEvaluator{
		LeftIsComplex:  true,
		RightIsComplex: false,
		GenerateLeft: func() (string, error) {
			return "    ; complex left expr\n    mov x2, #100\n", nil
		},
		GenerateRightToReg: func(reg string) (string, error) {
			return "    mov " + reg + ", #5\n", nil
		},
	}

	output, err := EmitBinaryExprSetup(eval)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"complex left expr",
		"str x2, [sp, #-16]!",
		"mov x1, #5",
		"ldr x0, [sp], #16",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitBinaryExprSetup_RightComplex(t *testing.T) {
	eval := &BinaryExprEvaluator{
		LeftIsComplex:  false,
		RightIsComplex: true,
		GenerateRight: func() (string, error) {
			return "    ; complex right expr\n    mov x2, #200\n", nil
		},
		GenerateLeftToReg: func(reg string) (string, error) {
			return "    mov " + reg + ", #3\n", nil
		},
	}

	output, err := EmitBinaryExprSetup(eval)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"complex right expr",
		"str x2, [sp, #-16]!",
		"mov x0, #3",
		"ldr x1, [sp], #16",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitBinaryExprSetup_BothComplex(t *testing.T) {
	eval := &BinaryExprEvaluator{
		LeftIsComplex:  true,
		RightIsComplex: true,
		GenerateLeft: func() (string, error) {
			return "    ; left\n    mov x2, #50\n", nil
		},
		GenerateRight: func() (string, error) {
			return "    ; right\n    mov x2, #60\n", nil
		},
	}

	output, err := EmitBinaryExprSetup(eval)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"; left",
		"str x2, [sp, #-16]!",
		"; right",
		"mov x1, x2",
		"ldr x0, [sp], #16",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitBinaryExprSetup_ErrorHandling(t *testing.T) {
	tests := []struct {
		name string
		eval *BinaryExprEvaluator
	}{
		{
			name: "left error in both complex",
			eval: &BinaryExprEvaluator{
				LeftIsComplex:  true,
				RightIsComplex: true,
				GenerateLeft: func() (string, error) {
					return "", &testError{"left failed"}
				},
			},
		},
		{
			name: "right error in both complex",
			eval: &BinaryExprEvaluator{
				LeftIsComplex:  true,
				RightIsComplex: true,
				GenerateLeft: func() (string, error) {
					return "ok", nil
				},
				GenerateRight: func() (string, error) {
					return "", &testError{"right failed"}
				},
			},
		},
		{
			name: "right error when right complex",
			eval: &BinaryExprEvaluator{
				LeftIsComplex:  false,
				RightIsComplex: true,
				GenerateRight: func() (string, error) {
					return "", &testError{"right failed"}
				},
			},
		},
		{
			name: "left error when left complex",
			eval: &BinaryExprEvaluator{
				LeftIsComplex:  true,
				RightIsComplex: false,
				GenerateLeft: func() (string, error) {
					return "", &testError{"left failed"}
				},
			},
		},
		{
			name: "left to reg error when right complex",
			eval: &BinaryExprEvaluator{
				LeftIsComplex:  false,
				RightIsComplex: true,
				GenerateRight: func() (string, error) {
					return "ok", nil
				},
				GenerateLeftToReg: func(reg string) (string, error) {
					return "", &testError{"left to reg failed"}
				},
			},
		},
		{
			name: "right to reg error when left complex",
			eval: &BinaryExprEvaluator{
				LeftIsComplex:  true,
				RightIsComplex: false,
				GenerateLeft: func() (string, error) {
					return "ok", nil
				},
				GenerateRightToReg: func(reg string) (string, error) {
					return "", &testError{"right to reg failed"}
				},
			},
		},
		{
			name: "left to reg error in simple case",
			eval: &BinaryExprEvaluator{
				LeftIsComplex:  false,
				RightIsComplex: false,
				GenerateLeftToReg: func(reg string) (string, error) {
					return "", &testError{"left to reg failed"}
				},
			},
		},
		{
			name: "right to reg error in simple case",
			eval: &BinaryExprEvaluator{
				LeftIsComplex:  false,
				RightIsComplex: false,
				GenerateLeftToReg: func(reg string) (string, error) {
					return "ok", nil
				},
				GenerateRightToReg: func(reg string) (string, error) {
					return "", &testError{"right to reg failed"}
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EmitBinaryExprSetup(tt.eval)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestEmitFloatBinaryExprSetup(t *testing.T) {
	leftCalled := false
	rightCalled := false

	output, err := EmitFloatBinaryExprSetup(
		func() (string, error) {
			leftCalled = true
			return "    ; left float\n    fmov d0, #1.0\n", nil
		},
		func() (string, error) {
			rightCalled = true
			return "    ; right float\n    fmov d0, #2.0\n", nil
		},
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !leftCalled {
		t.Error("left generator should have been called")
	}
	if !rightCalled {
		t.Error("right generator should have been called")
	}

	expected := []string{
		"; left float",
		"fmov d1, d0", // save left to d1
		"; right float",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitFloatBinaryExprSetup_LeftError(t *testing.T) {
	_, err := EmitFloatBinaryExprSetup(
		func() (string, error) {
			return "", &testError{"left failed"}
		},
		func() (string, error) {
			return "ok", nil
		},
	)

	if err == nil {
		t.Error("expected error")
	}
}

func TestEmitFloatBinaryExprSetup_RightError(t *testing.T) {
	_, err := EmitFloatBinaryExprSetup(
		func() (string, error) {
			return "ok", nil
		},
		func() (string, error) {
			return "", &testError{"right failed"}
		},
	)

	if err == nil {
		t.Error("expected error")
	}
}

func TestEmitCallSetup_NoArgs(t *testing.T) {
	output, err := EmitCallSetup(0, func(i int) (string, error) {
		t.Error("should not be called for 0 args")
		return "", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output != "" {
		t.Errorf("expected empty output for 0 args, got: %s", output)
	}
}

func TestEmitCallSetup_SingleArg(t *testing.T) {
	output, err := EmitCallSetup(1, func(i int) (string, error) {
		return "    mov x2, #42\n", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"sub sp, sp, #16",
		"mov x2, #42",
		"str x2, [sp, #0]",
		"ldr x0, [sp, #0]",
		"add sp, sp, #16",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitCallSetup_MultipleArgs(t *testing.T) {
	callCount := 0
	output, err := EmitCallSetup(3, func(i int) (string, error) {
		callCount++
		return "    mov x2, #" + intToStr(i*10) + "\n", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}

	expected := []string{
		"sub sp, sp, #48", // 3 * 16
		"str x2, [sp, #0]",
		"str x2, [sp, #16]",
		"str x2, [sp, #32]",
		"ldr x0, [sp, #0]",
		"ldr x1, [sp, #16]",
		"ldr x2, [sp, #32]",
		"add sp, sp, #48",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitCallSetup_Error(t *testing.T) {
	_, err := EmitCallSetup(2, func(i int) (string, error) {
		if i == 1 {
			return "", &testError{"arg 1 failed"}
		}
		return "ok", nil
	})

	if err == nil {
		t.Error("expected error")
	}
}

func TestIntToStr(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{100, "100"},
		{999, "999"},
		{-1, "-1"},
		{-42, "-42"},
		{12345, "12345"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := intToStr(tt.input)
			if result != tt.expected {
				t.Errorf("intToStr(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
