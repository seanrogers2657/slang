# Mach-O File Format Reference

This document provides a comprehensive reference for the Mach-O (Mach Object) file format used by macOS, iOS, and other Apple operating systems. It's based on studying the Go toolchain implementation and Apple's documentation.

## Overview

Mach-O is the native executable format for Apple platforms. It supports:
- **Object files** (`.o`) - Relocatable code from the assembler
- **Executables** - Directly runnable programs
- **Dynamic libraries** (`.dylib`) - Shared libraries
- **Bundles** - Loadable code modules

## File Structure

A Mach-O file consists of three main regions:

```
+------------------+
| Header           |  Mach header with magic number, CPU type, file type
+------------------+
| Load Commands    |  Instructions for loading the file
|  - Segments      |  Memory mapping instructions
|  - Dylinker      |  Dynamic linker path
|  - Entry Point   |  Main entry point
|  - Build Version |  OS version requirements
|  - UUID          |  Unique identifier
|  - Code Sign     |  Code signature location
+------------------+
| Data             |  Actual code and data
|  - __TEXT        |  Executable code (read-only)
|  - __DATA        |  Mutable data
|  - __LINKEDIT    |  Linker metadata
+------------------+
```

## Mach Header (64-bit)

The file begins with a 32-byte header:

```go
type MachHeader64 struct {
    Magic      uint32  // 0xfeedfacf for 64-bit
    CPUType    uint32  // CPU architecture
    CPUSubtype uint32  // CPU variant
    FileType   uint32  // Object, executable, dylib, etc.
    NCmds      uint32  // Number of load commands
    SizeofCmds uint32  // Total size of load commands
    Flags      uint32  // File flags
    Reserved   uint32  // Reserved (0)
}
```

### Magic Numbers

- **0xfeedface** - 32-bit Mach-O
- **0xfeedfacf** - 64-bit Mach-O (ARM64, x86-64)
- **0xcefaedfe** - 32-bit Mach-O (byte-swapped)
- **0xcffaedfe** - 64-bit Mach-O (byte-swapped)

### CPU Types (ARM64)

```go
const (
    CPU_TYPE_ARM64    = 0x0100000c  // ARM64 (64-bit ARM)
    CPU_SUBTYPE_ARM64 = 0x00000000  // All ARM64 variants
    CPU_SUBTYPE_ARM64_V8 = 0x00000001  // ARMv8
    CPU_SUBTYPE_ARM64E   = 0x00000002  // ARM64e (pointer authentication)
)
```

### File Types

```go
const (
    MH_OBJECT  = 0x1  // Relocatable object file (.o)
    MH_EXECUTE = 0x2  // Executable program
    MH_DYLIB   = 0x6  // Dynamic library
    MH_BUNDLE  = 0x8  // Loadable bundle
)
```

### Header Flags

```go
const (
    MH_NOUNDEFS = 0x1       // No undefined references
    MH_DYLDLINK = 0x4       // Uses dynamic linker
    MH_TWOLEVEL = 0x80      // Two-level namespace
    MH_PIE      = 0x200000  // Position independent executable
)
```

**Typical executable flags**: `MH_NOUNDEFS | MH_DYLDLINK | MH_TWOLEVEL | MH_PIE`

## Load Commands

Load commands immediately follow the header. Each command has:

```go
type LoadCommand struct {
    Cmd     uint32  // Command type
    Cmdsize uint32  // Command size including data
    // ... command-specific data
}
```

### Essential Load Commands for Executables

#### 1. LC_SEGMENT_64 (0x19) - Segment Definition

Defines a memory segment to be mapped when loading the file.

```go
type SegmentCommand64 struct {
    Cmd      uint32      // 0x19 (LC_SEGMENT_64)
    Cmdsize  uint32      // 72 + (80 * nsects)
    Segname  [16]byte    // Segment name (null-padded)
    VMAddr   uint64      // Virtual memory address
    VMSize   uint64      // Virtual memory size
    FileOff  uint64      // File offset
    FileSize uint64      // Bytes in file
    MaxProt  uint32      // Maximum protection
    InitProt uint32      // Initial protection
    NSects   uint32      // Number of sections
    Flags    uint32      // Segment flags
}
```

**Memory Protection Flags**:
```go
const (
    VM_PROT_READ    = 0x1  // Readable
    VM_PROT_WRITE   = 0x2  // Writable
    VM_PROT_EXECUTE = 0x4  // Executable
)
```

**Standard Segments**:

1. **__PAGEZERO** - Null pointer protection
   - VMAddr: `0x0`
   - VMSize: `0x100000000` (4GB on ARM64)
   - FileSize: `0`
   - Protection: None (catches null pointer dereferences)

2. **__TEXT** - Executable code (read-only)
   - VMAddr: `0x100000000` (standard base)
   - Protection: `VM_PROT_READ | VM_PROT_EXECUTE`
   - Contains `__text` section with actual code

3. **__DATA** - Mutable data
   - Protection: `VM_PROT_READ | VM_PROT_WRITE`
   - Contains initialized and uninitialized data

4. **__LINKEDIT** - Link-edit information
   - Protection: `VM_PROT_READ`
   - Contains symbol table, string table, code signatures

#### 2. Section Headers (within segments)

Each segment can contain multiple sections:

```go
type Section64 struct {
    Sectname  [16]byte  // Section name
    Segname   [16]byte  // Owning segment name
    Addr      uint64    // Virtual address
    Size      uint64    // Section size
    Offset    uint32    // File offset
    Align     uint32    // Alignment (power of 2)
    Reloff    uint32    // Relocation entries offset
    Nreloc    uint32    // Number of relocations
    Flags     uint32    // Section flags
    Reserved1 uint32    // Reserved
    Reserved2 uint32    // Reserved
    Reserved3 uint32    // Reserved (64-bit only)
}
```

**Section Flags**:
```go
const (
    S_REGULAR                  = 0x0           // Regular section
    S_ZEROFILL                 = 0x1           // Zero-filled on demand
    S_ATTR_PURE_INSTRUCTIONS   = 0x80000000    // Pure machine code
    S_ATTR_SOME_INSTRUCTIONS   = 0x00000400    // Contains instructions
)
```

**Standard Sections**:

- **__TEXT,__text** - Machine code
  - Flags: `S_REGULAR | S_ATTR_PURE_INSTRUCTIONS | S_ATTR_SOME_INSTRUCTIONS` (0x80000400)
  - Align: 2 (4-byte alignment)

- **__DATA,__data** - Initialized data
  - Flags: `S_REGULAR`

- **__DATA,__bss** - Uninitialized data
  - Flags: `S_ZEROFILL`

#### 3. LC_MAIN (0x80000028) - Entry Point

Specifies the main entry point for executables:

```go
type EntryPointCommand struct {
    Cmd       uint32  // 0x80000028
    Cmdsize   uint32  // 24
    EntryOff  uint64  // File offset of entry point
    StackSize uint64  // Initial stack size (0 = default)
}
```

**Key Points**:
- EntryOff is a **file offset**, not a virtual address
- Points to the start of the `__text` section for most programs
- Replaced the older LC_UNIXTHREAD command

#### 4. LC_LOAD_DYLINKER (0xe) - Dynamic Linker

Specifies which dynamic linker to use:

```go
type DylinkerCommand struct {
    Cmd     uint32  // 0xe
    Cmdsize uint32  // Header size + path length (aligned to 8 bytes)
    NameOff uint32  // Offset of name from start of command (usually 12)
}
// Followed by null-terminated path string
```

**Standard path**: `/usr/lib/dyld`

#### 5. LC_LOAD_DYLIB (0xc) - Load Dynamic Library

Specifies a dynamic library dependency:

```go
type DylibCommand struct {
    Cmd                  uint32  // 0xc
    Cmdsize              uint32  // Header size + path length (aligned)
    NameOff              uint32  // Offset of library path (usually 24)
    Timestamp            uint32  // Library build timestamp
    CurrentVersion       uint32  // Library version
    CompatibilityVersion uint32  // Minimum compatible version
}
// Followed by null-terminated path string
```

**libSystem Example**:
- Path: `/usr/lib/libSystem.B.dylib`
- CurrentVersion: `0x04e40e00` (1292.100.0)
- CompatibilityVersion: `0x00010000` (1.0.0)

**Version Encoding**: `(major << 16) | (minor << 8) | patch`

#### 6. LC_UUID (0x1b) - Unique Identifier

Provides a unique identifier for the binary:

```go
type UUIDCommand struct {
    Cmd     uint32    // 0x1b
    Cmdsize uint32    // 24
    UUID    [16]byte  // 128-bit UUID
}
```

**Purpose**:
- Identifies specific builds for debugging
- Used by crash reporting to match binaries with debug symbols
- Should be randomly generated or derived from build content

#### 7. LC_BUILD_VERSION (0x32) - Build Version

Specifies platform and SDK version requirements:

```go
type BuildVersionCommand struct {
    Cmd      uint32  // 0x32
    Cmdsize  uint32  // 24 + (8 * ntools)
    Platform uint32  // Platform type
    Minos    uint32  // Minimum OS version
    Sdk      uint32  // SDK version
    Ntools   uint32  // Number of tool entries
}

type BuildToolVersion struct {
    Tool    uint32  // Tool identifier
    Version uint32  // Tool version
}
```

**Platforms**:
```go
const (
    PLATFORM_MACOS       = 1
    PLATFORM_IOS         = 2
    PLATFORM_TVOS        = 3
    PLATFORM_WATCHOS     = 4
    PLATFORM_BRIDGEOS    = 5
    PLATFORM_MACCATALYST = 6
)
```

**Tools**:
```go
const (
    TOOL_CLANG = 1
    TOOL_SWIFT = 2
    TOOL_LD    = 3  // Linker
)
```

**Version Encoding**: `(major << 16) | (minor << 8) | patch`

**Example**:
- Platform: 1 (macOS)
- Minos: `0x000c0000` (12.0.0)
- SDK: `0x000f0000` (15.0.0)
- Tool: 3 (LD), Version: `0x048f0000` (1167.0)

#### 8. LC_SOURCE_VERSION (0x2a) - Source Version

Records the source version of the code:

```go
type SourceVersionCommand struct {
    Cmd     uint32  // 0x2a
    Cmdsize uint32  // 16
    Version uint64  // A.B.C.D.E packed as a24.b10.c10.d10.e10
}
```

**Encoding**: Five components packed into 64 bits
- A: 24 bits
- B, C, D, E: 10 bits each

**Example**: Version 1.0.0.0.0 = `0x0001000000000000`

#### 9. LC_CODE_SIGNATURE (0x1d) - Code Signature

Points to the code signature data:

```go
type CodeSignatureCommand struct {
    Cmd      uint32  // 0x1d
    Cmdsize  uint32  // 16
    DataOff  uint32  // File offset of signature
    DataSize uint32  // Size of signature data
}
```

**Important**:
- Added by `codesign` tool, not the assembler
- Points to data in the `__LINKEDIT` segment
- Required for modern macOS (especially on Apple Silicon)
- Ad-hoc signatures use: `codesign -s - -f <binary>`

## Code Signing

Modern macOS requires code signatures, especially on Apple Silicon.

### Code Signature Structure

The signature is stored in the `__LINKEDIT` segment and contains:

1. **SuperBlob Header** - Container for all signature data
2. **Code Directory** - Hashes of code pages
3. **Requirements** - Signing requirements
4. **Entitlements** - Security entitlements (optional)
5. **CMS Signature** - Cryptographic signature (or ad-hoc marker)

### Go's Approach (from cmd/internal/codesign)

The Go toolchain includes a minimal code signing implementation for ad-hoc signatures:

```go
// Minimal SuperBlob for ad-hoc signatures
type SuperBlob struct {
    Magic  uint32  // 0xfade0cc0
    Length uint32  // Total size
    Count  uint32  // Number of blob entries
}

type BlobIndex struct {
    Type   uint32  // Blob type (CodeDirectory, Requirements, etc.)
    Offset uint32  // Offset from SuperBlob start
}
```

### Ad-Hoc Signing Process

1. **Reserve space** in `__LINKEDIT` for signature (typically 8KB)
2. **Generate LC_CODE_SIGNATURE** load command pointing to signature location
3. **Compute hashes** of code pages (4KB pages)
4. **Create CodeDirectory** blob with page hashes
5. **Create Requirements** blob (minimal for ad-hoc)
6. **Assemble SuperBlob** containing all blobs
7. **Write signature** to reserved space

### Validation

After signing, validate with:
```bash
codesign -v <binary>           # Verify signature
codesign --verify --strict <binary>  # Strict validation
otool -l <binary> | grep -A 2 LC_CODE_SIGNATURE  # Check signature location
```

## Minimal Executable Example

Here's what a minimal ARM64 macOS executable needs:

### 1. Mach Header
```
Magic:      0xfeedfacf (MH_MAGIC_64)
CPUType:    0x0100000c (ARM64)
CPUSubtype: 0x00000000
FileType:   0x2 (MH_EXECUTE)
NCmds:      6
SizeofCmds: <calculated>
Flags:      0x200085 (PIE | DYLDLINK | TWOLEVEL | NOUNDEFS)
```

### 2. Load Commands

1. **__PAGEZERO segment** (72 bytes)
2. **__TEXT segment** with __text section (152 bytes: 72 + 80)
3. **__LINKEDIT segment** (72 bytes)
4. **LC_LOAD_DYLINKER** (32 bytes: 12 + 20 for path)
5. **LC_LOAD_DYLIB** for libSystem (56 bytes)
6. **LC_MAIN** (24 bytes)
7. **LC_UUID** (24 bytes)
8. **LC_BUILD_VERSION** (32 bytes with 1 tool)
9. **LC_SOURCE_VERSION** (16 bytes)

**Total**: ~480 bytes of load commands

### 3. Code Section

Machine code starting at file offset after load commands (page-aligned).

### 4. __LINKEDIT Section

Contains code signature (if signed).

## Memory Layout

Typical ARM64 executable memory layout:

```
0x000000000 - 0x100000000  __PAGEZERO (unmapped, catches null pointers)
0x100000000 - 0x100004000  __TEXT segment (code, read+exec)
0x100004000 - 0x100008000  __DATA segment (data, read+write)
0x100008000 - 0x10000C000  __LINKEDIT (metadata, read-only)
```

## File Layout Example

```
Offset      Size    Content
0x0000      32      Mach-O header
0x0020      480     Load commands
0x0200      4096    __TEXT,__text (code) [padded to page]
0x1200      8192    __LINKEDIT (signature)
```

## Load Command Order (from Go linker)

Go's linker writes load commands in this order:
1. All segment commands (__PAGEZERO, __TEXT, __DATA, __LINKEDIT)
2. LC_LOAD_DYLINKER
3. LC_LOAD_DYLIB commands (for each library)
4. LC_MAIN
5. LC_UUID
6. LC_BUILD_VERSION
7. LC_SOURCE_VERSION
8. LC_SYMTAB (if present)
9. LC_DYSYMTAB (if present)

## Tools for Inspection

```bash
# Display load commands
otool -l <binary>

# Display Mach header
otool -h <binary>

# Disassemble text section
otool -tV <binary>

# Check code signature
codesign -dv <binary>

# Display all sections
otool -l <binary> | grep -A 5 Section

# Verify signature
codesign --verify --strict <binary>

# Display file structure
size -m -x <binary>
```

## Common Issues

### 1. Code Signature Validation Fails

**Symptom**: `codesign --verify --strict` fails

**Causes**:
- Missing LC_UUID command
- Missing LC_BUILD_VERSION command
- Incorrect __LINKEDIT segment size
- Missing or malformed code signature blob
- File modified after signing

**Solution**: Ensure all required load commands are present and properly formatted

### 2. Binary Won't Execute

**Symptom**: "killed: 9" or security errors

**Causes**:
- No code signature on Apple Silicon
- Invalid entry point in LC_MAIN
- Missing __PAGEZERO segment
- Incorrect segment protections

**Solution**: Sign with `codesign -s - -f <binary>` and verify segment setup

### 3. Dynamic Linker Errors

**Symptom**: "dyld: Library not loaded"

**Causes**:
- Incorrect library path in LC_LOAD_DYLIB
- Missing MH_DYLDLINK flag
- Missing LC_LOAD_DYLINKER command

**Solution**: Ensure correct dylinker path (`/usr/lib/dyld`)

## References

- [Apple Mach-O Programming Topics](https://developer.apple.com/library/archive/documentation/DeveloperTools/Conceptual/MachOTopics/)
- [OS X ABI Mach-O File Format Reference](https://github.com/aidansteele/osx-abi-macho-file-format-reference)
- Go source: `cmd/link/internal/ld/macho.go`
- Go source: `cmd/internal/macho/macho.go`
- Go source: `cmd/internal/codesign/`

## Version History

- macOS 10.15+: Hardened runtime and notarization required
- macOS 11.0+: Apple Silicon support, stricter code signing
- macOS 12.0+: Minimum version for modern ARM64 binaries
