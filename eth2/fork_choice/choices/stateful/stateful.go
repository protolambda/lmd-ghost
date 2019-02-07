package stateful

import (
	"lmd-ghost/eth2/dag"
	"lmd-ghost/eth2/fork_choice"
)

func onAddWeight(n *dag.DagNode) {

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

func onRemoveWeight(n *dag.DagNode) {
	if n.Weight < 0 {
		panic("Error: removed too much weight! Weight of a node cannot be negative!")
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
		// check if we actually have to update the new-best
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

/// A simple take on using a DAG for the fork-choice.
/// Stores entries in DAG, and propagates target votes at the insertion time.
type StatefulLMDGhost struct {

	dag *dag.BeaconDag

}

func NewStatefulLMDGhost(d *dag.BeaconDag) fork_choice.ForkChoice {
	res := &StatefulLMDGhost{
		dag:          d,
	}
	return res
}

func (gh *StatefulLMDGhost) ApplyScoreChanges(changes []fork_choice.ScoreChange) {

	children := make([]*dag.DagNode, len(changes))
	// can be negative (i.e. attestation switch adjustment)
	lowestDelta := int64(1 << 63 - 1)
	nettoScoreChange := int64(0)
	for i, v := range changes {
		children[i] = v.Target
		if v.ScoreDelta < lowestDelta {
			lowestDelta = v.ScoreDelta
		}
		nettoScoreChange += v.ScoreDelta
	}
	// cutOff = (old total weight + netto change) / 2
	// cutOff2 = old total weight + netto change   (i.e., cutoff x 2)
	// weight of start point before changes = total old weight
	cutOff2 := gh.dag.Start.Weight + nettoScoreChange

	// TODO: implement dissolving between changes.

	// back-propagation cut-off strategy:
	// If n has sufficient weight to not lose its position on the canonical chain (i.e. n.Weight + v.Score > totalWeight / 2 + possible change)
	// Then the target will stay the same during this ApplyScoreChanges. Just propagate the weight adjustment.
	for _, v := range changes {
		n := v.Target
		// Propagate down the tree
		for n != nil {
			n.Weight += v.ScoreDelta
			// less possible change = better cutoff!
			cutOff2 -= v.ScoreDelta
			if v.ScoreDelta < 0 {
				onRemoveWeight(n)
			} else {
				onAddWeight(n)
			}
			if (n.Weight << 1) > cutOff2 {
				target := n.BestTarget
				n = n.Parent
				for n != nil {
					n.BestTarget = target
					n.Weight += v.ScoreDelta
					n = n.Parent
				}
				break
			}
			n = n.Parent
		}
	}
}

func (gh *StatefulLMDGhost) OnNewNode(node *dag.DagNode) {
	// best end-target is the block itself
	node.BestTarget = node
	if node.Parent != nil && len(node.Parent.Children) == 1 {
		// If this is the only/first node that is added,
		//  then it does not need attestations, it will just be the new target.

		// propagate the new best-target up, as far as necessary
		p := node.Parent
		c := node
		for p != nil {
			if c.IndexAsChild == 0 {
				p.BestTarget = node.BestTarget
				c = p
				p = p.Parent
			} else {
				// stop propagating when the child is not the best child of the parent
				break
			}
		}
	}
}

func (gh *StatefulLMDGhost) OnStartChange(newStart *dag.DagNode) {
	// nothing to do when the start changes
}

func (gh *StatefulLMDGhost) HeadFn() *dag.DagNode {
	// All the work has already been done, just pick the best-target of the root node.
	// *Bonus*: And this works for *every* node in the graph!
	// Changing the root is costless
	// (If you prune away old nodes it still costs something, but this also needs to be done for other algos)
	return gh.dag.Start.BestTarget
}
