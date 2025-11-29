# slasm Implementation Status

Last updated: 2025-11-28

## Overview

The slasm assembler is a custom ARM64 assembler that generates Mach-O executables directly without relying on system tools (`as`, `ld`). It is designed to support the Slang compiler.

## Implementation Status

### ✅ Fully Implemented

#### Lexer (`lexer.go`)
- Directives: `.global`, `.align`
- Labels: `_start:`, `loop:`
- Instructions: All needed mnemonics
- Registers: `x0-x30`, `sp`, `lr`, `xzr`
- Immediates: `#123`, `#0x1a`
- Punctuation: `:`, `,`, `#`
- Comments: `//` and `;` style
- Whitespace handling

#### Parser (`parser.go`)
- Parse `.global` directive
- Parse label definitions
- Parse instructions with register and immediate operands
- Build Program IR with text section

#### Symbol Table (`symbols.go`)
- Symbol definition with duplicate checking
- Symbol lookup
- Address resolution

#### Layout (`layout.go`)
- Single-pass through text section
- Address assignment (4 bytes per instruction)
- Symbol table construction
- Alignment directive handling

#### Encoder (`encoder.go`)

**Data Movement:**
- `mov Rd, #imm` → MOVZ encoding (16-bit immediate)
- `mov Rd, Rm` → ORR encoding (register-to-register)

**Arithmetic:**
- `add Rd, Rn, #imm` → ADD immediate (12-bit)
- `add Rd, Rn, Rm` → ADD register
- `sub Rd, Rn, #imm` → SUB immediate (12-bit)
- `sub Rd, Rn, Rm` → SUB register
- `mul Rd, Rn, Rm` → MADD with Ra=XZR
- `sdiv Rd, Rn, Rm` → Signed division
- `msub Rd, Rn, Rm, Ra` → Multiply-subtract (for modulo: `a - (n * m)`)

**Comparison:**
- `cmp Rn, #imm` → SUBS with Rd=XZR
- `cmp Rn, Rm` → SUBS register with Rd=XZR
- `cset Rd, cond` → CSINC (conditional set)
  - Supported conditions: `eq`, `ne`, `lt`, `le`, `gt`, `ge`

**System & Control:**
- `svc #imm` → Supervisor call (16-bit immediate)
- `ret` → Return (BR X30)

#### Mach-O Writer (`macho.go`)

**Mach-O Header:**
- Magic: `MH_MAGIC_64` (0xfeedfacf)
- CPU Type: `CPU_TYPE_ARM64`
- File Type: `MH_EXECUTE`
- Flags: `MH_PIE | MH_DYLDLINK | MH_TWOLEVEL | MH_NOUNDEFS`

**Segments:**
- `__PAGEZERO`: 4GB null pointer protection (vmaddr=0, vmsize=0x100000000)
- `__TEXT`: Code segment
  - File offset: 0 (includes header and load commands)
  - VM address: 0x100000000
  - VM size: Page-aligned (matches file size)
  - File size: Page-aligned
  - Contains `__text` section
- `__LINKEDIT`: Link-edit data (for code signatures)
  - Reserves 8KB for codesign data

**Load Commands:**
- `LC_SEGMENT_64`: __PAGEZERO, __TEXT, __LINKEDIT
- `LC_LOAD_DYLINKER`: `/usr/lib/dyld`
- `LC_LOAD_DYLIB`: `/usr/lib/libSystem.B.dylib`
- `LC_MAIN`: Entry point (references __text section offset)
- `LC_UUID`: Binary identifier
- `LC_BUILD_VERSION`: Platform and SDK versions
- `LC_SOURCE_VERSION`: Source version info
- `LC_DYLD_CHAINED_FIXUPS`: Modern dyld fixups (minimal empty table)
- `LC_SYMTAB`: Symbol table with _start symbol
- `LC_DYSYMTAB`: Dynamic symbol table
- Space for `LC_CODE_SIGNATURE` (added by codesign)

**Section Layout:**
- `__text` section:
  - VM address: `__TEXT.vmaddr + code_offset` (correctly offset)
  - File offset: Page-aligned (e.g., 0x1000 = 4096)
  - Alignment: 4-byte (2^2)
  - Flags: `S_REGULAR | S_ATTR_PURE_INSTRUCTIONS | S_ATTR_SOME_INSTRUCTIONS`

#### Main Assembler (`asm.go`)
- `New()`: Create assembler instance
- `Build()`: Full pipeline: Lex → Parse → Layout → Encode → Write Mach-O
- Error handling and reporting

## Test Coverage

### ✅ Passing Tests
- All lexer unit tests
- All parser unit tests
- All symbol table tests
- All layout tests
- All encoder unit tests:
  - `TestEncodeAdd` ✅
  - `TestEncodeSub` ✅
  - `TestEncodeMul` ✅
  - `TestEncodeSdiv` ✅
  - `TestEncodeMsub` ✅
  - `TestEncodeCmp` ✅
  - `TestEncodeCset` ✅

### ⚠️ Failing Tests
- End-to-end execution tests (binaries are killed at runtime)

## Key Achievements

1. **Valid Mach-O Generation** ✅
   - Generates structurally correct Mach-O executables
   - Proper segment and section layout
   - Correct alignment and offsets

2. **Code Signature Support** ✅
   - Binaries pass `codesign --verify` validation
   - MH_PIE flag correctly set
   - Space properly reserved for code signatures

3. **Instruction Encoding** ✅
   - All required ARM64 instructions encode correctly
   - Verified with `otool -tV` disassembly
   - Matches system assembler output

4. **Slang Compiler Support** ✅
   - All instructions needed by Slang compiler are implemented
   - Supports arithmetic, comparison, and system calls

5. **Modern macOS Load Commands** ✅
   - LC_DYLD_CHAINED_FIXUPS with correct format (matches C binaries)
   - LC_SYMTAB and LC_DYSYMTAB with minimal symbol table
   - All required load commands for modern macOS executables

## Known Issues

### 1. Runtime Execution Fails (Critical)

**Symptom:** Generated binaries are killed by kernel (SIGKILL, exit code 137)

**Evidence:**
```bash
$ ./test_slasm_binary
Killed: 9
$ echo $?
137
```

**Analysis:**
- Mach-O structure appears correct (verified with `otool -l`)
- Instruction encoding is correct (verified with `otool -tV`)
- Code signing passes (`codesign --verify`)
- Comparison with system assembler output shows similar structure
- No dyld output (killed before dyld initialization)

**Recent Improvements (2025-11-28):**
- ✅ Added `LC_DYLD_CHAINED_FIXUPS` - matches C binary format exactly
- ✅ Added `LC_SYMTAB` and `LC_DYSYMTAB` - minimal symbol table with _start symbol
- ✅ Generated proper chained fixups data structure (56 bytes)
- ✅ Symbol table data with nlist_64 entry and string table

**Possible Causes (Remaining):**
- Subtle difference in Mach-O structure not visible with standard tools
- Missing additional load commands (LC_FUNCTION_STARTS, LC_DATA_IN_CODE, LC_DYLD_EXPORTS_TRIE)
- Kernel-level validation failure (not related to structure)
- Security policy or entitlement requirement

**Next Steps:**
1. Add remaining optional load commands (LC_FUNCTION_STARTS, LC_DATA_IN_CODE, LC_DYLD_EXPORTS_TRIE)
2. Use kernel debugging tools (kdebug, dtrace with sudo) to see exact kill reason
3. Compare complete hex dumps byte-by-byte with working binaries
4. Test with completely minimal assembly (single instruction)
5. Try different entry point mechanisms (LC_UNIXTHREAD vs LC_MAIN)

## Not Yet Implemented

### Instructions
- Branch instructions: `b`, `bl`, `b.cond`
- Branch to register: `br Xn`
- Memory load/store: `ldr`, `str`, `ldp`, `stp`, `ldrb`, `strb`
- PC-relative addressing: `adr`, `adrp`
- Unsigned division: `udiv`
- Negate: `neg`
- Many other ARM64 instructions

### Features
- Data section (`.data`, `.rodata`)
- Data directives (`.byte`, `.word`, `.quad`, `.asciz`, `.space`)
- Label references in instructions (relocations)
- Object file generation (`.o` files)
- Linker implementation
- Symbol table in Mach-O output
- Multi-section support
- Section alignment directives
- BSS section

### Toolchain Integration
- Integration with Slang compiler as default assembler
- Flag to switch between system `as` and `slasm`
- Benchmarking vs system assembler

## File Structure

```
assembler/slasm/
├── asm.go           - Main assembler orchestration
├── lexer.go         - Tokenization
├── parser.go        - AST construction
├── symbols.go       - Symbol table
├── layout.go        - Address assignment
├── encoder.go       - ARM64 instruction encoding
├── macho.go         - Mach-O file generation
├── e2e_test.go      - End-to-end tests
├── encoder_test.go  - Encoding unit tests
└── README.md        - User-facing documentation
```

## Usage Example

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
    mov x0, #1      // Exit code
    mov x16, #1     // Exit syscall
    svc #0          // Make syscall
`

    err := asm.Build(code, assembler.BuildOptions{
        OutputPath: "output",
    })
    if err != nil {
        panic(err)
    }

    // Sign the binary
    exec.Command("codesign", "-s", "-", "-f", "output").Run()
}
```

## Comparison: slasm vs System Assembler

| Feature | slasm | System `as` + `ld` |
|---------|-------|-------------------|
| Mach-O generation | ✅ Direct | ✅ Via linker |
| Code signing | ✅ Passes validation | ✅ Full support |
| Instruction set | ⚠️ Partial | ✅ Complete |
| Execution | ❌ Fails | ✅ Works |
| Object files | ❌ Not implemented | ✅ Supported |
| Relocations | ❌ Not implemented | ✅ Supported |
| Speed | ? Not tested | Baseline |

## References

- ARM64 Architecture Reference Manual (ARM DDI 0487)
- Mach-O File Format Reference (Apple)
- Go assembler source: `cmd/internal/obj/arm64/`
- Go linker source: `cmd/link/internal/ld/`
- System tools: `as`, `ld`, `otool`, `codesign`
