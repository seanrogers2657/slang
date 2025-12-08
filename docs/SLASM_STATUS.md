# slasm Implementation Status

Last updated: 2025-12-06

**STATUS: ✅ FULLY WORKING** - Native ARM64 assembler with comprehensive instruction support!

## Overview

The slasm assembler is a custom ARM64 assembler that generates Mach-O executables directly without relying on system tools (`as`, `ld`). It is designed to support the Slang compiler and is now the **default backend** for the `slasm` command-line tool.

## Implementation Status

### ✅ Fully Implemented

#### Lexer (`lexer.go`)
- Directives: `.global`, `.align`, `.data`, `.text`
- Data directives: `.byte`, `.word`, `.quad`, `.asciz`, `.ascii`, `.space`, `.zero`, `.hword`, `.2byte`, `.4byte`, `.8byte`
- Labels: `_start:`, `loop:`
- Instructions: All needed mnemonics including branches and memory ops
- Registers: `x0-x30`, `sp`, `lr`, `xzr`
- Immediates: `#123`, `#0x1a` (decimal and hex)
- Memory operands: `[sp]`, `[sp, #16]`, `[x0, #8]`
- Writeback operands: `[sp, #-16]!` (pre-indexed), `[sp], #16` (post-indexed)
- Punctuation: `:`, `,`, `#`, `[`, `]`, `!`, `=`
- Comments: `//` and `;` style
- Conditional branches: `b.eq`, `b.ne`, `b.lt`, `b.gt`, `b.le`, `b.ge`, etc.
- Constant definitions: `NAME = value` (assembly-time constants)

#### Parser (`parser.go`)
- Parse `.global` directive
- Parse label definitions
- Parse constant definitions (`name = value`)
- Parse instructions with register, immediate, and memory operands
- Parse data directives into `DataDeclaration` items
- Build Program IR with multiple sections (`.text`, `.data`)
- Memory operand parsing with base register and offset
- Constants resolved during encoding (can use `#CONST_NAME` in immediates)

#### Symbol Table (`symbols.go`)
- Symbol definition with duplicate checking
- Symbol lookup
- Address resolution
- Section tracking

#### Layout (`layout.go`)
- **Two-pass layout** for forward reference resolution
- Address assignment (4 bytes per instruction)
- Symbol table construction
- Alignment directive handling (`.align` with NOP padding in code sections)
- Data section size calculation

#### Encoder (`encoder.go`)

**Data Movement:**
- `mov Rd, #imm` → MOVZ encoding (16-bit immediate)
- `mov Rd, Rm` → ORR encoding (register-to-register)

**Arithmetic:**
- `add Rd, Rn, #imm` → ADD immediate (12-bit)
- `add Rd, Rn, Rm` → ADD register
- `sub Rd, Rn, #imm` → SUB immediate (12-bit)
- `sub Rd, Rn, Rm` → SUB register
- `neg Rd, Rm` → Negate (SUB with Rn=XZR)
- `mul Rd, Rn, Rm` → MADD with Ra=XZR
- `sdiv Rd, Rn, Rm` → Signed division
- `udiv Rd, Rn, Rm` → Unsigned division
- `msub Rd, Rn, Rm, Ra` → Multiply-subtract (for modulo)

**Shift Operations:**
- `lsl Rd, Rn, #shift` → Logical shift left (immediate, 0-63)
- `lsl Rd, Rn, Rm` → Logical shift left (register)
- `lsr Rd, Rn, #shift` → Logical shift right (immediate, 0-63)
- `lsr Rd, Rn, Rm` → Logical shift right (register)
- `asr Rd, Rn, #shift` → Arithmetic shift right (immediate, 0-63)
- `asr Rd, Rn, Rm` → Arithmetic shift right (register)

**Logical Operations:**
- `and Rd, Rn, Rm` → Bitwise AND
- `orr Rd, Rn, Rm` → Bitwise OR
- `eor Rd, Rn, Rm` → Bitwise exclusive OR (XOR)
- `mvn Rd, Rm` → Bitwise NOT (move negated)
- `ands Rd, Rn, Rm` → Bitwise AND with flags update
- `tst Rn, Rm` → Test bits (ANDS with Rd=XZR)
- `bic Rd, Rn, Rm` → Bit clear (AND NOT)
- `orn Rd, Rn, Rm` → OR NOT
- `eon Rd, Rn, Rm` → Exclusive OR NOT

**Comparison:**
- `cmp Rn, #imm` → SUBS with Rd=XZR
- `cmp Rn, Rm` → SUBS register with Rd=XZR
- `cset Rd, cond` → CSINC (conditional set)
  - Supported conditions: `eq`, `ne`, `lt`, `le`, `gt`, `ge`, `cs`, `cc`, `mi`, `pl`, `vs`, `vc`, `hi`, `ls`

**Branch Instructions:**
- `b label` → Unconditional branch (PC-relative, ±128MB range)
- `bl label` → Branch with link (function call)
- `b.cond label` → Conditional branch (b.eq, b.ne, b.lt, b.gt, b.le, b.ge, etc.)
- `br Xn` → Branch to register
- `ret` → Return (BR X30)

**Memory Operations:**
- `ldr Rt, [Rn]` → Load register (unsigned offset)
- `ldr Rt, [Rn, #imm]` → Load with immediate offset
- `ldr Rt, [Rn, #imm]!` → Load with pre-indexed writeback
- `ldr Rt, [Rn], #imm` → Load with post-indexed writeback
- `str Rt, [Rn]` → Store register (unsigned offset)
- `str Rt, [Rn, #imm]` → Store with immediate offset
- `str Rt, [Rn, #imm]!` → Store with pre-indexed writeback
- `str Rt, [Rn], #imm` → Store with post-indexed writeback
- `ldp Rt1, Rt2, [Rn]` → Load pair
- `ldp Rt1, Rt2, [Rn, #imm]` → Load pair with signed offset
- `ldp Rt1, Rt2, [Rn, #imm]!` → Load pair with pre-indexed writeback
- `ldp Rt1, Rt2, [Rn], #imm` → Load pair with post-indexed writeback
- `stp Rt1, Rt2, [Rn]` → Store pair
- `stp Rt1, Rt2, [Rn, #imm]` → Store pair with signed offset
- `stp Rt1, Rt2, [Rn, #imm]!` → Store pair with pre-indexed writeback
- `stp Rt1, Rt2, [Rn], #imm` → Store pair with post-indexed writeback
- `ldrb Wt, [Xn, #imm]` → Load byte (unsigned offset)
- `strb Wt, [Xn, #imm]` → Store byte (unsigned offset)
- `ldrh Wt, [Xn, #imm]` → Load halfword (unsigned offset)
- `strh Wt, [Xn, #imm]` → Store halfword (unsigned offset)

**Data Encoding:**
- `.byte` values → 1-byte encoding
- `.hword`, `.2byte` → 2-byte little-endian
- `.word`, `.4byte` → 4-byte little-endian
- `.quad`, `.8byte` → 8-byte little-endian
- `.asciz`, `.string` → Null-terminated strings with escape sequences
- `.ascii` → Strings without null terminator
- `.space`, `.zero` → Zero-filled buffers

**Address Generation:**
- `adr Rd, label` → PC-relative address (±1MB range)
- `adrp Rd, label@PAGE` → Page address (4KB aligned, PC-relative)

**System & Control:**
- `svc #imm` → Supervisor call (16-bit immediate)

#### Mach-O Writer (`macho.go`)

**Mach-O Header:**
- Magic: `MH_MAGIC_64` (0xfeedfacf)
- CPU Type: `CPU_TYPE_ARM64`
- File Type: `MH_EXECUTE`
- Flags: `MH_PIE | MH_DYLDLINK | MH_TWOLEVEL | MH_NOUNDEFS`

**Segments:**
- `__PAGEZERO`: 4GB null pointer protection
- `__TEXT`: Code segment with `__text` section
- `__DATA`: Data segment with `__data` section (when data directives present)
- `__LINKEDIT`: Link-edit data (code signatures, symbol table)

**Load Commands (16-17 total):**
- `LC_SEGMENT_64`: __PAGEZERO, __TEXT, [__DATA], __LINKEDIT
- `LC_LOAD_DYLINKER`: `/usr/lib/dyld`
- `LC_LOAD_DYLIB`: `/usr/lib/libSystem.B.dylib`
- `LC_MAIN`: Entry point
- `LC_UUID`: Binary identifier
- `LC_BUILD_VERSION`: Platform and SDK versions
- `LC_SOURCE_VERSION`: Source version info
- `LC_DYLD_CHAINED_FIXUPS`: Modern dyld fixups
- `LC_DYLD_EXPORTS_TRIE`: Symbol exports
- `LC_SYMTAB`: Symbol table
- `LC_DYSYMTAB`: Dynamic symbol table
- `LC_FUNCTION_STARTS`: Function addresses
- `LC_DATA_IN_CODE`: Data in code markers
- `LC_CODE_SIGNATURE`: **Embedded inline** (no external codesign needed!)

#### Code Signing (`codesign/`)
- Native code signature generation
- Ad-hoc signing embedded during Mach-O generation
- No external `codesign` tool required

#### Main Assembler (`asm.go`)
- `New()`: Create assembler instance
- `Build()`: Full pipeline: Lex → Parse → Layout → Encode → Write Mach-O
- Error handling and reporting
- Verbose logging option

## Test Coverage

### Unit Tests
- All lexer tests ✅
- All parser tests ✅
- All symbol table tests ✅
- All layout tests ✅
- All encoder tests ✅
  - Arithmetic: ADD, SUB, NEG, MUL, SDIV, UDIV, MSUB
  - Shift: LSL, LSR, ASR (immediate and register forms)
  - Logical: AND, ORR, EOR, MVN, ANDS, TST, BIC, ORN, EON
  - Comparison: CMP, CSET
  - Branch: B, BL, BR, B.cond (all condition codes)
  - Address: ADR, ADRP
  - Memory: LDR, STR, LDP, STP, LDRB, STRB, LDRH, STRH (including pre/post-indexed writeback)
  - Data encoding: byte, word, quad, asciz, space
  - Constants: Resolution of assembly-time constants in immediates

### End-to-End Tests (`e2e_test.go`)
Table-driven tests covering:
- Basic exit codes (0, 1, 42, 255)
- Unconditional branches (forward, backward)
- Conditional branches (eq, ne, lt, gt, le, ge - taken and not taken)
- Branch with link and return
- Nested function calls
- Memory operations (str/ldr, with offsets, pair operations)
- Writeback addressing modes (pre-indexed `[sp, #-16]!` and post-indexed `[sp], #16`)
- Single-register indexed addressing (str/ldr with pre/post-indexed writeback)
- Arithmetic operations
- Division operations (udiv, sdiv, modulo using msub)
- Comparison operations
- Shift operations (lsl, lsr, asr)
- Address generation (adr, adrp for data access)
- Assembly-time constants (`name = value` definitions)
- Complex programs (factorial, fibonacci, sum loops, recursive functions)

## Key Achievements

1. **Complete Branch Support** ✅
   - Forward and backward branches with label resolution
   - Two-pass layout for forward references
   - All conditional branch types

2. **Memory Operations** ✅
   - Load/store single registers (with pre/post-indexed writeback)
   - Load/store pairs (for stack frames)
   - Scaled immediate offsets
   - **Pre-indexed writeback** (`[sp, #-16]!`) for push operations
   - **Post-indexed writeback** (`[sp], #16`) for pop operations

3. **Division Operations** ✅
   - Signed division (`sdiv`)
   - Unsigned division (`udiv`)
   - Modulo via multiply-subtract (`msub`)

4. **Data Section Parsing** ✅
   - Full directive support (.byte, .quad, .asciz, etc.)
   - Escape sequence handling
   - Multi-value directives

5. **Inline Code Signing** ✅
   - No external tools required
   - Binaries execute immediately after generation

6. **Default Backend** ✅
   - slasm is now the default for `cmd/slasm`
   - System backend available via `--backend system`

7. **Shift Instructions** ✅
   - LSL, LSR, ASR (immediate and register forms)
   - Encoded as UBFM/SBFM for immediate, variable shifts for register

8. **Assembly-Time Constants** ✅
   - Define with `NAME = value`
   - Use with `#NAME` in immediate positions
   - Resolved during encoding phase

## Not Yet Implemented

### Instructions
- Test bit and branch: `tbz`, `tbnz`
- Sign-extending loads: `ldrsh`, `ldrsw`
- 32-bit register variants for some operations

### Features
- BSS section
- Cross-file symbol references (symbols must be defined in the same file)

## Recently Added Features

### Logical Operations (2025-12-06)
Full support for bitwise logical operations:
- `and`, `orr`, `eor`, `mvn`
- `ands`, `tst`, `bic`, `orn`, `eon`

### Halfword Memory Operations (2025-12-06)
- `ldrh` - Load halfword (16-bit, zero-extended)
- `strh` - Store halfword

### Object File Generation (2025-12-06)
- `slasm assemble -o output.o input.s` generates relocatable `.o` files
- Mach-O MH_OBJECT format
- Symbol table with local and global symbols

### Multi-File Linking (2025-12-06)
- `slasm link -o output file1.o file2.o` links multiple object files
- Section merging (__text, __data)
- Symbol resolution across files
- Generates signed executable

**Note:** Cross-file symbol references are not yet supported. All symbols must be defined within the same source file.

## File Structure

```
assembler/slasm/
├── asm.go            - Main assembler orchestration
├── lexer.go          - Tokenization (with hex support)
├── parser.go         - AST construction (with data directives)
├── symbols.go        - Symbol table
├── layout.go         - Two-pass address assignment
├── encoder.go        - ARM64 instruction encoding
├── macho.go          - Mach-O file generation (executable + object)
├── ir.go             - Intermediate representation types
├── util.go           - Utility functions
├── logger.go         - Logging support
├── relocations.go    - Relocation types and structures
├── object_reader.go  - Mach-O object file parser
├── linker.go         - Multi-file linker
├── codesign/         - Native code signing
├── e2e_test.go       - End-to-end tests (table-driven)
├── encoder_test.go   - Encoding unit tests
├── lexer_test.go     - Lexer unit tests
└── parser_test.go    - Parser unit tests
```

## Usage

### Command Line (Default Backend)

```bash
# Build assembly to executable (uses slasm by default)
slasm build -o output input.s

# Use system assembler instead
slasm build --backend system -o output input.s

# Verbose output
slasm build -v -o output input.s

# Assemble to object file (.o)
slasm assemble -o output.o input.s

# Link multiple object files
slasm link -o output file1.o file2.o
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

    // Build directly to executable
    err := asm.Build(code, assembler.BuildOptions{
        OutputPath: "output",
    })
    if err != nil {
        panic(err)
    }
    // Binary is ready to execute - no codesign needed!

    // Or assemble to object file
    err = asm.Assemble("source.s", "output.o")
    if err != nil {
        panic(err)
    }

    // Link multiple object files
    err = asm.Link([]string{"file1.o", "file2.o"}, "output")
    if err != nil {
        panic(err)
    }
}
```

## Comparison: slasm vs System Assembler

| Feature | slasm | System `as` + `ld` |
|---------|-------|-------------------|
| Mach-O generation | ✅ Direct | ✅ Via linker |
| Code signing | ✅ Inline | ✅ Requires codesign |
| Branch instructions | ✅ Full support | ✅ Full support |
| Memory operations | ✅ ldr/str/ldp/stp | ✅ Complete |
| Writeback addressing | ✅ Pre/post-indexed | ✅ Complete |
| Data directives | ✅ Parsing/encoding | ✅ Complete |
| Instruction set | ⚠️ Core subset | ✅ Complete |
| Execution | ✅ Works | ✅ Works |
| Object files | ✅ Supported | ✅ Supported |
| Multi-file linking | ✅ Supported | ✅ Supported |
| External dependencies | ✅ None | ❌ Requires Xcode |

## References

- ARM64 Architecture Reference Manual (ARM DDI 0487)
- Mach-O File Format Reference (Apple)
- Go assembler source: `cmd/internal/obj/arm64/`
- Go linker source: `cmd/link/internal/ld/`
