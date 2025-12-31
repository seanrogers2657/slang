# Status

IMPLEMENTED, 2024-12-31

# Summary/Motivation

Add runtime boundary checking for all arithmetic operations with panic-on-violation semantics. When an overflow, underflow, or division by zero occurs, the program panics with a descriptive error message and stack trace showing source locations (file:line) for debugging.

# Goals/Non-Goals

- [goal] Detect signed integer overflow for addition, subtraction, multiplication
- [goal] Detect unsigned integer overflow/underflow for addition, subtraction, multiplication
- [goal] Detect division by zero and modulo by zero
- [goal] Provide panic messages with error type and stack trace
- [goal] Include source file and line number in stack traces
- [goal] Minimal hot-path overhead (1-4 instructions per checked operation)
- [non-goal] Array bounds checking (reserved for future)
- [non-goal] Null pointer checking (reserved for future)
- [non-goal] Stack overflow detection (reserved for future)
- [non-goal] Recoverable errors (all violations are fatal panics)

# APIs

- `_slang_panic` - Runtime panic handler that prints error message and stack trace, then exits with code 1.
- `_slang_symtab` - Symbol table mapping return addresses to function names and source locations.
- `_slang_symtab_lookup` - Looks up a return address in the symbol table.

**Error Codes:**

| Code | Constant | Message |
|------|----------|---------|
| 1 | `ErrOverflowAddSigned` | "integer overflow: addition" |
| 2 | `ErrOverflowSubSigned` | "integer overflow: subtraction" |
| 3 | `ErrOverflowMulSigned` | "integer overflow: multiplication" |
| 4 | `ErrOverflowAddUnsigned` | "unsigned overflow: addition" |
| 5 | `ErrUnderflowSubUnsigned` | "unsigned underflow: subtraction" |
| 6 | `ErrOverflowMulUnsigned` | "unsigned overflow: multiplication" |
| 7 | `ErrDivByZero` | "division by zero" |
| 8 | `ErrModByZero` | "modulo by zero" |

**Panic Message Format:**

```
panic: <error message>
    at <function>() <file>:<line>
    at <function>() <file>:<line>
    ...
```

# Description

## Step 1: Runtime Package

Create runtime support files:

- `compiler/runtime/errors.go` - Error code constants and message table
- `compiler/runtime/panic.go` - Embedded ARM64 panic handler assembly

The panic handler is embedded directly into the generated assembly rather than linked as a separate library.

## Step 2: Symbol Table Generation

The compiler generates a symbol table in the `.data` section mapping return addresses to function info:

```asm
.data
.align 3
_slang_symtab:
    .quad _fn_divide           // function start address
    .quad _fn_divide_end       // function end address
    .quad .Lname_divide        // pointer to name string
    .quad .Lfile_math          // pointer to filename string
    .quad 12                   // start line number
    .quad 0                    // sentinel (null terminator)

.Lname_divide:    .asciz "divide"
.Lfile_math:      .asciz "math.sl"
```

## Step 3: Frame Pointer Chain

Functions preserve frame pointers with the standard prologue:

```asm
stp x29, x30, [sp, #-16]!   // save frame pointer and return address
mov x29, sp                  // set new frame pointer
```

This creates a linked list on the stack that can be walked to build stack traces.

## Step 4: Check Generation

Each arithmetic operation emits a check after the computation. The check uses a forward branch to skip over the panic code on the happy path:

**Signed Addition:**
```asm
adds x2, x0, x1           // add with flags
b.vs .Lpanic_{id}         // branch on signed overflow
// ... continue ...
.Lpanic_{id}:
    mov x0, #1            // ErrOverflowAddSigned
    bl _slang_panic
```

**Signed Multiplication:**
```asm
mul x2, x0, x1            // low 64 bits
smulh x3, x0, x1          // high 64 bits
cmp x3, x2, asr #63       // compare high with sign extension
b.ne .Lpanic_{id}         // overflow if mismatch
```

**Division by Zero:**
```asm
cbz x1, .Lpanic_{id}      // check divisor before divide
sdiv x2, x0, x1
```

## Step 5: Stack Walking

The panic handler walks the frame pointer chain to build a stack trace:

1. Start with current frame pointer (x29)
2. For each frame:
   - Load return address from frame_ptr + 8
   - Look up return address in symbol table
   - If found, print "    at \<name\>() \<file\>:\<line\>"
   - Follow chain: frame_ptr = *frame_ptr
3. Stop when frame_ptr is null

## File Changes Summary

| File | Change |
|------|--------|
| `compiler/runtime/errors.go` | Error codes and messages |
| `compiler/runtime/panic.go` | Embedded panic handler assembly |
| `compiler/codegen/checks.go` | Check generation helpers |
| `compiler/codegen/symtab.go` | Symbol table generation |
| `compiler/codegen/typed_codegen.go` | Emit checks and function end labels |
| `assembler/slasm/encoder.go` | Support for shifted register CMP |
| `assembler/slasm/macho.go` | Chained fixups for data pointers |

# Alternatives

1. **No runtime checks**: Undefined behavior on overflow like C. Rejected because silent corruption is worse than crashing.

2. **Saturating arithmetic**: Clamp to min/max on overflow. Rejected because it hides bugs and changes program semantics.

3. **Wrapping arithmetic**: Two's complement wrap-around. Could be offered as an opt-in mode, but default should be checked.

4. **Exception-based handling**: Throw/catch for overflow. Rejected for MVP complexity; panic is simpler and matches Rust/Go philosophy.

5. **Return error codes**: Return (value, error) tuples. Rejected because it clutters every arithmetic expression.

6. **Compile-time only**: Prove absence of overflow statically. Too complex for MVP; would require dependent types or SMT solving.

# Testing

- **Unit tests**: Verify check generation produces correct ARM64 assembly patterns
- **E2E tests**: Each error type has a test file in `_examples/slang/runtime/` that triggers it and verifies the panic message
- **Negative tests**: Verify operations that don't overflow still produce correct results
- **Stack trace tests**: Verify multi-level stack traces show correct function names and line numbers

**E2E Test Files:**
- `overflow_add.sl` - Signed addition overflow
- `overflow_sub.sl` - Signed subtraction overflow
- `overflow_mul.sl` - Signed multiplication overflow
- `div_by_zero.sl` - Division by zero
- `mod_by_zero.sl` - Modulo by zero
- `unsigned_overflow_add.sl` - Unsigned addition overflow
- `unsigned_underflow_sub.sl` - Unsigned subtraction underflow
- `unsigned_overflow_mul.sl` - Unsigned multiplication overflow
- `stack_trace.sl` - Verifies stack trace output
- `no_panic.sl` - Verifies normal operations still work

# Code Examples

## Example 1: Signed Addition Overflow

Demonstrates panic on signed integer overflow.

```slang
// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: addition
main = () {
    val max: i64 = 9223372036854775807
    val result = max + 1  // panics
}
```

Output:
```
panic: integer overflow: addition
    at main() overflow_add.sl:5
```

## Example 2: Division by Zero

Demonstrates panic on division by zero.

```slang
// @test: exit_code=1
// @test: stderr_contains=panic: division by zero
main = () {
    val x = 10
    val y = 0
    val result = x / y  // panics
}
```

Output:
```
panic: division by zero
    at main() div_by_zero.sl:5
```

## Example 3: Stack Trace with Multiple Functions

Demonstrates stack trace through multiple function calls.

```slang
// @test: exit_code=1
// @test: stderr_contains=at divide()
// @test: stderr_contains=at calculate()
// @test: stderr_contains=at main()

divide = (a: i64, b: i64) -> i64 {
    a / b
}

calculate = (x: i64) -> i64 {
    divide(x, 0)
}

main = () {
    val result = calculate(42)
}
```

Output:
```
panic: division by zero
    at divide() stack_trace.sl:5
    at calculate() stack_trace.sl:9
    at main() stack_trace.sl:13
```

## Example 4: Normal Operations (No Panic)

Verifies that operations within bounds work correctly.

```slang
// @test: exit_code=0
// @test: stdout=42\n
main = () {
    val a = 20
    val b = 22
    val sum = a + b      // no overflow
    val diff = b - a     // no underflow
    val prod = a * 2     // no overflow
    val quot = b / 2     // no div by zero
    print(sum)
}
```

# Performance Notes

- **Hot path overhead**: 1-4 instructions depending on operation type
- **Cold path (panic)**: Only executed on error, not performance-critical
- **Branch prediction**: Error branches are predicted not-taken
- **Code size**: ~20 bytes per checked operation
- **Runtime library**: ~800 bytes (panic handler + stack walker)
- **Symbol table**: ~40 bytes per function
- **No runtime overhead** when no errors occur (checks are cheap, symbol table is only read on panic)
