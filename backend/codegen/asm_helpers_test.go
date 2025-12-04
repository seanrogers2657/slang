package codegen

import (
	"strings"
	"testing"
)

func TestEmitFunctionPrologue(t *testing.T) {
	tests := []struct {
		name      string
		stackSize int
		expected  []string
	}{
		{
			name:      "no locals",
			stackSize: 0,
			expected: []string{
				"stp x29, x30, [sp, #-16]!",
				"mov x29, sp",
			},
		},
		{
			name:      "with locals",
			stackSize: 32,
			expected: []string{
				"stp x29, x30, [sp, #-16]!",
				"mov x29, sp",
				"sub sp, sp, #32",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			EmitFunctionPrologue(&builder, tt.stackSize)
			output := builder.String()

			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected %q in output, got:\n%s", exp, output)
				}
			}
		})
	}
}

func TestEmitFunctionEpilogue(t *testing.T) {
	tests := []struct {
		name      string
		hasLocals bool
		expected  []string
		notExpect []string
	}{
		{
			name:      "with locals",
			hasLocals: true,
			expected: []string{
				"mov sp, x29",
				"ldp x29, x30, [sp], #16",
				"ret",
			},
		},
		{
			name:      "no locals",
			hasLocals: false,
			expected: []string{
				"ldp x29, x30, [sp], #16",
				"ret",
			},
			notExpect: []string{"mov sp, x29"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			EmitFunctionEpilogue(&builder, tt.hasLocals)
			output := builder.String()

			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected %q in output, got:\n%s", exp, output)
				}
			}
			for _, notExp := range tt.notExpect {
				if strings.Contains(output, notExp) {
					t.Errorf("did not expect %q in output, got:\n%s", notExp, output)
				}
			}
		})
	}
}

func TestEmitReturnEpilogue(t *testing.T) {
	var builder strings.Builder
	EmitReturnEpilogue(&builder)
	output := builder.String()

	expected := []string{
		"mov sp, x29",
		"ldp x29, x30, [sp], #16",
		"ret",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitExitSyscall(t *testing.T) {
	var builder strings.Builder
	EmitExitSyscall(&builder)
	output := builder.String()

	expected := []string{
		"mov x0, x2",
		"mov x16, #1",
		"svc #0",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitWriteSyscall(t *testing.T) {
	var builder strings.Builder
	EmitWriteSyscall(&builder, "x10", "x11")
	output := builder.String()

	expected := []string{
		"mov x2, x11",
		"mov x1, x10",
		"mov x0, #1",
		"mov x16, #4",
		"svc #0x80",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitNewline(t *testing.T) {
	var builder strings.Builder
	EmitNewline(&builder)
	output := builder.String()

	expected := []string{
		"adrp x1, newline@PAGE",
		"add x1, x1, newline@PAGEOFF",
		"mov x2, #1",
		"mov x0, #1",
		"mov x16, #4",
		"svc #0x80",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitPrintInt(t *testing.T) {
	var builder strings.Builder
	EmitPrintInt(&builder)
	output := builder.String()

	expected := []string{
		"mov x0, x2",
		"bl int_to_string",
		"mov x16, #4",
		"svc #0x80",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitDataSection(t *testing.T) {
	tests := []struct {
		name     string
		hasPrint bool
		expected []string
	}{
		{
			name:     "without print",
			hasPrint: false,
			expected: []string{
				".data",
				".align 3",
			},
		},
		{
			name:     "with print",
			hasPrint: true,
			expected: []string{
				".data",
				".align 3",
				"buffer: .space 32",
				"newline: .byte 10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			EmitDataSection(&builder, tt.hasPrint)
			output := builder.String()

			for _, exp := range tt.expected {
				if !strings.Contains(output, exp) {
					t.Errorf("expected %q in output, got:\n%s", exp, output)
				}
			}
		})
	}
}

func TestEmitProgramEntry(t *testing.T) {
	var builder strings.Builder
	EmitProgramEntry(&builder)
	output := builder.String()

	expected := []string{
		".global _start",
		".align 4",
		"_start:",
		"bl _main",
		"mov x16, #1",
		"svc #0",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitLegacyProgramEntry(t *testing.T) {
	var builder strings.Builder
	EmitLegacyProgramEntry(&builder)
	output := builder.String()

	expected := []string{
		".global _start",
		".align 4",
		"_start:",
		"b main",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitFunctionLabel(t *testing.T) {
	var builder strings.Builder
	EmitFunctionLabel(&builder, "myFunc")
	output := builder.String()

	expected := []string{
		".align 4",
		"_myFunc:",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEmitStoreToStack(t *testing.T) {
	var builder strings.Builder
	EmitStoreToStack(&builder, "x5", 32)
	output := builder.String()

	if !strings.Contains(output, "str x5, [x29, #-32]") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestEmitLoadFromStack(t *testing.T) {
	var builder strings.Builder
	EmitLoadFromStack(&builder, "x3", 16)
	output := builder.String()

	if !strings.Contains(output, "ldr x3, [x29, #-16]") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestEmitPushToStack(t *testing.T) {
	var builder strings.Builder
	EmitPushToStack(&builder, "x2")
	output := builder.String()

	if !strings.Contains(output, "str x2, [sp, #-16]!") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestEmitPopFromStack(t *testing.T) {
	var builder strings.Builder
	EmitPopFromStack(&builder, "x0")
	output := builder.String()

	if !strings.Contains(output, "ldr x0, [sp], #16") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestEmitMoveReg(t *testing.T) {
	var builder strings.Builder
	EmitMoveReg(&builder, "x0", "x2")
	output := builder.String()

	if !strings.Contains(output, "mov x0, x2") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestEmitMoveImm(t *testing.T) {
	var builder strings.Builder
	EmitMoveImm(&builder, "x2", "42")
	output := builder.String()

	if !strings.Contains(output, "mov x2, #42") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestEmitBranchLink(t *testing.T) {
	var builder strings.Builder
	EmitBranchLink(&builder, "myFunc")
	output := builder.String()

	if !strings.Contains(output, "bl _myFunc") {
		t.Errorf("unexpected output: %s", output)
	}
}

func TestEmitLoadAddress(t *testing.T) {
	var builder strings.Builder
	EmitLoadAddress(&builder, "x1", "str_0")
	output := builder.String()

	expected := []string{
		"adrp x1, str_0@PAGE",
		"add x1, x1, str_0@PAGEOFF",
	}

	for _, exp := range expected {
		if !strings.Contains(output, exp) {
			t.Errorf("expected %q in output, got:\n%s", exp, output)
		}
	}
}

func TestEscapeStringForAsm(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{"hello\nworld", "hello\\nworld"},
		{"tab\there", "tab\\there"},
		{"quote\"here", "quote\\\"here"},
		{"back\\slash", "back\\\\slash"},
		{"carriage\rreturn", "carriage\\rreturn"},
		{"mixed\n\t\"\\", "mixed\\n\\t\\\"\\\\"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := EscapeStringForAsm(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeStringForAsm(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
