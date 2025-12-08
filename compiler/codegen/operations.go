package codegen

// IntOperation generates assembly for an integer binary operation using the default emitter.
// The operands should already be in the left/right registers.
// Results are stored in the result register.
func IntOperation(op string, signed bool) (string, error) {
	return defaultEmitter.EmitIntOp(op, signed)
}

// FloatOperation generates assembly for a floating-point binary operation using the default emitter.
// The operands should already be in the float left/right registers.
// Results are stored in the float result register.
func FloatOperation(op string) (string, error) {
	return defaultEmitter.EmitFloatOp(op)
}
