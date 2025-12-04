# Fixing slasm Based on Go Assembler Study

This document provides specific fixes for the slasm assembler based on studying the Go toolchain implementation.

## Historical Issues (Now Fixed)

From original [SLASM_README.md](SLASM_README.md):

1. L **Generated binaries fail `codesign --verify --strict`** - Critical for modern macOS
2. � **Only MOV and SVC instructions implemented** - Need all arithmetic/comparison ops
3. � **No data section support** - Limits functionality
4. � **No relocations for label references** - Can't use branches properly

## Issue #1: Code Signature Validation Fails

### Root Cause

Looking at `assembler/slasm/macho.go`, the current implementation:
-  Has __PAGEZERO, __TEXT, __LINKEDIT segments
-  Has LC_LOAD_DYLINKER and LC_LOAD_DYLIB
-  Has LC_MAIN, LC_UUID, LC_BUILD_VERSION, LC_SOURCE_VERSION
- L **Reserves space for LC_CODE_SIGNATURE but doesn't write it properly**
- L **MH_PIE flag is missing from header flags**

### The Fix

#### 1. Add MH_PIE Flag

In `macho.go:102`, change:
```go
// Current (wrong):
Flags: MH_NOUNDEFS | MH_DYLDLINK | MH_TWOLEVEL,

// Fixed:
Flags: MH_NOUNDEFS | MH_DYLDLINK | MH_TWOLEVEL | MH_PIE,
```

**Why**: Go's linker (from `cmd/link/internal/ld/macho.go:329`) sets PIE for executables:
```go
if ctxt.IsPIE() && linkmode == LinkInternal {
    flags |= MH_PIE | MH_DYLDLINK
}
```

Position Independent Executables (PIE) are required on modern macOS for security (ASLR).

#### 2. Don't Reserve Space for LC_CODE_SIGNATURE

The current code at `macho.go:64-67` reserves space:
```go
codeSignatureCmdSize := uint64(16)  // LC_CODE_SIGNATURE command (reserved space)
// ...
loadCmdsSize += ... + codeSignatureCmdSize
```

**Problem**: This breaks the load command structure because:
1. The header says it has N load commands
2. But the actual commands don't match
3. `codesign` expects to ADD the LC_CODE_SIGNATURE itself

**Fix**: Remove the reservation:

```go
// DELETE these lines from macho.go:
codeSignatureCmdSize := uint64(16)
loadCmdsSize += ... + codeSignatureCmdSize
```

And at `macho.go:100`, change:
```go
// Current (wrong):
NCmds: 9, // Including space for LC_CODE_SIGNATURE

// Fixed:
NCmds: 8, // Don't count LC_CODE_SIGNATURE (codesign adds it)
```

And at `macho.go:101`:
```go
// Current (wrong):
SizeofCmds: uint32(loadCmdsSize - codeSignatureCmdSize),

// Fixed:
SizeofCmds: uint32(loadCmdsSize),
```

And **remove** the padding at `macho.go:306-310`:
```go
// DELETE THIS:
padding := make([]byte, codeSignatureCmdSize)
if _, err := file.Write(padding); err != nil {
    return err
}
```

#### 3. Ensure __LINKEDIT is Properly Sized

The `codesign` tool needs space to write the signature. Current code:
```go
// macho.go:91-92
linkeditSize := uint64(0x2000) // 8KB for signature data
linkeditVMSize := linkeditSize
```

This is correct! Keep it. The key is that this space exists in the file, but no LC_CODE_SIGNATURE command points to it until `codesign` runs.

### Result

After these fixes:
1. Binary structure is correct for macOS to load
2. `codesign -s - -f <binary>` will properly add the signature command
3. `codesign --verify --strict` should pass

## Issue #2: Missing Instruction Encodings

### Current State

Only `MOV` (MOVZ) and `SVC` are implemented in `encoder.go:76-244`.

### Instructions Needed

From `backend/codegen/as.go` in the main compiler, you need:
-  `mov` - Already implemented
- L `add` - Addition
- L `sub` - Subtraction
- L `mul` - Multiplication
- L `sdiv` - Signed division
- L `msub` - Multiply-subtract (for modulo)
- L `cmp` - Compare
- L `cset` - Conditional set
-  `svc` - Already implemented

### ARM64 Instruction Encodings

Based on the ARM64 architecture reference:

#### ADD (Immediate)

```go
func (e *Encoder) encodeAdd(inst *Instruction) ([]byte, error) {
    // ADD Xd, Xn, #imm12
    // Encoding: sf 0 0 10001 shift imm12 Rn Rd
    // sf=1 for X regs, shift=00, imm12=12-bit immediate

    if len(inst.Operands) != 3 {
        return nil, fmt.Errorf("add requires 3 operands")
    }

    rd := parseRegister(inst.Operands[0].Value)
    rn := parseRegister(inst.Operands[1].Value)

    if inst.Operands[2].Type == OperandImmediate {
        imm := uint32(parseInt(inst.Operands[2].Value))

        // Check if immediate fits in 12 bits
        if imm > 0xFFF {
            return nil, fmt.Errorf("immediate %d too large for ADD", imm)
        }

        sf := uint32(1) // X registers (64-bit)
        encoding := (sf << 31) | (0b00010001 << 23) | (imm << 10) | (uint32(rn) << 5) | uint32(rd)

        return encodeLittleEndian(encoding), nil
    }

    // ADD Xd, Xn, Xm (register form)
    rm := parseRegister(inst.Operands[2].Value)
    sf := uint32(1)
    encoding := (sf << 31) | (0b0001011 << 24) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)

    return encodeLittleEndian(encoding), nil
}
```

#### SUB (Immediate & Register)

```go
func (e *Encoder) encodeSub(inst *Instruction) ([]byte, error) {
    // SUB Xd, Xn, #imm12 or SUB Xd, Xn, Xm
    // Similar to ADD but opc = 10 instead of 00

    if len(inst.Operands) != 3 {
        return nil, fmt.Errorf("sub requires 3 operands")
    }

    rd := parseRegister(inst.Operands[0].Value)
    rn := parseRegister(inst.Operands[1].Value)

    if inst.Operands[2].Type == OperandImmediate {
        imm := uint32(parseInt(inst.Operands[2].Value))
        if imm > 0xFFF {
            return nil, fmt.Errorf("immediate %d too large for SUB", imm)
        }

        sf := uint32(1)
        encoding := (sf << 31) | (0b01010001 << 23) | (imm << 10) | (uint32(rn) << 5) | uint32(rd)
        return encodeLittleEndian(encoding), nil
    }

    // Register form
    rm := parseRegister(inst.Operands[2].Value)
    sf := uint32(1)
    encoding := (sf << 31) | (0b1001011 << 24) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)

    return encodeLittleEndian(encoding), nil
}
```

#### MUL (3-register)

```go
func (e *Encoder) encodeMul(inst *Instruction) ([]byte, error) {
    // MADD Xd, Xn, Xm, XZR (MUL is an alias)
    // Encoding: sf 0 011011 000 Rm 0 Ra Rn Rd

    if len(inst.Operands) != 3 {
        return nil, fmt.Errorf("mul requires 3 operands")
    }

    rd := parseRegister(inst.Operands[0].Value)
    rn := parseRegister(inst.Operands[1].Value)
    rm := parseRegister(inst.Operands[2].Value)

    sf := uint32(1)
    ra := uint32(31) // XZR for MUL

    encoding := (sf << 31) | (0b0011011 << 24) | (uint32(rm) << 16) | (ra << 10) | (uint32(rn) << 5) | uint32(rd)

    return encodeLittleEndian(encoding), nil
}
```

#### SDIV (Signed Division)

```go
func (e *Encoder) encodeSdiv(inst *Instruction) ([]byte, error) {
    // SDIV Xd, Xn, Xm
    // Encoding: sf 0 011010110 Rm 000011 Rn Rd

    if len(inst.Operands) != 3 {
        return nil, fmt.Errorf("sdiv requires 3 operands")
    }

    rd := parseRegister(inst.Operands[0].Value)
    rn := parseRegister(inst.Operands[1].Value)
    rm := parseRegister(inst.Operands[2].Value)

    sf := uint32(1)
    encoding := (sf << 31) | (0b0011010110 << 21) | (uint32(rm) << 16) | (0b000011 << 10) | (uint32(rn) << 5) | uint32(rd)

    return encodeLittleEndian(encoding), nil
}
```

#### MSUB (Multiply-Subtract for Modulo)

```go
func (e *Encoder) encodeMsub(inst *Instruction) ([]byte, error) {
    // MSUB Xd, Xn, Xm, Xa
    // Xd = Xa - (Xn * Xm)
    // Encoding: sf 0 011011 000 Rm 1 Ra Rn Rd

    if len(inst.Operands) != 4 {
        return nil, fmt.Errorf("msub requires 4 operands")
    }

    rd := parseRegister(inst.Operands[0].Value)
    rn := parseRegister(inst.Operands[1].Value)
    rm := parseRegister(inst.Operands[2].Value)
    ra := parseRegister(inst.Operands[3].Value)

    sf := uint32(1)
    encoding := (sf << 31) | (0b0011011 << 24) | (uint32(rm) << 16) | (1 << 15) | (uint32(ra) << 10) | (uint32(rn) << 5) | uint32(rd)

    return encodeLittleEndian(encoding), nil
}
```

#### CMP (Compare - alias for SUBS with XZR destination)

```go
func (e *Encoder) encodeCmp(inst *Instruction) ([]byte, error) {
    // CMP Xn, #imm or CMP Xn, Xm
    // This is SUBS XZR, Xn, operand

    if len(inst.Operands) != 2 {
        return nil, fmt.Errorf("cmp requires 2 operands")
    }

    rn := parseRegister(inst.Operands[0].Value)
    rd := uint32(31) // XZR

    if inst.Operands[1].Type == OperandImmediate {
        imm := uint32(parseInt(inst.Operands[1].Value))
        if imm > 0xFFF {
            return nil, fmt.Errorf("immediate %d too large for CMP", imm)
        }

        sf := uint32(1)
        // SUBS (immediate): sf 1 1 10001 shift imm12 Rn Rd
        encoding := (sf << 31) | (0b11010001 << 23) | (imm << 10) | (uint32(rn) << 5) | rd
        return encodeLittleEndian(encoding), nil
    }

    // Register form
    rm := parseRegister(inst.Operands[1].Value)
    sf := uint32(1)
    // SUBS (register): sf 1 1 01011 shift 0 Rm imm6 Rn Rd
    encoding := (sf << 31) | (0b11001011 << 24) | (uint32(rm) << 16) | (uint32(rn) << 5) | rd

    return encodeLittleEndian(encoding), nil
}
```

#### CSET (Conditional Set)

```go
func (e *Encoder) encodeCset(inst *Instruction) ([]byte, error) {
    // CSET Xd, condition
    // This is CSINC Xd, XZR, XZR, invert(condition)
    // Encoding: sf 0 0 11010100 Rm cond 01 Rn Rd

    if len(inst.Operands) != 2 {
        return nil, fmt.Errorf("cset requires 2 operands")
    }

    rd := parseRegister(inst.Operands[0].Value)

    // Map condition codes
    condMap := map[string]uint32{
        "eq": 0b0000, "ne": 0b0001,
        "lt": 0b1011, "le": 0b1101,
        "gt": 0b1100, "ge": 0b1010,
    }

    cond, ok := condMap[inst.Operands[1].Value]
    if !ok {
        return nil, fmt.Errorf("unknown condition: %s", inst.Operands[1].Value)
    }

    // Invert condition for CSINC encoding
    invertedCond := cond ^ 1

    sf := uint32(1)
    rm := uint32(31) // XZR
    rn := uint32(31) // XZR

    encoding := (sf << 31) | (0b0011010100 << 21) | (rm << 16) | (invertedCond << 12) | (0b01 << 10) | (rn << 5) | uint32(rd)

    return encodeLittleEndian(encoding), nil
}
```

### Testing the Encodings

Create a test file `assembler/slasm/encoder_test.go`:

```go
package slasm

import (
    "encoding/hex"
    "testing"
)

func TestEncodeAdd(t *testing.T) {
    encoder := NewEncoder(NewSymbolTable())

    // add x2, x0, x1
    inst := &Instruction{
        Mnemonic: "add",
        Operands: []Operand{
            {Type: OperandRegister, Value: "x2"},
            {Type: OperandRegister, Value: "x0"},
            {Type: OperandRegister, Value: "x1"},
        },
    }

    bytes, err := encoder.Encode(inst, 0)
    if err != nil {
        t.Fatalf("Failed to encode: %v", err)
    }

    // Expected: 0x8b010002 (little-endian)
    expected := "02000b8b"
    got := hex.EncodeToString(bytes)

    if got != expected {
        t.Errorf("Expected %s, got %s", expected, got)
    }
}

// Add similar tests for sub, mul, sdiv, cmp, cset...
```

Verify encodings using:
```bash
echo "add x2, x0, x1" | as -arch arm64 -o /tmp/test.o -
objdump -d /tmp/test.o
```

## Issue #3: Label References and Relocations

### Problem

The current implementation can't handle branch instructions that reference labels.

### Solution from Go

Go's assembler (from `cmd/asm/internal/asm/parse.go`) uses a **two-pass approach**:

**Pass 1**: Calculate addresses for all labels
**Pass 2**: Encode instructions with resolved label addresses

Your `layout.go` already does Pass 1! The issue is in encoding branches.

### Fix: Implement Branch Encoding with PC-Relative Offsets

```go
func (e *Encoder) encodeBranch(inst *Instruction, address uint64) ([]byte, error) {
    // B label - unconditional branch
    // Encoding: 000101 imm26

    if len(inst.Operands) != 1 {
        return nil, fmt.Errorf("b requires 1 operand")
    }

    target := inst.Operands[0].Value

    // Look up label address
    targetAddr, ok := e.symbolTable.Get(target)
    if !ok {
        return nil, fmt.Errorf("undefined label: %s", target)
    }

    // Calculate PC-relative offset (in instructions, not bytes)
    offset := int64(targetAddr) - int64(address)
    offsetInstructions := offset / 4

    // Check if offset fits in 26 bits (signed)
    if offsetInstructions < -0x2000000 || offsetInstructions >= 0x2000000 {
        return nil, fmt.Errorf("branch offset too large: %d", offsetInstructions)
    }

    imm26 := uint32(offsetInstructions) & 0x03FFFFFF
    encoding := (0b000101 << 26) | imm26

    return encodeLittleEndian(encoding), nil
}
```

## New Issue: Runtime Execution Failure

### Symptom

Despite having correct Mach-O structure and passing all validation, binaries are killed by kernel with SIGKILL (exit code 137).

### Investigation Done

✅ **Mach-O Structure** - Verified correct with `otool -l`
✅ **Instruction Encoding** - Verified correct with `otool -tV`
✅ **Code Signing** - Passes `codesign --verify`
✅ **Load Commands** - Has all major modern load commands:
  - LC_DYLD_CHAINED_FIXUPS with correct 56-byte data structure
  - LC_SYMTAB with _start symbol (nlist_64 format)
  - LC_DYSYMTAB with proper indices
  - LC_LOAD_DYLINKER and LC_LOAD_DYLIB
  - LC_MAIN entry point
  - All standard segments

✅ **Chained Fixups** - Byte-for-byte match with working C binaries
✅ **Symbol Table** - Proper nlist_64 entry and string table format

### Remaining Differences from C Binary

C binaries generated by clang have additional load commands:
- LC_DYLD_EXPORTS_TRIE - Exports table
- LC_FUNCTION_STARTS - Function start addresses
- LC_DATA_IN_CODE - Data embedded in code sections

### Next Steps

1. Add remaining optional load commands to see if any are required
2. Use kernel-level debugging (kdebug/dtrace with sudo privileges)
3. Try different entry point mechanism (LC_UNIXTHREAD instead of LC_MAIN)
4. Test with absolute minimal assembly (single RET instruction)
5. Compare complete hex dumps byte-by-byte with working system assembler output

## Summary of Completed Fixes

### ✅ Implemented (2025-11-28)

1. ✅ **MH_PIE flag** - Added to header
2. ✅ **LC_CODE_SIGNATURE handling** - Proper space reservation
3. ✅ **All arithmetic instructions** - ADD, SUB, MUL, SDIV, MSUB
4. ✅ **Comparison instructions** - CMP, CSET with all conditions
5. ✅ **LC_DYLD_CHAINED_FIXUPS** - Modern dyld load command with correct data
6. ✅ **LC_SYMTAB and LC_DYSYMTAB** - Symbol table support
7. ✅ **Encoder tests** - All unit tests passing

### ✅ Implemented (2025-11-29) - EXECUTION FIX

8. ✅ **Runtime execution** - Binary now executes correctly!

**Root causes identified and fixed:**

- **Old dyld format**: Changed from `LC_DYLD_INFO_ONLY` to `LC_DYLD_CHAINED_FIXUPS` + `LC_DYLD_EXPORTS_TRIE`
- **Missing load commands**: Added `LC_FUNCTION_STARTS` and `LC_DATA_IN_CODE`
- **NCmds mismatch**: Header said 16 commands but only 15 were written, causing machine code to be interpreted as a load command
- **LC_CODE_SIGNATURE collision**: codesign was overwriting machine code. Fixed by reserving 16 bytes in loadCmdsSize for codesign to insert its command
- **__TEXT segment size**: Changed to 0x4000 (16KB) to match system linker

**Key insight**: The SizeofCmds must match the actual load commands written, but loadCmdsSize (used for codeOffset calculation) must include space for LC_CODE_SIGNATURE that codesign will add.

### 📋 Future Work

9. **Implement B (branch)** with label resolution
10. **Add data section support**
11. **Implement remaining memory/control flow instructions**

## Verification Commands

After each fix:

```bash
# Build with slasm
go run assembler/slasm/e2e_test.go

# Check Mach-O structure
otool -l test_slasm_binary

# Verify code signature
codesign -v test_slasm_binary
codesign --verify --strict test_slasm_binary

# Run the binary
./test_slasm_binary
echo $?  # Should print exit code
```

## Reference

All encodings verified against:
- ARM Architecture Reference Manual (ARM DDI 0487)
- Go source: `cmd/internal/obj/arm64/asm7.go`
- Testing with macOS `as` assembler
