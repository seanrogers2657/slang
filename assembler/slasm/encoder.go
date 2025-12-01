package slasm

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Encoder encodes ARM64 instructions to machine code
type Encoder struct {
	symbolTable *SymbolTable
}

// NewEncoder creates a new instruction encoder
func NewEncoder(symbolTable *SymbolTable) *Encoder {
	return &Encoder{
		symbolTable: symbolTable,
	}
}

// Encode encodes an instruction to machine code (4 bytes for ARM64)
func (e *Encoder) Encode(inst *Instruction, address uint64) ([]byte, error) {
	// TODO: Implement instruction encoding
	// This is the core of the assembler - converts mnemonics to machine code

	switch inst.Mnemonic {
	case "mov":
		return e.encodeMov(inst)
	case "add":
		return e.encodeAdd(inst)
	case "sub":
		return e.encodeSub(inst)
	case "mul":
		return e.encodeMul(inst)
	case "sdiv":
		return e.encodeSdiv(inst)
	case "udiv":
		return e.encodeUdiv(inst)
	case "msub":
		return e.encodeMsub(inst)
	case "neg":
		return e.encodeNeg(inst)
	case "cmp":
		return e.encodeCmp(inst)
	case "cset":
		return e.encodeCset(inst)
	case "b":
		return e.encodeBranch(inst, address)
	case "bl":
		return e.encodeBranchLink(inst, address)
	case "br":
		return e.encodeBranchRegister(inst)
	case "b.eq", "b.ne", "b.cs", "b.hs", "b.cc", "b.lo", "b.mi", "b.pl",
		"b.vs", "b.vc", "b.hi", "b.ls", "b.ge", "b.lt", "b.gt", "b.le", "b.al":
		return e.encodeBranchConditional(inst, address)
	case "ret":
		return e.encodeRet(inst)
	case "ldr":
		return e.encodeLdr(inst)
	case "str":
		return e.encodeStr(inst)
	case "ldp":
		return e.encodeLdp(inst)
	case "stp":
		return e.encodeStp(inst)
	case "ldrb":
		return e.encodeLdrb(inst)
	case "strb":
		return e.encodeStrb(inst)
	case "adr":
		return e.encodeAdr(inst, address)
	case "adrp":
		return e.encodeAdrp(inst, address)
	case "svc":
		return e.encodeSvc(inst)
	default:
		return nil, fmt.Errorf("unsupported instruction: %s", inst.Mnemonic)
	}
}

// Placeholder encoding functions - these will be implemented with actual ARM64 encoding logic

func (e *Encoder) encodeMov(inst *Instruction) ([]byte, error) {
	// MOV is typically encoded as MOVZ (Move with Zero)
	// MOVZ Xd, #imm16, LSL #shift
	// Encoding: sf 10 100101 hw imm16 Rd
	// sf=1 for X registers, hw=shift/16 (00=0, 01=16, 10=32, 11=48)

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: mov requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: mov destination: %w", inst.Line, inst.Column, err)
	}

	if inst.Operands[1].Type == OperandImmediate {
		immVal, err := ParseInt(inst.Operands[1].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: mov immediate: %w", inst.Line, inst.Column, err)
		}
		imm := uint32(immVal)

		// Check if immediate fits in 16 bits
		if imm > 0xFFFF {
			return nil, fmt.Errorf("line %d:%d: immediate %d too large for MOV (max 65535)",
				inst.Line, inst.Column, imm)
		}

		sf := uint32(1)  // X registers (64-bit)
		hw := uint32(0)  // No shift
		encoding := (sf << 31) | (0b10100101 << 23) | (hw << 21) | (imm << 5) | uint32(rd)

		return EncodeLittleEndian(encoding), nil
	}

	// MOV Xd, Xm (register to register - alias for ORR Xd, XZR, Xm)
	rm, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: mov source: %w", inst.Line, inst.Column, err)
	}
	sf := uint32(1)
	// ORR (shifted register): sf 01 01010 shift 0 Rm imm6 Rn Rd
	// With Rn=XZR(31), shift=00, imm6=0
	encoding := (sf << 31) | (0b0101010 << 24) | (uint32(rm) << 16) | (uint32(31) << 5) | uint32(rd)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeAdd(inst *Instruction) ([]byte, error) {
    // ADD Xd, Xn, #imm12
    // Encoding: sf 0 0 10001 shift imm12 Rn Rd
    // sf=1 for X regs, shift=00, imm12=12-bit immediate

    if len(inst.Operands) != 3 {
        return nil, fmt.Errorf("line %d:%d: add requires 3 operands, got %d",
            inst.Line, inst.Column, len(inst.Operands))
    }

    rd, err := ParseRegister(inst.Operands[0].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: add destination: %w", inst.Line, inst.Column, err)
    }

    rn, err := ParseRegister(inst.Operands[1].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: add operand 1: %w", inst.Line, inst.Column, err)
    }

    if inst.Operands[2].Type == OperandImmediate {
        immVal, err := ParseInt(inst.Operands[2].Value)
        if err != nil {
            return nil, fmt.Errorf("line %d:%d: add immediate: %w", inst.Line, inst.Column, err)
        }
        imm := uint32(immVal)

        // Check if immediate fits in 12 bits
        if imm > 0xFFF {
            return nil, fmt.Errorf("line %d:%d: immediate %d too large for ADD (max 4095)",
                inst.Line, inst.Column, imm)
        }

        sf := uint32(1) // X registers (64-bit)
        encoding := (sf << 31) | (0b0010001 << 24) | (imm << 10) | (uint32(rn) << 5) | uint32(rd)

        return EncodeLittleEndian(encoding), nil
    }

    // ADD Xd, Xn, Xm (register form)
    rm, err := ParseRegister(inst.Operands[2].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: add operand 2: %w", inst.Line, inst.Column, err)
    }
    sf := uint32(1)
    encoding := (sf << 31) | (0b0001011 << 24) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)

    return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeSub(inst *Instruction) ([]byte, error) {
    // SUB Xd, Xn, #imm12 or SUB Xd, Xn, Xm
    // Similar to ADD but opc = 10 instead of 00

    if len(inst.Operands) != 3 {
        return nil, fmt.Errorf("line %d:%d: sub requires 3 operands, got %d",
            inst.Line, inst.Column, len(inst.Operands))
    }

    rd, err := ParseRegister(inst.Operands[0].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: sub destination: %w", inst.Line, inst.Column, err)
    }

    rn, err := ParseRegister(inst.Operands[1].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: sub operand 1: %w", inst.Line, inst.Column, err)
    }

    if inst.Operands[2].Type == OperandImmediate {
        immVal, err := ParseInt(inst.Operands[2].Value)
        if err != nil {
            return nil, fmt.Errorf("line %d:%d: sub immediate: %w", inst.Line, inst.Column, err)
        }
        imm := uint32(immVal)
        if imm > 0xFFF {
            return nil, fmt.Errorf("line %d:%d: immediate %d too large for SUB (max 4095)",
                inst.Line, inst.Column, imm)
        }

        // SUB (immediate): sf 1 0 1 0 0 0 1 sh imm12 Rn Rd
        // sf=1 for 64-bit, op=1 (SUB), S=0 (no flags), 10001, shift=00
        // Fixed bits for SUB without flags: 0xD1000000
        encoding := uint32(0xD1000000) | (imm << 10) | (uint32(rn) << 5) | uint32(rd)
        return EncodeLittleEndian(encoding), nil
    }

    // Register form
    rm, err := ParseRegister(inst.Operands[2].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: sub operand 2: %w", inst.Line, inst.Column, err)
    }
    sf := uint32(1)
    encoding := (sf << 31) | (0b1001011 << 24) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)

    return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeMul(inst *Instruction) ([]byte, error) {
    // MADD Xd, Xn, Xm, XZR (MUL is an alias)
    // Encoding: sf 0 011011 000 Rm 0 Ra Rn Rd

    if len(inst.Operands) != 3 {
        return nil, fmt.Errorf("line %d:%d: mul requires 3 operands, got %d",
            inst.Line, inst.Column, len(inst.Operands))
    }

    rd, err := ParseRegister(inst.Operands[0].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: mul destination: %w", inst.Line, inst.Column, err)
    }

    rn, err := ParseRegister(inst.Operands[1].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: mul operand 1: %w", inst.Line, inst.Column, err)
    }

    rm, err := ParseRegister(inst.Operands[2].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: mul operand 2: %w", inst.Line, inst.Column, err)
    }

    sf := uint32(1)
    ra := uint32(31) // XZR for MUL

    encoding := (sf << 31) | (0b0011011 << 24) | (uint32(rm) << 16) | (ra << 10) | (uint32(rn) << 5) | uint32(rd)

    return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeSdiv(inst *Instruction) ([]byte, error) {
    // SDIV Xd, Xn, Xm
    // Encoding: sf 0 011010110 Rm 000011 Rn Rd

    if len(inst.Operands) != 3 {
        return nil, fmt.Errorf("line %d:%d: sdiv requires 3 operands, got %d",
            inst.Line, inst.Column, len(inst.Operands))
    }

    rd, err := ParseRegister(inst.Operands[0].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: sdiv destination: %w", inst.Line, inst.Column, err)
    }

    rn, err := ParseRegister(inst.Operands[1].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: sdiv operand 1: %w", inst.Line, inst.Column, err)
    }

    rm, err := ParseRegister(inst.Operands[2].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: sdiv operand 2: %w", inst.Line, inst.Column, err)
    }

    sf := uint32(1)
    encoding := (sf << 31) | (0b0011010110 << 21) | (uint32(rm) << 16) | (0b000011 << 10) | (uint32(rn) << 5) | uint32(rd)

    return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeUdiv(inst *Instruction) ([]byte, error) {
	// TODO: Implement udiv encoding
	return make([]byte, 4), nil
}

func (e *Encoder) encodeMsub(inst *Instruction) ([]byte, error) {
    // MSUB Xd, Xn, Xm, Xa
    // Xd = Xa - (Xn * Xm)
    // Encoding: sf 0 011011 000 Rm 1 Ra Rn Rd

    if len(inst.Operands) != 4 {
        return nil, fmt.Errorf("line %d:%d: msub requires 4 operands, got %d",
            inst.Line, inst.Column, len(inst.Operands))
    }

    rd, err := ParseRegister(inst.Operands[0].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: msub destination: %w", inst.Line, inst.Column, err)
    }

    rn, err := ParseRegister(inst.Operands[1].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: msub operand 1: %w", inst.Line, inst.Column, err)
    }

    rm, err := ParseRegister(inst.Operands[2].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: msub operand 2: %w", inst.Line, inst.Column, err)
    }

    ra, err := ParseRegister(inst.Operands[3].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: msub operand 3: %w", inst.Line, inst.Column, err)
    }

    sf := uint32(1)
    encoding := (sf << 31) | (0b0011011 << 24) | (uint32(rm) << 16) | (1 << 15) | (uint32(ra) << 10) | (uint32(rn) << 5) | uint32(rd)

    return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeNeg(inst *Instruction) ([]byte, error) {
	// TODO: Implement neg encoding (negate)
	return make([]byte, 4), nil
}

func (e *Encoder) encodeCmp(inst *Instruction) ([]byte, error) {
    // CMP Xn, #imm or CMP Xn, Xm
    // This is SUBS XZR, Xn, operand

    if len(inst.Operands) != 2 {
        return nil, fmt.Errorf("line %d:%d: cmp requires 2 operands, got %d",
            inst.Line, inst.Column, len(inst.Operands))
    }

    rn, err := ParseRegister(inst.Operands[0].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: cmp operand 1: %w", inst.Line, inst.Column, err)
    }
    rd := uint32(31) // XZR

    if inst.Operands[1].Type == OperandImmediate {
        immVal, err := ParseInt(inst.Operands[1].Value)
        if err != nil {
            return nil, fmt.Errorf("line %d:%d: cmp immediate: %w", inst.Line, inst.Column, err)
        }
        imm := uint32(immVal)
        if imm > 0xFFF {
            return nil, fmt.Errorf("line %d:%d: immediate %d too large for CMP (max 4095)",
                inst.Line, inst.Column, imm)
        }

        sf := uint32(1)
        // SUBS (immediate): sf 1 1 10001 shift imm12 Rn Rd
        encoding := (sf << 31) | (0b1110001 << 24) | (imm << 10) | (uint32(rn) << 5) | rd
        return EncodeLittleEndian(encoding), nil
    }

    // Register form
    rm, err := ParseRegister(inst.Operands[1].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: cmp operand 2: %w", inst.Line, inst.Column, err)
    }
    sf := uint32(1)
    // SUBS (register): sf 1 1 01011 shift 0 Rm imm6 Rn Rd
    encoding := (sf << 31) | (0b1101011 << 24) | (uint32(rm) << 16) | (uint32(rn) << 5) | rd

    return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeCset(inst *Instruction) ([]byte, error) {
    // CSET Xd, condition
    // This is CSINC Xd, XZR, XZR, invert(condition)
    // Encoding: sf 0 0 11010100 Rm cond 01 Rn Rd

    if len(inst.Operands) != 2 {
        return nil, fmt.Errorf("line %d:%d: cset requires 2 operands, got %d",
            inst.Line, inst.Column, len(inst.Operands))
    }

    rd, err := ParseRegister(inst.Operands[0].Value)
    if err != nil {
        return nil, fmt.Errorf("line %d:%d: cset destination: %w", inst.Line, inst.Column, err)
    }

    // Map condition codes
    condMap := map[string]uint32{
        "eq": 0b0000, "ne": 0b0001,
        "lt": 0b1011, "le": 0b1101,
        "gt": 0b1100, "ge": 0b1010,
    }

    cond, ok := condMap[inst.Operands[1].Value]
    if !ok {
        return nil, fmt.Errorf("line %d:%d: unknown condition '%s' (valid: eq, ne, lt, le, gt, ge)",
            inst.Line, inst.Column, inst.Operands[1].Value)
    }

    // Invert condition for CSINC encoding
    invertedCond := cond ^ 1

    sf := uint32(1)
    rm := uint32(31) // XZR
    rn := uint32(31) // XZR

    encoding := (sf << 31) | (0b0011010100 << 21) | (rm << 16) | (invertedCond << 12) | (0b01 << 10) | (rn << 5) | uint32(rd)

    return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeBranch(inst *Instruction, address uint64) ([]byte, error) {
	// B label (unconditional branch)
	// Encoding: 0 00101 imm26
	// imm26 is a signed offset in instructions (not bytes) from PC

	if len(inst.Operands) != 1 {
		return nil, fmt.Errorf("line %d:%d: b requires 1 operand, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	if inst.Operands[0].Type != OperandLabel {
		return nil, fmt.Errorf("line %d:%d: b requires a label operand",
			inst.Line, inst.Column)
	}

	labelName := inst.Operands[0].Value
	symbol, found := e.symbolTable.Lookup(labelName)
	if !found {
		return nil, fmt.Errorf("line %d:%d: undefined label '%s'",
			inst.Line, inst.Column, labelName)
	}

	// Calculate offset in instructions (each instruction is 4 bytes)
	// offset = (target - current) / 4
	targetAddr := symbol.Address
	offset := (int64(targetAddr) - int64(address)) / 4

	// Check if offset fits in 26 bits (signed)
	if offset < -0x2000000 || offset > 0x1FFFFFF {
		return nil, fmt.Errorf("line %d:%d: branch target '%s' is too far away (offset %d)",
			inst.Line, inst.Column, labelName, offset)
	}

	// Encode: 000101 followed by 26-bit signed offset
	imm26 := uint32(offset) & 0x03FFFFFF
	encoding := (0b000101 << 26) | imm26

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeBranchLink(inst *Instruction, address uint64) ([]byte, error) {
	// BL label (branch with link)
	// Encoding: 1 00101 imm26
	// imm26 is a signed offset in instructions (not bytes) from PC

	if len(inst.Operands) != 1 {
		return nil, fmt.Errorf("line %d:%d: bl requires 1 operand, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	if inst.Operands[0].Type != OperandLabel {
		return nil, fmt.Errorf("line %d:%d: bl requires a label operand",
			inst.Line, inst.Column)
	}

	labelName := inst.Operands[0].Value
	symbol, found := e.symbolTable.Lookup(labelName)
	if !found {
		return nil, fmt.Errorf("line %d:%d: undefined label '%s'",
			inst.Line, inst.Column, labelName)
	}

	// Calculate offset in instructions (each instruction is 4 bytes)
	targetAddr := symbol.Address
	offset := (int64(targetAddr) - int64(address)) / 4

	// Check if offset fits in 26 bits (signed)
	if offset < -0x2000000 || offset > 0x1FFFFFF {
		return nil, fmt.Errorf("line %d:%d: branch target '%s' is too far away (offset %d)",
			inst.Line, inst.Column, labelName, offset)
	}

	// Encode: 100101 followed by 26-bit signed offset
	imm26 := uint32(offset) & 0x03FFFFFF
	encoding := (0b100101 << 26) | imm26

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeBranchRegister(inst *Instruction) ([]byte, error) {
	// BR Xn (branch to register)
	// Encoding: 1101011 0000 11111 000000 Rn 00000
	// 0xD61F0000 | (Rn << 5)

	if len(inst.Operands) != 1 {
		return nil, fmt.Errorf("line %d:%d: br requires 1 operand, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: br requires a register operand",
			inst.Line, inst.Column)
	}

	rn, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: br register: %w", inst.Line, inst.Column, err)
	}

	// BR Xn encoding: 1101011 0000 11111 000000 Rn 00000
	encoding := uint32(0xD61F0000) | (uint32(rn) << 5)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeBranchConditional(inst *Instruction, address uint64) ([]byte, error) {
	// B.cond label (conditional branch)
	// Encoding: 0101010 0 imm19 0 cond
	// imm19 is a signed offset in instructions (not bytes) from PC

	if len(inst.Operands) != 1 {
		return nil, fmt.Errorf("line %d:%d: %s requires 1 operand, got %d",
			inst.Line, inst.Column, inst.Mnemonic, len(inst.Operands))
	}

	if inst.Operands[0].Type != OperandLabel {
		return nil, fmt.Errorf("line %d:%d: %s requires a label operand",
			inst.Line, inst.Column, inst.Mnemonic)
	}

	// Extract condition code from mnemonic (e.g., "b.eq" -> "eq")
	cond := inst.Mnemonic[2:] // skip "b."

	// Map condition codes to their 4-bit encoding
	condMap := map[string]uint32{
		"eq": 0b0000, // Equal
		"ne": 0b0001, // Not equal
		"cs": 0b0010, // Carry set / unsigned higher or same
		"hs": 0b0010, // (alias for cs)
		"cc": 0b0011, // Carry clear / unsigned lower
		"lo": 0b0011, // (alias for cc)
		"mi": 0b0100, // Minus / negative
		"pl": 0b0101, // Plus / positive or zero
		"vs": 0b0110, // Overflow
		"vc": 0b0111, // No overflow
		"hi": 0b1000, // Unsigned higher
		"ls": 0b1001, // Unsigned lower or same
		"ge": 0b1010, // Signed greater than or equal
		"lt": 0b1011, // Signed less than
		"gt": 0b1100, // Signed greater than
		"le": 0b1101, // Signed less than or equal
		"al": 0b1110, // Always (unconditional)
	}

	condCode, ok := condMap[cond]
	if !ok {
		return nil, fmt.Errorf("line %d:%d: unknown condition code '%s'",
			inst.Line, inst.Column, cond)
	}

	labelName := inst.Operands[0].Value
	symbol, found := e.symbolTable.Lookup(labelName)
	if !found {
		return nil, fmt.Errorf("line %d:%d: undefined label '%s'",
			inst.Line, inst.Column, labelName)
	}

	// Calculate offset in instructions (each instruction is 4 bytes)
	targetAddr := symbol.Address
	offset := (int64(targetAddr) - int64(address)) / 4

	// Check if offset fits in 19 bits (signed)
	if offset < -0x40000 || offset > 0x3FFFF {
		return nil, fmt.Errorf("line %d:%d: branch target '%s' is too far away (offset %d)",
			inst.Line, inst.Column, labelName, offset)
	}

	// Encode: 01010100 imm19 0 cond
	imm19 := uint32(offset) & 0x7FFFF
	encoding := (0b01010100 << 24) | (imm19 << 5) | condCode

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeRet(inst *Instruction) ([]byte, error) {
	// RET is an alias for BR X30
	// Encoding: 1101011 00101 11111 00000 011110 00000
	return EncodeLittleEndian(0xd65f03c0), nil
}

func (e *Encoder) encodeLdr(inst *Instruction) ([]byte, error) {
	// LDR Xt, [Xn, #imm] - Load register (unsigned offset)
	// Encoding: 11 111 00100 01 imm12 Rn Rt
	// imm12 is scaled by 8 for 64-bit registers

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: ldr requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	// First operand: destination register
	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: ldr destination must be a register",
			inst.Line, inst.Column)
	}
	rt, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ldr destination: %w", inst.Line, inst.Column, err)
	}

	// Second operand: memory operand [base, #offset]
	if inst.Operands[1].Type != OperandMemory {
		return nil, fmt.Errorf("line %d:%d: ldr source must be a memory operand",
			inst.Line, inst.Column)
	}

	rn, err := ParseRegister(inst.Operands[1].Base)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ldr base register: %w", inst.Line, inst.Column, err)
	}

	// Parse offset (default to 0 if not specified)
	offset := int64(0)
	if inst.Operands[1].Offset != "" {
		offset, err = ParseInt64(inst.Operands[1].Offset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: ldr offset: %w", inst.Line, inst.Column, err)
		}
	}

	// For 64-bit LDR, offset must be multiple of 8 and within range
	if offset < 0 || offset > 32760 || offset%8 != 0 {
		return nil, fmt.Errorf("line %d:%d: ldr offset must be 0-32760 and multiple of 8, got %d",
			inst.Line, inst.Column, offset)
	}

	// Scale offset for encoding (divide by 8)
	imm12 := uint32(offset / 8)

	// LDR (unsigned offset): 11 111 00100 01 imm12 Rn Rt
	// Bits 31-30: 11 (size for 64-bit)
	// Bits 29-27: 111
	// Bits 26-24: 001
	// Bits 23-22: 00 (V=0 for GPR, not SIMD)
	// Bits 21-10: 01 imm12
	// Bits 9-5: Rn
	// Bits 4-0: Rt
	encoding := uint32(0xF9400000) | (imm12 << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeStr(inst *Instruction) ([]byte, error) {
	// STR Xt, [Xn, #imm] - Store register (unsigned offset)
	// Encoding: 11 111 00100 00 imm12 Rn Rt
	// imm12 is scaled by 8 for 64-bit registers

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: str requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	// First operand: source register
	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: str source must be a register",
			inst.Line, inst.Column)
	}
	rt, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: str source: %w", inst.Line, inst.Column, err)
	}

	// Second operand: memory operand [base, #offset]
	if inst.Operands[1].Type != OperandMemory {
		return nil, fmt.Errorf("line %d:%d: str destination must be a memory operand",
			inst.Line, inst.Column)
	}

	rn, err := ParseRegister(inst.Operands[1].Base)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: str base register: %w", inst.Line, inst.Column, err)
	}

	// Parse offset (default to 0 if not specified)
	offset := int64(0)
	if inst.Operands[1].Offset != "" {
		offset, err = ParseInt64(inst.Operands[1].Offset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: str offset: %w", inst.Line, inst.Column, err)
		}
	}

	// For 64-bit STR, offset must be multiple of 8 and within range
	if offset < 0 || offset > 32760 || offset%8 != 0 {
		return nil, fmt.Errorf("line %d:%d: str offset must be 0-32760 and multiple of 8, got %d",
			inst.Line, inst.Column, offset)
	}

	// Scale offset for encoding (divide by 8)
	imm12 := uint32(offset / 8)

	// STR (unsigned offset): 11 111 00100 00 imm12 Rn Rt
	// = 0xF9000000 | (imm12 << 10) | (Rn << 5) | Rt
	encoding := uint32(0xF9000000) | (imm12 << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeLdp(inst *Instruction) ([]byte, error) {
	// LDP Xt1, Xt2, [Xn, #imm] - Load pair (signed offset)
	// Encoding: 10 101 0010 1 imm7 Rt2 Rn Rt
	// imm7 is signed and scaled by 8 for 64-bit registers

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: ldp requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	// First operand: first destination register
	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: ldp first operand must be a register",
			inst.Line, inst.Column)
	}
	rt, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ldp first register: %w", inst.Line, inst.Column, err)
	}

	// Second operand: second destination register
	if inst.Operands[1].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: ldp second operand must be a register",
			inst.Line, inst.Column)
	}
	rt2, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ldp second register: %w", inst.Line, inst.Column, err)
	}

	// Third operand: memory operand [base, #offset]
	if inst.Operands[2].Type != OperandMemory {
		return nil, fmt.Errorf("line %d:%d: ldp third operand must be a memory operand",
			inst.Line, inst.Column)
	}

	rn, err := ParseRegister(inst.Operands[2].Base)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ldp base register: %w", inst.Line, inst.Column, err)
	}

	// Parse offset (default to 0 if not specified)
	offset := int64(0)
	if inst.Operands[2].Offset != "" {
		offset, err = ParseInt64(inst.Operands[2].Offset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: ldp offset: %w", inst.Line, inst.Column, err)
		}
	}

	// For 64-bit LDP, offset must be multiple of 8 and within signed 7-bit range * 8
	if offset < -512 || offset > 504 || offset%8 != 0 {
		return nil, fmt.Errorf("line %d:%d: ldp offset must be -512 to 504 and multiple of 8, got %d",
			inst.Line, inst.Column, offset)
	}

	// Scale offset for encoding (divide by 8, signed 7-bit)
	imm7 := uint32(offset/8) & 0x7F

	// LDP (signed offset): 10 101 0010 1 imm7 Rt2 Rn Rt
	// opc=10 (64-bit), 1010010 (fixed), L=1 (load)
	// = 0xA9400000 | (imm7 << 15) | (Rt2 << 10) | (Rn << 5) | Rt
	encoding := uint32(0xA9400000) | (imm7 << 15) | (uint32(rt2) << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeStp(inst *Instruction) ([]byte, error) {
	// STP Xt1, Xt2, [Xn, #imm] - Store pair (signed offset)
	// Encoding: 10 101 0010 0 imm7 Rt2 Rn Rt
	// imm7 is signed and scaled by 8 for 64-bit registers

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: stp requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	// First operand: first source register
	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: stp first operand must be a register",
			inst.Line, inst.Column)
	}
	rt, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: stp first register: %w", inst.Line, inst.Column, err)
	}

	// Second operand: second source register
	if inst.Operands[1].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: stp second operand must be a register",
			inst.Line, inst.Column)
	}
	rt2, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: stp second register: %w", inst.Line, inst.Column, err)
	}

	// Third operand: memory operand [base, #offset]
	if inst.Operands[2].Type != OperandMemory {
		return nil, fmt.Errorf("line %d:%d: stp third operand must be a memory operand",
			inst.Line, inst.Column)
	}

	rn, err := ParseRegister(inst.Operands[2].Base)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: stp base register: %w", inst.Line, inst.Column, err)
	}

	// Parse offset (default to 0 if not specified)
	offset := int64(0)
	if inst.Operands[2].Offset != "" {
		offset, err = ParseInt64(inst.Operands[2].Offset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: stp offset: %w", inst.Line, inst.Column, err)
		}
	}

	// For 64-bit STP, offset must be multiple of 8 and within signed 7-bit range * 8
	if offset < -512 || offset > 504 || offset%8 != 0 {
		return nil, fmt.Errorf("line %d:%d: stp offset must be -512 to 504 and multiple of 8, got %d",
			inst.Line, inst.Column, offset)
	}

	// Scale offset for encoding (divide by 8, signed 7-bit)
	imm7 := uint32(offset/8) & 0x7F

	// STP (signed offset): 10 101 0010 0 imm7 Rt2 Rn Rt
	// opc=10 (64-bit), 1010010 (fixed), L=0 (store)
	// = 0xA9000000 | (imm7 << 15) | (Rt2 << 10) | (Rn << 5) | Rt
	encoding := uint32(0xA9000000) | (imm7 << 15) | (uint32(rt2) << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeLdrb(inst *Instruction) ([]byte, error) {
	// TODO: Implement ldrb encoding (load byte)
	return make([]byte, 4), nil
}

func (e *Encoder) encodeStrb(inst *Instruction) ([]byte, error) {
	// TODO: Implement strb encoding (store byte)
	return make([]byte, 4), nil
}

func (e *Encoder) encodeAdr(inst *Instruction, address uint64) ([]byte, error) {
	// TODO: Implement adr encoding (address to register, PC-relative)
	return make([]byte, 4), nil
}

func (e *Encoder) encodeAdrp(inst *Instruction, address uint64) ([]byte, error) {
	// TODO: Implement adrp encoding (address to register, page)
	return make([]byte, 4), nil
}

func (e *Encoder) encodeSvc(inst *Instruction) ([]byte, error) {
	// SVC (supervisor call)
	// Format: 11010100 000 imm16 00001

	imm := uint32(0)
	if len(inst.Operands) > 0 {
		if inst.Operands[0].Type != OperandImmediate {
			return nil, fmt.Errorf("line %d:%d: svc requires immediate operand",
				inst.Line, inst.Column)
		}
		immVal, err := ParseInt(inst.Operands[0].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: svc immediate: %w", inst.Line, inst.Column, err)
		}
		imm = uint32(immVal)

		if imm > 0xFFFF {
			return nil, fmt.Errorf("line %d:%d: immediate %d too large for SVC (max 65535)",
				inst.Line, inst.Column, imm)
		}
	}

	// Build SVC instruction
	encoding := uint32(0xD4000001) // Base SVC #0 encoding
	encoding |= (imm & 0xFFFF) << 5  // Insert imm16

	return EncodeLittleEndian(encoding), nil
}

// EncodeData encodes a data declaration to bytes
func (e *Encoder) EncodeData(data *DataDeclaration) ([]byte, error) {
	switch data.Type {
	case "byte":
		return e.encodeByteValues(data.Value, 1)
	case "2byte", "hword":
		return e.encodeByteValues(data.Value, 2)
	case "4byte", "word":
		return e.encodeByteValues(data.Value, 4)
	case "8byte", "quad":
		return e.encodeByteValues(data.Value, 8)
	case "space", "zero":
		size, err := ParseInt(data.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid size for .%s: %w", data.Type, err)
		}
		return make([]byte, size), nil
	case "asciz", "string":
		// Null-terminated string
		s := unescapeDataString(data.Value)
		result := make([]byte, len(s)+1)
		copy(result, s)
		result[len(s)] = 0 // null terminator
		return result, nil
	case "ascii":
		// String without null terminator
		s := unescapeDataString(data.Value)
		return []byte(s), nil
	default:
		return nil, fmt.Errorf("unsupported data directive: .%s", data.Type)
	}
}

// encodeByteValues encodes comma-separated integer values to bytes
func (e *Encoder) encodeByteValues(value string, size int) ([]byte, error) {
	if value == "" {
		return nil, nil
	}

	parts := strings.Split(value, ",")
	result := make([]byte, 0, len(parts)*size)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		val, err := ParseInt64(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value '%s': %w", part, err)
		}

		bytes := make([]byte, size)
		switch size {
		case 1:
			bytes[0] = byte(val)
		case 2:
			binary.LittleEndian.PutUint16(bytes, uint16(val))
		case 4:
			binary.LittleEndian.PutUint32(bytes, uint32(val))
		case 8:
			binary.LittleEndian.PutUint64(bytes, uint64(val))
		}
		result = append(result, bytes...)
	}

	return result, nil
}

// unescapeDataString converts escape sequences in a string (same as in layout.go)
func unescapeDataString(s string) string {
	result := ""
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result += "\n"
			case 't':
				result += "\t"
			case 'r':
				result += "\r"
			case '\\':
				result += "\\"
			case '"':
				result += "\""
			case '0':
				result += "\x00"
			default:
				result += string(s[i+1])
			}
			i += 2
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}

