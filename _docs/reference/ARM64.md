# ARM64 (AArch64) Instruction Encoding Reference

This document provides a reference for ARM64 instruction encoding needed for the slasm assembler.

## Sources
- [ARM Developer: A64 Instruction Set Architecture](https://developer.arm.com/documentation/ddi0596/latest/)
- [Structure of the ARM A64 instruction set](https://weinholt.se/articles/arm-a64-instruction-set/)
- [ARM64 Quick Reference (UW)](https://courses.cs.washington.edu/courses/cse469/19wi/arm64.pdf)
- [ARM64 Cheat Sheet (Swarthmore)](https://www.cs.swarthmore.edu/~kwebb/cs31/resources/ARM64_Cheat_Sheet.pdf)

## Overview

All ARM64 instructions are exactly **32 bits (4 bytes)** wide. Instructions are stored in **little-endian** format on macOS.

## Register Encoding

### General Purpose Registers

- **x0-x30**: 64-bit general purpose registers
- **w0-w30**: 32-bit variants (lower 32 bits of x0-x30)
- **sp**: Stack pointer (register 31 in some contexts)
- **xzr/wzr**: Zero register (register 31 in some contexts)
- **lr (x30)**: Link register

Register encoding uses 5 bits (0-31):
- Registers x0-x30 / w0-w30: encoded as 0-30
- Register 31 context-dependent: sp or zr

### Register Field Names

- **Rd**: Destination register
- **Rn**: First source register (or base register for memory ops)
- **Rm**: Second source register
- **Rt**: Transfer register (for load/store)

## Instruction Format Overview

Instructions follow a hierarchical encoding with common patterns:

```
31 30 29 28 27 26 25 24 23 22 21 20 19 18 17 16 15 14 13 12 11 10  9  8  7  6  5  4  3  2  1  0
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
|sf|op|           opcode/immediate/registers...                                              |
+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
```

- **sf** (bit 31): Size flag (0 = 32-bit/w, 1 = 64-bit/x)
- **op** bits: Operation type

## Instruction Encodings for slasm

### MOV (Move - immediate)

MOV is actually MOVZ (move with zero) or ORR (register variant).

**MOVZ Rd, #imm** - Move immediate with zero
```
sf=1 (x) or 0 (w)
Encoding: 1101 0010 1XX0 IIII IIII IIII IIID DDDD
          |sf| 10100|hw|     imm16        | Rd  |
```
- sf: 1 for x registers, 0 for w
- hw: shift amount (0=no shift, 1=16, 2=32, 3=48)
- imm16: 16-bit immediate value
- Rd: destination register (5 bits)

**Example:** `mov x0, #1`
- Binary: `1101 0010 1000 0000 0000 0000 0010 0000`
- Hex: `0xd2800020`

### ADD (Add)

**ADD Rd, Rn, Rm** - Add register to register
```
sf=1, 0001011, shift=00, Rm, 000000, Rn, Rd
Encoding: 1000 1011 000R RRRR 0000 00NN NNND DDDD
```

**ADD Rd, Rn, #imm** - Add immediate
```
sf=1, 0010001, shift, imm12, Rn, Rd
Encoding: 1001 0001 00II IIII IIII IINN NNND DDDD
```

**Example:** `add x2, x0, x1`
- sf=1, Rm=1, Rn=0, Rd=2
- Binary: `1000 1011 0000 0001 0000 0000 0000 0010`
- Hex: `0x8b010002`

### SUB (Subtract)

**SUB Rd, Rn, Rm** - Subtract register from register
```
sf=1, 1001011, shift=00, Rm, 000000, Rn, Rd
Encoding: 1100 1011 000R RRRR 0000 00NN NNND DDDD
```

**Example:** `sub x2, x0, x1`
- Hex: `0xcb010002`

### MUL (Multiply)

**MUL Rd, Rn, Rm** - Multiply
```
Actually MADD with Ra=31 (zero)
sf=1, 0011011, 000, Rm, 0, 31, Rn, Rd
Encoding: 1001 1011 000R RRRR 0111 11NN NNND DDDD
```

**Example:** `mul x2, x0, x1`
- Hex: `0x9b017c02`

### SDIV (Signed Divide)

**SDIV Rd, Rn, Rm** - Signed divide
```
sf=1, 0011010, 110, Rm, 000011, Rn, Rd
Encoding: 1001 1010 110R RRRR 0000 11NN NNND DDDD
```

**Example:** `sdiv x2, x0, x1`
- Hex: `0x9ac10c02`

### CMP (Compare)

**CMP Rn, Rm** - Compare registers
```
Actually SUBS with Rd=31 (discard result)
sf=1, 1101011, shift=00, Rm, 000000, Rn, 11111
Encoding: 1110 1011 000R RRRR 0000 00NN NNN1 1111
```

**Example:** `cmp x0, x1`
- Hex: `0xeb01001f`

### CSET (Conditional Set)

**CSET Rd, cond** - Set register if condition true
```
Actually CSINC with Rn=31, Rm=31
sf=1, 0011010, 100, 31, cond^1, 01, 31, Rd
Encoding: 1001 1010 1001 1111 CCCC 0111 111D DDDD
```

Condition codes:
- EQ (equal): 0000
- NE (not equal): 0001
- LT (less than): 1011
- GT (greater than): 1100
- LE (less or equal): 1101
- GE (greater or equal): 1010

**Example:** `cset x2, eq`
- Hex: `0x9a9f17e2`

### B (Branch)

**B label** - Unconditional branch
```
Encoding: 000101 IIIIIIIIIIIIIIIIIIIIIIIIII
          |  5 |      26-bit offset       |
```

Offset is PC-relative, signed, in units of 4 bytes (word-aligned).
Range: ±128MB

**Example:** `b main` (forward 4 instructions = offset +4)
- Binary: `0001 0100 0000 0000 0000 0000 0000 0100`
- Hex: `0x14000004`

### BL (Branch with Link)

**BL label** - Branch with link (function call)
```
Encoding: 100101 IIIIIIIIIIIIIIIIIIIIIIIIII
          | 37 |      26-bit offset       |
```

**Example:** `bl func`
- Hex: `0x94000000` + offset

### RET (Return)

**RET** - Return from subroutine (defaults to lr/x30)
```
Encoding: 1101 0110 0101 1111 0000 0011 1100 0000
```
- Hex: `0xd65f03c0`

### LDR (Load Register)

**LDR Rt, [Rn, #imm]** - Load from memory with immediate offset
```
sf=1, 11 1001 01, imm12, Rn, Rt
Encoding: 1111 1001 01II IIII IIII IINN NNNT TTTT
```

imm12 is scaled by 8 for 64-bit loads (so actual offset = imm12 * 8).

**Example:** `ldr x0, [sp, #16]`
- imm12 = 16/8 = 2
- Hex: `0xf94003e0`

### STR (Store Register)

**STR Rt, [Rn, #imm]** - Store to memory with immediate offset
```
sf=1, 11 1001 00, imm12, Rn, Rt
Encoding: 1111 1001 00II IIII IIII IINN NNNT TTTT
```

**Example:** `str x0, [sp, #16]`
- Hex: `0xf90003e0`

### LDP (Load Pair)

**LDP Rt1, Rt2, [Rn, #imm]** - Load register pair
```
sf=1, 010 1001, 01, imm7, Rt2, Rn, Rt1
Encoding: 1010 1001 01II IIII ITTT TTNN NNNT TTTT
```

imm7 is signed, scaled by 8.

**Example:** `ldp x29, x30, [sp], #16` (post-index)
- Hex: `0xa8c107fd`

### STP (Store Pair)

**STP Rt1, Rt2, [Rn, #imm]** - Store register pair
```
sf=1, 010 1001, 00, imm7, Rt2, Rn, Rt1
Encoding: 1010 1001 00II IIII ITTT TTNN NNNT TTTT
```

**Example:** `stp x29, x30, [sp, #-16]!` (pre-index)
- Hex: `0xa9bf7bfd`

### ADRP (Address Page)

**ADRP Rd, label@PAGE** - Form PC-relative page address
```
Encoding: 1 IIIIIIII IIII IIII IIII IIII IIID DDDD
          |1|  immlo |    immhi      |1 0000| Rd |
```

PC-relative address to 4KB page (clears bottom 12 bits).
21-bit immediate (2 bits immlo, 19 bits immhi).

**Example:** `adrp x0, buffer@PAGE`
- Requires relocation: ARM64_RELOC_PAGE21

### SVC (Supervisor Call)

**SVC #imm** - Make system call
```
Encoding: 1101 0100 000I IIII IIII IIII IIII 0001
          |11010100|000|  imm16     | 00001|
```

**Example:** `svc #0`
- Binary: `1101 0100 0000 0000 0000 0000 0000 0001`
- Hex: `0xd4000001`

**Example:** `svc #0x80`
- Hex: `0xd4001001`

## Notes for Implementation

1. **All instructions are 32 bits**, stored little-endian
2. **Register encoding**: 5 bits for each register field
3. **Immediates**: Different instructions have different immediate formats
4. **Branches**: Offsets are in 4-byte units (word-aligned)
5. **Memory operations**: Offsets may be scaled based on operation size
6. **sf bit**: Almost always bit 31, determines 32/64-bit operation

## Minimal Instruction Set for slasm

For a simple end-to-end implementation, start with:
- **MOV** (immediate only)
- **ADD** (register and immediate)
- **SUB** (register)
- **B** (branch)
- **RET** (return)
- **SVC** (syscall)

This is enough to assemble basic programs like the simple.s example.
