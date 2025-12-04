package arm64

import (
	"strings"
	"testing"

	"github.com/seanrogers2657/slang/backend/arch"
)

// TestEmitterImplementsInterface verifies that Emitter implements arch.Emitter.
func TestEmitterImplementsInterface(t *testing.T) {
	var _ arch.Emitter = (*Emitter)(nil)
	var _ arch.Emitter = New()
}

// =============================================================================
// ABI Accessor Tests
// =============================================================================

func TestABIConstants(t *testing.T) {
	e := New()

	t.Run("StackAlignment", func(t *testing.T) {
		if got := e.StackAlignment(); got != 16 {
			t.Errorf("StackAlignment() = %d, want 16", got)
		}
	})

	t.Run("ResultReg", func(t *testing.T) {
		if got := e.ResultReg(); got != "x2" {
			t.Errorf("ResultReg() = %q, want %q", got, "x2")
		}
	})

	t.Run("FloatResultReg", func(t *testing.T) {
		if got := e.FloatResultReg(); got != "d0" {
			t.Errorf("FloatResultReg() = %q, want %q", got, "d0")
		}
	})

	t.Run("LeftReg", func(t *testing.T) {
		if got := e.LeftReg(); got != "x0" {
			t.Errorf("LeftReg() = %q, want %q", got, "x0")
		}
	})

	t.Run("RightReg", func(t *testing.T) {
		if got := e.RightReg(); got != "x1" {
			t.Errorf("RightReg() = %q, want %q", got, "x1")
		}
	})

	t.Run("FloatLeftReg", func(t *testing.T) {
		if got := e.FloatLeftReg(); got != "d1" {
			t.Errorf("FloatLeftReg() = %q, want %q", got, "d1")
		}
	})

	t.Run("FloatRightReg", func(t *testing.T) {
		if got := e.FloatRightReg(); got != "d0" {
			t.Errorf("FloatRightReg() = %q, want %q", got, "d0")
		}
	})

	t.Run("FramePointer", func(t *testing.T) {
		if got := e.FramePointer(); got != "x29" {
			t.Errorf("FramePointer() = %q, want %q", got, "x29")
		}
	})

	t.Run("LinkReg", func(t *testing.T) {
		if got := e.LinkReg(); got != "x30" {
			t.Errorf("LinkReg() = %q, want %q", got, "x30")
		}
	})

	t.Run("ArgRegs", func(t *testing.T) {
		got := e.ArgRegs()
		want := []string{"x0", "x1", "x2", "x3", "x4", "x5", "x6", "x7"}
		if len(got) != len(want) {
			t.Errorf("ArgRegs() length = %d, want %d", len(got), len(want))
		}
		for i, r := range want {
			if got[i] != r {
				t.Errorf("ArgRegs()[%d] = %q, want %q", i, got[i], r)
			}
		}
	})
}

// =============================================================================
// Program Structure Tests
// =============================================================================

func TestEmitDataSection(t *testing.T) {
	e := New()

	t.Run("without print support", func(t *testing.T) {
		output := e.EmitDataSection(false)

		mustContain := []string{".data", ".align 3"}
		mustNotContain := []string{"buffer:", "newline:"}

		for _, s := range mustContain {
			if !strings.Contains(output, s) {
				t.Errorf("expected %q in output:\n%s", s, output)
			}
		}
		for _, s := range mustNotContain {
			if strings.Contains(output, s) {
				t.Errorf("did not expect %q in output:\n%s", s, output)
			}
		}
	})

	t.Run("with print support", func(t *testing.T) {
		output := e.EmitDataSection(true)

		mustContain := []string{
			".data",
			".align 3",
			"buffer: .space 32",
			"newline: .byte 10",
		}

		for _, s := range mustContain {
			if !strings.Contains(output, s) {
				t.Errorf("expected %q in output:\n%s", s, output)
			}
		}
	})
}

func TestEmitProgramEntry(t *testing.T) {
	e := New()
	output := e.EmitProgramEntry()

	mustContain := []string{
		".global _start",
		".align 4",
		"_start:",
		"bl _main",
		"mov x16, #1",
		"svc #0",
	}

	for _, s := range mustContain {
		if !strings.Contains(output, s) {
			t.Errorf("expected %q in output:\n%s", s, output)
		}
	}
}

func TestEmitLegacyProgramEntry(t *testing.T) {
	e := New()
	output := e.EmitLegacyProgramEntry()

	mustContain := []string{
		".global _start",
		".align 4",
		"_start:",
		"b main",
	}

	for _, s := range mustContain {
		if !strings.Contains(output, s) {
			t.Errorf("expected %q in output:\n%s", s, output)
		}
	}

	// Should NOT have bl (branch-link), just b (branch)
	if strings.Contains(output, "bl main") {
		t.Errorf("expected 'b main' not 'bl main' in output:\n%s", output)
	}
}

func TestEmitFunctionLabel(t *testing.T) {
	e := New()

	tests := []struct {
		name     string
		expected string
	}{
		{"main", "_main:"},
		{"foo", "_foo:"},
		{"myFunc", "_myFunc:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := e.EmitFunctionLabel(tt.name)

			if !strings.Contains(output, ".align 4") {
				t.Errorf("expected .align 4 in output:\n%s", output)
			}
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output:\n%s", tt.expected, output)
			}
		})
	}
}

// =============================================================================
// Function Prologue/Epilogue Tests
// =============================================================================

func TestEmitFunctionPrologue(t *testing.T) {
	e := New()

	t.Run("no stack allocation", func(t *testing.T) {
		output := e.EmitFunctionPrologue(0)

		mustContain := []string{
			"stp x29, x30, [sp, #-16]!",
			"mov x29, sp",
		}
		mustNotContain := []string{"sub sp, sp"}

		for _, s := range mustContain {
			if !strings.Contains(output, s) {
				t.Errorf("expected %q in output:\n%s", s, output)
			}
		}
		for _, s := range mustNotContain {
			if strings.Contains(output, s) {
				t.Errorf("did not expect %q in output:\n%s", s, output)
			}
		}
	})

	t.Run("with stack allocation", func(t *testing.T) {
		output := e.EmitFunctionPrologue(48)

		mustContain := []string{
			"stp x29, x30, [sp, #-16]!",
			"mov x29, sp",
			"sub sp, sp, #48",
		}

		for _, s := range mustContain {
			if !strings.Contains(output, s) {
				t.Errorf("expected %q in output:\n%s", s, output)
			}
		}
	})
}

func TestEmitFunctionEpilogue(t *testing.T) {
	e := New()

	t.Run("with locals", func(t *testing.T) {
		output := e.EmitFunctionEpilogue(true)

		mustContain := []string{
			"mov sp, x29",
			"ldp x29, x30, [sp], #16",
			"ret",
		}

		for _, s := range mustContain {
			if !strings.Contains(output, s) {
				t.Errorf("expected %q in output:\n%s", s, output)
			}
		}
	})

	t.Run("without locals", func(t *testing.T) {
		output := e.EmitFunctionEpilogue(false)

		mustContain := []string{
			"ldp x29, x30, [sp], #16",
			"ret",
		}
		mustNotContain := []string{"mov sp, x29"}

		for _, s := range mustContain {
			if !strings.Contains(output, s) {
				t.Errorf("expected %q in output:\n%s", s, output)
			}
		}
		for _, s := range mustNotContain {
			if strings.Contains(output, s) {
				t.Errorf("did not expect %q in output:\n%s", s, output)
			}
		}
	})
}

func TestEmitReturnEpilogue(t *testing.T) {
	e := New()
	output := e.EmitReturnEpilogue()

	mustContain := []string{
		"mov sp, x29",
		"ldp x29, x30, [sp], #16",
		"ret",
	}

	for _, s := range mustContain {
		if !strings.Contains(output, s) {
			t.Errorf("expected %q in output:\n%s", s, output)
		}
	}
}

// =============================================================================
// Integer Operation Tests
// =============================================================================

func TestEmitIntOp_Arithmetic(t *testing.T) {
	e := New()

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
			output, err := e.EmitIntOp(tt.op, tt.signed)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected %q in output:\n%s", exp, output)
				}
			}
		})
	}
}

func TestEmitIntOp_Comparison_Signed(t *testing.T) {
	e := New()

	tests := []struct {
		op       string
		condCode string
	}{
		{"==", "eq"},
		{"!=", "ne"},
		{"<", "lt"},
		{">", "gt"},
		{"<=", "le"},
		{">=", "ge"},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			output, err := e.EmitIntOp(tt.op, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(output, "cmp x0, x1") {
				t.Errorf("expected 'cmp x0, x1' in output:\n%s", output)
			}
			expected := "cset x2, " + tt.condCode
			if !strings.Contains(output, expected) {
				t.Errorf("expected %q in output:\n%s", expected, output)
			}
		})
	}
}

func TestEmitIntOp_Comparison_Unsigned(t *testing.T) {
	e := New()

	tests := []struct {
		op       string
		condCode string
	}{
		{"==", "eq"}, // same for signed/unsigned
		{"!=", "ne"}, // same for signed/unsigned
		{"<", "lo"},  // unsigned: lower
		{">", "hi"},  // unsigned: higher
		{"<=", "ls"}, // unsigned: lower or same
		{">=", "hs"}, // unsigned: higher or same
	}

	for _, tt := range tests {
		t.Run(tt.op+"_unsigned", func(t *testing.T) {
			output, err := e.EmitIntOp(tt.op, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(output, "cmp x0, x1") {
				t.Errorf("expected 'cmp x0, x1' in output:\n%s", output)
			}
			expected := "cset x2, " + tt.condCode
			if !strings.Contains(output, expected) {
				t.Errorf("expected %q in output:\n%s", expected, output)
			}
		})
	}
}

func TestEmitIntOp_Unsupported(t *testing.T) {
	e := New()

	unsupported := []string{"&", "|", "^", "<<", ">>", "&&", "||", "invalid"}

	for _, op := range unsupported {
		t.Run(op, func(t *testing.T) {
			_, err := e.EmitIntOp(op, true)
			if err == nil {
				t.Errorf("expected error for unsupported operation %q", op)
			}
			if !strings.Contains(err.Error(), "unsupported") {
				t.Errorf("error should mention 'unsupported', got: %v", err)
			}
		})
	}
}

// =============================================================================
// Float Operation Tests
// =============================================================================

func TestEmitFloatOp_Arithmetic(t *testing.T) {
	e := New()

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
			output, err := e.EmitFloatOp(tt.op)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output:\n%s", tt.expected, output)
			}
		})
	}
}

func TestEmitFloatOp_Comparison(t *testing.T) {
	e := New()

	tests := []struct {
		op       string
		condCode string
	}{
		{"==", "eq"},
		{"!=", "ne"},
		{"<", "mi"},
		{">", "gt"},
		{"<=", "ls"},
		{">=", "ge"},
	}

	for _, tt := range tests {
		t.Run(tt.op, func(t *testing.T) {
			output, err := e.EmitFloatOp(tt.op)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !strings.Contains(output, "fcmp d1, d0") {
				t.Errorf("expected 'fcmp d1, d0' in output:\n%s", output)
			}
			expected := "cset x2, " + tt.condCode
			if !strings.Contains(output, expected) {
				t.Errorf("expected %q in output:\n%s", expected, output)
			}
		})
	}
}

func TestEmitFloatOp_Unsupported(t *testing.T) {
	e := New()

	unsupported := []string{"%", "&", "|", "^", "invalid"}

	for _, op := range unsupported {
		t.Run(op, func(t *testing.T) {
			_, err := e.EmitFloatOp(op)
			if err == nil {
				t.Errorf("expected error for unsupported operation %q", op)
			}
			if !strings.Contains(err.Error(), "unsupported") {
				t.Errorf("error should mention 'unsupported', got: %v", err)
			}
		})
	}
}

// =============================================================================
// Register and Memory Operation Tests
// =============================================================================

func TestEmitMoveReg(t *testing.T) {
	e := New()

	tests := []struct {
		dst, src string
		expected string
	}{
		{"x0", "x2", "mov x0, x2"},
		{"x1", "x0", "mov x1, x0"},
		{"d0", "d1", "mov d0, d1"},
	}

	for _, tt := range tests {
		t.Run(tt.dst+"_"+tt.src, func(t *testing.T) {
			output := e.EmitMoveReg(tt.dst, tt.src)
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output:\n%s", tt.expected, output)
			}
		})
	}
}

func TestEmitMoveImm(t *testing.T) {
	e := New()

	tests := []struct {
		reg, value string
		expected   string
	}{
		{"x0", "0", "mov x0, #0"},
		{"x2", "42", "mov x2, #42"},
		{"x1", "255", "mov x1, #255"},
	}

	for _, tt := range tests {
		t.Run(tt.reg+"_"+tt.value, func(t *testing.T) {
			output := e.EmitMoveImm(tt.reg, tt.value)
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output:\n%s", tt.expected, output)
			}
		})
	}
}

func TestEmitStoreToStack(t *testing.T) {
	e := New()

	tests := []struct {
		reg      string
		offset   int
		expected string
	}{
		{"x2", 16, "str x2, [x29, #-16]"},
		{"x0", 32, "str x0, [x29, #-32]"},
		{"x5", 48, "str x5, [x29, #-48]"},
	}

	for _, tt := range tests {
		t.Run(tt.reg, func(t *testing.T) {
			output := e.EmitStoreToStack(tt.reg, tt.offset)
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output:\n%s", tt.expected, output)
			}
		})
	}
}

func TestEmitLoadFromStack(t *testing.T) {
	e := New()

	tests := []struct {
		reg      string
		offset   int
		expected string
	}{
		{"x2", 16, "ldr x2, [x29, #-16]"},
		{"x0", 32, "ldr x0, [x29, #-32]"},
		{"x5", 48, "ldr x5, [x29, #-48]"},
	}

	for _, tt := range tests {
		t.Run(tt.reg, func(t *testing.T) {
			output := e.EmitLoadFromStack(tt.reg, tt.offset)
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output:\n%s", tt.expected, output)
			}
		})
	}
}

func TestEmitPushToStack(t *testing.T) {
	e := New()

	tests := []struct {
		reg      string
		expected string
	}{
		{"x2", "str x2, [sp, #-16]!"},
		{"x0", "str x0, [sp, #-16]!"},
	}

	for _, tt := range tests {
		t.Run(tt.reg, func(t *testing.T) {
			output := e.EmitPushToStack(tt.reg)
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output:\n%s", tt.expected, output)
			}
		})
	}
}

func TestEmitPopFromStack(t *testing.T) {
	e := New()

	tests := []struct {
		reg      string
		expected string
	}{
		{"x0", "ldr x0, [sp], #16"},
		{"x2", "ldr x2, [sp], #16"},
	}

	for _, tt := range tests {
		t.Run(tt.reg, func(t *testing.T) {
			output := e.EmitPopFromStack(tt.reg)
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output:\n%s", tt.expected, output)
			}
		})
	}
}

func TestEmitLoadAddress(t *testing.T) {
	e := New()

	tests := []struct {
		reg, label string
	}{
		{"x1", "str_0"},
		{"x0", "buffer"},
		{"x8", "float_1"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			output := e.EmitLoadAddress(tt.reg, tt.label)

			// ARM64 uses adrp/add pattern for address loading
			expectedAdrp := "adrp " + tt.reg + ", " + tt.label + "@PAGE"
			expectedAdd := "add " + tt.reg + ", " + tt.reg + ", " + tt.label + "@PAGEOFF"

			if !strings.Contains(output, expectedAdrp) {
				t.Errorf("expected %q in output:\n%s", expectedAdrp, output)
			}
			if !strings.Contains(output, expectedAdd) {
				t.Errorf("expected %q in output:\n%s", expectedAdd, output)
			}
		})
	}
}

func TestEmitBranchLink(t *testing.T) {
	e := New()

	tests := []struct {
		label    string
		expected string
	}{
		{"main", "bl _main"},
		{"foo", "bl _foo"},
		{"helper", "bl _helper"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			output := e.EmitBranchLink(tt.label)
			if !strings.Contains(output, tt.expected) {
				t.Errorf("expected %q in output:\n%s", tt.expected, output)
			}
		})
	}
}

// =============================================================================
// Syscall Tests
// =============================================================================

func TestEmitExitSyscall(t *testing.T) {
	e := New()
	output := e.EmitExitSyscall()

	mustContain := []string{
		"mov x0, x2",  // exit code from x2 to x0
		"mov x16, #1", // syscall number for exit
		"svc #0",      // supervisor call
	}

	for _, s := range mustContain {
		if !strings.Contains(output, s) {
			t.Errorf("expected %q in output:\n%s", s, output)
		}
	}
}

func TestEmitWriteSyscall(t *testing.T) {
	e := New()

	t.Run("standard registers", func(t *testing.T) {
		output := e.EmitWriteSyscall("x10", "x11")

		mustContain := []string{
			"mov x2, x11", // length
			"mov x1, x10", // buffer
			"mov x0, #1",  // stdout
			"mov x16, #4", // syscall number for write
			"svc #0x80",   // supervisor call (macOS)
		}

		for _, s := range mustContain {
			if !strings.Contains(output, s) {
				t.Errorf("expected %q in output:\n%s", s, output)
			}
		}
	})

	t.Run("different registers", func(t *testing.T) {
		output := e.EmitWriteSyscall("x0", "x1")

		if !strings.Contains(output, "mov x1, x0") {
			t.Errorf("expected buffer register move in output:\n%s", output)
		}
		if !strings.Contains(output, "mov x2, x1") {
			t.Errorf("expected length register move in output:\n%s", output)
		}
	})
}

func TestEmitNewline(t *testing.T) {
	e := New()
	output := e.EmitNewline()

	mustContain := []string{
		"adrp x1, newline@PAGE",
		"add x1, x1, newline@PAGEOFF",
		"mov x2, #1",  // length = 1
		"mov x0, #1",  // stdout
		"mov x16, #4", // write syscall
		"svc #0x80",
	}

	for _, s := range mustContain {
		if !strings.Contains(output, s) {
			t.Errorf("expected %q in output:\n%s", s, output)
		}
	}
}

func TestEmitPrintInt(t *testing.T) {
	e := New()
	output := e.EmitPrintInt()

	mustContain := []string{
		"mov x0, x2",      // value to print
		"bl int_to_string", // call conversion routine
		"mov x16, #4",     // write syscall
		"svc #0x80",       // supervisor call
	}

	for _, s := range mustContain {
		if !strings.Contains(output, s) {
			t.Errorf("expected %q in output:\n%s", s, output)
		}
	}
}

// =============================================================================
// Int-to-String Function Test
// =============================================================================

func TestIntToStringFunction(t *testing.T) {
	e := New()
	output := e.IntToStringFunction()

	// Verify it's a complete function
	mustContain := []string{
		"int_to_string:",             // function label
		"stp x29, x30, [sp, #-16]!",  // prologue
		"mov x29, sp",
		"buffer@PAGE",                // uses buffer
		"check_negative:",            // handles negative numbers
		"convert_loop:",              // digit conversion loop
		"finalize:",                  // finalization
		"restore_regs:",              // cleanup
		"ret",                        // return
	}

	for _, s := range mustContain {
		if !strings.Contains(output, s) {
			t.Errorf("expected %q in output:\n%s", s, output)
		}
	}

	// Verify it handles zero case
	if !strings.Contains(output, "cmp x20, #0") {
		t.Errorf("expected zero check in output:\n%s", output)
	}

	// Verify digit extraction (divide by 10)
	if !strings.Contains(output, "mov x10, #10") {
		t.Errorf("expected division by 10 in output:\n%s", output)
	}
}

// =============================================================================
// Edge Cases and Integration
// =============================================================================

func TestEmitterConsistency(t *testing.T) {
	e := New()

	// Verify that the result register matches what operations use
	t.Run("result register consistency", func(t *testing.T) {
		resultReg := e.ResultReg()
		addOp, _ := e.EmitIntOp("+", true)

		// The add operation should store result in the result register
		expected := "add " + resultReg
		if !strings.Contains(addOp, expected) {
			t.Errorf("IntOp doesn't use ResultReg. Got:\n%s\nExpected to contain: %s", addOp, expected)
		}
	})

	// Verify that left/right registers match operation expectations
	t.Run("operand register consistency", func(t *testing.T) {
		leftReg := e.LeftReg()
		rightReg := e.RightReg()
		addOp, _ := e.EmitIntOp("+", true)

		expected := leftReg + ", " + rightReg
		if !strings.Contains(addOp, expected) {
			t.Errorf("IntOp doesn't use Left/Right regs. Got:\n%s\nExpected to contain: %s", addOp, expected)
		}
	})

	// Verify float register consistency
	t.Run("float register consistency", func(t *testing.T) {
		floatLeft := e.FloatLeftReg()
		floatRight := e.FloatRightReg()
		faddOp, _ := e.EmitFloatOp("+")

		expected := floatLeft + ", " + floatRight
		if !strings.Contains(faddOp, expected) {
			t.Errorf("FloatOp doesn't use Float Left/Right regs. Got:\n%s\nExpected to contain: %s", faddOp, expected)
		}
	})
}
