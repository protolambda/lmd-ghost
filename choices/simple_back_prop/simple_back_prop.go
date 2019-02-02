package simple_back_prop

import "lmd-ghost/sim"

/// A simple take on using a DAG for the fork-choice.
/// Stores entries in DAG, but back-propagates target votes every time the head is computed.
type SimpleBackPropLMDGhost struct {

	chain *sim.SimChain

	maxKnownSlot uint32
}

func NewSimpleBackPropLMDGhost() sim.ForkChoice {
	return new(SimpleBackPropLMDGhost)
}

func (gh *SimpleBackPropLMDGhost) SetChain(chain *sim.SimChain) {
	gh.chain = chain
}

func (gh *SimpleBackPropLMDGhost) AttestIn(hash256 sim.Hash256) {
	// free, at cost of head-function.
}

func (gh *SimpleBackPropLMDGhost) BlockIn(block *sim.Block) {
	// almost free, we back-propagate all at once, when we need to.
	if block.Slot > gh.maxKnownSlot {
		gh.maxKnownSlot = block.Slot
	}
}

type ChildScore struct {
	ChildHash sim.Hash256
	ChildVotes uint32
}

func (gh *SimpleBackPropLMDGhost) HeadFn() sim.Hash256 {
	// Keep track of votes for each block, per height
	votesAtHeight := make([]map[sim.Hash256]uint32, gh.maxKnownSlot + 1)
	for i := uint32(0); i <= gh.maxKnownSlot; i++ {
		votesAtHeight[i] = make(map[sim.Hash256]uint32)
	}
	// put all initial votes in the "DAG" (or tree, if non-justified roots would be removed)
	for _, t := range gh.chain.Targets {
		targetBlock := gh.chain.Blocks[t]
		votesAtHeight[targetBlock.Slot][t] = votesAtHeight[targetBlock.Slot][t] + 1
	}
	bestChildMapping := make(map[sim.Hash256]ChildScore)
	// Now back-propagate, per slot height
	for i := gh.maxKnownSlot; i > 0; i-- {
		// Propagate all higher-slot votes back to the root of the tree,
		//  while keeping track of the most-voted child.
		for k, v := range votesAtHeight[i] {
			block := gh.chain.Blocks[k]
			// Propagate votes for child to parent
			votesAtHeight[i-1][block.ParentHash] = votesAtHeight[i-1][block.ParentHash] + v
			// keep track of the best child for this parent block
			mapping, initialized := bestChildMapping[block.ParentHash]
			if !initialized || v > mapping.ChildVotes {
				bestChildMapping[block.ParentHash] = ChildScore{ChildHash: k, ChildVotes: v}
			}
		}
	}
	// Now walk back from the root of the tree, picking the best child every step.
	best := gh.chain.Justified
	for {
		// Stop when we reach a leaf, the end of the tree
		block := gh.chain.Blocks[best]
		if len(block.Children) == 0 {
			break
		}
		if bestChild, hasBest := bestChildMapping[best]; hasBest {
			// Pick the best child of the current best block
			best = bestChild.ChildHash
		} else {
			// just pick the first child if none of the children has received any attestation (making none the best)
			best = block.Children[0].Hash
		}
	}

	return best
}
