# AMD64 Assembly Reference

This document provides a comprehensive reference for AMD64 (x86-64) assembly language syntax.

## Table of Contents

- [Registers](#registers)
- [Addressing Modes](#addressing-modes)
- [Instruction Suffixes](#instruction-suffixes)
- [Data Movement Instructions](#data-movement-instructions)
- [Arithmetic & Bitwise Operations](#arithmetic--bitwise-operations)
- [Condition Codes & Branching](#condition-codes--branching)
- [Function Calling Convention](#function-calling-convention)
- [Stack Operations](#stack-operations)
- [Compilation & Linking](#compilation--linking)
- [GDB Debugging](#gdb-debugging)

---

## Registers

AMD64 provides sixteen general-purpose registers plus special-purpose registers, each 64 bits wide. Lower portions are accessible through pseudo-register names for 32-, 16-, and 8-bit operations.

### General Purpose Registers

| 64-bit | 32-bit | 16-bit | 8-bit (low) | Purpose |
|--------|--------|--------|-------------|---------|
| `%rax` | `%eax` | `%ax`  | `%al`       | Accumulator, return value |
| `%rbx` | `%ebx` | `%bx`  | `%bl`       | Base register (callee-saved) |
| `%rcx` | `%ecx` | `%cx`  | `%cl`       | Counter, 4th argument |
| `%rdx` | `%edx` | `%dx`  | `%dl`       | Data register, 3rd argument |
| `%rsi` | `%esi` | `%si`  | `%sil`      | Source index, 2nd argument |
| `%rdi` | `%edi` | `%di`  | `%dil`      | Destination index, 1st argument |
| `%rbp` | `%ebp` | `%bp`  | `%bpl`      | Base pointer (callee-saved) |
| `%rsp` | `%esp` | `%sp`  | `%spl`      | Stack pointer |
| `%r8`  | `%r8d` | `%r8w` | `%r8b`      | 5th argument |
| `%r9`  | `%r9d` | `%r9w` | `%r9b`      | 6th argument |
| `%r10` | `%r10d`| `%r10w`| `%r10b`     | Scratch/temporary |
| `%r11` | `%r11d`| `%r11w`| `%r11b`     | Scratch/temporary |
| `%r12` | `%r12d`| `%r12w`| `%r12b`     | Callee-saved |
| `%r13` | `%r13d`| `%r13w`| `%r13b`     | Callee-saved |
| `%r14` | `%r14d`| `%r14w`| `%r14b`     | Callee-saved |
| `%r15` | `%r15d`| `%r15w`| `%r15b`     | Callee-saved |

### Special Registers

| Register | Purpose |
|----------|---------|
| `%rip`   | Instruction pointer (read-only) |
| `%rflags`| Status/condition code flags |

### Register Categories

**Caller-saved (scratch) registers** - caller must preserve before function calls:
- `%rax`, `%rcx`, `%rdx`, `%rsi`, `%rdi`, `%r8`, `%r9`, `%r10`, `%r11`

**Callee-saved registers** - callee must preserve if used:
- `%rbx`, `%rbp`, `%r12`, `%r13`, `%r14`, `%r15`, `%rsp`

---

## Addressing Modes

AMD64 supports multiple addressing modes for memory access:

| Mode | Syntax | Description |
|------|--------|-------------|
| Immediate | `$42` | Constant value |
| Register | `%rax` | Value in register |
| Direct | `0x604892` | Value at constant address |
| Indirect | `(%rax)` | Value at address in register |
| Displacement | `-24(%rbp)` | Base register plus offset |
| Indexed | `(%rbx,%rcx)` | Base + index |
| Scaled-index | `8(%rsp,%rdi,4)` | Base + displacement + (index × scale) |

**General form:** `displacement(base, index, scale)`
- Result: `base + (index × scale) + displacement`
- Scale must be 1, 2, 4, or 8

---

## Instruction Suffixes

Instructions use suffixes to indicate operand size:

| Suffix | Size | Example |
|--------|------|---------|
| `b` | 1 byte (8-bit) | `movb %al, (%rax)` |
| `w` | 2 bytes (16-bit) | `movw %ax, (%rax)` |
| `l` | 4 bytes (32-bit) | `movl %eax, (%rax)` |
| `q` | 8 bytes (64-bit) | `movq %rax, (%rax)` |

**Important:** A 32-bit instruction zeroes the high-order 32 bits of the destination register.

---

## Data Movement Instructions

### Basic Move

```asm
mov src, dst           # Copy src to dst
movb %al, 0x409892     # Write low-byte of %rax to address
movq 8(%rsp), %rax     # Read 8 bytes from stack into %rax
movl $42, %eax         # Load immediate into register
```

### Load Effective Address (LEA)

Computes an address without dereferencing memory:

```asm
lea 0x20(%rsp), %rdi   # %rdi = %rsp + 0x20
lea (%rdi,%rdx,1), %rax # %rax = %rdi + %rdx
lea (%rax,%rax,4), %rax # %rax = %rax * 5 (multiply trick)
```

### Sign/Zero Extension

```asm
movsbl %al, %edx       # Sign-extend 1-byte to 4-byte
movzbl %al, %edx       # Zero-extend 1-byte to 4-byte
movslq %eax, %rax      # Sign-extend 4-byte to 8-byte
cltq                   # Sign-extend %eax to %rax in-place
cqto                   # Sign-extend %rax to %rdx:%rax
```

---

## Arithmetic & Bitwise Operations

### Arithmetic

```asm
add src, dst           # dst += src
sub src, dst           # dst -= src
imul src, dst          # dst *= src (signed)
imul src               # %rdx:%rax = %rax * src
idiv src               # %rax = %rdx:%rax / src, %rdx = remainder

inc dst                # dst += 1
dec dst                # dst -= 1
neg dst                # dst = -dst
```

### Bitwise

```asm
and src, dst           # dst &= src
or src, dst            # dst |= src
xor src, dst           # dst ^= src
not dst                # dst = ~dst

shl count, dst         # Left shift (logical)
shr count, dst         # Right shift (logical)
sar count, dst         # Right shift (arithmetic, preserves sign)
rol count, dst         # Rotate left
ror count, dst         # Rotate right
```

---

## Condition Codes & Branching

### Condition Code Flags

| Flag | Name | Description |
|------|------|-------------|
| ZF | Zero | Result was zero |
| SF | Sign | Result was negative |
| OF | Overflow | Signed overflow occurred |
| CF | Carry | Unsigned overflow occurred |

### Comparison Instructions

```asm
cmp op2, op1           # Sets flags based on op1 - op2
test op2, op1          # Sets flags based on op1 & op2
```

### Conditional Jumps

| Instruction | Condition | Description |
|-------------|-----------|-------------|
| `jmp label` | Always | Unconditional jump |
| `je label`  | ZF=1 | Jump if equal |
| `jne label` | ZF=0 | Jump if not equal |
| `jg label`  | Signed > | Jump if greater |
| `jge label` | Signed >= | Jump if greater or equal |
| `jl label`  | Signed < | Jump if less |
| `jle label` | Signed <= | Jump if less or equal |
| `ja label`  | Unsigned > | Jump if above |
| `jae label` | Unsigned >= | Jump if above or equal |
| `jb label`  | Unsigned < | Jump if below |
| `jbe label` | Unsigned <= | Jump if below or equal |
| `js label`  | SF=1 | Jump if sign (negative) |
| `jns label` | SF=0 | Jump if not sign |
| `jo label`  | OF=1 | Jump if overflow |
| `jno label` | OF=0 | Jump if not overflow |
| `jz label`  | ZF=1 | Jump if zero (same as je) |
| `jnz label` | ZF=0 | Jump if not zero (same as jne) |

### Conditional Set

```asm
sete dst               # Set byte to 1 if equal, 0 otherwise
setne dst              # Set byte if not equal
setg dst               # Set byte if greater (signed)
setl dst               # Set byte if less (signed)
setge dst              # Set byte if greater or equal
setle dst              # Set byte if less or equal
seta dst               # Set byte if above (unsigned)
setb dst               # Set byte if below (unsigned)
```

### Conditional Move

```asm
cmove src, dst         # Move if equal
cmovne src, dst        # Move if not equal
cmovg src, dst         # Move if greater (signed)
cmovl src, dst         # Move if less (signed)
cmovns src, dst        # Move if not sign
```

---

## Function Calling Convention

### System V AMD64 ABI (Linux/macOS)

**Arguments (in order):**
1. `%rdi` - 1st argument
2. `%rsi` - 2nd argument
3. `%rdx` - 3rd argument
4. `%rcx` - 4th argument
5. `%r8`  - 5th argument
6. `%r9`  - 6th argument
7. Stack - additional arguments (pushed right-to-left)

**Return values:**
- `%rax` - Primary return value
- `%rdx` - Secondary return value (for 128-bit returns)

**Stack alignment:** Must be 16-byte aligned before `call` instruction.

### Calling a Function

```asm
# Call function with arguments (3, 7)
mov $3, %rdi           # First argument
mov $7, %rsi           # Second argument
call function_name     # Push return address, jump to function
# Return value is now in %rax
```

### Writing a Function

```asm
my_function:
    push %rbp              # Save old base pointer
    mov %rsp, %rbp         # Set up new base pointer
    push %rbx              # Save callee-saved registers if used
    sub $16, %rsp          # Allocate local variables

    # ... function body ...
    # Arguments in %rdi, %rsi, %rdx, %rcx, %r8, %r9

    add $16, %rsp          # Deallocate locals
    pop %rbx               # Restore callee-saved registers
    pop %rbp               # Restore old base pointer
    ret                    # Return (pops return address into %rip)
```

---

## Stack Operations

The stack grows **downward** toward lower addresses.

```asm
push %rax              # Decrement %rsp by 8, store %rax at (%rsp)
pop %rax               # Load (%rsp) into %rax, increment %rsp by 8

sub $32, %rsp          # Allocate 32 bytes on stack
add $32, %rsp          # Deallocate 32 bytes

enter $32, $0          # Equivalent to: push %rbp; mov %rsp,%rbp; sub $32,%rsp
leave                  # Equivalent to: mov %rbp,%rsp; pop %rbp

call label             # Push return address, jump to label
ret                    # Pop return address into %rip
```

---

## Compilation & Linking

### Using GCC

```bash
# Standard compilation (with C library)
gcc -o program program.s

# Without position-independent executable
gcc -no-pie -o program program.s

# Without C library
gcc -nostdlib -no-pie -o program program.s
```

### Using Assembler and Linker Directly

```bash
# Assemble
as -o program.o program.s

# Link (Linux)
ld -o program program.o

# Link (macOS)
ld -o program program.o -lSystem -syslibroot $(xcrun --show-sdk-path) -e _start
```

### Assembly File Structure

```asm
.data                      # Data section
message:
    .asciz "Hello\n"       # Null-terminated string

.bss                       # Uninitialized data
buffer:
    .space 1024            # Reserve 1024 bytes

.text                      # Code section
.global _start             # Export entry point

_start:
    # ... code ...
```

---

## GDB Debugging

### Compilation for Debugging

```bash
gcc -g -o program program.s    # Include debug symbols
```

### Common Commands

```gdb
break main              # Set breakpoint at main
break *0x08048375       # Set breakpoint at address
break *main+7           # Set breakpoint at offset

run                     # Start program
continue                # Continue execution
stepi                   # Step one instruction
nexti                   # Next instruction (skip over calls)

p $rax                  # Print register value
p/x $rax                # Print in hexadecimal
p/d $rax                # Print in decimal
info reg                # Show all registers

x/8i main               # Disassemble 8 instructions at main
x/4xg $rsp              # Examine 4 quad-words at stack pointer
disassemble main        # Disassemble function

quit                    # Exit GDB
```

---

## Sources

This documentation was compiled from:

- [CS107 Guide to x86-64 (Stanford)](https://web.stanford.edu/class/cs107/guide/x86-64.html)
- [AMD64 Assembly Quick Reference](https://homework.quest/classes/2025-01/cs4310/asm/cheatsheet/)
- [AMD64 Assembly Tutorial](https://www.vikaskumar.org/amd64/)
- [x86 and amd64 instruction reference](https://www.felixcloutier.com/x86/)

For the official AMD documentation, see:
- [AMD64 Architecture Programmer's Manual](https://www.amd.com/system/files/TechDocs/24594.pdf)
