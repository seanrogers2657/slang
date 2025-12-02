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

func TestEndToEnd_WritebackAddressing(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "stp_preindex_ldp_postindex",
			assembly: `.global _start
_start:
    // Test pre-indexed STP and post-indexed LDP
    mov x0, #10
    mov x1, #20

    // Pre-indexed: sp = sp - 16, then store
    stp x0, x1, [sp, #-16]!

    // Clear registers to prove we restore from stack
    mov x0, #0
    mov x1, #0

    // Post-indexed: load, then sp = sp + 16
    ldp x0, x1, [sp], #16

    // x0 + x1 should be 30
    add x0, x0, x1
    mov x16, #1
    svc #0
`,
			expectedExit: 30,
		},
		{
			name: "nested_preindex_postindex",
			assembly: `.global _start
_start:
    mov x0, #5
    mov x1, #3

    // Push two pairs using pre-indexed
    stp x0, x1, [sp, #-16]!
    mov x2, #7
    mov x3, #11
    stp x2, x3, [sp, #-16]!

    // Clear all
    mov x0, #0
    mov x1, #0
    mov x2, #0
    mov x3, #0

    // Pop in reverse order using post-indexed
    ldp x2, x3, [sp], #16
    ldp x0, x1, [sp], #16

    // x0=5, x1=3, x2=7, x3=11
    // Return x0 + x2 = 12
    add x0, x0, x2
    mov x16, #1
    svc #0
`,
			expectedExit: 12,
		},
		{
			name: "function_with_preindex_postindex",
			assembly: `.global _start
_start:
    mov x0, #7
    bl double_it
    mov x16, #1
    svc #0

double_it:
    // Save link register with pre-indexed
    stp x29, x30, [sp, #-16]!
    mov x29, sp

    add x0, x0, x0

    // Restore with post-indexed
    ldp x29, x30, [sp], #16
    ret
`,
			expectedExit: 14,
		},
		{
			name: "multiple_callee_saved_registers",
			assembly: `.global _start
_start:
    mov x0, #2
    bl compute_sum
    mov x16, #1
    svc #0

compute_sum:
    // Save frame pointer and link register
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    // Save callee-saved registers
    stp x19, x20, [sp, #-16]!
    stp x21, x22, [sp, #-16]!

    // Use callee-saved registers
    mov x19, x0       // x19 = 2
    add x20, x19, #3  // x20 = 5
    add x21, x20, #7  // x21 = 12
    add x22, x21, #8  // x22 = 20

    // Result is sum of all
    add x0, x19, x20
    add x0, x0, x21
    add x0, x0, x22   // 2 + 5 + 12 + 20 = 39

    // Restore callee-saved registers (post-indexed)
    ldp x21, x22, [sp], #16
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    ret
`,
			expectedExit: 39,
		},
		{
			name: "preindex_negative_offset",
			assembly: `.global _start
_start:
    mov x0, #42
    mov x1, #0

    // Pre-indexed with negative offset (push pattern)
    stp x0, x1, [sp, #-16]!

    // Overwrite x0
    mov x0, #99

    // Post-indexed restore (pop pattern)
    ldp x0, x1, [sp], #16

    // x0 should be restored to 42
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "recursive_function_with_writeback",
			assembly: `.global _start
_start:
    mov x0, #4
    bl countdown
    mov x16, #1
    svc #0

// countdown: returns sum of n + (n-1) + ... + 1
countdown:
    stp x29, x30, [sp, #-16]!
    mov x29, sp
    stp x19, x20, [sp, #-16]!

    mov x19, x0           // save n

    cmp x0, #1
    b.le base_case

    sub x0, x0, #1        // n - 1
    bl countdown          // recursive call
    add x0, x0, x19       // result + n
    b epilogue

base_case:
    mov x0, #1

epilogue:
    ldp x19, x20, [sp], #16
    ldp x29, x30, [sp], #16
    ret
`,
			expectedExit: 10, // 4 + 3 + 2 + 1 = 10
		},
	}
	runE2ETests(t, tests)
}

func TestEndToEnd_Division(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "udiv_simple",
			assembly: `.global _start
_start:
    mov x0, #100
    mov x1, #20
    udiv x2, x0, x1     // x2 = 100 / 20 = 5
    mov x0, x2
    mov x16, #1
    svc #0
`,
			expectedExit: 5,
		},
		{
			name: "udiv_with_remainder",
			assembly: `.global _start
_start:
    mov x0, #47
    mov x1, #10
    udiv x2, x0, x1     // x2 = 47 / 10 = 4
    mov x0, x2
    mov x16, #1
    svc #0
`,
			expectedExit: 4,
		},
		{
			name: "modulo_using_msub",
			assembly: `.global _start
_start:
    // Compute 47 % 10 = 7
    mov x0, #47         // dividend
    mov x1, #10         // divisor
    udiv x2, x0, x1     // x2 = 47 / 10 = 4
    msub x3, x2, x1, x0 // x3 = x0 - (x2 * x1) = 47 - 40 = 7
    mov x0, x3
    mov x16, #1
    svc #0
`,
			expectedExit: 7,
		},
		{
			name: "sdiv_positive",
			assembly: `.global _start
_start:
    mov x0, #42
    mov x1, #6
    sdiv x2, x0, x1     // x2 = 42 / 6 = 7
    mov x0, x2
    mov x16, #1
    svc #0
`,
			expectedExit: 7,
		},
	}
	runE2ETests(t, tests)
}

func TestEndToEnd_SingleRegisterIndexed(t *testing.T) {
	tests := []e2eTestCase{
		{
			name: "str_ldr_preindex",
			assembly: `.global _start
_start:
    mov x0, #42

    // Pre-indexed store: decrement sp, then store
    str x0, [sp, #-16]!

    // Overwrite x0
    mov x0, #99

    // Pre-indexed load: load, then increment sp
    ldr x0, [sp], #16

    // x0 should be restored to 42
    mov x16, #1
    svc #0
`,
			expectedExit: 42,
		},
		{
			name: "str_ldr_postindex",
			assembly: `.global _start
_start:
    // Allocate stack space
    sub sp, sp, #16
    mov x0, #55

    // Store at current sp
    str x0, [sp]

    // Overwrite x0
    mov x0, #0

    // Load from sp, then deallocate
    ldr x0, [sp], #16

    // x0 should be 55
    mov x16, #1
    svc #0
`,
			expectedExit: 55,
		},
		{
			name: "multiple_push_pop_pattern",
			assembly: `.global _start
_start:
    mov x0, #10
    mov x1, #20
    mov x2, #30

    // Push all three values using pre-indexed str
    str x0, [sp, #-16]!
    str x1, [sp, #-16]!
    str x2, [sp, #-16]!

    // Clear registers
    mov x0, #0
    mov x1, #0
    mov x2, #0

    // Pop in reverse order using post-indexed ldr
    ldr x2, [sp], #16   // x2 = 30
    ldr x1, [sp], #16   // x1 = 20
    ldr x0, [sp], #16   // x0 = 10

    // Return x0 + x1 + x2 = 60
    add x0, x0, x1
    add x0, x0, x2
    mov x16, #1
    svc #0
`,
			expectedExit: 60,
		},
	}
	runE2ETests(t, tests)
}
