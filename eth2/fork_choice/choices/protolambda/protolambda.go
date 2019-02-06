package protolambda

import (
	"lmd-ghost/eth2/dag"
	"lmd-ghost/eth2/fork_choice"
)

func addVote(n *dag.DagNode) {
	// increment the actual vote count
	n.Weight += 1
	//log.Printf("Added vote to %d, new count: %d\n", n.Block.Hash[0], n.Votes)

	// if we're not the best child of the parent, than we have a chance to become it.
	if n.Parent != nil && n.IndexAsChild != 0 {
		if oldBest := n.Parent.Children[0]; n.Weight > oldBest.Weight {
			// the best has been overthrown, now swap it out
			n.Parent.Children[0] = n
			n.Parent.Children[n.IndexAsChild] = oldBest
			oldBest.IndexAsChild = n.IndexAsChild
			n.IndexAsChild = 0

			// Update the target of the parent, it inherits it from this node
			n.Parent.BestTarget = n.BestTarget
		}
	}
}

func removeVote(n *dag.DagNode) {
	// decrement the actual vote count
	n.Weight -= 1
	//log.Printf("Removed vote from %d, new count: %d\n", n.Block.Hash[0], n.Votes)

	if n.Weight < 0 {
		panic("Warning: removed too many votes!")
	}

	// if we're the best child of the parent, than we have a chance to lose our position to the second best.
	if n.Parent != nil && n.IndexAsChild == 0 && len(n.Parent.Children) > 1 {
		// TODO: with some sorting it may be faster to keep track of the 2nd best child
		newBest := n
		for i := 1; i < len(n.Parent.Children); i++ {
			c := n.Parent.Children[i]
			if c.Weight > newBest.Weight {
				newBest = c
			}
		}
		// check if we actually had to update the new-best
		if newBest != n {
			// we have been overthrown, now swap in the new best
			n.Parent.Children[0] = newBest
			n.Parent.Children[newBest.IndexAsChild] = n
			n.IndexAsChild = newBest.IndexAsChild
			newBest.IndexAsChild = 0

			// Update the target of the parent, it inherits it from the new-best
			n.Parent.BestTarget = newBest.BestTarget
		}
	}
}

func PropagateBestTargetUp(n *dag.DagNode) {
	// propagate the new best-target up, as far as necessary
	p := n.Parent
	c := n
	for p != nil {
		if c.IndexAsChild == 0 {
			p.BestTarget = n.BestTarget
			c = p
			p = p.Parent
		} else {
			// stop propagating when the child is not the best child of the parent
			break
		}
	}
}

/// A simple take on using a DAG for the fork-choice.
/// Stores entries in DAG, but re-propagates target votes every time the head is computed.
type ProtolambdaLMDGhost struct {

	dag *dag.BeaconDag

}

func NewProtolambdaLMDGhost() fork_choice.ForkChoice {
	return new(ProtolambdaLMDGhost)
}

func (gh *ProtolambdaLMDGhost) SetDag(dag *dag.BeaconDag) {
	gh.dag = dag
}

func (gh *ProtolambdaLMDGhost) ApplyScoreChanges(changes []fork_choice.ScoreChange) {

}

//func (gh *ProtolambdaLMDGhost) AttestIn(blockHash sim.Hash256, attester sim.ValidatorID) {
//
//	// remove previous attest by validator, if there is any
//	prevTarget, hasPrev := gh.chain.Targets[attester]
//	//log.Println("=======================")
//
//	//for k, n := range gh.nodes {
//	//	log.Printf("%d: %d\n", k[0], n.Votes)
//	//}
//	//log.Printf("Attesation: %d prev %d, by %d\n", blockHash[0], prevTarget[0], attester)
//	if prevTarget == blockHash {
//		// nothing to do, attest does not change vote
//		return
//	}
//
//	if hasPrev {
//		// there is a previous attestation, that becomes invalid,
//		// but it may share a path to the root, which could stay as-is regarding vote count,
//		//  and possibly also regarding the bestTarget
//		prevBranch := gh.nodes[prevTarget]
//		newBranch := gh.nodes[blockHash]
//		for {
//			if prevBranch.Parent == newBranch.Parent {
//				sharedParent := prevBranch.Parent
//				// Check if prev-branch and new-branch are unconnected due to lack of history
//				//  (edge case, if you prune not everything)
//				if sharedParent == nil {
//					// nothing to propagate up, we're done
//					break
//				}
//
//				oldBestTarget := sharedParent.BestTarget
//				prevBranch.RemoveVote()
//				newBranch.AddVote()
//				// if the target changed, we have to propagate it,
//				// but the vote count will stay the same (Sum of branches)
//				if sharedParent.BestTarget != oldBestTarget {
//					sharedParent.PropagateBestTargetUp()
//				}
//				break
//			} else if prevBranch.Block.Slot > newBranch.Block.Slot {
//				// Depending on the heights of the branches, we walk down.
//				// Walk down the prev-branch
//				prevBranch.RemoveVote()
//				prevBranch = prevBranch.Parent
//			} else {
//				// Walk down the new-branch
//				newBranch.AddVote()
//				newBranch = newBranch.Parent
//			}
//		}
//	} else {
//		// just add the vote, and propagate back all the way, nothing else to do.
//		n := gh.nodes[blockHash]
//		for n != nil {
//			n.AddVote()
//			n = n.Parent
//		}
//	}
//}

func (gh *ProtolambdaLMDGhost) OnNewNode(node *dag.DagNode) {
	// best end-target is the block itself
	node.BestTarget = node
	if node.Parent != nil && len(node.Parent.Children) == 1 {
		// If this is the only/first node that is added,
		//  then it does not need attestations, it will just be the new target.
		PropagateBestTargetUp(node)
	}
}

func (gh *ProtolambdaLMDGhost) OnStartChange(newStart *dag.DagNode) {
	// nothing to do when the start changes
}

func (gh *ProtolambdaLMDGhost) HeadFn() *dag.DagNode {
	// All the work has already been done, just pick the best-target of the root node.
	// *Bonus*: And this works for *every* node in the graph!
	// Changing the root is costless
	// (If you prune away old nodes it still costs something, but this also needs to be done for other algos)
	return gh.dag.Start.BestTarget
}
