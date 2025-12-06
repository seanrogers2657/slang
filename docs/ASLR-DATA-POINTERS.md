# ASLR and Data Section Pointers

This document describes the challenges of handling pointers in the data section for Position Independent Executables (PIE) on macOS ARM64, and the current workarounds implemented in slasm.

## Background

### Position Independent Executables (PIE)

Modern macOS requires executables to be position-independent (PIE) for security. The `MH_PIE` flag in the Mach-O header enables Address Space Layout Randomization (ASLR), which loads the binary at a random address each time it runs.

- **Preferred load address**: 0x100000000 (4GB)
- **Actual load address**: Random, e.g., 0x102345000
- **ASLR slide**: actual - preferred = 0x2345000

### The Problem

When assembly code uses `.quad label` directives to store pointers in the data section, the assembler embeds the absolute address based on the preferred load address:

```asm
.data
_pointer_table:
    .quad _my_function      // Stores 0x100001000 (preferred address)
    .quad _another_function // Stores 0x100001100
```

At runtime with ASLR:
- The actual address of `_my_function` is 0x102346000 (preferred + slide)
- But the data section still contains 0x100001000
- Loading from `_pointer_table` gives the wrong address
- Dereferencing causes a crash or undefined behavior

### Proper Solution: Chained Fixups

The correct solution is to use **chained fixups** (LC_DYLD_CHAINED_FIXUPS), which tell dyld to rebase pointers at load time. This requires:

1. Storing pointer values in a special chained format (DYLD_CHAINED_PTR_64)
2. Generating proper chained fixups data in __LINKEDIT
3. Pointing LC_DYLD_CHAINED_FIXUPS to this data

The chained pointer format packs:
- Target offset (36 bits)
- High8 bits (8 bits)
- Reserved (7 bits)
- Next pointer delta (12 bits)
- Bind flag (1 bit)

This is complex to implement correctly and is tracked as future work.

## Current Workarounds

### 1. Error Message Lookup (runtime.go)

**Problem**: The panic handler used a table of pointers to error messages:
```asm
_slang_error_messages:
    .quad _err_msg_1    // Absolute pointer - broken with ASLR
    .quad _err_msg_2
```

**Solution**: Use PC-relative addressing with a switch/case pattern:
```asm
    cmp x19, #1
    beq _panic_msg_1
    // ...

_panic_msg_1:
    adrp x1, _err_msg_1@PAGE      // PC-relative, works with ASLR
    add x1, x1, _err_msg_1@PAGEOFF
    mov x2, #26
    b _panic_msg_print
```

The `adrp`/`add` instructions compute addresses relative to the program counter, which is always correct regardless of load address.

### 2. Symbol Table Lookup (runtime.go, symtab.go)

**Problem**: The symbol table stores function addresses and string pointers:
```asm
_slang_symtab:
    .quad _main           // Function start - broken with ASLR
    .quad _main_end       // Function end
    .quad _symtab_name_0  // Name pointer - broken with ASLR
```

**Solution**: Compute the ASLR slide at runtime and adjust all loaded pointers:

1. Store a reference pointer that we can compare:
```asm
_slang_symtab_ref:
    .quad _slang_symtab   // Expected address
```

2. At lookup time, compute the slide:
```asm
    // Get actual runtime address
    adrp x20, _slang_symtab@PAGE
    add x20, x20, _slang_symtab@PAGEOFF

    // Get expected address from data
    adrp x8, _slang_symtab_ref@PAGE
    add x8, x8, _slang_symtab_ref@PAGEOFF
    ldr x21, [x8]

    // Compute slide
    sub x22, x20, x21     // slide = actual - expected
```

3. Add the slide to all pointer loads:
```asm
    ldr x9, [x8]          // Load stored address
    add x9, x9, x22       // Apply slide to get actual address
```

### 3. Relocation Tracking (encoder.go, macho.go)

The encoder now tracks which data locations contain label references:

```go
type DataRelocation struct {
    Offset     uint64 // Offset within data section
    Size       int    // Pointer size (typically 8)
    TargetAddr uint64 // The absolute address that was written
}
```

This information is passed through to the Mach-O writer for future implementation of proper chained fixups.

## Files Involved

| File | Purpose |
|------|---------|
| `assembler/slasm/ir.go` | DataRelocation struct definition |
| `assembler/slasm/encoder.go` | EncodeDataWithRelocations() tracks label refs |
| `assembler/slasm/asm.go` | Collects relocations during encoding |
| `assembler/slasm/macho.go` | generateChainedFixupsWithRelocations() |
| `backend/codegen/runtime.go` | ASLR-safe panic handler |
| `backend/codegen/symtab.go` | Symbol table with reference pointer |

## ✅ Chained Fixups Implementation (COMPLETED)

Proper chained fixups are now implemented in `generateChainedFixupsWithRelocations()`:

1. **Data encoding**: Pointers are encoded in DYLD_CHAINED_PTR_64_OFFSET format:
   - bits 0-35: target offset from image base
   - bits 36-43: high8 (for addresses > 36 bits)
   - bits 51-62: next pointer delta (in 4-byte units)
   - bit 63: bind flag (0 for rebase)

2. **Fixup chain**: Pointers within each 16KB page are linked together via the `next` field

3. **Fixups header**: Complete structure generated:
   - `dyld_chained_fixups_header` with proper offsets
   - `dyld_chained_starts_in_image` with 4 segments (PAGEZERO, TEXT, DATA, LINKEDIT)
   - `dyld_chained_starts_in_segment` for __DATA with `pointer_format = 6`

4. **LC_DYLD_CHAINED_FIXUPS**: Points to the generated fixups data in __LINKEDIT

### References

- [Apple Mach-O Format](https://developer.apple.com/documentation/kernel/mach-o)
- [dyld source code](https://github.com/apple-oss-distributions/dyld) - see MachOFile.cpp
- `<mach-o/fixup-chains.h>` - Chained fixups structures

### Testing

Verified with:
1. Runtime panic tests with stack traces (use symbol table pointers)
2. Multiple runs to verify ASLR handling
3. `otool -chained_fixups` shows correct structure matching system assembler output

## Why Not Disable PIE?

Removing the `MH_PIE` flag would load the binary at a fixed address, avoiding ASLR issues. However:

1. **macOS rejects non-PIE binaries** on Apple Silicon - they are killed immediately
2. **Security implications** - fixed addresses make exploits easier
3. **Not future-proof** - Apple may further restrict non-PIE binaries

The current workarounds are the correct approach for code we control (runtime). For arbitrary user code with data pointers, proper chained fixups are needed.
