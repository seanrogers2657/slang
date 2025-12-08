package slasm

import (
	"encoding/binary"
	"fmt"
	"strings"
)

// Encoder encodes ARM64 instructions to machine code
type Encoder struct {
	symbolTable *SymbolTable
	constants   map[string]int64
}

// NewEncoder creates a new instruction encoder
func NewEncoder(symbolTable *SymbolTable, constants map[string]int64) *Encoder {
	return &Encoder{
		symbolTable: symbolTable,
		constants:   constants,
	}
}

// Encoder helper functions to reduce code duplication

// errAt creates a formatted error at the instruction's location
func errAt(inst *Instruction, format string, args ...any) error {
	prefix := fmt.Sprintf("line %d:%d: ", inst.Line, inst.Column)
	return fmt.Errorf(prefix+format, args...)
}

// validateOperandCount checks if the instruction has the expected number of operands
func validateOperandCount(inst *Instruction, expected int) error {
	if len(inst.Operands) != expected {
		return errAt(inst, "%s requires %d operand(s), got %d", inst.Mnemonic, expected, len(inst.Operands))
	}
	return nil
}

// validateOperandCountRange checks if the operand count is within a range
func validateOperandCountRange(inst *Instruction, min, max int) error {
	if len(inst.Operands) < min || len(inst.Operands) > max {
		return errAt(inst, "%s requires %d-%d operands, got %d", inst.Mnemonic, min, max, len(inst.Operands))
	}
	return nil
}

// parseRegOperand parses a register at the given operand index
func parseRegOperand(inst *Instruction, idx int, name string) (int, error) {
	if idx >= len(inst.Operands) {
		return 0, errAt(inst, "%s: missing %s operand", inst.Mnemonic, name)
	}
	reg, err := ParseRegister(inst.Operands[idx].Value)
	if err != nil {
		return 0, errAt(inst, "%s %s: %w", inst.Mnemonic, name, err)
	}
	return reg, nil
}

// parseImmOperand parses an immediate value at the given operand index
func (e *Encoder) parseImmOperand(inst *Instruction, idx int, name string) (int64, error) {
	if idx >= len(inst.Operands) {
		return 0, errAt(inst, "%s: missing %s operand", inst.Mnemonic, name)
	}
	if inst.Operands[idx].Type != OperandImmediate {
		return 0, errAt(inst, "%s %s must be an immediate value", inst.Mnemonic, name)
	}
	val, err := e.ResolveImmediate(inst.Operands[idx].Value)
	if err != nil {
		return 0, errAt(inst, "%s %s: %w", inst.Mnemonic, name, err)
	}
	return val, nil
}

// validateImm12 checks if an immediate fits in 12 bits (unsigned)
func validateImm12(inst *Instruction, imm int64) error {
	if imm < 0 || imm > 0xFFF {
		return errAt(inst, "immediate %d out of range for %s (0-4095)", imm, inst.Mnemonic)
	}
	return nil
}

// validateImm16 checks if an immediate fits in 16 bits (unsigned)
func validateImm16(inst *Instruction, imm int64) error {
	if imm < 0 || imm > 0xFFFF {
		return errAt(inst, "immediate %d out of range for %s (0-65535)", imm, inst.Mnemonic)
	}
	return nil
}

// parseBranchTarget looks up a label and calculates the branch offset
func (e *Encoder) parseBranchTarget(inst *Instruction, address uint64) (int64, error) {
	if len(inst.Operands) < 1 || inst.Operands[0].Type != OperandLabel {
		return 0, errAt(inst, "%s requires a label operand", inst.Mnemonic)
	}

	labelName := inst.Operands[0].Value
	symbol, found := e.symbolTable.Lookup(labelName)
	if !found {
		return 0, errAt(inst, "undefined label '%s'", labelName)
	}

	// Calculate offset in instructions (each instruction is 4 bytes)
	offset := (int64(symbol.Address) - int64(address)) / 4
	return offset, nil
}

// validateBranchOffset26 checks if offset fits in 26-bit signed range
func validateBranchOffset26(inst *Instruction, offset int64, labelName string) error {
	if offset < -0x2000000 || offset > 0x1FFFFFF {
		return errAt(inst, "branch target '%s' too far (offset %d)", labelName, offset)
	}
	return nil
}

// validateBranchOffset19 checks if offset fits in 19-bit signed range
func validateBranchOffset19(inst *Instruction, offset int64, labelName string) error {
	if offset < -0x40000 || offset > 0x3FFFF {
		return errAt(inst, "branch target '%s' too far (offset %d)", labelName, offset)
	}
	return nil
}

// ResolveImmediate resolves an immediate value, which may be a constant name
func (e *Encoder) ResolveImmediate(value string) (int64, error) {
	// First try to parse as integer
	if val, err := ParseInt64(value); err == nil {
		return val, nil
	}

	// Try to look up as a constant
	if e.constants != nil {
		if constVal, found := e.constants[value]; found {
			return constVal, nil
		}
	}

	return 0, fmt.Errorf("unknown constant or invalid integer: %s", value)
}

// Encode encodes an instruction to machine code (4 bytes for ARM64)
func (e *Encoder) Encode(inst *Instruction, address uint64) ([]byte, error) {
	// TODO: Implement instruction encoding
	// This is the core of the assembler - converts mnemonics to machine code

	switch inst.Mnemonic {
	case "mov":
		return e.encodeMov(inst)
	case "movz":
		return e.encodeMovz(inst)
	case "movk":
		return e.encodeMovk(inst)
	case "add":
		return e.encodeAdd(inst)
	case "adds":
		return e.encodeAdds(inst)
	case "sub":
		return e.encodeSub(inst)
	case "subs":
		return e.encodeSubs(inst)
	case "mul":
		return e.encodeMul(inst)
	case "smulh":
		return e.encodeSmulh(inst)
	case "umulh":
		return e.encodeUmulh(inst)
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
		"b.vs", "b.vc", "b.hi", "b.ls", "b.ge", "b.lt", "b.gt", "b.le", "b.al",
		"beq", "bne", "bcs", "bhs", "bcc", "blo", "bmi", "bpl",
		"bvs", "bvc", "bhi", "bls", "bge", "blt", "bgt", "ble", "bal":
		return e.encodeBranchConditional(inst, address)
	case "ret":
		return e.encodeRet(inst)
	case "cbz":
		return e.encodeCbz(inst, address)
	case "cbnz":
		return e.encodeCbnz(inst, address)
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
	case "ldrh":
		return e.encodeLdrh(inst)
	case "strh":
		return e.encodeStrh(inst)
	case "adr":
		return e.encodeAdr(inst, address)
	case "adrp":
		return e.encodeAdrp(inst, address)
	case "svc":
		return e.encodeSvc(inst)
	case "lsl":
		return e.encodeLsl(inst)
	case "lsr":
		return e.encodeLsr(inst)
	case "asr":
		return e.encodeAsr(inst)
	case "and":
		return e.encodeAnd(inst)
	case "orr":
		return e.encodeOrr(inst)
	case "eor":
		return e.encodeEor(inst)
	case "mvn":
		return e.encodeMvn(inst)
	case "ands":
		return e.encodeAnds(inst)
	case "tst":
		return e.encodeTst(inst)
	case "bic":
		return e.encodeBic(inst)
	case "orn":
		return e.encodeOrn(inst)
	case "eon":
		return e.encodeEon(inst)
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
		immVal, err := e.ResolveImmediate(inst.Operands[1].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: mov immediate: %w", inst.Line, inst.Column, err)
		}
		imm := uint32(immVal)

		// Check if immediate fits in 16 bits
		if imm > 0xFFFF {
			return nil, fmt.Errorf("line %d:%d: immediate %d too large for MOV (max 65535)",
				inst.Line, inst.Column, imm)
		}

		sf := uint32(1) // X registers (64-bit)
		hw := uint32(0) // No shift
		encoding := (sf << 31) | (0b10100101 << 23) | (hw << 21) | (imm << 5) | uint32(rd)

		return EncodeLittleEndian(encoding), nil
	}

	// MOV Xd, Xm (register to register)
	rm, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: mov source: %w", inst.Line, inst.Column, err)
	}

	// Check if either operand is SP (stack pointer)
	// In ARM64, register 31 is interpreted as XZR in most instructions, but as SP in some
	// When moving to/from SP, we must use ADD Rd, Rn, #0 instead of ORR
	srcIsSP := inst.Operands[1].Value == "sp"
	dstIsSP := inst.Operands[0].Value == "sp"

	if srcIsSP || dstIsSP {
		// Use ADD Xd, Xn, #0 to move involving SP
		// ADD (immediate): sf 0 0 10001 shift imm12 Rn Rd
		sf := uint32(1) // 64-bit
		encoding := (sf << 31) | (0b0010001 << 24) | (0 << 10) | (uint32(rm) << 5) | uint32(rd)
		return EncodeLittleEndian(encoding), nil
	}

	// Normal register move: use ORR Xd, XZR, Xm
	sf := uint32(1)
	// ORR (shifted register): sf 01 01010 shift 0 Rm imm6 Rn Rd
	// With Rn=XZR(31), shift=00, imm6=0
	encoding := (sf << 31) | (0b0101010 << 24) | (uint32(rm) << 16) | (uint32(31) << 5) | uint32(rd)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeMovz(inst *Instruction) ([]byte, error) {
	// MOVZ Xd, #imm16, LSL #shift - Move with zero
	// Encoding: sf 10 100101 hw imm16 Rd
	// sf=1 for X registers, hw=shift/16 (00=0, 01=16, 10=32, 11=48)

	if len(inst.Operands) < 2 || len(inst.Operands) > 3 {
		return nil, fmt.Errorf("line %d:%d: movz requires 2-3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: movz destination: %w", inst.Line, inst.Column, err)
	}

	if inst.Operands[1].Type != OperandImmediate {
		return nil, fmt.Errorf("line %d:%d: movz requires immediate operand", inst.Line, inst.Column)
	}

	immVal, err := e.ResolveImmediate(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: movz immediate: %w", inst.Line, inst.Column, err)
	}
	imm := uint32(immVal)

	if imm > 0xFFFF {
		return nil, fmt.Errorf("line %d:%d: immediate %d too large for MOVZ (max 65535)",
			inst.Line, inst.Column, imm)
	}

	// Parse optional shift (lsl #N)
	hw := uint32(0)
	if len(inst.Operands) == 3 {
		if inst.Operands[2].Type != OperandShift {
			return nil, fmt.Errorf("line %d:%d: movz third operand must be shift", inst.Line, inst.Column)
		}
		shiftVal, err := ParseInt(inst.Operands[2].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: movz shift: %w", inst.Line, inst.Column, err)
		}
		switch shiftVal {
		case 0:
			hw = 0
		case 16:
			hw = 1
		case 32:
			hw = 2
		case 48:
			hw = 3
		default:
			return nil, fmt.Errorf("line %d:%d: movz shift must be 0, 16, 32, or 48", inst.Line, inst.Column)
		}
	}

	sf := uint32(1) // X registers (64-bit)
	// MOVZ: sf 10 100101 hw imm16 Rd = 0xD2800000
	encoding := (sf << 31) | (0b10100101 << 23) | (hw << 21) | (imm << 5) | uint32(rd)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeMovk(inst *Instruction) ([]byte, error) {
	// MOVK Xd, #imm16, LSL #shift - Move with keep
	// Encoding: sf 11 100101 hw imm16 Rd
	// sf=1 for X registers, hw=shift/16 (00=0, 01=16, 10=32, 11=48)

	if len(inst.Operands) < 2 || len(inst.Operands) > 3 {
		return nil, fmt.Errorf("line %d:%d: movk requires 2-3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: movk destination: %w", inst.Line, inst.Column, err)
	}

	if inst.Operands[1].Type != OperandImmediate {
		return nil, fmt.Errorf("line %d:%d: movk requires immediate operand", inst.Line, inst.Column)
	}

	immVal, err := e.ResolveImmediate(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: movk immediate: %w", inst.Line, inst.Column, err)
	}
	imm := uint32(immVal)

	if imm > 0xFFFF {
		return nil, fmt.Errorf("line %d:%d: immediate %d too large for MOVK (max 65535)",
			inst.Line, inst.Column, imm)
	}

	// Parse required shift (lsl #N) for MOVK
	hw := uint32(0)
	if len(inst.Operands) == 3 {
		if inst.Operands[2].Type != OperandShift {
			return nil, fmt.Errorf("line %d:%d: movk third operand must be shift", inst.Line, inst.Column)
		}
		shiftVal, err := ParseInt(inst.Operands[2].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: movk shift: %w", inst.Line, inst.Column, err)
		}
		switch shiftVal {
		case 0:
			hw = 0
		case 16:
			hw = 1
		case 32:
			hw = 2
		case 48:
			hw = 3
		default:
			return nil, fmt.Errorf("line %d:%d: movk shift must be 0, 16, 32, or 48", inst.Line, inst.Column)
		}
	}

	sf := uint32(1) // X registers (64-bit)
	// MOVK: sf 11 100101 hw imm16 Rd = 0xF2800000
	encoding := (sf << 31) | (0b11100101 << 23) | (hw << 21) | (imm << 5) | uint32(rd)

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

	// Check for label with @PAGEOFF suffix (for ADRP+ADD pattern)
	if inst.Operands[2].Type == OperandLabel {
		labelName := inst.Operands[2].Value
		if strings.HasSuffix(labelName, "@PAGEOFF") {
			// Strip @PAGEOFF suffix and look up label
			labelName = strings.TrimSuffix(labelName, "@PAGEOFF")
			symbol, found := e.symbolTable.Lookup(labelName)
			if !found {
				return nil, fmt.Errorf("line %d:%d: undefined label '%s'",
					inst.Line, inst.Column, labelName)
			}

			// Use low 12 bits of the label address as immediate
			imm := uint32(symbol.Address) & 0xFFF

			sf := uint32(1) // X registers (64-bit)
			encoding := (sf << 31) | (0b0010001 << 24) | (imm << 10) | (uint32(rn) << 5) | uint32(rd)

			return EncodeLittleEndian(encoding), nil
		}
		return nil, fmt.Errorf("line %d:%d: add with label requires @PAGEOFF suffix",
			inst.Line, inst.Column)
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

func (e *Encoder) encodeAdds(inst *Instruction) ([]byte, error) {
	// ADDS Xd, Xn, #imm12 or ADDS Xd, Xn, Xm (add with flags)
	// Same as ADD but with S=1 to set flags

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: adds requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: adds destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: adds operand 1: %w", inst.Line, inst.Column, err)
	}

	if inst.Operands[2].Type == OperandImmediate {
		immVal, err := ParseInt(inst.Operands[2].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: adds immediate: %w", inst.Line, inst.Column, err)
		}
		imm := uint32(immVal)

		if imm > 0xFFF {
			return nil, fmt.Errorf("line %d:%d: immediate %d too large for ADDS (max 4095)",
				inst.Line, inst.Column, imm)
		}

		// ADDS (immediate): sf 0 1 10001 shift imm12 Rn Rd
		// sf=1 for 64-bit, op=0 (ADD), S=1 (set flags)
		// 0xB1000000 = 10110001 00000000 00000000 00000000
		encoding := uint32(0xB1000000) | (imm << 10) | (uint32(rn) << 5) | uint32(rd)
		return EncodeLittleEndian(encoding), nil
	}

	// Register form: ADDS Xd, Xn, Xm
	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: adds operand 2: %w", inst.Line, inst.Column, err)
	}

	// ADDS (shifted register): sf 0 1 01011 shift 0 Rm imm6 Rn Rd
	// sf=1, op=0, S=1 => 10101011...
	// 0xAB000000 = 10101011 00000000 00000000 00000000
	encoding := uint32(0xAB000000) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
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

func (e *Encoder) encodeSubs(inst *Instruction) ([]byte, error) {
	// SUBS Xd, Xn, #imm12 or SUBS Xd, Xn, Xm (subtract with flags)
	// Same as SUB but with S=1 to set flags

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: subs requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: subs destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: subs operand 1: %w", inst.Line, inst.Column, err)
	}

	if inst.Operands[2].Type == OperandImmediate {
		immVal, err := ParseInt(inst.Operands[2].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: subs immediate: %w", inst.Line, inst.Column, err)
		}
		imm := uint32(immVal)

		if imm > 0xFFF {
			return nil, fmt.Errorf("line %d:%d: immediate %d too large for SUBS (max 4095)",
				inst.Line, inst.Column, imm)
		}

		// SUBS (immediate): sf 1 1 10001 shift imm12 Rn Rd
		// sf=1 for 64-bit, op=1 (SUB), S=1 (set flags)
		// 0xF1000000 = 11110001 00000000 00000000 00000000
		encoding := uint32(0xF1000000) | (imm << 10) | (uint32(rn) << 5) | uint32(rd)
		return EncodeLittleEndian(encoding), nil
	}

	// Register form: SUBS Xd, Xn, Xm
	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: subs operand 2: %w", inst.Line, inst.Column, err)
	}

	// SUBS (shifted register): sf 1 1 01011 shift 0 Rm imm6 Rn Rd
	// sf=1, op=1, S=1 => 11101011...
	// 0xEB000000 = 11101011 00000000 00000000 00000000
	encoding := uint32(0xEB000000) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
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

func (e *Encoder) encodeSmulh(inst *Instruction) ([]byte, error) {
	// SMULH Xd, Xn, Xm - Signed multiply high (upper 64 bits of 128-bit result)
	// Encoding: 1 0011011 010 Rm 0 11111 Rn Rd
	// = 0x9B400000 | (Rm << 16) | (0x1F << 10) | (Rn << 5) | Rd

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: smulh requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: smulh destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: smulh operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: smulh operand 2: %w", inst.Line, inst.Column, err)
	}

	// SMULH: 1 00 11011 010 Rm 0 11111 Rn Rd
	// 0x9B407C00 = base with Ra=11111
	encoding := uint32(0x9B407C00) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeUmulh(inst *Instruction) ([]byte, error) {
	// UMULH Xd, Xn, Xm - Unsigned multiply high (upper 64 bits of 128-bit result)
	// Encoding: 1 0011011 110 Rm 0 11111 Rn Rd
	// = 0x9BC00000 | (Rm << 16) | (0x1F << 10) | (Rn << 5) | Rd

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: umulh requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: umulh destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: umulh operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: umulh operand 2: %w", inst.Line, inst.Column, err)
	}

	// UMULH: 1 00 11011 110 Rm 0 11111 Rn Rd
	// 0x9BC07C00 = base with Ra=11111
	encoding := uint32(0x9BC07C00) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
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
	// UDIV Xd, Xn, Xm
	// Encoding: sf 0 011010110 Rm 000010 Rn Rd
	// Same as SDIV but opcode bits [11:10] = 10 instead of 11

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: udiv requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: udiv destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: udiv operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: udiv operand 2: %w", inst.Line, inst.Column, err)
	}

	sf := uint32(1)
	encoding := (sf << 31) | (0b0011010110 << 21) | (uint32(rm) << 16) | (0b000010 << 10) | (uint32(rn) << 5) | uint32(rd)

	return EncodeLittleEndian(encoding), nil
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
	// NEG Xd, Xm is an alias for SUB Xd, XZR, Xm
	// SUB (register): sf 1001011 shift 0 Rm imm6 Rn Rd
	// With Rn = XZR (31), shift = 0, imm6 = 0

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: neg requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: neg destination: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: neg source: %w", inst.Line, inst.Column, err)
	}

	sf := uint32(1)  // 64-bit
	rn := uint32(31) // XZR

	// SUB (register): sf 1001011 shift 0 Rm imm6 Rn Rd
	encoding := (sf << 31) | (0b1001011 << 24) | (uint32(rm) << 16) | (rn << 5) | uint32(rd)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeCmp(inst *Instruction) ([]byte, error) {
	// CMP Xn, #imm or CMP Xn, Xm or CMP Xn, Xm, shift #amount
	// This is SUBS XZR, Xn, operand

	if len(inst.Operands) < 2 || len(inst.Operands) > 3 {
		return nil, fmt.Errorf("line %d:%d: cmp requires 2-3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rn, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: cmp operand 1: %w", inst.Line, inst.Column, err)
	}
	rd := uint32(31) // XZR

	if inst.Operands[1].Type == OperandImmediate {
		immVal, err := e.ResolveImmediate(inst.Operands[1].Value)
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

	// Parse optional shift
	shiftType := uint32(0) // LSL
	shiftAmount := uint32(0)
	if len(inst.Operands) == 3 {
		if inst.Operands[2].Type != OperandShift {
			return nil, fmt.Errorf("line %d:%d: cmp third operand must be shift", inst.Line, inst.Column)
		}
		switch inst.Operands[2].ShiftType {
		case "lsl":
			shiftType = 0b00
		case "lsr":
			shiftType = 0b01
		case "asr":
			shiftType = 0b10
		default:
			return nil, fmt.Errorf("line %d:%d: cmp unsupported shift type: %s",
				inst.Line, inst.Column, inst.Operands[2].ShiftType)
		}
		amount, err := ParseInt(inst.Operands[2].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: cmp shift amount: %w", inst.Line, inst.Column, err)
		}
		if amount < 0 || amount > 63 {
			return nil, fmt.Errorf("line %d:%d: cmp shift amount must be 0-63, got %d",
				inst.Line, inst.Column, amount)
		}
		shiftAmount = uint32(amount)
	}

	sf := uint32(1)
	// SUBS (shifted register): sf 1 1 01011 shift(2) 0 Rm imm6(6) Rn Rd
	encoding := (sf << 31) | (0b1101011 << 24) | (shiftType << 22) |
		(uint32(rm) << 16) | (shiftAmount << 10) | (uint32(rn) << 5) | rd

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

	// Extract condition code from mnemonic (e.g., "b.eq" -> "eq", "beq" -> "eq")
	var cond string
	if strings.HasPrefix(inst.Mnemonic, "b.") {
		cond = inst.Mnemonic[2:] // skip "b."
	} else {
		cond = inst.Mnemonic[1:] // skip "b" (for "beq", "bne", etc.)
	}

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

func (e *Encoder) encodeCbz(inst *Instruction, address uint64) ([]byte, error) {
	// CBZ Xn, label - Compare and branch if zero
	// Encoding: sf 0 11010 0 imm19 Rt
	// sf=1 for X registers (64-bit)

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: cbz requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: cbz first operand must be a register",
			inst.Line, inst.Column)
	}
	rt, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: cbz register: %w", inst.Line, inst.Column, err)
	}

	if inst.Operands[1].Type != OperandLabel {
		return nil, fmt.Errorf("line %d:%d: cbz second operand must be a label",
			inst.Line, inst.Column)
	}

	labelName := inst.Operands[1].Value
	symbol, found := e.symbolTable.Lookup(labelName)
	if !found {
		return nil, fmt.Errorf("line %d:%d: undefined label '%s'",
			inst.Line, inst.Column, labelName)
	}

	// Calculate offset in instructions
	targetAddr := symbol.Address
	offset := (int64(targetAddr) - int64(address)) / 4

	// Check if offset fits in 19 bits (signed)
	if offset < -0x40000 || offset > 0x3FFFF {
		return nil, fmt.Errorf("line %d:%d: cbz target '%s' is too far away (offset %d)",
			inst.Line, inst.Column, labelName, offset)
	}

	// CBZ (64-bit): 1 011010 0 imm19 Rt = 0xB4000000
	imm19 := uint32(offset) & 0x7FFFF
	encoding := uint32(0xB4000000) | (imm19 << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeCbnz(inst *Instruction, address uint64) ([]byte, error) {
	// CBNZ Xn, label - Compare and branch if not zero
	// Encoding: sf 0 11010 1 imm19 Rt
	// sf=1 for X registers (64-bit)

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: cbnz requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: cbnz first operand must be a register",
			inst.Line, inst.Column)
	}
	rt, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: cbnz register: %w", inst.Line, inst.Column, err)
	}

	if inst.Operands[1].Type != OperandLabel {
		return nil, fmt.Errorf("line %d:%d: cbnz second operand must be a label",
			inst.Line, inst.Column)
	}

	labelName := inst.Operands[1].Value
	symbol, found := e.symbolTable.Lookup(labelName)
	if !found {
		return nil, fmt.Errorf("line %d:%d: undefined label '%s'",
			inst.Line, inst.Column, labelName)
	}

	// Calculate offset in instructions
	targetAddr := symbol.Address
	offset := (int64(targetAddr) - int64(address)) / 4

	// Check if offset fits in 19 bits (signed)
	if offset < -0x40000 || offset > 0x3FFFF {
		return nil, fmt.Errorf("line %d:%d: cbnz target '%s' is too far away (offset %d)",
			inst.Line, inst.Column, labelName, offset)
	}

	// CBNZ (64-bit): 1 011010 1 imm19 Rt = 0xB5000000
	imm19 := uint32(offset) & 0x7FFFF
	encoding := uint32(0xB5000000) | (imm19 << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeLdr(inst *Instruction) ([]byte, error) {
	// LDR Xt, [Xn, #imm] - Load register
	// Multiple encoding forms:
	// 1. Unsigned offset: 11 111 00100 01 imm12 Rn Rt (imm12 scaled by 8, range 0-32760)
	// 2. Unscaled (LDUR): 11 111 000 01 0 imm9 00 Rn Rt (imm9 signed, range -256 to 255)
	// 3. Pre-indexed:     11 111 000 01 0 imm9 11 Rn Rt (with writeback)
	// 4. Post-indexed:    11 111 000 01 0 imm9 01 Rn Rt (with writeback)

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

	// Check for post-indexed mode: [base], #offset
	if inst.Operands[1].PostIndexOffset != "" {
		offset, err := ParseInt64(inst.Operands[1].PostIndexOffset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: ldr post-index offset: %w", inst.Line, inst.Column, err)
		}
		if offset < -256 || offset > 255 {
			return nil, fmt.Errorf("line %d:%d: ldr post-index offset must be -256 to 255, got %d",
				inst.Line, inst.Column, offset)
		}
		// LDR (post-index): 11 111 000 01 0 imm9 01 Rn Rt = 0xF8400400
		imm9 := uint32(offset) & 0x1FF
		encoding := uint32(0xF8400400) | (imm9 << 12) | (uint32(rn) << 5) | uint32(rt)
		return EncodeLittleEndian(encoding), nil
	}

	// Parse offset (default to 0 if not specified)
	offset := int64(0)
	if inst.Operands[1].Offset != "" {
		offset, err = ParseInt64(inst.Operands[1].Offset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: ldr offset: %w", inst.Line, inst.Column, err)
		}
	}

	// Check for pre-indexed mode: [base, #offset]!
	if inst.Operands[1].Writeback {
		if offset < -256 || offset > 255 {
			return nil, fmt.Errorf("line %d:%d: ldr pre-index offset must be -256 to 255, got %d",
				inst.Line, inst.Column, offset)
		}
		// LDR (pre-index): 11 111 000 01 0 imm9 11 Rn Rt = 0xF8400C00
		imm9 := uint32(offset) & 0x1FF
		encoding := uint32(0xF8400C00) | (imm9 << 12) | (uint32(rn) << 5) | uint32(rt)
		return EncodeLittleEndian(encoding), nil
	}

	// Auto-detect which encoding to use:
	// - If offset is negative or not aligned to 8, use LDUR (unscaled)
	// - Otherwise use LDR with unsigned offset
	if offset < 0 || offset%8 != 0 {
		// Use LDUR (unscaled) encoding for negative or unaligned offsets
		if offset < -256 || offset > 255 {
			return nil, fmt.Errorf("line %d:%d: ldr unscaled offset must be -256 to 255, got %d",
				inst.Line, inst.Column, offset)
		}

		// LDUR: 11 111 000 01 0 imm9 00 Rn Rt
		// = 0xF8400000 | (imm9 << 12) | (Rn << 5) | Rt
		imm9 := uint32(offset) & 0x1FF
		encoding := uint32(0xF8400000) | (imm9 << 12) | (uint32(rn) << 5) | uint32(rt)
		return EncodeLittleEndian(encoding), nil
	}

	// Use LDR with unsigned offset (must be 0-32760 and multiple of 8)
	if offset > 32760 {
		return nil, fmt.Errorf("line %d:%d: ldr offset must be 0-32760 and multiple of 8, got %d",
			inst.Line, inst.Column, offset)
	}

	// Scale offset for encoding (divide by 8)
	imm12 := uint32(offset / 8)

	// LDR (unsigned offset): 11 111 00100 01 imm12 Rn Rt
	// = 0xF9400000 | (imm12 << 10) | (Rn << 5) | Rt
	encoding := uint32(0xF9400000) | (imm12 << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeStr(inst *Instruction) ([]byte, error) {
	// STR Xt, [Xn, #imm] - Store register
	// Multiple encoding forms:
	// 1. Unsigned offset: 11 111 00100 00 imm12 Rn Rt (imm12 scaled by 8, range 0-32760)
	// 2. Unscaled (STUR): 11 111 000 00 0 imm9 00 Rn Rt (imm9 signed, range -256 to 255)
	// 3. Pre-indexed:     11 111 000 00 0 imm9 11 Rn Rt (with writeback)
	// 4. Post-indexed:    11 111 000 00 0 imm9 01 Rn Rt (with writeback)

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

	// Check for post-indexed mode: [base], #offset
	if inst.Operands[1].PostIndexOffset != "" {
		offset, err := ParseInt64(inst.Operands[1].PostIndexOffset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: str post-index offset: %w", inst.Line, inst.Column, err)
		}
		if offset < -256 || offset > 255 {
			return nil, fmt.Errorf("line %d:%d: str post-index offset must be -256 to 255, got %d",
				inst.Line, inst.Column, offset)
		}
		// STR (post-index): 11 111 000 00 0 imm9 01 Rn Rt = 0xF8000400
		imm9 := uint32(offset) & 0x1FF
		encoding := uint32(0xF8000400) | (imm9 << 12) | (uint32(rn) << 5) | uint32(rt)
		return EncodeLittleEndian(encoding), nil
	}

	// Parse offset (default to 0 if not specified)
	offset := int64(0)
	if inst.Operands[1].Offset != "" {
		offset, err = ParseInt64(inst.Operands[1].Offset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: str offset: %w", inst.Line, inst.Column, err)
		}
	}

	// Check for pre-indexed mode: [base, #offset]!
	if inst.Operands[1].Writeback {
		if offset < -256 || offset > 255 {
			return nil, fmt.Errorf("line %d:%d: str pre-index offset must be -256 to 255, got %d",
				inst.Line, inst.Column, offset)
		}
		// STR (pre-index): 11 111 000 00 0 imm9 11 Rn Rt = 0xF8000C00
		imm9 := uint32(offset) & 0x1FF
		encoding := uint32(0xF8000C00) | (imm9 << 12) | (uint32(rn) << 5) | uint32(rt)
		return EncodeLittleEndian(encoding), nil
	}

	// Auto-detect which encoding to use:
	// - If offset is negative or not aligned to 8, use STUR (unscaled)
	// - Otherwise use STR with unsigned offset
	if offset < 0 || offset%8 != 0 {
		// Use STUR (unscaled) encoding for negative or unaligned offsets
		if offset < -256 || offset > 255 {
			return nil, fmt.Errorf("line %d:%d: str unscaled offset must be -256 to 255, got %d",
				inst.Line, inst.Column, offset)
		}

		// STUR: 11 111 000 00 0 imm9 00 Rn Rt
		// = 0xF8000000 | (imm9 << 12) | (Rn << 5) | Rt
		imm9 := uint32(offset) & 0x1FF
		encoding := uint32(0xF8000000) | (imm9 << 12) | (uint32(rn) << 5) | uint32(rt)
		return EncodeLittleEndian(encoding), nil
	}

	// Use STR with unsigned offset (must be 0-32760 and multiple of 8)
	if offset > 32760 {
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
	// LDP Xt1, Xt2, [Xn, #imm] - Load pair
	// Supports three addressing modes:
	// - Signed offset: [Xn, #imm]     - base 0xA9400000 (bits [25:23] = 010)
	// - Pre-indexed:   [Xn, #imm]!    - base 0xA9C00000 (bits [25:23] = 011)
	// - Post-indexed:  [Xn], #imm     - base 0xA8C00000 (bits [25:23] = 001)
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

	// Determine addressing mode and get offset
	var offset int64
	var baseEncoding uint32

	if inst.Operands[2].PostIndexOffset != "" {
		// Post-indexed: [Xn], #imm
		offset, err = ParseInt64(inst.Operands[2].PostIndexOffset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: ldp post-index offset: %w", inst.Line, inst.Column, err)
		}
		baseEncoding = 0xA8C00000
	} else if inst.Operands[2].Writeback {
		// Pre-indexed: [Xn, #imm]!
		if inst.Operands[2].Offset != "" {
			offset, err = ParseInt64(inst.Operands[2].Offset)
			if err != nil {
				return nil, fmt.Errorf("line %d:%d: ldp offset: %w", inst.Line, inst.Column, err)
			}
		}
		baseEncoding = 0xA9C00000
	} else {
		// Signed offset: [Xn, #imm]
		if inst.Operands[2].Offset != "" {
			offset, err = ParseInt64(inst.Operands[2].Offset)
			if err != nil {
				return nil, fmt.Errorf("line %d:%d: ldp offset: %w", inst.Line, inst.Column, err)
			}
		}
		baseEncoding = 0xA9400000
	}

	// For 64-bit LDP, offset must be multiple of 8 and within signed 7-bit range * 8
	if offset < -512 || offset > 504 || offset%8 != 0 {
		return nil, fmt.Errorf("line %d:%d: ldp offset must be -512 to 504 and multiple of 8, got %d",
			inst.Line, inst.Column, offset)
	}

	// Scale offset for encoding (divide by 8, signed 7-bit)
	imm7 := uint32(offset/8) & 0x7F

	encoding := baseEncoding | (imm7 << 15) | (uint32(rt2) << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeStp(inst *Instruction) ([]byte, error) {
	// STP Xt1, Xt2, [Xn, #imm] - Store pair
	// Supports three addressing modes:
	// - Signed offset: [Xn, #imm]     - base 0xA9000000 (bits [25:23] = 010)
	// - Pre-indexed:   [Xn, #imm]!    - base 0xA9800000 (bits [25:23] = 011)
	// - Post-indexed:  [Xn], #imm     - base 0xA8800000 (bits [25:23] = 001)
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

	// Determine addressing mode and get offset
	var offset int64
	var baseEncoding uint32

	if inst.Operands[2].PostIndexOffset != "" {
		// Post-indexed: [Xn], #imm
		offset, err = ParseInt64(inst.Operands[2].PostIndexOffset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: stp post-index offset: %w", inst.Line, inst.Column, err)
		}
		baseEncoding = 0xA8800000
	} else if inst.Operands[2].Writeback {
		// Pre-indexed: [Xn, #imm]!
		if inst.Operands[2].Offset != "" {
			offset, err = ParseInt64(inst.Operands[2].Offset)
			if err != nil {
				return nil, fmt.Errorf("line %d:%d: stp offset: %w", inst.Line, inst.Column, err)
			}
		}
		baseEncoding = 0xA9800000
	} else {
		// Signed offset: [Xn, #imm]
		if inst.Operands[2].Offset != "" {
			offset, err = ParseInt64(inst.Operands[2].Offset)
			if err != nil {
				return nil, fmt.Errorf("line %d:%d: stp offset: %w", inst.Line, inst.Column, err)
			}
		}
		baseEncoding = 0xA9000000
	}

	// For 64-bit STP, offset must be multiple of 8 and within signed 7-bit range * 8
	if offset < -512 || offset > 504 || offset%8 != 0 {
		return nil, fmt.Errorf("line %d:%d: stp offset must be -512 to 504 and multiple of 8, got %d",
			inst.Line, inst.Column, offset)
	}

	// Scale offset for encoding (divide by 8, signed 7-bit)
	imm7 := uint32(offset/8) & 0x7F

	encoding := baseEncoding | (imm7 << 15) | (uint32(rt2) << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeLdrb(inst *Instruction) ([]byte, error) {
	// LDRB Wt, [Xn, #imm] - Load byte (unsigned offset)
	// Encoding: 00 111 00101 01 imm12 Rn Rt
	// imm12 is NOT scaled for byte operations (range 0-4095)

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: ldrb requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	// First operand: destination register (must be W register)
	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: ldrb destination must be a register",
			inst.Line, inst.Column)
	}
	rt, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ldrb destination: %w", inst.Line, inst.Column, err)
	}

	// Second operand: memory operand [base, #offset]
	if inst.Operands[1].Type != OperandMemory {
		return nil, fmt.Errorf("line %d:%d: ldrb source must be a memory operand",
			inst.Line, inst.Column)
	}

	rn, err := ParseRegister(inst.Operands[1].Base)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ldrb base register: %w", inst.Line, inst.Column, err)
	}

	// Parse offset (default to 0 if not specified)
	offset := int64(0)
	if inst.Operands[1].Offset != "" {
		offset, err = ParseInt64(inst.Operands[1].Offset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: ldrb offset: %w", inst.Line, inst.Column, err)
		}
	}

	// For LDRB, offset must be 0-4095 (no scaling for bytes)
	if offset < 0 || offset > 4095 {
		return nil, fmt.Errorf("line %d:%d: ldrb offset must be 0-4095, got %d",
			inst.Line, inst.Column, offset)
	}

	imm12 := uint32(offset)

	// LDRB (unsigned offset): 00 111 00101 01 imm12 Rn Rt
	// = 0x39400000 | (imm12 << 10) | (Rn << 5) | Rt
	encoding := uint32(0x39400000) | (imm12 << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeStrb(inst *Instruction) ([]byte, error) {
	// STRB Wt, [Xn, #imm] - Store byte (unsigned offset)
	// Encoding: 00 111 00100 00 imm12 Rn Rt
	// imm12 is NOT scaled for byte operations (range 0-4095)

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: strb requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	// First operand: source register (must be W register)
	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: strb source must be a register",
			inst.Line, inst.Column)
	}
	rt, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: strb source: %w", inst.Line, inst.Column, err)
	}

	// Second operand: memory operand [base, #offset]
	if inst.Operands[1].Type != OperandMemory {
		return nil, fmt.Errorf("line %d:%d: strb destination must be a memory operand",
			inst.Line, inst.Column)
	}

	rn, err := ParseRegister(inst.Operands[1].Base)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: strb base register: %w", inst.Line, inst.Column, err)
	}

	// Parse offset (default to 0 if not specified)
	offset := int64(0)
	if inst.Operands[1].Offset != "" {
		offset, err = ParseInt64(inst.Operands[1].Offset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: strb offset: %w", inst.Line, inst.Column, err)
		}
	}

	// For STRB, offset must be 0-4095 (no scaling for bytes)
	if offset < 0 || offset > 4095 {
		return nil, fmt.Errorf("line %d:%d: strb offset must be 0-4095, got %d",
			inst.Line, inst.Column, offset)
	}

	imm12 := uint32(offset)

	// STRB (unsigned offset): 00 111 00100 00 imm12 Rn Rt
	// = 0x39000000 | (imm12 << 10) | (Rn << 5) | Rt
	encoding := uint32(0x39000000) | (imm12 << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeLdrh(inst *Instruction) ([]byte, error) {
	// LDRH Wt, [Xn, #imm] - Load halfword (unsigned offset, zero-extended to 32-bit)
	// Encoding: 01 111 00101 01 imm12 Rn Rt
	// imm12 is scaled by 2 for halfword operations (range 0-8190, must be even)

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: ldrh requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	// First operand: destination register
	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: ldrh destination must be a register",
			inst.Line, inst.Column)
	}
	rt, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ldrh destination: %w", inst.Line, inst.Column, err)
	}

	// Second operand: memory operand [base, #offset]
	if inst.Operands[1].Type != OperandMemory {
		return nil, fmt.Errorf("line %d:%d: ldrh source must be a memory operand",
			inst.Line, inst.Column)
	}

	rn, err := ParseRegister(inst.Operands[1].Base)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ldrh base register: %w", inst.Line, inst.Column, err)
	}

	// Parse offset (default to 0 if not specified)
	offset := int64(0)
	if inst.Operands[1].Offset != "" {
		offset, err = ParseInt64(inst.Operands[1].Offset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: ldrh offset: %w", inst.Line, inst.Column, err)
		}
	}

	// For LDRH, offset must be halfword-aligned (divisible by 2)
	if offset%2 != 0 {
		return nil, fmt.Errorf("line %d:%d: ldrh offset must be halfword-aligned (divisible by 2), got %d",
			inst.Line, inst.Column, offset)
	}

	// Scale offset by 2 for encoding
	scaledOffset := offset / 2
	if scaledOffset < 0 || scaledOffset > 4095 {
		return nil, fmt.Errorf("line %d:%d: ldrh offset must be 0-8190, got %d",
			inst.Line, inst.Column, offset)
	}

	imm12 := uint32(scaledOffset)

	// LDRH (unsigned offset): 01 111 00101 01 imm12 Rn Rt
	// = 0x79400000 | (imm12 << 10) | (Rn << 5) | Rt
	encoding := uint32(0x79400000) | (imm12 << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeStrh(inst *Instruction) ([]byte, error) {
	// STRH Wt, [Xn, #imm] - Store halfword (unsigned offset)
	// Encoding: 01 111 00100 00 imm12 Rn Rt
	// imm12 is scaled by 2 for halfword operations (range 0-8190, must be even)

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: strh requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	// First operand: source register
	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: strh source must be a register",
			inst.Line, inst.Column)
	}
	rt, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: strh source: %w", inst.Line, inst.Column, err)
	}

	// Second operand: memory operand [base, #offset]
	if inst.Operands[1].Type != OperandMemory {
		return nil, fmt.Errorf("line %d:%d: strh destination must be a memory operand",
			inst.Line, inst.Column)
	}

	rn, err := ParseRegister(inst.Operands[1].Base)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: strh base register: %w", inst.Line, inst.Column, err)
	}

	// Parse offset (default to 0 if not specified)
	offset := int64(0)
	if inst.Operands[1].Offset != "" {
		offset, err = ParseInt64(inst.Operands[1].Offset)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: strh offset: %w", inst.Line, inst.Column, err)
		}
	}

	// For STRH, offset must be halfword-aligned (divisible by 2)
	if offset%2 != 0 {
		return nil, fmt.Errorf("line %d:%d: strh offset must be halfword-aligned (divisible by 2), got %d",
			inst.Line, inst.Column, offset)
	}

	// Scale offset by 2 for encoding
	scaledOffset := offset / 2
	if scaledOffset < 0 || scaledOffset > 4095 {
		return nil, fmt.Errorf("line %d:%d: strh offset must be 0-8190, got %d",
			inst.Line, inst.Column, offset)
	}

	imm12 := uint32(scaledOffset)

	// STRH (unsigned offset): 01 111 00100 00 imm12 Rn Rt
	// = 0x79000000 | (imm12 << 10) | (Rn << 5) | Rt
	encoding := uint32(0x79000000) | (imm12 << 10) | (uint32(rn) << 5) | uint32(rt)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeAdr(inst *Instruction, address uint64) ([]byte, error) {
	// ADR Xd, label - Form PC-relative address (±1MB range)
	// Encoding: 0 immlo 10000 immhi Rd
	// immlo (2 bits), immhi (19 bits) = 21-bit signed byte offset from PC

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: adr requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	// First operand: destination register
	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: adr destination must be a register",
			inst.Line, inst.Column)
	}
	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: adr destination: %w", inst.Line, inst.Column, err)
	}

	// Second operand: label
	if inst.Operands[1].Type != OperandLabel {
		return nil, fmt.Errorf("line %d:%d: adr requires a label operand",
			inst.Line, inst.Column)
	}

	labelName := inst.Operands[1].Value
	symbol, found := e.symbolTable.Lookup(labelName)
	if !found {
		return nil, fmt.Errorf("line %d:%d: undefined label '%s'",
			inst.Line, inst.Column, labelName)
	}

	// Calculate byte offset from PC to target
	targetAddr := symbol.Address
	offset := int64(targetAddr) - int64(address)

	// Check if offset fits in 21 bits (signed, byte offset)
	// Range: ±1MB = ±2^20 bytes
	if offset < -0x100000 || offset > 0xFFFFF {
		return nil, fmt.Errorf("line %d:%d: adr target '%s' is too far away (offset %d bytes, max ±1MB)",
			inst.Line, inst.Column, labelName, offset)
	}

	// ADR encoding: 0 immlo[1:0] 10000 immhi[18:0] Rd[4:0]
	// op=0 for ADR (vs op=1 for ADRP)
	immlo := uint32(offset) & 0x3            // bits 1:0
	immhi := (uint32(offset) >> 2) & 0x7FFFF // bits 20:2

	encoding := (immlo << 29) | (0b10000 << 24) | (immhi << 5) | uint32(rd)

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeAdrp(inst *Instruction, address uint64) ([]byte, error) {
	// ADRP Xd, label@PAGE - Address of page (PC-relative)
	// Encoding: 1 immlo 10000 immhi Rd
	// immlo (2 bits), immhi (19 bits) = 21-bit signed page offset from PC

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: adrp requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	// First operand: destination register
	if inst.Operands[0].Type != OperandRegister {
		return nil, fmt.Errorf("line %d:%d: adrp destination must be a register",
			inst.Line, inst.Column)
	}
	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: adrp destination: %w", inst.Line, inst.Column, err)
	}

	// Second operand: label (optionally with @PAGE suffix)
	if inst.Operands[1].Type != OperandLabel {
		return nil, fmt.Errorf("line %d:%d: adrp requires a label operand",
			inst.Line, inst.Column)
	}

	// Strip @PAGE suffix if present
	labelName := inst.Operands[1].Value
	labelName = strings.TrimSuffix(labelName, "@PAGE")

	symbol, found := e.symbolTable.Lookup(labelName)
	if !found {
		return nil, fmt.Errorf("line %d:%d: undefined label '%s'",
			inst.Line, inst.Column, labelName)
	}

	// Calculate page offset
	// ADRP loads the page address (4KB aligned) of the label
	// Page offset = (labelPage - currentPage)
	// Note: The symbol table contains section-relative addresses
	// For now, we use the relative addresses and assume text starts at 0
	targetPage := int64(symbol.Address) &^ 0xFFF
	currentPage := int64(address) &^ 0xFFF
	pageOffset := (targetPage - currentPage) >> 12

	// Check if offset fits in 21 bits (signed)
	if pageOffset < -0x100000 || pageOffset > 0xFFFFF {
		return nil, fmt.Errorf("line %d:%d: adrp target '%s' is too far away (page offset %d)",
			inst.Line, inst.Column, labelName, pageOffset)
	}

	// ADRP encoding: 1 immlo[1:0] 10000 immhi[18:0] Rd[4:0]
	immlo := uint32(pageOffset) & 0x3            // bits 1:0
	immhi := (uint32(pageOffset) >> 2) & 0x7FFFF // bits 20:2

	encoding := uint32(1<<31) | (immlo << 29) | (0b10000 << 24) | (immhi << 5) | uint32(rd)

	return EncodeLittleEndian(encoding), nil
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
	encoding := uint32(0xD4000001)  // Base SVC #0 encoding
	encoding |= (imm & 0xFFFF) << 5 // Insert imm16

	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeLsl(inst *Instruction) ([]byte, error) {
	// LSL Xd, Xn, #shift (immediate) or LSL Xd, Xn, Xm (register)
	// Immediate form is encoded as UBFM Xd, Xn, #(-shift MOD 64), #(63-shift)
	// Register form is encoded as LSLV Xd, Xn, Xm

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: lsl requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: lsl destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: lsl operand 1: %w", inst.Line, inst.Column, err)
	}

	if inst.Operands[2].Type == OperandImmediate {
		// Immediate form: LSL Xd, Xn, #shift
		// Encoded as UBFM Xd, Xn, #(-shift MOD 64), #(63-shift)
		shiftVal, err := ParseInt(inst.Operands[2].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: lsl shift amount: %w", inst.Line, inst.Column, err)
		}
		shift := uint32(shiftVal)

		if shift > 63 {
			return nil, fmt.Errorf("line %d:%d: lsl shift amount must be 0-63, got %d",
				inst.Line, inst.Column, shift)
		}

		// UBFM encoding for 64-bit: sf=1, opc=10, N=1
		// immr = (-shift) mod 64 = (64 - shift) & 0x3F
		// imms = 63 - shift
		immr := (64 - shift) & 0x3F
		imms := 63 - shift

		// UBFM: 1 1 0 100110 1 immr imms Rn Rd = 0xD3400000
		encoding := uint32(0xD3400000) | (immr << 16) | (imms << 10) | (uint32(rn) << 5) | uint32(rd)
		return EncodeLittleEndian(encoding), nil
	}

	// Register form: LSL Xd, Xn, Xm (LSLV)
	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: lsl operand 2: %w", inst.Line, inst.Column, err)
	}

	// LSLV: sf=1, 0 0 11010110 Rm 0010 00 Rn Rd = 0x9AC02000
	encoding := uint32(0x9AC02000) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeLsr(inst *Instruction) ([]byte, error) {
	// LSR Xd, Xn, #shift (immediate) or LSR Xd, Xn, Xm (register)
	// Immediate form is encoded as UBFM Xd, Xn, #shift, #63
	// Register form is encoded as LSRV Xd, Xn, Xm

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: lsr requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: lsr destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: lsr operand 1: %w", inst.Line, inst.Column, err)
	}

	if inst.Operands[2].Type == OperandImmediate {
		// Immediate form: LSR Xd, Xn, #shift
		// Encoded as UBFM Xd, Xn, #shift, #63
		shiftVal, err := ParseInt(inst.Operands[2].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: lsr shift amount: %w", inst.Line, inst.Column, err)
		}
		shift := uint32(shiftVal)

		if shift > 63 {
			return nil, fmt.Errorf("line %d:%d: lsr shift amount must be 0-63, got %d",
				inst.Line, inst.Column, shift)
		}

		// UBFM encoding for 64-bit: sf=1, opc=10, N=1
		// immr = shift, imms = 63
		immr := shift
		imms := uint32(63)

		// UBFM: 1 1 0 100110 1 immr imms Rn Rd = 0xD3400000
		encoding := uint32(0xD3400000) | (immr << 16) | (imms << 10) | (uint32(rn) << 5) | uint32(rd)
		return EncodeLittleEndian(encoding), nil
	}

	// Register form: LSR Xd, Xn, Xm (LSRV)
	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: lsr operand 2: %w", inst.Line, inst.Column, err)
	}

	// LSRV: sf=1, 0 0 11010110 Rm 0010 01 Rn Rd = 0x9AC02400
	encoding := uint32(0x9AC02400) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeAsr(inst *Instruction) ([]byte, error) {
	// ASR Xd, Xn, #shift (immediate) or ASR Xd, Xn, Xm (register)
	// Immediate form is encoded as SBFM Xd, Xn, #shift, #63
	// Register form is encoded as ASRV Xd, Xn, Xm

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: asr requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: asr destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: asr operand 1: %w", inst.Line, inst.Column, err)
	}

	if inst.Operands[2].Type == OperandImmediate {
		// Immediate form: ASR Xd, Xn, #shift
		// Encoded as SBFM Xd, Xn, #shift, #63
		shiftVal, err := ParseInt(inst.Operands[2].Value)
		if err != nil {
			return nil, fmt.Errorf("line %d:%d: asr shift amount: %w", inst.Line, inst.Column, err)
		}
		shift := uint32(shiftVal)

		if shift > 63 {
			return nil, fmt.Errorf("line %d:%d: asr shift amount must be 0-63, got %d",
				inst.Line, inst.Column, shift)
		}

		// SBFM encoding for 64-bit: sf=1, opc=00, N=1
		// immr = shift, imms = 63
		immr := shift
		imms := uint32(63)

		// SBFM: 1 0 0 100110 1 immr imms Rn Rd = 0x9340FC00
		encoding := uint32(0x93400000) | (immr << 16) | (imms << 10) | (uint32(rn) << 5) | uint32(rd)
		return EncodeLittleEndian(encoding), nil
	}

	// Register form: ASR Xd, Xn, Xm (ASRV)
	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: asr operand 2: %w", inst.Line, inst.Column, err)
	}

	// ASRV: sf=1, 0 0 11010110 Rm 0010 10 Rn Rd = 0x9AC02800
	encoding := uint32(0x9AC02800) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeAnd(inst *Instruction) ([]byte, error) {
	// AND Xd, Xn, Xm - Bitwise AND (register)
	// Encoding: sf 00 01010 shift N Rm imm6 Rn Rd
	// sf=1 for 64-bit, opc=00 (AND), shift=00 (LSL), N=0, imm6=0
	// Base: 0x8A000000

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: and requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: and destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: and operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: and operand 2: %w", inst.Line, inst.Column, err)
	}

	// AND (shifted register): sf 00 01010 shift N Rm imm6 Rn Rd
	// 0x8A000000 = 10001010 00000000 00000000 00000000
	encoding := uint32(0x8A000000) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeOrr(inst *Instruction) ([]byte, error) {
	// ORR Xd, Xn, Xm - Bitwise OR (register)
	// Encoding: sf 01 01010 shift N Rm imm6 Rn Rd
	// sf=1, opc=01 (ORR)
	// Base: 0xAA000000

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: orr requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: orr destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: orr operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: orr operand 2: %w", inst.Line, inst.Column, err)
	}

	// ORR (shifted register): sf 01 01010 shift N Rm imm6 Rn Rd
	// 0xAA000000 = 10101010 00000000 00000000 00000000
	encoding := uint32(0xAA000000) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeEor(inst *Instruction) ([]byte, error) {
	// EOR Xd, Xn, Xm - Bitwise exclusive OR (register)
	// Encoding: sf 10 01010 shift N Rm imm6 Rn Rd
	// sf=1, opc=10 (EOR)
	// Base: 0xCA000000

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: eor requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: eor destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: eor operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: eor operand 2: %w", inst.Line, inst.Column, err)
	}

	// EOR (shifted register): sf 10 01010 shift N Rm imm6 Rn Rd
	// 0xCA000000 = 11001010 00000000 00000000 00000000
	encoding := uint32(0xCA000000) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeMvn(inst *Instruction) ([]byte, error) {
	// MVN Xd, Xm - Bitwise NOT (alias for ORN Xd, XZR, Xm)
	// Encoding: sf 01 01010 shift 1 Rm imm6 Rn Rd (with Rn=XZR)
	// Base: 0xAA200000 (ORN with N=1)

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: mvn requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: mvn destination: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: mvn source: %w", inst.Line, inst.Column, err)
	}

	// ORN (shifted register) with Rn=XZR: sf 01 01010 shift 1 Rm imm6 11111 Rd
	// 0xAA2003E0 = base with N=1 and Rn=31 (XZR)
	rn := uint32(31) // XZR
	encoding := uint32(0xAA200000) | (uint32(rm) << 16) | (rn << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeAnds(inst *Instruction) ([]byte, error) {
	// ANDS Xd, Xn, Xm - Bitwise AND with flags (register)
	// Encoding: sf 11 01010 shift N Rm imm6 Rn Rd
	// sf=1, opc=11 (ANDS)
	// Base: 0xEA000000

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: ands requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ands destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ands operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: ands operand 2: %w", inst.Line, inst.Column, err)
	}

	// ANDS (shifted register): sf 11 01010 shift N Rm imm6 Rn Rd
	// 0xEA000000 = 11101010 00000000 00000000 00000000
	encoding := uint32(0xEA000000) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeTst(inst *Instruction) ([]byte, error) {
	// TST Xn, Xm - Test bits (alias for ANDS XZR, Xn, Xm)
	// Encoding: sf 11 01010 shift N Rm imm6 Rn 11111 (Rd=XZR)
	// Base: 0xEA00001F

	if len(inst.Operands) != 2 {
		return nil, fmt.Errorf("line %d:%d: tst requires 2 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rn, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: tst operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: tst operand 2: %w", inst.Line, inst.Column, err)
	}

	// ANDS with Rd=XZR (31): sf 11 01010 shift N Rm imm6 Rn 11111
	rd := uint32(31) // XZR
	encoding := uint32(0xEA000000) | (uint32(rm) << 16) | (uint32(rn) << 5) | rd
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeBic(inst *Instruction) ([]byte, error) {
	// BIC Xd, Xn, Xm - Bitwise bit clear (AND NOT)
	// Encoding: sf 00 01010 shift 1 Rm imm6 Rn Rd (N=1 for NOT)
	// Base: 0x8A200000

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: bic requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: bic destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: bic operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: bic operand 2: %w", inst.Line, inst.Column, err)
	}

	// BIC (shifted register): sf 00 01010 shift 1 Rm imm6 Rn Rd
	// 0x8A200000 = AND with N=1 (inverted Rm)
	encoding := uint32(0x8A200000) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeOrn(inst *Instruction) ([]byte, error) {
	// ORN Xd, Xn, Xm - Bitwise OR NOT
	// Encoding: sf 01 01010 shift 1 Rm imm6 Rn Rd (N=1 for NOT)
	// Base: 0xAA200000

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: orn requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: orn destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: orn operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: orn operand 2: %w", inst.Line, inst.Column, err)
	}

	// ORN (shifted register): sf 01 01010 shift 1 Rm imm6 Rn Rd
	// 0xAA200000 = ORR with N=1 (inverted Rm)
	encoding := uint32(0xAA200000) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
	return EncodeLittleEndian(encoding), nil
}

func (e *Encoder) encodeEon(inst *Instruction) ([]byte, error) {
	// EON Xd, Xn, Xm - Bitwise exclusive OR NOT
	// Encoding: sf 10 01010 shift 1 Rm imm6 Rn Rd (N=1 for NOT)
	// Base: 0xCA200000

	if len(inst.Operands) != 3 {
		return nil, fmt.Errorf("line %d:%d: eon requires 3 operands, got %d",
			inst.Line, inst.Column, len(inst.Operands))
	}

	rd, err := ParseRegister(inst.Operands[0].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: eon destination: %w", inst.Line, inst.Column, err)
	}

	rn, err := ParseRegister(inst.Operands[1].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: eon operand 1: %w", inst.Line, inst.Column, err)
	}

	rm, err := ParseRegister(inst.Operands[2].Value)
	if err != nil {
		return nil, fmt.Errorf("line %d:%d: eon operand 2: %w", inst.Line, inst.Column, err)
	}

	// EON (shifted register): sf 10 01010 shift 1 Rm imm6 Rn Rd
	// 0xCA200000 = EOR with N=1 (inverted Rm)
	encoding := uint32(0xCA200000) | (uint32(rm) << 16) | (uint32(rn) << 5) | uint32(rd)
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

// encodeByteValues encodes comma-separated integer values or labels to bytes
func (e *Encoder) encodeByteValues(value string, size int) ([]byte, error) {
	bytes, _, err := e.encodeByteValuesWithRelocations(value, size, 0)
	return bytes, err
}

// EncodeDataWithRelocations encodes a data declaration and returns relocations for label references
func (e *Encoder) EncodeDataWithRelocations(data *DataDeclaration, baseOffset uint64) ([]byte, []DataRelocation, error) {
	switch data.Type {
	case "byte":
		return e.encodeByteValuesWithRelocations(data.Value, 1, baseOffset)
	case "2byte", "hword":
		return e.encodeByteValuesWithRelocations(data.Value, 2, baseOffset)
	case "4byte", "word":
		return e.encodeByteValuesWithRelocations(data.Value, 4, baseOffset)
	case "8byte", "quad":
		return e.encodeByteValuesWithRelocations(data.Value, 8, baseOffset)
	case "space", "zero":
		size, err := ParseInt(data.Value)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid size for .%s: %w", data.Type, err)
		}
		return make([]byte, size), nil, nil
	case "asciz", "string":
		s := unescapeDataString(data.Value)
		result := make([]byte, len(s)+1)
		copy(result, s)
		result[len(s)] = 0
		return result, nil, nil
	case "ascii":
		s := unescapeDataString(data.Value)
		return []byte(s), nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported data directive: .%s", data.Type)
	}
}

// encodeByteValuesWithRelocations encodes comma-separated integer values or labels to bytes
// and returns relocation info for any label references
func (e *Encoder) encodeByteValuesWithRelocations(value string, size int, baseOffset uint64) ([]byte, []DataRelocation, error) {
	if value == "" {
		return nil, nil, nil
	}

	parts := strings.Split(value, ",")
	result := make([]byte, 0, len(parts)*size)
	var relocations []DataRelocation

	for _, part := range parts {
		part = strings.TrimSpace(part)
		currentOffset := baseOffset + uint64(len(result))

		// First try to parse as integer
		val, err := ParseInt64(part)
		isLabel := false
		if err != nil {
			// Not an integer, try as a label reference
			if e.symbolTable != nil {
				symbol, found := e.symbolTable.Lookup(part)
				if found {
					val = int64(symbol.Address)
					isLabel = true
				} else {
					return nil, nil, fmt.Errorf("invalid value '%s': not a number or known label", part)
				}
			} else {
				return nil, nil, fmt.Errorf("invalid value '%s': %w", part, err)
			}
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

		// If this was a label reference, record a relocation
		if isLabel && size >= 4 {
			relocations = append(relocations, DataRelocation{
				Offset:     currentOffset,
				Size:       size,
				TargetAddr: uint64(val),
			})
		}
	}

	return result, relocations, nil
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
