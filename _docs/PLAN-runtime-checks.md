# Runtime Boundary Checking Implementation Plan

## Implementation Status

### ✅ Completed
1. **`runtime/errors.go`** - Error codes and messages
2. **`runtime/panic_arm64.asm`** - Panic handler assembly with stack walking
3. **`backend/codegen/checks.go`** - Check generation helpers for all arithmetic operations
4. **`backend/codegen/symtab.go`** - Symbol table generation for stack traces
5. **`backend/codegen/runtime.go`** - Embedded runtime panic code (fixed stack walking bug)
6. **`backend/codegen/typed_codegen.go`** - Integration with code generator
7. **slasm fixes:**
   - Added support for shifted register CMP (`cmp x3, x2, asr #63`) in `assembler/slasm/encoder.go`
   - Added `ShiftType` field to `Operand` struct in `assembler/slasm/ir.go`
   - Updated parser to capture shift type in `assembler/slasm/parser.go`
   - Fixed data section alignment padding in `assembler/slasm/asm.go`
8. **Test framework:**
   - Added `stderr_contains` directive support to `test/testutil/expectations.go`
   - Updated `test/sl/e2e_test.go` to capture stdout/stderr separately
9. **E2E test files** in `_examples/slang/runtime/`:
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
10. **Chained fixups for ASLR support** in `assembler/slasm/macho.go`:
    - `generateChainedFixupsWithRelocations()` - Generates proper chained fixups for data pointers
    - `sortDataRelocations()` - Sorts relocations for chain building
    - Proper `DYLD_CHAINED_PTR_64_OFFSET` encoding for all `.quad label` references

### ✅ Chained Fixups for Data Pointers (FIXED)

**Previous Problem:** The slasm assembler incorrectly handled `.quad label` references in the data section. The pointers were stored with raw VM addresses, but macOS requires **chained fixups** for dyld to resolve them at load time.

**Solution Implemented:**
The `generateChainedFixupsWithRelocations()` function in `assembler/slasm/macho.go` now:
1. Tracks which DATA section locations contain label references (via `DataRelocation` structs)
2. Generates proper chained fixup entries with `DYLD_CHAINED_PTR_64_OFFSET` format
3. Encodes pointers in the chained format with:
   - Target offset from image base (bits 0-35)
   - High8 bits (bits 36-43)
   - Next pointer delta in 4-byte units (bits 51-62)
4. Populates the `dyld_chained_starts_in_segment` structure for the __DATA segment

All runtime tests now work with slasm. The slasm encoder supports MOVZ/MOVK with shift operands, enabling proper encoding of large immediate values.

### ✅ Completed

---

## Overview

Add runtime boundary checking for all arithmetic operations with panic-on-violation semantics. Errors include source location (file:line) for debugging.

## Error Codes

```go
// runtime/errors.go
const (
    // Signed overflow
    ErrOverflowAddSigned    = 1  // "integer overflow: addition"
    ErrOverflowSubSigned    = 2  // "integer overflow: subtraction"
    ErrOverflowMulSigned    = 3  // "integer overflow: multiplication"

    // Unsigned overflow/underflow
    ErrOverflowAddUnsigned  = 4  // "unsigned overflow: addition"
    ErrUnderflowSubUnsigned = 5  // "unsigned underflow: subtraction"
    ErrOverflowMulUnsigned  = 6  // "unsigned overflow: multiplication"

    // Division errors
    ErrDivByZero            = 7  // "division by zero"
    ErrModByZero            = 8  // "modulo by zero"

    // Reserved for future
    // ErrIndexOutOfBounds  = 9
    // ErrNilPointer        = 10
    // ErrStackOverflow     = 11
)
```

## Panic Message Format

```
panic: <error message>
    at <function>() <file>:<line>
    at <function>() <file>:<line>
    ...
```

Examples:
```
panic: division by zero
    at divide() math.sl:15
    at calculate() main.sl:8
    at main() main.sl:3
```

```
panic: integer overflow: addition
    at add_values() calc.sl:42
    at main() calc.sl:5
```

## Stack Trace Implementation

### Symbol Table

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

    .quad _fn_calculate
    .quad _fn_calculate_end
    .quad .Lname_calculate
    .quad .Lfile_main
    .quad 5

    .quad 0                    // sentinel (null terminator)

.Lname_divide:    .asciz "divide"
.Lname_calculate: .asciz "calculate"
.Lname_main:      .asciz "main"
.Lfile_math:      .asciz "math.sl"
.Lfile_main:      .asciz "main.sl"

_slang_symtab_count:
    .quad 3                    // number of entries
```

### Frame Pointer Chain

We already preserve frame pointers with the function prologue:
```asm
stp x29, x30, [sp, #-16]!   // save frame pointer and return address
mov x29, sp                  // set new frame pointer
```

This creates a linked list on the stack:
```
┌─────────────────┐
│ return addr (lr)│  ← x29 + 8
├─────────────────┤
│ prev frame ptr  │  ← x29
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ return addr (lr)│  ← prev_fp + 8
├─────────────────┤
│ prev frame ptr  │  ← prev_fp
└────────┬────────┘
         │
         ▼
        ...
```

### Stack Walking Algorithm

```
1. Start with current frame pointer (x29)
2. While frame_ptr != 0:
   a. return_addr = *(frame_ptr + 8)
   b. Look up return_addr in symbol table
   c. If found, print "    at <name>() <file>:<line>"
   d. frame_ptr = *frame_ptr (follow chain)
3. Stop when frame_ptr is null or invalid
```

### Symbol Table Lookup

Binary search or linear scan through symbol table:
```
for each entry in symtab:
    if entry.start <= return_addr < entry.end:
        return entry
return null (unknown function)
```

## Implementation Steps

### Step 1: Create Runtime Package

Create `runtime/` package with:

**`runtime/errors.go`** - Error code constants and message table
```go
package runtime

var ErrorMessages = map[int]string{
    1: "integer overflow: addition",
    2: "integer overflow: subtraction",
    // ...
}
```

**`runtime/panic_arm64.s`** - ARM64 assembly for panic handler
```asm
.global _slang_panic
_slang_panic:
    // x0 = error code
    // x1 = filename pointer
    // x2 = line number
    // Write "panic: " prefix
    // Lookup and write error message
    // Write " at "
    // Write filename
    // Write ":"
    // Write line number
    // Write newline
    // Exit with code 1
```

### Step 2: Add Location Tracking to AST

Modify AST nodes to carry source location:

**`frontend/ast/ast.go`** - Add Position field
```go
type Position struct {
    Filename string
    Line     int
    Column   int
}

type BinaryExpr struct {
    Left     Expr
    Op       string
    Right    Expr
    Position Position  // Add this
}
```

### Step 3: Update Parser to Track Positions

**`frontend/parser/parser.go`** - Capture token positions
- Track current filename (passed to parser)
- Store line/column from tokens into AST nodes

### Step 4: Propagate Positions Through Semantic Analysis

**`frontend/semantic/analyzer.go`** - Preserve positions in TypedAST
- Ensure TypedBinaryExpr includes Position
- Pass through unchanged from parser

### Step 5: Add Check Generation to Code Generator

**`backend/codegen/checks.go`** - New file with check helpers
```go
type CheckContext struct {
    Filename  string
    Line      int
    ErrorCode int
    LabelID   int  // unique label suffix
}

func (g *Generator) EmitOverflowCheck(ctx CheckContext) string
func (g *Generator) EmitDivZeroCheck(ctx CheckContext) string
```

**`backend/codegen/typed_codegen.go`** - Integrate checks
- After each arithmetic op, emit appropriate check
- Pass source location to check emitter

### Step 6: Add Symbol Table Generation

**`backend/codegen/symtab.go`** - New file for symbol table generation
```go
type SymbolEntry struct {
    Name      string
    Filename  string
    StartLine int
    Label     string    // assembly label for function start
    EndLabel  string    // assembly label for function end
}

type SymbolTable struct {
    Entries []SymbolEntry
}

func (s *SymbolTable) Add(name, filename string, line int, label string)
func (s *SymbolTable) Generate() string  // produces .data section
```

**`backend/codegen/typed_codegen.go`** - Emit function end labels
- After each function body, emit `_fn_{name}_end:` label
- Register function in symbol table during codegen

### Step 7: Update Linker to Include Runtime

**`cmd/sl/main.go`** - Link runtime library
- Assemble runtime/panic_arm64.s
- Link with user code

### Step 8: Add E2E Tests

**`_examples/slang/runtime/`** - Test files for each error type
```slang
// overflow_add.sl
// @test: exit_code=1
// @test: stderr_contains=panic: integer overflow: addition
fn main(): void {
    val max: i64 = 9223372036854775807
    val result = max + 1  // should panic
}
```

## File Changes Summary

| File | Change |
|------|--------|
| `runtime/errors.go` | New - error codes and messages |
| `runtime/panic_arm64.s` | New - panic handler with stack walker |
| `frontend/ast/ast.go` | Add Position to nodes |
| `frontend/lexer/lexer.go` | Track line/column in tokens |
| `frontend/parser/parser.go` | Attach positions to AST |
| `frontend/semantic/types.go` | Add Position to typed AST |
| `backend/codegen/checks.go` | New - check generation helpers |
| `backend/codegen/symtab.go` | New - symbol table generation |
| `backend/codegen/typed_codegen.go` | Emit checks, function end labels, symtab |
| `cmd/sl/main.go` | Link runtime library |
| `_examples/slang/runtime/*.sl` | New - test files |
| `test/sl/e2e_test.go` | Add stderr_contains support |

## ARM64 Check Patterns

### Signed Addition
```asm
adds x2, x0, x1           // add with flags
b.vs .Lpanic_{id}         // branch on signed overflow
// ... continue ...
.Lpanic_{id}:
    mov x0, #1            // ErrOverflowAddSigned
    adrp x1, .Lfile@PAGE
    add x1, x1, .Lfile@PAGEOFF
    mov x2, #{line}
    bl _slang_panic
```

### Signed Subtraction
```asm
subs x2, x0, x1           // sub with flags
b.vs .Lpanic_{id}         // branch on signed overflow
```

### Signed Multiplication
```asm
mul x2, x0, x1            // low 64 bits
smulh x3, x0, x1          // high 64 bits
cmp x3, x2, asr #63       // compare high with sign extension
b.ne .Lpanic_{id}         // overflow if mismatch
```

### Unsigned Addition
```asm
adds x2, x0, x1           // add with flags
b.cs .Lpanic_{id}         // branch on carry (unsigned overflow)
```

### Unsigned Subtraction
```asm
subs x2, x0, x1           // sub with flags
b.cc .Lpanic_{id}         // branch on no carry (underflow)
```

### Unsigned Multiplication
```asm
mul x2, x0, x1            // low 64 bits
umulh x3, x0, x1          // high 64 bits (unsigned)
cbnz x3, .Lpanic_{id}     // overflow if high bits non-zero
```

### Division by Zero
```asm
cbz x1, .Lpanic_{id}      // check divisor before divide
sdiv x2, x0, x1           // or udiv for unsigned
```

### Modulo by Zero
```asm
cbz x1, .Lpanic_{id}      // check divisor before modulo
sdiv x3, x0, x1
msub x2, x3, x1, x0
```

## Panic Handler Assembly (Sketch)

```asm
.data
.Lpanic_prefix:  .asciz "panic: "
.Lpanic_at:      .asciz "    at "
.Lpanic_paren:   .asciz "() "
.Lpanic_colon:   .asciz ":"
.Lpanic_nl:      .asciz "\n"
.Lpanic_unknown: .asciz "<unknown>"

// Error messages (indexed by error code - 1)
.Lerror_messages:
    .quad .Lerr_1, .Lerr_2, .Lerr_3, .Lerr_4, .Lerr_5, .Lerr_6, .Lerr_7, .Lerr_8
.Lerr_1: .asciz "integer overflow: addition"
.Lerr_2: .asciz "integer overflow: subtraction"
.Lerr_3: .asciz "integer overflow: multiplication"
.Lerr_4: .asciz "unsigned overflow: addition"
.Lerr_5: .asciz "unsigned underflow: subtraction"
.Lerr_6: .asciz "unsigned overflow: multiplication"
.Lerr_7: .asciz "division by zero"
.Lerr_8: .asciz "modulo by zero"

.text
.global _slang_panic
_slang_panic:
    // Arguments: x0 = error code
    // Save frame pointer for stack walking
    stp x29, x30, [sp, #-16]!
    mov x29, sp

    // Save error code
    mov x19, x0

    // 1. Write "panic: " to stderr
    // 2. Lookup and write error message
    // 3. Write newline

    // 4. Walk stack and print trace
    mov x20, x29                    // current frame pointer

.Lwalk_loop:
    cbz x20, .Lwalk_done            // null frame pointer = done

    ldr x21, [x20, #8]              // x21 = return address
    ldr x20, [x20]                  // x20 = previous frame pointer

    // Lookup return address in symbol table
    // x21 = return address to look up
    bl _slang_symtab_lookup         // returns: x0=name, x1=file, x2=line (or all 0)

    cbz x0, .Lwalk_loop             // skip if not found

    // Print "    at <name>() <file>:<line>\n"
    // ... write x0 (name), "()", x1 (file), ":", x2 (line), newline

    b .Lwalk_loop

.Lwalk_done:
    // Exit with code 1
    mov x0, #1
    mov x16, #1
    svc #0

// Symbol table lookup function
// Input: x21 = return address
// Output: x0 = name ptr, x1 = file ptr, x2 = line (or all 0 if not found)
_slang_symtab_lookup:
    // Load symbol table base address
    adrp x8, _slang_symtab@PAGE
    add x8, x8, _slang_symtab@PAGEOFF

.Llookup_loop:
    ldr x9, [x8]                    // start address
    cbz x9, .Llookup_notfound       // sentinel = end of table

    ldr x10, [x8, #8]               // end address
    cmp x21, x9
    b.lt .Llookup_next              // return_addr < start
    cmp x21, x10
    b.ge .Llookup_next              // return_addr >= end

    // Found! Load name, file, line
    ldr x0, [x8, #16]               // name pointer
    ldr x1, [x8, #24]               // file pointer
    ldr x2, [x8, #32]               // line number
    ret

.Llookup_next:
    add x8, x8, #40                 // sizeof(SymbolEntry) = 5 * 8
    b .Llookup_loop

.Llookup_notfound:
    mov x0, #0
    mov x1, #0
    mov x2, #0
    ret
```

## Testing Strategy

1. **Unit tests** - Check generation produces correct assembly patterns
2. **E2E tests** - Each error type has a test file that triggers it
3. **Negative tests** - Operations that don't overflow still work

## Performance Notes

- Hot path overhead: 1-4 instructions depending on operation
- Cold path (panic): only executed on error, not performance-critical
- Branch prediction: error branches predicted not-taken
- Code size per checked operation: ~20 bytes
- Runtime library size: ~800 bytes (panic handler + stack walker)
- Symbol table overhead: ~40 bytes per function
- No runtime overhead when no errors occur (checks are cheap, symbol table is only read on panic)
