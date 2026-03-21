package slasm

// ARM64 instruction base encodings.
// These represent the fixed-bit patterns for ARM64 instructions.
// Operand fields (Rd, Rn, Rm, imm, etc.) are OR'd into these bases at encoding time.
//
// Reference: ARM Architecture Reference Manual for A-profile architecture (DDI 0487)

const (
	// Special instructions
	ARM64_NOP uint32 = 0xD503201F // NOP (no operation)
	ARM64_RET uint32 = 0xD65F03C0 // RET (return, alias for BR X30)
	ARM64_SVC uint32 = 0xD4000001 // SVC #0 (supervisor call)

	// Branch instructions
	ARM64_B      uint32 = 0x14000000 // B label (unconditional branch, imm26)
	ARM64_BL     uint32 = 0x94000000 // BL label (branch with link, imm26)
	ARM64_BR     uint32 = 0xD61F0000 // BR Xn (branch to register)
	ARM64_CBZ_X  uint32 = 0xB4000000 // CBZ Xt, label (64-bit compare and branch if zero)
	ARM64_CBNZ_X uint32 = 0xB5000000 // CBNZ Xt, label (64-bit compare and branch if not zero)

	// Data processing - immediate (with flags)
	ARM64_ADDS_IMM uint32 = 0xB1000000 // ADDS Xd, Xn, #imm12 (64-bit, set flags)
	ARM64_SUB_IMM  uint32 = 0xD1000000 // SUB Xd, Xn, #imm12 (64-bit, no flags)
	ARM64_SUBS_IMM uint32 = 0xF1000000 // SUBS Xd, Xn, #imm12 (64-bit, set flags)

	// Data processing - register
	ARM64_ADDS_REG uint32 = 0xAB000000 // ADDS Xd, Xn, Xm (64-bit, set flags)
	ARM64_SUBS_REG uint32 = 0xEB000000 // SUBS Xd, Xn, Xm (64-bit, set flags)
	ARM64_SMULH    uint32 = 0x9B407C00 // SMULH Xd, Xn, Xm (signed multiply high)
	ARM64_UMULH    uint32 = 0x9BC07C00 // UMULH Xd, Xn, Xm (unsigned multiply high)

	// Data processing - variable shift
	ARM64_LSLV_X uint32 = 0x9AC02000 // LSLV Xd, Xn, Xm (logical shift left variable, 64-bit)
	ARM64_LSRV_X uint32 = 0x9AC02400 // LSRV Xd, Xn, Xm (logical shift right variable, 64-bit)
	ARM64_ASRV_X uint32 = 0x9AC02800 // ASRV Xd, Xn, Xm (arithmetic shift right variable, 64-bit)

	// Bitfield operations
	ARM64_UBFM_X uint32 = 0xD3400000 // UBFM Xd, Xn, #immr, #imms (unsigned bitfield move, 64-bit)
	ARM64_SBFM_X uint32 = 0x93400000 // SBFM Xd, Xn, #immr, #imms (signed bitfield move, 64-bit)

	// Logical - register
	ARM64_AND_REG  uint32 = 0x8A000000 // AND Xd, Xn, Xm
	ARM64_ORR_REG  uint32 = 0xAA000000 // ORR Xd, Xn, Xm
	ARM64_EOR_REG  uint32 = 0xCA000000 // EOR Xd, Xn, Xm
	ARM64_ANDS_REG uint32 = 0xEA000000 // ANDS Xd, Xn, Xm (set flags)
	ARM64_BIC_REG  uint32 = 0x8A200000 // BIC Xd, Xn, Xm (bit clear = AND NOT)
	ARM64_ORN_REG  uint32 = 0xAA200000 // ORN Xd, Xn, Xm (OR NOT, also used for MVN)
	ARM64_EON_REG  uint32 = 0xCA200000 // EON Xd, Xn, Xm (EOR NOT)

	// Load/Store 64-bit (Xt)
	ARM64_LDR_POST uint32 = 0xF8400400 // LDR Xt, [Xn], #imm9 (post-index)
	ARM64_LDR_PRE  uint32 = 0xF8400C00 // LDR Xt, [Xn, #imm9]! (pre-index)
	ARM64_LDUR     uint32 = 0xF8400000 // LDUR Xt, [Xn, #imm9] (unscaled offset)
	ARM64_LDR_UOFF uint32 = 0xF9400000 // LDR Xt, [Xn, #imm12] (unsigned offset, scaled by 8)
	ARM64_STR_POST uint32 = 0xF8000400 // STR Xt, [Xn], #imm9 (post-index)
	ARM64_STR_PRE  uint32 = 0xF8000C00 // STR Xt, [Xn, #imm9]! (pre-index)
	ARM64_STUR     uint32 = 0xF8000000 // STUR Xt, [Xn, #imm9] (unscaled offset)
	ARM64_STR_UOFF uint32 = 0xF9000000 // STR Xt, [Xn, #imm12] (unsigned offset, scaled by 8)

	// Load/Store pair 64-bit (Xt1, Xt2)
	ARM64_LDP_POST uint32 = 0xA8C00000 // LDP Xt1, Xt2, [Xn], #imm7 (post-index)
	ARM64_LDP_PRE  uint32 = 0xA9C00000 // LDP Xt1, Xt2, [Xn, #imm7]! (pre-index)
	ARM64_LDP_SOFF uint32 = 0xA9400000 // LDP Xt1, Xt2, [Xn, #imm7] (signed offset)
	ARM64_STP_POST uint32 = 0xA8800000 // STP Xt1, Xt2, [Xn], #imm7 (post-index)
	ARM64_STP_PRE  uint32 = 0xA9800000 // STP Xt1, Xt2, [Xn, #imm7]! (pre-index)
	ARM64_STP_SOFF uint32 = 0xA9000000 // STP Xt1, Xt2, [Xn, #imm7] (signed offset)

	// Load/Store byte (8-bit, Wt)
	ARM64_LDRB_UOFF uint32 = 0x39400000 // LDRB Wt, [Xn, #imm12] (unsigned offset)
	ARM64_STRB_UOFF uint32 = 0x39000000 // STRB Wt, [Xn, #imm12] (unsigned offset)

	// Load/Store halfword (16-bit, Wt)
	ARM64_LDRH_UOFF uint32 = 0x79400000 // LDRH Wt, [Xn, #imm12] (unsigned offset, scaled by 2)
	ARM64_STRH_UOFF uint32 = 0x79000000 // STRH Wt, [Xn, #imm12] (unsigned offset, scaled by 2)
)

// Branch encoding masks
const (
	ARM64_IMM26_MASK     uint32 = 0x03FFFFFF // 26-bit immediate mask for B/BL instructions
	ARM64_BRANCH_OP_MASK uint32 = 0xFC000000 // Branch opcode mask (upper 6 bits)
)

// Mach-O section numbers (1-based, as stored in nlist64.n_sect)
const (
	MachOSectText uint8 = 1 // __text section number
	MachOSectData uint8 = 2 // __data section number
)
