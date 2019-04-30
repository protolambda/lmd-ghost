package proto_array

import (
	"lmd-ghost/eth2/dag"
)

type ProtoArrayLMDGhost struct {
	dag *dag.BeaconDag

	lastIndex uint64

	// node best-child
	b []uint64
	// node values
	w []int64
	// node parents
	p []uint64
	// node best-targets
	t []uint64

	// if we prune, indices need to be adjusted
	indexOffset uint64

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
	d := make([]int64, gh.lastIndex)
	for _, c := range changes {
		i := gh.indices[c.Target]
		d[i] = c.ScoreDelta
	}
	// back-prop diff values
	for i := len(d) - 1; i >= 0; i-- {
		pi := gh.p[i]
		if pi != 0 {
			d[pi] += d[i]
		}
	}
	// apply diffs to weights
	// TODO: could be parallel / SIMD
	for i, v := range d {
		gh.w[i] += v
	}
	// back-prop best-child/target updates
	for i := uint64(len(d) - 1); i >= 0; i-- {
		pi := gh.p[i] - gh.indexOffset
		bpi := gh.b[pi] - gh.indexOffset
		if bpi != i && d[i] > d[bpi] {
			if gh.w[i] > gh.w[bpi] {
				gh.b[pi] = i + gh.indexOffset
				gh.t[pi] = gh.t[i]
			}
		}
	}
}

func (gh *ProtoArrayLMDGhost) OnNewNode(block *dag.DagNode) {
	gh.lastIndex += 1
	gh.indices[block] = gh.lastIndex
	gh.b = append(gh.b, 0)
	gh.w = append(gh.w, 0)
	if block.Parent == nil {
		gh.p = append(gh.p, 0)
	} else {
		gh.p = append(gh.p, gh.indices[block.Parent])
	}
	gh.t = append(gh.t, gh.lastIndex)
}

func (gh *ProtoArrayLMDGhost) OnPrune() {
	i := gh.indices[gh.dag.Finalized]
	i -= gh.indexOffset
	gh.b = gh.b[i:]
	gh.w = gh.w[i:]
	gh.p = gh.p[i:]
	gh.t = gh.t[i:]
	// TODO recycle backing arrays?

	for x := uint64(0); x < i; x++ {
		n := gh.nodes[x]
		delete(gh.indices, n)
	}
	gh.nodes = gh.nodes[i:]
}

func (gh *ProtoArrayLMDGhost) HeadFn() *dag.DagNode {
	i := gh.indices[gh.dag.Finalized]
	targetI := gh.t[i - gh.indexOffset]
	target := gh.nodes[targetI - gh.indexOffset]
	return target
}
