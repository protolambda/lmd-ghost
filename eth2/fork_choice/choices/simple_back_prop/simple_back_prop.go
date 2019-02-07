package simple_back_prop

import (
	"lmd-ghost/eth2/dag"
	"lmd-ghost/eth2/fork_choice"
)

/// A simple take on using a DAG for the fork-choice.
/// Stores entries in DAG, but back-propagates target votes every time the head is computed.
type SimpleBackPropLMDGhost struct {

	dag *dag.BeaconDag

	maxKnownSlot uint64

	latestScores map[*dag.DagNode]int64
}

func NewSimpleBackPropLMDGhost(d *dag.BeaconDag) fork_choice.ForkChoice {
	res := &SimpleBackPropLMDGhost{
		dag:          d,
		latestScores: make(map[*dag.DagNode]int64),
		maxKnownSlot: 0,
	}
	return res
}

func (gh *SimpleBackPropLMDGhost) ApplyScoreChanges(changes []fork_choice.ScoreChange) {
	for _, v := range changes {
		gh.latestScores[v.Target] += v.ScoreDelta
	}
	// delete targets that have a 0 score
	for k, v := range gh.latestScores {
		if v == 0 {
			// deletion during map iteration, safe in Go
			delete(gh.latestScores, k)
		}
	}
}

func (gh *SimpleBackPropLMDGhost) OnNewNode(block *dag.DagNode) {
	// almost free, we back-propagate all at once, when we need to.
	if block.Slot > gh.maxKnownSlot {
		gh.maxKnownSlot = block.Slot
	}
}

func (gh *SimpleBackPropLMDGhost) OnStartChange(newStart *dag.DagNode) {
	// nothing to do when the start changes
}

type ChildScore struct {
	BestTarget *dag.DagNode
	ChildScore int64
}

func (gh *SimpleBackPropLMDGhost) HeadFn() *dag.DagNode {
	// Keep track of weight for each block, per height
	weightedBlocksAtHeight := make([]map[*dag.DagNode]int64, gh.maxKnownSlot + 1)
	for i := uint64(0); i <= gh.maxKnownSlot; i++ {
		weightedBlocksAtHeight[i] = make(map[*dag.DagNode]int64)
	}
	// compute cutoff: sum all scores, and divide by 2.
	cutOff := int64(0)
	// put all initial weights in the "DAG" (or tree, if non-justified roots would be removed)
	for t, w := range gh.latestScores {
		weightedBlocksAtHeight[t.Slot][t] = weightedBlocksAtHeight[t.Slot][t] + w
		cutOff += w
	}
	cutOff /= 2
	bestChildMapping := make(map[*dag.DagNode]ChildScore)
	// Now back-propagate, per slot height
	for i := gh.maxKnownSlot; i > 0; i-- {
		// Propagate all higher-slot votes back to the root of the tree,
		//  while keeping track of the most-voted child.
		for block, w := range weightedBlocksAtHeight[i] {
			// check for cutOff, if the block weight is heavy enough, then we can just stop at this block, and use the bestChildMapping to get the final head.
			if w > cutOff {
				if myBest, hasBest := bestChildMapping[block]; hasBest {
					return myBest.BestTarget
				} else {
					return block
				}
			}
			// Propagate weight of child to parent
			weightedBlocksAtHeight[i-1][block.Parent] = weightedBlocksAtHeight[i-1][block.Parent] + w
			// keep track of the best child for this parent block
			mapping, initialized := bestChildMapping[block.Parent]
			if !initialized || w > mapping.ChildScore {
				if myBest, hasBest := bestChildMapping[block]; hasBest {
					// inherit the best-target if there is one
					bestChildMapping[block.Parent] = ChildScore{BestTarget: myBest.BestTarget, ChildScore: w}
				} else {
					// otherwise just put this node as the best target, if it has no entry in the bestChildMapping, then the node has no children
					bestChildMapping[block.Parent] = ChildScore{BestTarget: block, ChildScore: w}
				}
			}
		}
	}
	if myBest, hasBest := bestChildMapping[gh.dag.Start]; hasBest {
		return myBest.BestTarget
	} else {
		return gh.dag.Start
	}
}
