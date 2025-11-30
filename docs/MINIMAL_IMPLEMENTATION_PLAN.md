# Minimal End-to-End Implementation Plan

## Goal
Implement the simplest possible slasm assembler that can assemble and execute a basic ARM64 program.

## Target Program

We'll aim to assemble this minimal program (`_examples/arm64/simple.s`):

```asm
.global _start

_start:
    mov x0, #1
    mov x16, #1
    svc #0
```

This program exits with status code 1.

## Minimal Feature Set

### Phase 1: Lexer (lexer.go)
**Implement only what we need:**
- ✅ Directives: `.global`
- ✅ Labels: `_start:`
- ✅ Instructions: `mov`, `svc`
- ✅ Registers: `x0-x30` (64-bit only, skip w registers for now)
- ✅ Immediates: `#1`, `#0`
- ✅ Punctuation: `:`, `,`, `#`
- ✅ Comments: `//` and `;` style
- ✅ Whitespace handling

**Skip:**
- String literals
- Data directives (.space, .byte, .asciz)
- Memory operands ([sp, #16])
- @PAGE/@PAGEOFF relocations
- w registers

### Phase 2: Parser (parser.go)
**Implement only what we need:**
- ✅ Parse `.global` directive
- ✅ Parse label definitions
- ✅ Parse simple instructions with register and immediate operands
- ✅ Build minimal Program IR with one text section

**Skip:**
- .data/.text section switching
- Data declarations
- Memory operands
- Complex operand types

### Phase 3: Symbol Table (symbols.go)
**Already mostly done, just implement:**
- ✅ Define() with duplicate checking
- ✅ Lookup() (already works)

### Phase 4: Layout (layout.go)
**Implement only what we need:**
- ✅ Simple single-pass through .text section
- ✅ Assign address 0 to first label
- ✅ Each instruction is 4 bytes
- ✅ Build symbol table

**Skip:**
- Alignment directives
- Data section
- Two-pass layout

### Phase 5: Encoder (encoder.go)
**Minimal instructions (Phase 1):**
- ✅ `mov Rd, #imm` → MOVZ encoding (with proper operand parsing)
- ✅ `mov Rd, Rm` → ORR encoding (register-to-register)
- ✅ `svc #imm` → SVC encoding

**Extended instructions (Phase 2 - for Slang compiler):**
- ✅ `add Rd, Rn, #imm` → ADD immediate encoding
- ✅ `add Rd, Rn, Rm` → ADD register encoding
- ✅ `sub Rd, Rn, #imm` → SUB immediate encoding
- ✅ `sub Rd, Rn, Rm` → SUB register encoding
- ✅ `mul Rd, Rn, Rm` → MADD encoding (MUL alias)
- ✅ `sdiv Rd, Rn, Rm` → SDIV encoding
- ✅ `msub Rd, Rn, Rm, Ra` → MSUB encoding (for modulo)
- ✅ `cmp Rn, #imm` → SUBS immediate encoding
- ✅ `cmp Rn, Rm` → SUBS register encoding
- ✅ `cset Rd, cond` → CSINC encoding
- ✅ `ret` → BR x30 encoding

**Still TODO:**
- Branch instructions (b, bl, br)
- Memory operations (ldr, str, ldp, stp)
- ADR/ADRP for PC-relative addressing

### Phase 6: Mach-O Writer (macho.go)
**Minimal executable generation:**
- ✅ Define basic Mach-O structures (header, segment, section)
- ✅ WriteExecutable() for minimal executable:
  - ✅ Mach header (ARM64, executable, with MH_PIE flag)
  - ✅ __PAGEZERO segment for memory protection
  - ✅ __TEXT segment with __text section (fileoff=0, includes headers)
  - ✅ __LINKEDIT segment for code signatures
  - ✅ LC_LOAD_DYLINKER and LC_LOAD_DYLIB commands
  - ✅ LC_MAIN load command pointing to entry point
  - ✅ LC_UUID, LC_BUILD_VERSION, LC_SOURCE_VERSION
  - ✅ Proper segment/section alignment (page-aligned)
  - ✅ __TEXT segment vmsize = filesize (both page-aligned)
  - ✅ __text section vmaddr correctly offset from __TEXT base
  - ✅ Space reserved for codesign to add LC_CODE_SIGNATURE

**Fixed issues:**
- ✅ __TEXT segment starts at file offset 0 (includes headers and load commands)
- ✅ __text section vmaddr = __TEXT vmaddr + code offset
- ✅ vmsize and filesize both page-aligned and matching
- ✅ Code offset page-aligned to leave room for codesign
- ✅ Passes `codesign --verify` validation

**Skip:**
- Object file generation
- Symbol table/string table
- Relocations
- __DATA segment

### Phase 7: Main Assembler (asm.go)
**Implement Build() method:**
1. ✅ Read assembly source string
2. ✅ Lex → Parse → Layout → Encode
3. ✅ Write executable Mach-O
4. ✅ Error handling

**Skip:**
- Assemble() and Link() methods
- File I/O (work with strings for now)

## Success Criteria

When done, this should work:

```go
asm := slasm.New()
code := `.global _start
_start:
    mov x0, #1
    mov x16, #1
    svc #0
`
err := asm.Build(code, assembler.BuildOptions{
    OutputPath: "test_output",
})
// Should create executable that exits with code 1
```

## Estimated Implementation

- **Lexer**: ~100 lines
- **Parser**: ~80 lines
- **Symbol Table**: ~20 lines (mostly done)
- **Layout**: ~50 lines
- **Encoder**: ~60 lines (just 2 instructions)
- **Mach-O Writer**: ~150 lines (minimal)
- **Main Assembler**: ~80 lines

**Total**: ~540 lines of actual implementation code

## Test Strategy

We'll use the existing tests but only expect a subset to pass:
- Lexer tests: Most should pass
- Parser tests: Simple instruction tests should pass
- Symbol tests: All should pass
- Layout tests: Simple tests should pass
- Encoder: We won't have tests, will verify with execution

## Current Status (Updated 2025-11-29)

### ✅ What Works

The assembler successfully:
- Lexes and parses assembly source code
- Encodes ARM64 instructions to machine code
- Generates valid Mach-O executables with proper structure
- Passes `codesign --verify` validation
- All unit tests for encoding pass (ADD, SUB, MUL, SDIV, MSUB, CMP, CSET)
- Generates LC_DYLD_CHAINED_FIXUPS with correct format
- Generates LC_SYMTAB and LC_DYSYMTAB with minimal symbol table
- **Generated binaries execute correctly!**

Generated binaries have:
- Correct instruction encoding (verified with `otool -tV`)
- Proper Mach-O structure (segments, sections, load commands)
- Valid code signatures
- Proper alignment and offsets
- Modern macOS load commands (chained fixups, symbol tables)
- Chained fixups data that matches C binaries
- Symbol table with _start symbol in nlist_64 format

### ⚠️ Current Limitations

1. **Limited instruction set** - Only implements instructions needed by Slang compiler
2. **No data section support** - Cannot handle `.data` sections or data directives
3. **No branch instructions with label resolution** - Branch encoding exists but label resolution is incomplete
4. **No relocations** - Cannot handle label references that require relocation

### Test Results

```bash
# All encoding unit tests pass
go test ./assembler/slasm/... -run TestEncode
# PASS: TestEncodeAdd, TestEncodeSub, TestEncodeMul, TestEncodeSdiv, TestEncodeMsub, TestEncodeCmp, TestEncodeCset

# Layout tests pass
go test ./assembler/slasm/... -run TestLayout
# PASS: All layout tests

# E2E tests pass - binaries execute correctly
go test ./assembler/slasm/... -run TestEndToEnd
# PASS: All end-to-end tests
```

## Next Steps

### Feature Development
1. **Add branch instructions** - `b`, `bl`, `b.cond` with label resolution
2. **Add memory operations** - `ldr`, `str`, `ldp`, `stp`
3. **Add data section support** - `.data`, `.rodata`, `.asciz`, `.byte`, `.word`
4. **Add relocations** - Support label references in instructions
5. **Generate object files** - Implement `Assemble()` for `.o` files
6. **Implement linker** - Implement `Link()` to combine object files

## Implementation Progress

### Phase 6: Mach-O Writer - ✅ COMPLETE

**All features implemented:**
- ✅ LC_DYLD_CHAINED_FIXUPS command and data generation
- ✅ LC_SYMTAB command with symbol table data
- ✅ LC_DYSYMTAB command with proper indices
- ✅ Minimal symbol table (_start symbol in nlist_64 format)
- ✅ String table with proper null-termination
- ✅ Chained fixups data structure

**Mach-O structure:**
- ✅ Mach header (ARM64, executable, with MH_PIE flag)
- ✅ __PAGEZERO segment for memory protection
- ✅ __TEXT segment with __text section
- ✅ __LINKEDIT segment with chained fixups and symbol table data
- ✅ LC_LOAD_DYLINKER and LC_LOAD_DYLIB commands
- ✅ LC_MAIN load command pointing to entry point
- ✅ LC_UUID, LC_BUILD_VERSION, LC_SOURCE_VERSION
- ✅ LC_DYLD_CHAINED_FIXUPS with correct data format
- ✅ LC_SYMTAB and LC_DYSYMTAB with minimal symbol table
- ✅ Proper segment/section alignment (page-aligned)
- ✅ Space reserved for codesign to add LC_CODE_SIGNATURE
- ✅ Passes `codesign --verify` validation
- ✅ **Generated binaries execute correctly!**
