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

	LatestScores map[*dag.DagNode]int64
}

func NewSimpleBackPropLMDGhost() fork_choice.ForkChoice {
	return new(SimpleBackPropLMDGhost)
}

func (gh *SimpleBackPropLMDGhost) SetDag(dag *dag.BeaconDag) {
	gh.dag = dag
}

func (gh *SimpleBackPropLMDGhost) ApplyScoreChanges(changes []fork_choice.ScoreChange) {
	for _, v := range changes {
		gh.LatestScores[v.Target] += v.ScoreDelta
	}
	// delete targets that have a 0 score
	for k, v := range gh.LatestScores {
		if v == 0 {
			// deletion during map iteration, safe in Go
			delete(gh.LatestScores, k)
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
	Child *dag.DagNode
	ChildScore int64
}

func (gh *SimpleBackPropLMDGhost) HeadFn() *dag.DagNode {
	// Keep track of weight for each block, per height
	weightedBlocksAtHeight := make([]map[*dag.DagNode]int64, gh.maxKnownSlot + 1)
	for i := uint64(0); i <= gh.maxKnownSlot; i++ {
		weightedBlocksAtHeight[i] = make(map[*dag.DagNode]int64)
	}
	// put all initial weights in the "DAG" (or tree, if non-justified roots would be removed)
	for t, w := range gh.LatestScores {
		weightedBlocksAtHeight[t.Slot][t] = weightedBlocksAtHeight[t.Slot][t] + w
	}
	bestChildMapping := make(map[*dag.DagNode]ChildScore)
	// Now back-propagate, per slot height
	for i := gh.maxKnownSlot; i > 0; i-- {
		// Propagate all higher-slot votes back to the root of the tree,
		//  while keeping track of the most-voted child.
		for block, w := range weightedBlocksAtHeight[i] {
			// Propagate weight of child to parent
			weightedBlocksAtHeight[i-1][block.Parent] = weightedBlocksAtHeight[i-1][block.Parent] + w
			// keep track of the best child for this parent block
			mapping, initialized := bestChildMapping[block.Parent]
			if !initialized || w > mapping.ChildScore {
				bestChildMapping[block.Parent] = ChildScore{Child: block, ChildScore: w}
			}
		}
	}
	// Now walk back from the root of the tree, picking the best child every step.
	best := gh.dag.Start
	for {
		// Stop when we reach a leaf, the end of the tree
		if len(best.Children) == 0 {
			break
		}
		if bestChildData, hasBest := bestChildMapping[best]; hasBest {
			// Pick the best child of the current best block
			best = bestChildData.Child
		} else {
			// just pick the first child if none of the children has received any attestation (making none the best)
			best = best.Children[0]
		}
	}

	return best
}
