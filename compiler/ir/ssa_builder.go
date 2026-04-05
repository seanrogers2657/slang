package ir

// SSABuilder implements the "Simple and Efficient Construction of SSA Form" algorithm.
// It tracks variable definitions across blocks and handles phi node insertion.
//
// Reference: Braun et al., "Simple and Efficient Construction of Static Single
// Assignment Form", CC 2013.
type SSABuilder struct {
	// varDefs maps variable name -> block -> current definition
	varDefs map[string]map[*Block]*Value

	// incompletePhis tracks phi nodes that need operands filled in after sealing
	incompletePhis map[*Block]map[string]*Value

	// fn is the current function being built (needed for phi replacement)
	fn *Function
}

// NewSSABuilder creates a new SSA builder.
func NewSSABuilder() *SSABuilder {
	return &SSABuilder{
		varDefs:        make(map[string]map[*Block]*Value),
		incompletePhis: make(map[*Block]map[string]*Value),
	}
}

// Reset clears all SSA state for reuse with a new function.
func (s *SSABuilder) Reset() {
	s.varDefs = make(map[string]map[*Block]*Value)
	s.incompletePhis = make(map[*Block]map[string]*Value)
	s.fn = nil
}

// SetFunction sets the current function context.
func (s *SSABuilder) SetFunction(fn *Function) {
	s.fn = fn
}

// WriteVariable records a definition of a variable in a block.
func (s *SSABuilder) WriteVariable(name string, block *Block, val *Value) {
	if s.varDefs[name] == nil {
		s.varDefs[name] = make(map[*Block]*Value)
	}
	s.varDefs[name][block] = val
}

// ReadVariable returns the current definition of a variable in a block.
// If the variable is not defined locally, it searches predecessors and
// inserts phi nodes as needed.
func (s *SSABuilder) ReadVariable(name string, block *Block) *Value {
	// Check local definition first
	if defs, ok := s.varDefs[name]; ok {
		if val, ok := defs[block]; ok {
			return val
		}
	}
	// Recurse to predecessors
	return s.readVariableRecursive(name, block)
}

// IsVariableDefinedOnAllPaths checks if a variable is defined on all paths to the block.
// This is used to avoid creating invalid phi nodes for variables defined only in loops.
func (s *SSABuilder) IsVariableDefinedOnAllPaths(name string, block *Block) bool {
	return s.isDefinedOnAllPathsRecursive(name, block, make(map[*Block]bool))
}

func (s *SSABuilder) isDefinedOnAllPathsRecursive(name string, block *Block, visited map[*Block]bool) bool {
	if visited[block] {
		return true // Already visited (loop), assume OK
	}
	visited[block] = true

	// Check local definition
	if defs, ok := s.varDefs[name]; ok {
		if _, ok := defs[block]; ok {
			return true
		}
	}

	// Entry block with no definition
	if len(block.Preds) == 0 {
		return false
	}

	// Must be defined on all predecessors
	for _, pred := range block.Preds {
		if !s.isDefinedOnAllPathsRecursive(name, pred, visited) {
			return false
		}
	}
	return true
}

// readVariableRecursive implements the core SSA construction algorithm.
func (s *SSABuilder) readVariableRecursive(name string, block *Block) *Value {
	var val *Value

	if !block.Sealed {
		// Block not sealed - create incomplete phi placeholder
		val = s.createIncompletePhi(name, block)
	} else if len(block.Preds) == 0 {
		// Entry block with no definition - undefined variable
		return nil
	} else if len(block.Preds) == 1 {
		// Single predecessor - no phi needed
		val = s.ReadVariable(name, block.Preds[0])
	} else {
		// Multiple predecessors - need phi node
		phi := block.NewPhiValue(nil)
		s.WriteVariable(name, block, phi)
		val = s.addPhiOperands(name, phi)
	}

	s.WriteVariable(name, block, val)
	return val
}

// createIncompletePhi creates or returns an existing incomplete phi for a variable.
func (s *SSABuilder) createIncompletePhi(name string, block *Block) *Value {
	if s.incompletePhis[block] == nil {
		s.incompletePhis[block] = make(map[string]*Value)
	}
	if phi, ok := s.incompletePhis[block][name]; ok {
		return phi
	}
	phi := block.NewPhiValue(nil)
	s.incompletePhis[block][name] = phi
	return phi
}

// addPhiOperands adds operands to a phi node from all predecessors.
func (s *SSABuilder) addPhiOperands(name string, phi *Value) *Value {
	for _, pred := range phi.Block.Preds {
		predVal := s.ReadVariable(name, pred)
		if predVal != nil {
			phi.PhiArgs = append(phi.PhiArgs, &PhiArg{From: pred, Value: predVal})
		}
	}

	// Set phi type from first operand with a known type.
	// In nested loops, some operands might be incomplete phis with nil type,
	// so we search for any operand with a valid type.
	for _, arg := range phi.PhiArgs {
		if arg.Value.Type != nil {
			phi.Type = arg.Value.Type
			break
		}
	}

	return s.tryRemoveTrivialPhi(phi)
}

// tryRemoveTrivialPhi removes a phi if it has only one unique non-self value.
// After replacement, recursively checks phi users that may have become trivial.
func (s *SSABuilder) tryRemoveTrivialPhi(phi *Value) *Value {
	same := findUniquePhi(phi)
	if same == nil {
		return phi // No unique value or multiple values - keep phi
	}

	// Collect phi users before replacement (they may become trivial)
	var phiUsers []*Value
	for _, use := range phi.Uses {
		if use.Op == OpPhi && use != phi {
			phiUsers = append(phiUsers, use)
		}
	}
	if s.fn != nil {
		for _, block := range s.fn.Blocks {
			for _, v := range block.Values {
				if v.Op == OpPhi && v != phi {
					for _, pa := range v.PhiArgs {
						if pa.Value == phi {
							phiUsers = append(phiUsers, v)
							break
						}
					}
				}
			}
		}
	}

	// Replace all uses of phi with the unique value
	s.replacePhiUses(phi, same)

	// Update varDefs to point to replacement
	s.replaceInVarDefs(phi, same)

	// Remove from block
	phi.Block.RemoveValue(phi)

	// Recursively check phi users that may have become trivial
	for _, user := range phiUsers {
		if user.Block != nil { // still in a block (not already removed)
			s.tryRemoveTrivialPhi(user)
		}
	}

	return same
}

// findUniquePhi returns the single unique non-self value in a phi, or nil if none/multiple.
func findUniquePhi(phi *Value) *Value {
	var same *Value
	for _, arg := range phi.PhiArgs {
		if arg.Value == same || arg.Value == phi {
			continue
		}
		if same != nil {
			return nil // Multiple unique values
		}
		same = arg.Value
	}
	return same
}

// replacePhiUses replaces all uses of oldVal with newVal.
func (s *SSABuilder) replacePhiUses(oldVal, newVal *Value) {
	// Update Args references and transfer Uses to newVal
	for _, use := range oldVal.Uses {
		for i, arg := range use.Args {
			if arg == oldVal {
				use.Args[i] = newVal
				newVal.Uses = append(newVal.Uses, use)
			}
		}
	}

	// Update PhiArgs of other phi nodes that reference this phi
	if s.fn != nil {
		for _, block := range s.fn.Blocks {
			for _, v := range block.Values {
				if v.Op == OpPhi && v != oldVal {
					for _, phiArg := range v.PhiArgs {
						if phiArg.Value == oldVal {
							phiArg.Value = newVal
						}
					}
				}
			}
		}
	}
}

// replaceInVarDefs updates varDefs to replace oldVal with newVal.
func (s *SSABuilder) replaceInVarDefs(oldVal, newVal *Value) {
	for _, blockDefs := range s.varDefs {
		for block, val := range blockDefs {
			if val == oldVal {
				blockDefs[block] = newVal
			}
		}
	}
}

// SealBlock marks a block as having all predecessors known and fills incomplete phis.
func (s *SSABuilder) SealBlock(block *Block) {
	if block.Sealed {
		return
	}

	// Fill in incomplete phis
	if phis, ok := s.incompletePhis[block]; ok {
		for name, phi := range phis {
			if len(block.Preds) == 0 {
				// Entry block - remove useless phi
				block.RemoveValue(phi)
				continue
			}
			s.addPhiOperands(name, phi)
		}
		delete(s.incompletePhis, block)
	}

	block.Sealed = true
}

// HasDefinition returns true if a variable has any definition.
func (s *SSABuilder) HasDefinition(name string) bool {
	_, ok := s.varDefs[name]
	return ok
}

// DefinedVariables returns all variable names that have definitions.
func (s *SSABuilder) DefinedVariables() []string {
	names := make([]string, 0, len(s.varDefs))
	for name := range s.varDefs {
		names = append(names, name)
	}
	return names
}
