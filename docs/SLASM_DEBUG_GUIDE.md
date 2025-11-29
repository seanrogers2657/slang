# SLASM Assembler - Debug Guide

This guide explains how to use the comprehensive debugging tools built into the slasm assembler.

## Quick Start

### Option 1: Run the Standalone Debug Program

```bash
go run cmd/slasm-debug/main.go
```

This creates a simple test binary with full debug output showing every stage of the build process.

### Option 2: Run Debug Tests

```bash
# Minimal program (just 2 instructions)
go test -v ./assembler/slasm -run TestDebugExample_MinimalProgram

# Syscall program (3 instructions with svc)
go test -v ./assembler/slasm -run TestDebugExample_SimpleSyscall

# Arithmetic program (5 instructions with add/sub)
go test -v ./assembler/slasm -run TestDebugExample_Arithmetic
```

## What You'll See

The debug output shows 7 stages of the build pipeline:

### STEP 1: LEXER
Shows all tokens generated from the assembly source:
```
Lexer produced 16 tokens:
  [  0] 6              : .global
  [  1] 3              : _start
  [  2] 1              :
  [  3] 3              : mov
  [  4] 7              : x0
  [  5] 8              : ,
  [  6] 10             : #
  [  7] 4              : 42
  ...
```

**What to look for:**
- Token count matches expectations
- Token types are correct (6=directive, 3=identifier, 7=register, etc.)
- Values are extracted correctly

### STEP 2: PARSER
Shows the parsed program structure:
```
Parser produced 1 section(s):
  Section 0: 1 (4 items)
    [  0] Directive: .global [_start]
    [  1] Label: _start
    [  2] Instruction: mov    x0, 42
    [  3] Instruction: ret
```

**What to look for:**
- Correct number of sections (usually 1 for simple programs)
- Labels are recognized
- Instructions parsed with correct operands
- Directives processed correctly

### STEP 3: LAYOUT & SYMBOL TABLE
Shows symbol addresses:
```
Symbol table:
  _start         : addr=0x0000 section=1
```

**What to look for:**
- All labels appear in symbol table
- Addresses are sequential (0x0000, 0x0004, 0x0008, etc. - each instruction is 4 bytes)
- Global symbols marked with [GLOBAL]

### STEP 4: INSTRUCTION ENCODING
Shows machine code for each instruction:
```
  [0x0000] mov x0, 42           -> 40 05 80 d2 (0xd2800540)
  [0x0004] ret                  -> c0 03 5f d6 (0xd65f03c0)

Encoded 2 instructions (8 bytes total)
Complete machine code: 400580d2c0035fd6
```

**What to look for:**
- Each instruction shows its address, assembly form, and hex encoding
- Byte order is correct (little-endian: bytes reversed in final output)
- Total byte count = instruction count × 4
- You can verify encodings with `otool -tV` on the final binary

**Common encodings to recognize:**
- `mov x0, #N` → `d2 80 0N xx` (MOVZ instruction)
- `ret` → `c0 03 5f d6` (always the same)
- `svc #0` → `01 00 00 d4` (syscall)
- `add x2, x0, x1` → register add pattern

### STEP 5: MACH-O GENERATION
Shows the Mach-O file structure:
```
Mach-O Structure:
  Header:            size=32 bytes
  Load commands:     size=480 bytes, count=9
  Code offset:       0x1000 (4096 bytes)
  Code size:         12 bytes

Segments:
  __PAGEZERO:        vm=0x0-0x100000000 (size=0x100000000)
  __TEXT:            vm=0x100000000-0x100002000 (size=0x2000), file=0x0-0x2000
    __text section:  vm=0x100001000-0x10000100c (size=0xc), file=0x1000
  __LINKEDIT:        vm=0x100002000-0x100004000 (size=0x2000), file=0x2000

Entry point:         0x100001000 (file offset 0x1000)
Total file size:     16384 bytes
```

**What to look for:**
- Code offset should be page-aligned (0x1000 = 4096 = 4KB)
- Entry point = VM base (0x100000000) + code offset (0x1000)
- __PAGEZERO protects against null pointer dereferences
- __LINKEDIT is reserved for code signatures (filled by codesign)
- Total file size is reasonable for the code size

### STEP 6: FILE PERMISSIONS
```
Set executable permissions (0755)
```

### STEP 7: CODE SIGNING
```
Successfully signed binary with ad-hoc signature
```

**What to look for:**
- Should succeed (required on modern macOS)
- If it fails, check if codesign is available

### BUILD SUMMARY
Final statistics:
```
========== BUILD SUMMARY ==========
Output file:       ./test_slasm_binary
Architecture:      arm64
Entry point:       _start
Instructions:      3
Code size:         12 bytes
Symbols:           1
===================================
```

## Verification Tools

The debug program also runs verification commands:

### 1. File Type
```bash
file ./test_slasm_binary
```
Should show: `Mach-O 64-bit executable arm64`

### 2. Disassembly
```bash
otool -tV ./test_slasm_binary
```
Shows the actual machine code decoded back to assembly - should match your input!

Example:
```
(__TEXT,__text) section
0000000100001000	mov	x0, #0x2a
0000000100001004	mov	x16, #0x1
0000000100001008	svc	#0
```

### 3. Code Signature
```bash
codesign --verify --verbose ./test_slasm_binary
```
Should show: `valid on disk` and `satisfies its Designated Requirement`

### 4. Mach-O Header
```bash
otool -hv ./test_slasm_binary
```
Shows file type, CPU type, flags, and load command count.

### 5. Load Commands
```bash
otool -l ./test_slasm_binary
```
Shows all load commands in detail (segments, dylinker, entry point, etc.)

## Debugging Common Issues

### Issue: "Lexer error"
- Check for syntax errors in assembly source
- Verify token types are recognized (registers, immediates, etc.)

### Issue: "Parser error"
- Check instruction format (correct number of operands)
- Verify labels end with `:`
- Check directive syntax (`.global`, `.text`, etc.)

### Issue: "Layout error"
- Check for duplicate label definitions
- Verify all referenced labels are defined

### Issue: "Encoding error for instruction"
- Check if instruction is supported (see README for supported instructions)
- Verify operand types (register vs immediate)
- Check immediate value range (16-bit for mov, 12-bit for add/sub)

### Issue: "Mach-O generation error"
- Usually indicates a bug in the assembler
- Check the debug output to see what was generated

### Issue: Binary created but execution fails
This is the **known issue** - binaries are valid but fail at runtime. The debug output helps narrow down whether the issue is:
- Instruction encoding (check with otool -tV)
- Mach-O structure (check with otool -l)
- Code signing (check with codesign --verify)
- Something at the kernel level (likely the current issue)

## Tips

1. **Start simple**: Test with the minimal program first
2. **Compare encodings**: Use otool -tV to verify your machine code
3. **Check addresses**: Symbol addresses should increment by 4 for each instruction
4. **Verify hex bytes**: Little-endian can be confusing - the bytes are reversed
5. **Use the standalone program**: It's the easiest way to see everything at once

## Example: Debugging a New Instruction

If you're adding support for a new instruction, here's how to debug it:

1. Write a test program using just that instruction
2. Run the debug build program
3. Check STEP 1: Verify the instruction is tokenized correctly
4. Check STEP 2: Verify it's parsed as an instruction (not a label or directive)
5. Check STEP 4: Verify the encoding matches ARM64 spec
6. Run `otool -tV` on the output to see if the system assembler agrees

Example for adding `neg x0, x1`:
```assembly
.global _start
_start:
    neg x0, x1
    ret
```

Then check:
- Lexer shows: `TokenIdentifier: neg`, `TokenRegister: x0`, `TokenRegister: x1`
- Parser shows: `Instruction: neg    x0, x1`
- Encoder shows: `neg x0, x1 -> [bytes]` (verify these against ARM64 spec)
- otool shows: `neg x0, x1` (system agrees with your encoding)

## Files

- `cmd/slasm-debug/main.go` - Standalone debug build program
- `debug_example_test.go` - Debug tests with comprehensive output
- `asm.go` - Build() method contains all debug logging
- `macho.go` - Mach-O generation with structure details

## More Information

See the main README.md for:
- Supported instructions
- ARM64 encoding reference
- Mach-O structure details
- Known issues and future work
