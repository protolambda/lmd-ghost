package spec

import (
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/dag"
	"lmd-ghost/eth2/data/attestation"
	"lmd-ghost/eth2/fork_choice"
)

/// The naive, but readable, spec implementation
type SpecLMDGhost struct {

	dag *dag.BeaconDag

}

func NewSpecLMDGhost() fork_choice.ForkChoice {
	return new(SpecLMDGhost)
}

func (gh *SpecLMDGhost) SetDag(dag *dag.BeaconDag) {
	gh.dag = dag
}

func (gh *SpecLMDGhost) AttestationIn(attestation *attestation.Attestation) {
	// free, at cost of head-function.
}

func (gh *SpecLMDGhost) BlockIn(block *dag.DagNode) {
	// free, at cost of head-function
}

func (gh *SpecLMDGhost) StartIn(newStart *dag.DagNode) {
	// nothing to do when the start changes
}

/// Retrieves the head by *recursively* looking for the highest voted block
//   at *every* block in the path from start to head.
func (gh *SpecLMDGhost) HeadFn() *dag.DagNode {
	// Minor difference:
	// Normally you would have to filter for the active validators, and get their targets.
	// We can just iterate over the values in the common-chain.
	// This difference only really matters when there's many validators inactive,
	//  and the client implementation doesn't store them separately.

	head := gh.dag.Start
	for {
		if len(head.Children) == 0 {
			return head
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

func (gh *SpecLMDGhost) getVoteCount(block *common.Block) uint32 {
	count := uint32(0)
	for _, target := range gh.dag.LatestTargets {
		if anc := gh.getAncestor(gh.chain.Blocks[target], block.Slot); anc != nil && anc.Hash == block.Hash {
			count++
		}
	}
	return count
}

/// Gets the ancestor of `block` at `slot`
func (gh *SpecLMDGhost) getAncestor(block *dag.DagNode, slot uint64) *dag.DagNode {
	if block.Slot == slot {
		return block
	} else if block.Slot < slot {
		return nil
	} else {
		return gh.getAncestor(block.Parent, slot)
	}
}
