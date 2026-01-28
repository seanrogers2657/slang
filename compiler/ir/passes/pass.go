// Package passes provides the infrastructure for IR optimization passes.
package passes

import (
	"github.com/seanrogers2657/slang/compiler/ir"
)

// Pass is the interface that all optimization passes must implement.
type Pass interface {
	// Name returns the name of this pass.
	Name() string

	// Run applies the pass to a function and returns true if changes were made.
	Run(fn *ir.Function) bool
}

// ProgramPass is an interface for passes that operate on the entire program.
type ProgramPass interface {
	// Name returns the name of this pass.
	Name() string

	// Run applies the pass to the program and returns true if changes were made.
	Run(prog *ir.Program) bool
}

// Manager manages and runs optimization passes.
type Manager struct {
	passes []Pass
}

// NewManager creates a new pass manager.
func NewManager() *Manager {
	return &Manager{}
}

// Add adds a pass to the manager.
func (m *Manager) Add(pass Pass) {
	m.passes = append(m.passes, pass)
}

// Run runs all passes on a function.
// Returns true if any pass made changes.
func (m *Manager) Run(fn *ir.Function) bool {
	changed := false
	for _, pass := range m.passes {
		if pass.Run(fn) {
			changed = true
		}
	}
	return changed
}

// RunUntilFixed runs all passes repeatedly until no changes are made.
// This is useful for passes that may enable other passes.
func (m *Manager) RunUntilFixed(fn *ir.Function) {
	for {
		if !m.Run(fn) {
			break
		}
	}
}

// ProgramManager manages and runs program-level optimization passes.
type ProgramManager struct {
	passes []ProgramPass
}

// NewProgramManager creates a new program pass manager.
func NewProgramManager() *ProgramManager {
	return &ProgramManager{}
}

// Add adds a pass to the manager.
func (m *ProgramManager) Add(pass ProgramPass) {
	m.passes = append(m.passes, pass)
}

// Run runs all passes on a program.
// Returns true if any pass made changes.
func (m *ProgramManager) Run(prog *ir.Program) bool {
	changed := false
	for _, pass := range m.passes {
		if pass.Run(prog) {
			changed = true
		}
	}
	return changed
}

// AnalysisPass is an interface for analysis passes that compute information
// without modifying the IR.
type AnalysisPass interface {
	// Name returns the name of this analysis.
	Name() string

	// Analyze computes analysis results for a function.
	Analyze(fn *ir.Function)

	// Invalidate marks the analysis as needing recomputation.
	Invalidate()
}

// DominatorInfo holds dominance information for a function.
type DominatorInfo struct {
	fn *ir.Function

	// Idom maps each block to its immediate dominator.
	Idom map[*ir.Block]*ir.Block

	// DomTree maps each block to its children in the dominator tree.
	DomTree map[*ir.Block][]*ir.Block
}

// NewDominatorInfo creates dominator info for a function.
func NewDominatorInfo(fn *ir.Function) *DominatorInfo {
	info := &DominatorInfo{
		fn:      fn,
		Idom:    make(map[*ir.Block]*ir.Block),
		DomTree: make(map[*ir.Block][]*ir.Block),
	}
	info.compute()
	return info
}

// compute calculates dominance information using the simple algorithm.
// This could be replaced with Lengauer-Tarjan for better performance.
func (d *DominatorInfo) compute() {
	if len(d.fn.Blocks) == 0 {
		return
	}

	entry := d.fn.Entry()

	// Initialize: entry dominates only itself, others dominated by all
	doms := make(map[*ir.Block]map[*ir.Block]bool)
	allBlocks := make(map[*ir.Block]bool)
	for _, b := range d.fn.Blocks {
		allBlocks[b] = true
		doms[b] = make(map[*ir.Block]bool)
	}

	// Entry is dominated only by itself
	doms[entry][entry] = true

	// All other blocks start dominated by all blocks
	for _, b := range d.fn.Blocks {
		if b != entry {
			for _, bb := range d.fn.Blocks {
				doms[b][bb] = true
			}
		}
	}

	// Iterate until fixed point
	changed := true
	for changed {
		changed = false
		for _, b := range d.fn.Blocks {
			if b == entry {
				continue
			}

			// Dom(b) = {b} ∪ ∩{Dom(p) : p ∈ pred(b)}
			newDom := make(map[*ir.Block]bool)
			newDom[b] = true

			if len(b.Preds) > 0 {
				// Start with first predecessor's dominators
				for bb := range doms[b.Preds[0]] {
					newDom[bb] = true
				}
				// Intersect with other predecessors
				for i := 1; i < len(b.Preds); i++ {
					for bb := range newDom {
						if !doms[b.Preds[i]][bb] {
							delete(newDom, bb)
						}
					}
				}
			}

			// Check for change
			if len(newDom) != len(doms[b]) {
				doms[b] = newDom
				changed = true
			} else {
				for bb := range newDom {
					if !doms[b][bb] {
						doms[b] = newDom
						changed = true
						break
					}
				}
			}
		}
	}

	// Compute immediate dominators
	for _, b := range d.fn.Blocks {
		if b == entry {
			continue
		}

		// Idom(b) is the dominator of b that is dominated by all other dominators
		for dom := range doms[b] {
			if dom == b {
				continue
			}
			isImmediate := true
			for other := range doms[b] {
				if other == b || other == dom {
					continue
				}
				// If dom dominates other, dom is not the immediate dominator
				if doms[other][dom] {
					isImmediate = false
					break
				}
			}
			if isImmediate {
				d.Idom[b] = dom
				break
			}
		}
	}

	// Build dominator tree
	for b, idom := range d.Idom {
		d.DomTree[idom] = append(d.DomTree[idom], b)
	}
}

// Dominates returns true if a dominates b.
func (d *DominatorInfo) Dominates(a, b *ir.Block) bool {
	if a == b {
		return true
	}

	// Walk up the dominator tree from b
	for {
		idom := d.Idom[b]
		if idom == nil {
			return false
		}
		if idom == a {
			return true
		}
		b = idom
	}
}

// Children returns the children of a block in the dominator tree.
func (d *DominatorInfo) Children(b *ir.Block) []*ir.Block {
	return d.DomTree[b]
}
