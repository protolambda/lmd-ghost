package spec

import (
	"lmd-ghost/sim"
)

/// The naive, but readable, spec implementation
type SpecLMDGhost struct {

	chain *sim.SimChain

}

func NewSpecLMDGhost() sim.ForkChoice {
	return new(SpecLMDGhost)
}

func (gh *SpecLMDGhost) SetChain(chain *sim.SimChain) {
	gh.chain = chain
}

func (gh *SpecLMDGhost) AttestIn(blockHash sim.Hash256, attester sim.ValidatorID) {
	// free, at cost of head-function.
}

func (gh *SpecLMDGhost) BlockIn(block *sim.Block) {
	// free, at cost of head-function
}

/// Retrieves the head by *recursively* looking for the highest voted block
//   at *every* block in the path from start to head.
func (gh *SpecLMDGhost) HeadFn() sim.Hash256 {
	// Minor difference:
	// Normally you would have to filter for the active validators, and get their targets.
	// We can just iterate over the values in the sim-chain.
	// This difference only really matters when there's many validators inactive,
	//  and the client implementation doesn't store them separately.

	head := gh.chain.Blocks[gh.chain.Justified]
	for {
		if len(head.Children) == 0 {
			return head.Hash
		}
		bestItem := head.Children[0]
		var bestScore uint32 = 0
		for _, child := range head.Children {
			childVotes := gh.getVoteCount(child)
			if childVotes > bestScore {
				bestScore = childVotes
				bestItem = child
			}
		}
		head = bestItem
	}
}

func (gh *SpecLMDGhost) getVoteCount(block *sim.Block) uint32 {
	count := uint32(0)
	for _, target := range gh.chain.Targets {
		if anc := gh.getAncestor(gh.chain.Blocks[target], block.Slot); anc != nil && anc.Hash == block.Hash {
			count++
		}
	}
	return count
}

/// Gets the ancestor of `block` at `slot`
func (gh *SpecLMDGhost) getAncestor(block *sim.Block, slot uint32) *sim.Block {
	if block.Slot == slot {
		return block
	} else if block.Slot < slot {
		return nil
	} else {
		return gh.getAncestor(gh.chain.Blocks[block.ParentHash], slot)
	}
}
