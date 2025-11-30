package slasm

import "fmt"

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

        sf := uint32(1)
        encoding := (sf << 31) | (0b01010001 << 23) | (imm << 10) | (uint32(rn) << 5) | uint32(rd)
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
	// TODO: Implement branch with link encoding
	return make([]byte, 4), nil
}

func (e *Encoder) encodeBranchRegister(inst *Instruction) ([]byte, error) {
	// TODO: Implement branch to register encoding
	return make([]byte, 4), nil
}

func (e *Encoder) encodeRet(inst *Instruction) ([]byte, error) {
	// RET is an alias for BR X30
	// Encoding: 1101011 00101 11111 00000 011110 00000
	return EncodeLittleEndian(0xd65f03c0), nil
}

func (e *Encoder) encodeLdr(inst *Instruction) ([]byte, error) {
	// TODO: Implement ldr encoding (load register)
	return make([]byte, 4), nil
}

func (e *Encoder) encodeStr(inst *Instruction) ([]byte, error) {
	// TODO: Implement str encoding (store register)
	return make([]byte, 4), nil
}

func (e *Encoder) encodeLdp(inst *Instruction) ([]byte, error) {
	// TODO: Implement ldp encoding (load pair)
	return make([]byte, 4), nil
}

func (e *Encoder) encodeStp(inst *Instruction) ([]byte, error) {
	// TODO: Implement stp encoding (store pair)
	return make([]byte, 4), nil
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

