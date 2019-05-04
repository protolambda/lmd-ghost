package proto_array

import (
	"lmd-ghost/eth2/dag"
)

const nonExistentNode = ^uint64(0)

type ProtoArrayLMDGhost struct {
	dag *dag.BeaconDag

	// node best-child
	b []uint64
	// node values
	w []int64
	// node parents
	p []uint64
	// node best-targets
	t []uint64

	// nodes in order
	nodes []*dag.DagNode

	// root -> node index
	indices map[*dag.DagNode]uint64

}

func NewProtoArrayLMDGhost(d *dag.BeaconDag) dag.ForkChoice {
	res := &ProtoArrayLMDGhost{
		dag:          d,
		indices: make(map[*dag.DagNode]uint64),
	}
	return res
}

func (gh *ProtoArrayLMDGhost) ApplyScoreChanges(changes []dag.ScoreChange) {
	// diff values: we map changes into an array mirroring the current state arrays.
	// These diff values can be propagated first, and then applied to the weights after.
	// This makes it easy to recognize when a "change" cancels out somewhere:
	//  the diff value will be 0, just like any non-change.
	d := make([]int64, len(gh.nodes), len(gh.nodes))
	start := int64(gh.indices[gh.dag.Finalized])
	for _, c := range changes {
		i := gh.indices[c.Target]
		d[i] += c.ScoreDelta
	}
	// back-prop diff values
	for i := int64(len(d)) - 1; i >= start; i-- {
		pi := gh.p[i]
		if pi == nonExistentNode {
			continue
		}
		if pi != 0 {
			d[pi] += d[i]
		}
	}
	// apply diffs to weights
	// (note: array ADD, doesn't have to be a loop)
	for i, v := range d {
		gh.w[i] += v
	}
	// back-prop best-child/target updates
	for i := int64(len(d)) - 1; i >= start; i-- {
		// propagate best-target
		if bi := gh.b[i]; bi != nonExistentNode {
			gh.t[i] = gh.t[bi]
		}
		// if this note did not change at all, then skip it
		if d[i] == 0 {
			continue
		}
		// parent of i (may not exist)
		pi := gh.p[i]
		if pi == nonExistentNode {
			continue
		}
		ui := uint64(i)
		// best child of the parent of i
		bpi := gh.b[pi]
		if bpi == nonExistentNode {
			// i is better than nothing, easy
			gh.b[pi] = ui
			continue
		}
		// if i is already the best, don't do any work
		// if i changed better than the bpi, then it's worth looking at.
		if bpi != ui && d[i] > d[bpi] {
			// now look at the weights: if i is better than bpi,
			// it becomes the best-child for its parent, and the parent is assigned the best-target of i
			if gh.w[i] > gh.w[bpi] {
				gh.b[pi] = ui
			}
		}
	}
}

func (gh *ProtoArrayLMDGhost) OnNewNode(block *dag.DagNode) {
	i := uint64(len(gh.nodes))
	gh.indices[block] = i
	// the new node does not have a best-child
	gh.b = append(gh.b, nonExistentNode)
	// new node is weighted 0
	gh.w = append(gh.w, 0)
	// the new node may not have a parent
	if block.Parent == nil {
		gh.p = append(gh.p, nonExistentNode)
	} else {
		// or the parent may be out of scope
		if pi, ok := gh.indices[block.Parent]; ok {
			gh.p = append(gh.p, pi)
			// if it is the first child, it is also the best.
			if gh.b[pi] == nonExistentNode {
				gh.b[pi] = i
			}
		} else {
			gh.p = append(gh.p, nonExistentNode)
		}
	}
	// new node points to itself as a best-target, since it is a leaf.
	gh.t = append(gh.t, i)
	gh.nodes = append(gh.nodes, block)
}

func (gh *ProtoArrayLMDGhost) OnPrune() {
	// get the index of the finalized node, and adjust to current array-space.
	start := gh.indices[gh.dag.Finalized]
	// Small pruning does not help more than it costs to do. Postpone pruning in such case.
	// For implementers: tune this parameter, or trigger pruning based on this value going over a threshold.
	if start < 200 {
		return
	}
	// Note: the elements pruned at the start will stay in the backing array.
	// However, since we are appending to the slice, append() may re-allocate
	// the slice to a new backing array: eventually the pruned parts will be GC'd.
	gh.b = gh.b[start:]
	gh.w = gh.w[start:]
	gh.p = gh.p[start:]
	gh.t = gh.t[start:]

	// now delete all pruned nodes from the key->index lookup-map.
	for i := uint64(0); i < start; i++ {
		n := gh.nodes[i]
		delete(gh.indices, n)
	}
	gh.nodes = gh.nodes[start:]

	// adjust indices back to 0
	for i, n := range gh.nodes {
		// best-child may not exist, i.e. does not need to be adjusted
		if gh.b[i] != nonExistentNode {
			gh.b[i] -= start
		}
		gh.t[i] -= start
		// parent may not exist anymore
		if gh.p[i] < start {
			gh.p[i] = nonExistentNode
		} else {
			gh.p[i] -= start
		}
		gh.indices[n] -= start
	}
}

func (gh *ProtoArrayLMDGhost) HeadFn() *dag.DagNode {
	// look up the index of the finalized node, this is our starting point
	i := gh.indices[gh.dag.Finalized]
	for {
		if bi := gh.b[i]; bi != nonExistentNode {
			i = bi
		} else {
			break
		}
	}
	// Get the target (again, adjust index)
	return gh.nodes[i]
}
