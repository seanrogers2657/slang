# slasm - Native ARM64 Assembler

A custom assembler implementation that directly generates ARM64 machine code and Mach-O executables without relying on system tools (`as`, `ld`).

## Status

**✅ WORKING!** (as of 2025-11-29)

The assembler successfully:
- ✅ Lexes assembly source code
- ✅ Parses into intermediate representation
- ✅ Calculates symbol addresses
- ✅ Encodes ARM64 instructions to machine code
- ✅ Generates valid Mach-O executables
- ✅ Passes `codesign --verify` validation
- ✅ All instruction encodings verified correct
- ✅ **Generated binaries execute correctly!**

**Current Limitations:**
- ⚠️ No data section support yet
- ⚠️ No relocations for label references
- ⚠️ Branch instructions not fully implemented (no label resolution)

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
- `cset Rd, cond` - Conditional set (eq, ne, lt, le, gt, ge)

**System & Control:**
- `svc #imm` - Supervisor call / syscall
- `ret` - Return from function

### Example

```assembly
.global _start

_start:
    mov x0, #1      ; Exit code
    mov x16, #1     ; Exit syscall number
    svc #0          ; Make syscall
```

This assembles to a working ARM64 Mach-O executable that exits with code 1.

## Implementation

### Pipeline

1. **Lexer** (`lexer.go`) - Tokenizes assembly source
2. **Parser** (`parser.go`) - Builds IR from tokens
3. **Layout** (`layout.go`) - Assigns addresses to labels
4. **Encoder** (`encoder.go`) - Generates ARM64 machine code
5. **Mach-O Writer** (`macho.go`) - Creates executable file
6. **Assembler** (`asm.go`) - Orchestrates the pipeline

### Generated Mach-O Structure

- **Mach Header** - ARM64, EXECUTE, with PIE/DYLDLINK/TWOLEVEL/NOUNDEFS flags
- **__PAGEZERO** segment - Memory protection (0x0 - 0x100000000)
- **__TEXT** segment with **__text** section - Executable code
- **__LINKEDIT** segment - Link-edit data (chained fixups, symbol table, code signature)
- **LC_LOAD_DYLINKER** - Loads `/usr/lib/dyld`
- **LC_LOAD_DYLIB** - Links `/usr/lib/libSystem.B.dylib`
- **LC_MAIN** - Entry point command
- **LC_DYLD_CHAINED_FIXUPS** - Modern dyld fixups (56-byte minimal structure)
- **LC_SYMTAB** - Symbol table with _start symbol
- **LC_DYSYMTAB** - Dynamic symbol table
- **LC_UUID, LC_BUILD_VERSION, LC_SOURCE_VERSION** - Metadata
- **LC_CODE_SIGNATURE** - Code signature (added by codesign)

### Instruction Encoding

**MOVZ (Move with Zero)**
```
Encoding: sf 10 100101 hw imm16 Rd
- sf=1 for x registers, 0 for w
- hw=00 (no shift), 01 (shift 16), 10 (shift 32), 11 (shift 48)
- imm16 = immediate value
- Rd = destination register
```

**SVC (Supervisor Call)**
```
Encoding: 11010100 000 imm16 00001
- imm16 = syscall immediate
```

## Testing

All unit tests pass:
- ✅ Lexer tests (100% pass rate)
- ✅ Parser tests (100% pass rate)
- ✅ Symbol table tests (100% pass rate)
- ✅ Layout tests (100% pass rate)
- ✅ Encoder tests (100% pass rate)
  - ADD, SUB, MUL, SDIV, MSUB, CMP, CSET all verified

End-to-end tests:
- ✅ Generate valid Mach-O executables
- ✅ Pass `codesign --verify` validation
- ✅ Correct instruction encoding (verified with `otool -tV`)
- ✅ Binaries execute correctly and return expected exit codes

### Debug Testing

The assembler includes comprehensive debug output to help diagnose issues:

**Run the debug build program:**
```bash
go run cmd/slasm-debug/main.go
```

This will show detailed output for every stage:
1. **Lexer** - All tokens with types and values
2. **Parser** - Parsed program structure (sections, labels, instructions)
3. **Layout** - Symbol table with addresses
4. **Encoder** - Machine code for each instruction (with hex bytes)
5. **Mach-O** - Complete file structure (segments, load commands, offsets)
6. **Verification** - File info, disassembly, code signature, etc.

**Run the debug tests:**
```bash
# Minimal program test
go test -v ./assembler/slasm -run TestDebugExample_MinimalProgram

# Syscall test
go test -v ./assembler/slasm -run TestDebugExample_SimpleSyscall

# Arithmetic test
go test -v ./assembler/slasm -run TestDebugExample_Arithmetic
```

Each test shows:
- Complete build pipeline output
- Instruction encoding with hex bytes
- Symbol table addresses
- Mach-O structure details
- otool disassembly verification
- Code signature verification
- Execution attempt with exit code

**Example Debug Output:**
```
========== SLASM ASSEMBLER - BUILD PIPELINE ==========

STEP 1: LEXER
-------------
Lexer produced 16 tokens:
  [  0] TokenDirective : .global
  [  1] TokenIdentifier: _start
  ...

STEP 2: PARSER
--------------
Parser produced 1 section(s):
  Section 0: SectionText (4 items)
    [  0] Directive: .global [_start]
    [  1] Label: _start
    [  2] Instruction: mov    x0, 42
    [  3] Instruction: ret

STEP 3: LAYOUT & SYMBOL TABLE
------------------------------
Symbol table:
  _start         : addr=0x0000 section=SectionText [GLOBAL]

STEP 4: INSTRUCTION ENCODING
----------------------------
  [0x0000] mov x0, 42           -> 40 05 80 d2 (0xd2800540)
  [0x0004] ret                  -> c0 03 5f d6 (0xd65f03c0)

Encoded 2 instructions (8 bytes total)
Complete machine code: 400580d2c0035fd6

STEP 5: MACH-O GENERATION
-------------------------
Mach-O Structure:
  Header:            size=32 bytes
  Load commands:     size=480 bytes, count=9
  Code offset:       0x1000 (4096 bytes)
  ...

========== BUILD SUMMARY ==========
Output file:       /tmp/test_slasm_minimal
Architecture:      arm64
Entry point:       _start
Instructions:      2
Code size:         8 bytes
Symbols:           1
===================================
```

The debug output helps identify issues at each stage of the assembly process, making it easy to diagnose encoding errors, symbol resolution problems, or Mach-O structure issues.

## Known Issues

1. **Limited instruction set** - Only implements instructions needed for the Slang compiler. Many ARM64 instructions are not yet implemented:
   - Branch instructions with label resolution (`b`, `bl`, `b.cond`)
   - Memory operations (`ldr`, `str`, `ldp`, `stp`)
   - PC-relative addressing (`adr`, `adrp`)

2. **No data section support** - Cannot assemble programs with `.data` sections, string literals, or data directives (`.byte`, `.word`, `.asciz`, etc.).

3. **No relocations** - Cannot handle label references in instructions that require relocation.

## Usage

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
    mov x0, #1
    mov x16, #1
    svc #0
`

    err := asm.Build(code, assembler.BuildOptions{
        OutputPath: "output",
    })
    if err != nil {
        panic(err)
    }
}
```

## Future Work

### Next Steps
1. **Implement branch instructions** - `b`, `bl`, `b.cond` with label resolution
2. **Implement memory operations** - `ldr`, `str`, `ldp`, `stp`
3. **Data section support** - `.data`, `.rodata`, `.space`, `.asciz`, etc.
4. **Relocations** - Support label references in instructions
5. **Object file generation** - Implement `Assemble()` for `.o` files
6. **Linker** - Implement `Link()` to combine object files

## Documentation

See `/docs/reference/`:
- `ARM64.md` - ARM64 instruction encoding reference
- `MACH-O.md` - Mach-O file format reference

## Acknowledgments

Built as part of the Slang compiler project. This is a minimal proof-of-concept assembler demonstrating:
- ARM64 instruction encoding
- Mach-O file format generation
- Native code generation without external tools

The assembler successfully generates working ARM64 executables from scratch!
