# slasm - Native ARM64 Assembler

A custom assembler implementation that directly generates ARM64 machine code and Mach-O executables without relying on system tools (`as`, `ld`).

## Status

**STATUS: FULLY WORKING** (as of 2025-12-01)

The assembler successfully:
- Lexes assembly source code
- Parses into intermediate representation
- Calculates symbol addresses with two-pass layout for forward references
- Encodes ARM64 instructions to machine code
- Encodes data directives (.byte, .word, .quad, .asciz, etc.)
- Generates valid Mach-O executables with inline code signatures
- **No external tools required** - completely self-contained

**slasm is now the DEFAULT backend** for the `cmd/slasm` command-line tool.

## What Works

### Supported Instructions

**Data Movement:**
- `mov Rd, #imm` - Move immediate to register (MOVZ encoding, 16-bit)
- `mov Rd, Rm` - Move register to register (ORR encoding)

**Arithmetic:**
- `add Rd, Rn, #imm` - Add immediate (12-bit)
- `add Rd, Rn, Rm` - Add registers
- `sub Rd, Rn, #imm` - Subtract immediate (12-bit)
- `sub Rd, Rn, Rm` - Subtract registers
- `mul Rd, Rn, Rm` - Multiply
- `sdiv Rd, Rn, Rm` - Signed division
- `msub Rd, Rn, Rm, Ra` - Multiply-subtract (for modulo)

**Comparison:**
- `cmp Rn, #imm` - Compare with immediate
- `cmp Rn, Rm` - Compare registers
- `cset Rd, cond` - Conditional set (eq, ne, lt, le, gt, ge, cs, cc, mi, pl, vs, vc, hi, ls)

**Branch Instructions:**
- `b label` - Unconditional branch (PC-relative, +/-128MB range)
- `bl label` - Branch with link (function call)
- `b.cond label` - Conditional branch (b.eq, b.ne, b.lt, b.gt, b.le, b.ge, etc.)
- `br Xn` - Branch to register
- `ret` - Return from function (BR X30)

**Memory Operations:**
- `ldr Rt, [Rn]` - Load register (unsigned offset)
- `ldr Rt, [Rn, #imm]` - Load with immediate offset
- `str Rt, [Rn]` - Store register (unsigned offset)
- `str Rt, [Rn, #imm]` - Store with immediate offset
- `ldp Rt1, Rt2, [Rn]` - Load pair
- `ldp Rt1, Rt2, [Rn, #imm]` - Load pair with signed offset
- `ldp Rt1, Rt2, [Rn, #imm]!` - Load pair with pre-indexed writeback
- `ldp Rt1, Rt2, [Rn], #imm` - Load pair with post-indexed writeback
- `stp Rt1, Rt2, [Rn]` - Store pair
- `stp Rt1, Rt2, [Rn, #imm]` - Store pair with signed offset
- `stp Rt1, Rt2, [Rn, #imm]!` - Store pair with pre-indexed writeback
- `stp Rt1, Rt2, [Rn], #imm` - Store pair with post-indexed writeback

**Data Directives:**
- `.byte` - 1-byte values
- `.hword`, `.2byte` - 2-byte little-endian
- `.word`, `.4byte` - 4-byte little-endian
- `.quad`, `.8byte` - 8-byte little-endian
- `.asciz`, `.string` - Null-terminated strings with escape sequences
- `.ascii` - Strings without null terminator
- `.space`, `.zero` - Zero-filled buffers

**System & Control:**
- `svc #imm` - Supervisor call / syscall

### Example

```assembly
.global _start

_start:
    mov x0, #5
    bl add_five
    mov x16, #1
    svc #0

add_five:
    // Proper function prologue with pre-indexed writeback (push)
    stp x29, x30, [sp, #-16]!
    mov x29, sp

    add x0, x0, #5

    // Epilogue with post-indexed writeback (pop)
    ldp x29, x30, [sp], #16
    ret
```

This assembles to a working ARM64 Mach-O executable that exits with code 10.

## Usage

### Command Line (Default Backend)

```bash
# Build assembly to executable (uses slasm by default)
slasm build -o output input.s

# Use system assembler instead
slasm build --backend system -o output input.s

# Verbose output
slasm build -v -o output input.s
```

### Go API

```go
package main

import (
    "github.com/seanrogers2657/slang/assembler"
    "github.com/seanrogers2657/slang/assembler/slasm"
)

func main() {
    asm := slasm.New()

    code := `.global _start
_start:
    mov x0, #42
    mov x16, #1
    svc #0
`

    err := asm.Build(code, assembler.BuildOptions{
        OutputPath: "output",
    })
    if err != nil {
        panic(err)
    }
    // Binary is ready to execute - no codesign needed!
}
```

## Implementation

### Pipeline

1. **Lexer** (`lexer.go`) - Tokenizes assembly source (supports hex numbers, `!` for writeback)
2. **Parser** (`parser.go`) - Builds IR from tokens, including data directives
3. **Layout** (`layout.go`) - Two-pass address assignment for forward references
4. **Encoder** (`encoder.go`) - Generates ARM64 machine code and encodes data
5. **Mach-O Writer** (`macho.go`) - Creates executable file with inline code signature
6. **Assembler** (`asm.go`) - Orchestrates the pipeline

### Generated Mach-O Structure

- **Mach Header** - ARM64, EXECUTE, with PIE/DYLDLINK/TWOLEVEL/NOUNDEFS flags
- **__PAGEZERO** segment - Memory protection (0x0 - 0x100000000)
- **__TEXT** segment with **__text** section - Executable code
- **__LINKEDIT** segment - Link-edit data (code signature, symbol table)
- **LC_LOAD_DYLINKER** - Loads `/usr/lib/dyld`
- **LC_LOAD_DYLIB** - Links `/usr/lib/libSystem.B.dylib`
- **LC_MAIN** - Entry point command
- **LC_DYLD_CHAINED_FIXUPS** - Modern dyld fixups
- **LC_DYLD_EXPORTS_TRIE** - Symbol exports
- **LC_SYMTAB** - Symbol table
- **LC_DYSYMTAB** - Dynamic symbol table
- **LC_FUNCTION_STARTS** - Function addresses
- **LC_DATA_IN_CODE** - Data in code markers
- **LC_CODE_SIGNATURE** - Inline ad-hoc code signature (no external codesign needed!)
- **LC_UUID, LC_BUILD_VERSION, LC_SOURCE_VERSION** - Metadata

### Instruction Encoding

**MOVZ (Move with Zero)**
```
Encoding: sf 10 100101 hw imm16 Rd
- sf=1 for x registers, 0 for w
- hw=00 (no shift), 01 (shift 16), 10 (shift 32), 11 (shift 48)
- imm16 = immediate value
- Rd = destination register
```

**Branch (B)**
```
Encoding: 000101 imm26
- imm26 = signed offset / 4 (PC-relative)
```

**SVC (Supervisor Call)**
```
Encoding: 11010100 000 imm16 00001
- imm16 = syscall immediate
```

## Testing

All unit tests pass:
- Lexer tests (100% pass rate)
- Parser tests (100% pass rate)
- Symbol table tests (100% pass rate)
- Layout tests (100% pass rate)
- Encoder tests (100% pass rate)
  - Arithmetic: ADD, SUB, MUL, SDIV, MSUB
  - Comparison: CMP, CSET
  - Branch: B, BL, BR, B.cond (all condition codes)
  - Memory: LDR, STR, LDP, STP
  - Data encoding: byte, word, quad, asciz, space

End-to-end tests (table-driven):
- Basic exit codes (0, 1, 42, 255)
- Unconditional branches (forward, backward)
- Conditional branches (eq, ne, lt, gt, le, ge - taken and not taken)
- Branch with link and return
- Nested function calls
- Memory operations (str/ldr, with offsets, pair operations)
- Writeback addressing modes (pre-indexed and post-indexed for ldp/stp)
- Arithmetic operations
- Comparison operations
- Complex programs (factorial, fibonacci, sum loops, recursive functions)

### Debug Testing

The assembler includes comprehensive debug output:

```bash
# Run with verbose flag
slasm build -v -o output input.s

# Run the debug build program
go run cmd/slasm-debug/main.go

# Run specific tests
go test -v ./assembler/slasm -run TestEndToEnd_BasicExitCodes
go test -v ./assembler/slasm -run TestEndToEnd_BranchLink
go test -v ./assembler/slasm -run TestEndToEnd_MemoryOperations
go test -v ./assembler/slasm -run TestEndToEnd_ComplexPrograms
```

## Not Yet Implemented

### Instructions
- PC-relative addressing: `adr`, `adrp` (needed for data access)
- Unsigned division: `udiv`
- Logical operations: `and`, `orr`, `eor`, `mvn`
- Shift operations: `lsl`, `lsr`, `asr`
- 32-bit register variants: `w0-w30`
- Byte/half-word loads: `ldrb`, `ldrh`, `strb`, `strh`

### Features
- `__DATA` segment in Mach-O (data is parsed but not linked)
- Object file generation (`.o` files)
- Multi-file linking
- Relocations for external symbols
- BSS section

## Comparison: slasm vs System Assembler

| Feature | slasm | System `as` + `ld` |
|---------|-------|-------------------|
| Mach-O generation | Direct | Via linker |
| Code signing | Inline | Requires codesign |
| Branch instructions | Full support | Full support |
| Memory operations | ldr/str/ldp/stp | Complete |
| Writeback addressing | Pre/post-indexed | Complete |
| Data directives | Parsing/encoding | Complete |
| Instruction set | Core subset | Complete |
| Execution | Works | Works |
| Object files | Not implemented | Supported |
| External dependencies | None | Requires Xcode |

## File Structure

```
assembler/slasm/
├── asm.go           - Main assembler orchestration
├── lexer.go         - Tokenization (with hex support)
├── parser.go        - AST construction (with data directives)
├── symbols.go       - Symbol table
├── layout.go        - Two-pass address assignment
├── encoder.go       - ARM64 instruction encoding
├── macho.go         - Mach-O file generation
├── ir.go            - Intermediate representation types
├── util.go          - Utility functions
├── logger.go        - Logging support
├── codesign/        - Native code signing
├── e2e_test.go      - End-to-end tests (table-driven)
├── encoder_test.go  - Encoding unit tests
├── lexer_test.go    - Lexer unit tests
└── parser_test.go   - Parser unit tests
```

## Documentation

See `/docs/`:
- `SLASM_STATUS.md` - Detailed implementation status
- `reference/ARM64.md` - ARM64 instruction encoding reference
- `reference/MACH-O.md` - Mach-O file format reference

## Acknowledgments

Built as part of the Slang compiler project. This is a native assembler demonstrating:
- ARM64 instruction encoding
- Mach-O file format generation
- Native code generation without external tools
- Inline code signing

The assembler generates working ARM64 executables completely from scratch!
