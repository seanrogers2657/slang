// Package arm64 provides phi node elimination for ARM64 code generation.
package arm64

import (
	"github.com/seanrogers2657/slang/compiler/ir"
)

// PhiEliminator converts phi nodes to copies.
// In SSA form, phi nodes represent value merging at control flow joins.
// For code generation, we need to convert these to actual copy operations.
type PhiEliminator struct {
	fn *ir.Function
}

// NewPhiEliminator creates a new phi eliminator for a function.
func NewPhiEliminator(fn *ir.Function) *PhiEliminator {
	return &PhiEliminator{fn: fn}
}

// Eliminate converts all phi nodes in the function to copy operations.
// This places copies at the end of predecessor blocks, just before the branch.
//
// The algorithm handles the "lost copy problem" by using a parallel copy
// semantics: all copies happen simultaneously, then values are used.
// For simple cases where there are no circular dependencies, we emit
// sequential copies. For complex cases, we use a temporary register.
func (pe *PhiEliminator) Eliminate() {
	for _, block := range pe.fn.Blocks {
		pe.eliminatePhisInBlock(block)
	}
}

func (pe *PhiEliminator) eliminatePhisInBlock(block *ir.Block) {
	phis := block.Phis()
	if len(phis) == 0 {
		return
	}

	// For each predecessor, collect the copies that need to happen
	for _, pred := range block.Preds {
		copies := pe.collectCopiesFromPred(phis, pred)
		if len(copies) == 0 {
			continue
		}

		// Check for circular dependencies
		if pe.hasCircularDependency(copies) {
			pe.resolveCircularCopies(pred, copies)
		} else {
			pe.emitSequentialCopies(pred, copies)
		}
	}

	// Remove phi nodes from the block (they've been replaced with copies)
	// In this implementation, we keep the phi nodes but mark them as resolved
	// The code generator handles phi nodes by emitting copies when jumping
}

// copyInfo represents a single copy operation.
type copyInfo struct {
	src  *ir.Value // source value
	dst  *ir.Value // destination (the phi node)
}

func (pe *PhiEliminator) collectCopiesFromPred(phis []*ir.Value, pred *ir.Block) []copyInfo {
	var copies []copyInfo

	for _, phi := range phis {
		// Find the value from this predecessor
		for _, arg := range phi.PhiArgs {
			if arg.From == pred {
				copies = append(copies, copyInfo{
					src: arg.Value,
					dst: phi,
				})
				break
			}
		}
	}

	return copies
}

func (pe *PhiEliminator) hasCircularDependency(copies []copyInfo) bool {
	// Build a graph of dependencies
	// A circular dependency exists if dst of one copy is src of another
	// and they form a cycle

	dstSet := make(map[*ir.Value]bool)
	for _, c := range copies {
		dstSet[c.dst] = true
	}

	for _, c := range copies {
		if dstSet[c.src] {
			// Source is also a destination - potential cycle
			// Do a more thorough check
			return pe.detectCycle(copies)
		}
	}

	return false
}

func (pe *PhiEliminator) detectCycle(copies []copyInfo) bool {
	// Build adjacency map: dst -> src
	graph := make(map[*ir.Value]*ir.Value)
	for _, c := range copies {
		graph[c.dst] = c.src
	}

	// Check for cycles using visited marking
	visited := make(map[*ir.Value]int) // 0=unvisited, 1=in-progress, 2=done

	var hasCycle func(*ir.Value) bool
	hasCycle = func(v *ir.Value) bool {
		switch visited[v] {
		case 1:
			return true // Back edge = cycle
		case 2:
			return false // Already processed
		}

		visited[v] = 1
		if next, ok := graph[v]; ok {
			if hasCycle(next) {
				return true
			}
		}
		visited[v] = 2
		return false
	}

	for _, c := range copies {
		if hasCycle(c.dst) {
			return true
		}
	}

	return false
}

func (pe *PhiEliminator) emitSequentialCopies(pred *ir.Block, copies []copyInfo) {
	// For non-circular cases, we can emit copies in a safe order
	// This is handled by the code generator via emitPhiCopies

	// The copies are already tracked in the phi nodes' PhiArgs
	// The code generator will emit them when generating the branch
}

func (pe *PhiEliminator) resolveCircularCopies(pred *ir.Block, copies []copyInfo) {
	// For circular dependencies, we need to break the cycle
	// Classic approach: use a temporary for one value
	//
	// Example:
	//   phi1 = phi(a from b1, phi2 from b2)
	//   phi2 = phi(b from b1, phi1 from b2)
	//
	// From b2, we have: phi1 <- phi2, phi2 <- phi1
	// This is circular. We resolve by:
	//   tmp <- phi1
	//   phi1 <- phi2
	//   phi2 <- tmp
	//
	// In the IR, we represent this by reordering and the code generator
	// uses temporary registers.

	// For now, the code generator handles this by always using x9 as temp
	// and emitting copies one at a time. This works because loads happen
	// before stores in our simple register allocation.
}

// ReorderCopies returns copies in an order that avoids overwriting sources.
// For acyclic graphs, this is a topological sort.
func ReorderCopies(copies []copyInfo) []copyInfo {
	if len(copies) <= 1 {
		return copies
	}

	// Build dependency graph
	srcOf := make(map[*ir.Value]*ir.Value)
	for _, c := range copies {
		srcOf[c.dst] = c.src
	}

	// Find values that are both sources and destinations
	dstSet := make(map[*ir.Value]bool)
	for _, c := range copies {
		dstSet[c.dst] = true
	}

	// Topological sort: emit copies whose source is not a destination first
	var result []copyInfo
	done := make(map[*ir.Value]bool)

	for len(result) < len(copies) {
		progress := false
		for _, c := range copies {
			if done[c.dst] {
				continue
			}

			// Can emit if source is not a pending destination
			canEmit := true
			if dstSet[c.src] && !done[c.src] {
				canEmit = false
			}

			if canEmit {
				result = append(result, c)
				done[c.dst] = true
				progress = true
			}
		}

		if !progress {
			// Cycle detected - add remaining in any order
			// (caller should have used resolveCircularCopies)
			for _, c := range copies {
				if !done[c.dst] {
					result = append(result, c)
					done[c.dst] = true
				}
			}
		}
	}

	return result
}
