package slasm

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/seanrogers2657/slang/assembler"
)

// e2eTestCase defines a single end-to-end test
type e2eTestCase struct {
	name         string
	assembly     string
	expectedExit int
	disassemble  bool // set to true to log disassembly output
}

// runE2ETest is a helper that builds, executes, and validates an assembly program
func runE2ETest(t *testing.T, tc e2eTestCase) {
	t.Helper()

	asm := New()
	outputPath := fmt.Sprintf("/tmp/test_slasm_%s", tc.name)
	defer os.Remove(outputPath)

	err := asm.Build(tc.assembly, assembler.BuildOptions{
		OutputPath: outputPath,
	})
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if tc.disassemble {
		otoolCmd := exec.Command("otool", "-tV", outputPath)
		otoolBytes, err := otoolCmd.CombinedOutput()
		if err != nil {
			t.Fatalf("otool -tV failed: %v", err)
		}
		t.Logf("Disassembly:\n%s", string(otoolBytes))
	}

	cmd := exec.Command(outputPath)
	err = cmd.Run()

	actualExit := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		actualExit = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("Failed to execute program: %v", err)
	}

	if actualExit != tc.expectedExit {
		t.Errorf("Expected exit code %d, got %d", tc.expectedExit, actualExit)
	}
}

// runE2ETests runs a slice of test cases as subtests
func runE2ETests(t *testing.T, tests []e2eTestCase) {
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runE2ETest(t, tc)
		})
	}
}

func TestEndToEnd_BasicExitCodes(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "exit_0",
			assembly: `.global _start
_start:
    mov x0, #0
    mov x16, #1
    svc #0
`,
			expectedExit: 0,
		},
		{
			name: "exit_1",
			assembly: `.global _start
_start:
    mov x0, #1
    mov x16, #1
    svc #0
`,
			expectedExit: 1,
		},
		{
			name: "exit_42",
			assembly: `.global _start
_start:
    mov x0, #42
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "exit_255",
			assembly: `.global _start
_start:
    mov x0, #255
    mov x16, #1
    svc #0
`,
			expectedExit: 255,
		},
	}
	runE2ETests(t, tests)
}

func TestEndToEnd_UnconditionalBranch(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "branch_forward",
			assembly: `.global _start
_start:
    b main
skip_this:
    mov x0, #99
main:
    mov x0, #25
    mov x16, #1
    svc #0
`,
			expectedExit: 25,
		},
		{
			name: "branch_backward_loop",
			assembly: `.global _start
_start:
    mov x0, #0
    mov x1, #3
loop:
    add x0, x0, #1
    sub x1, x1, #1
    cmp x1, #0
    b.ne loop
    mov x16, #1
    svc #0
`,
			expectedExit: 3,
		},
	}
	runE2ETests(t, tests)
}

func TestEndToEnd_ConditionalBranch(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "b.eq_taken",
			assembly: `.global _start
_start:
    mov x0, #5
    mov x1, #5
    cmp x0, x1
    b.eq equal
    mov x0, #0
    b done
equal:
    mov x0, #42
done:
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "b.eq_not_taken",
			assembly: `.global _start
_start:
    mov x0, #5
    mov x1, #10
    cmp x0, x1
    b.eq equal
    mov x0, #99
    b done
equal:
    mov x0, #42
done:
    mov x16, #1
    svc #0
`,
			expectedExit: 99,
		},
		{
			name: "b.ne_taken",
			assembly: `.global _start
_start:
    mov x0, #5
    mov x1, #10
    cmp x0, x1
    b.ne not_equal
    mov x0, #0
    b done
not_equal:
    mov x0, #42
done:
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "b.lt_taken",
			assembly: `.global _start
_start:
    mov x0, #5
    mov x1, #10
    cmp x0, x1
    b.lt less_than
    mov x0, #0
    b done
less_than:
    mov x0, #42
done:
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "b.gt_taken",
			assembly: `.global _start
_start:
    mov x0, #10
    mov x1, #5
    cmp x0, x1
    b.gt greater_than
    mov x0, #0
    b done
greater_than:
    mov x0, #42
done:
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "b.le_taken_equal",
			assembly: `.global _start
_start:
    mov x0, #5
    mov x1, #5
    cmp x0, x1
    b.le less_or_equal
    mov x0, #0
    b done
less_or_equal:
    mov x0, #42
done:
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "b.ge_taken_equal",
			assembly: `.global _start
_start:
    mov x0, #5
    mov x1, #5
    cmp x0, x1
    b.ge greater_or_equal
    mov x0, #0
    b done
greater_or_equal:
    mov x0, #42
done:
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
	}
	runE2ETests(t, tests)
}

func TestEndToEnd_BranchLink(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "bl_and_ret",
			assembly: `.global _start
_start:
    mov x0, #5
    bl add_five
    mov x16, #1
    svc #0

add_five:
    add x0, x0, #5
    ret
`,
			expectedExit: 10,
		},
		{
			name: "nested_function_calls",
			assembly: `.global _start
_start:
    mov x0, #2
    bl double
    mov x16, #1
    svc #0

double:
    stp x29, x30, [sp]
    bl add_self
    ldp x29, x30, [sp]
    ret

add_self:
    add x0, x0, x0
    ret
`,
			expectedExit: 4,
		},
	}
	runE2ETests(t, tests)
}

func TestEndToEnd_MemoryOperations(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "str_ldr_simple",
			assembly: `.global _start
_start:
    mov x0, #42
    str x0, [sp]
    mov x0, #0
    ldr x0, [sp]
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "str_ldr_with_offset",
			assembly: `.global _start
_start:
    mov x0, #3
    str x0, [sp]
    mov x0, #7
    str x0, [sp, #8]
    ldr x1, [sp]
    ldr x2, [sp, #8]
    add x0, x1, x2
    mov x16, #1
    svc #0
`,
			expectedExit: 10,
		},
		{
			name: "stp_ldp_pair",
			assembly: `.global _start
_start:
    mov x0, #10
    mov x1, #20
    stp x0, x1, [sp]
    mov x0, #0
    mov x1, #0
    ldp x2, x3, [sp]
    add x0, x2, x3
    mov x16, #1
    svc #0
`,
			expectedExit: 30,
		},
		{
			name: "function_with_frame",
			assembly: `.global _start
_start:
    mov x0, #50
    bl multiply_by_two
    mov x16, #1
    svc #0

multiply_by_two:
    stp x29, x30, [sp]
    add x0, x0, x0
    ldp x29, x30, [sp]
    ret
`,
			expectedExit: 100,
		},
		{
			name: "multiple_offsets",
			assembly: `.global _start
_start:
    mov x0, #1
    str x0, [sp]
    mov x0, #2
    str x0, [sp, #8]
    mov x0, #4
    str x0, [sp, #16]
    ldr x1, [sp]
    ldr x2, [sp, #8]
    ldr x3, [sp, #16]
    add x0, x1, x2
    add x0, x0, x3
    mov x16, #1
    svc #0
`,
			expectedExit: 7,
		},
	}
	runE2ETests(t, tests)
}

func TestEndToEnd_Arithmetic(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "add_registers",
			assembly: `.global _start
_start:
    mov x0, #10
    mov x1, #20
    add x0, x0, x1
    mov x16, #1
    svc #0
`,
			expectedExit: 30,
		},
		{
			name: "add_immediate",
			assembly: `.global _start
_start:
    mov x0, #10
    add x0, x0, #5
    mov x16, #1
    svc #0
`,
			expectedExit: 15,
		},
		{
			name: "sub_registers",
			assembly: `.global _start
_start:
    mov x0, #50
    mov x1, #20
    sub x0, x0, x1
    mov x16, #1
    svc #0
`,
			expectedExit: 30,
		},
		{
			name: "sub_immediate",
			assembly: `.global _start
_start:
    mov x0, #50
    sub x0, x0, #8
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "mul_registers",
			assembly: `.global _start
_start:
    mov x0, #7
    mov x1, #6
    mul x0, x0, x1
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "sdiv_registers",
			assembly: `.global _start
_start:
    mov x0, #100
    mov x1, #10
    sdiv x0, x0, x1
    mov x16, #1
    svc #0
`,
			expectedExit: 10,
		},
	}
	runE2ETests(t, tests)
}

func TestEndToEnd_Comparison(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "cmp_and_cset_eq",
			assembly: `.global _start
_start:
    mov x0, #5
    mov x1, #5
    cmp x0, x1
    cset x0, eq
    mov x16, #1
    svc #0
`,
			expectedExit: 1,
		},
		{
			name: "cmp_and_cset_ne",
			assembly: `.global _start
_start:
    mov x0, #5
    mov x1, #10
    cmp x0, x1
    cset x0, ne
    mov x16, #1
    svc #0
`,
			expectedExit: 1,
		},
		{
			name: "cmp_immediate",
			assembly: `.global _start
_start:
    mov x0, #42
    cmp x0, #42
    cset x0, eq
    mov x16, #1
    svc #0
`,
			expectedExit: 1,
		},
	}
	runE2ETests(t, tests)
}

func TestEndToEnd_ComplexPrograms(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "factorial_3",
			assembly: `.global _start
_start:
    mov x0, #3       ; n = 3
    mov x1, #1       ; result = 1
factorial_loop:
    cmp x0, #1
    b.le factorial_done
    mul x1, x1, x0   ; result *= n
    sub x0, x0, #1   ; n--
    b factorial_loop
factorial_done:
    mov x0, x1       ; return result (3! = 6)
    mov x16, #1
    svc #0
`,
			expectedExit: 6,
		},
		{
			name: "fibonacci_6",
			assembly: `.global _start
_start:
    mov x0, #6       ; compute fib(6)
    mov x1, #0       ; fib(0) = 0
    mov x2, #1       ; fib(1) = 1
fib_loop:
    cmp x0, #0
    b.eq fib_done
    add x3, x1, x2   ; next = a + b
    mov x1, x2       ; a = b
    mov x2, x3       ; b = next
    sub x0, x0, #1   ; n--
    b fib_loop
fib_done:
    mov x0, x1       ; return fib(6) = 8
    mov x16, #1
    svc #0
`,
			expectedExit: 8,
		},
		{
			name: "sum_1_to_5",
			assembly: `.global _start
_start:
    mov x0, #0       ; sum = 0
    mov x1, #1       ; i = 1
sum_loop:
    cmp x1, #6       ; while i < 6
    b.ge sum_done
    add x0, x0, x1   ; sum += i
    add x1, x1, #1   ; i++
    b sum_loop
sum_done:
    mov x16, #1      ; sum = 1+2+3+4+5 = 15
    svc #0
`,
			expectedExit: 15,
		},
	}
	runE2ETests(t, tests)
}
