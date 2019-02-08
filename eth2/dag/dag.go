package dag

import (
	"lmd-ghost/eth2/attestations"
	"lmd-ghost/eth2/attestations/attestation"
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/common"
)

/// Beacon-Dag: a collection of the blocks in the canonical chain, and all its unfinalized branches.

type BeaconDag struct {

	ForkChoice ForkChoice

	// Aggegate, the effective "latest-targets", but every attestation is grouped by block.
	agor *attestations.AttestationsAggregator
	synced bool

	Nodes map[common.Hash256]*DagNode

	Start *DagNode
}

func NewBeaconDag(initForkChoice InitForkChoice) *BeaconDag {
	res := &BeaconDag{
		synced: false,
		Nodes: make(map[common.Hash256]*DagNode),
	}
	res.ForkChoice = initForkChoice(res)
	res.agor = attestations.NewAttestationsAggregator(func(blockHash common.Hash256) uint64 {
		return res.Nodes[blockHash].Slot
	})
	return res
}

func (dag *BeaconDag) BlockIn(block *block.BeaconBlock) {
	dag.synced = false
	// Create a node in the DAG for the block
	node := &DagNode{
		Parent: dag.Nodes[block.ParentHash],
		Proposer: block.Proposer,
		// expected branch factor is 2 (??), capacity of 8 should be fine? (TODO)
		Children: make([]*DagNode, 0, 8),
		Key: block.Hash,
		Slot: block.Slot,
		Height: 0,
		Weight: 0,
	}
	// append to parent's children if there is a parent
	if node.Parent != nil {
		node.IndexAsChild = uint32(len(node.Parent.Children))
		node.Parent.Children = append(node.Parent.Children, node)
		node.Height = node.Parent.Height + 1
	}
	dag.Nodes[block.Hash] = node
	if dag.Start == nil {
		dag.Start = node
		dag.ForkChoice.OnStartChange()
	}
	dag.ForkChoice.OnNewNode(node)
}

func (dag *BeaconDag) AttestationIn(atIn *attestation.Attestation) {
	dag.synced = false
	// input the attestation into the attestation aggregator.
	dag.agor.AttestationIn(atIn)
}

func (dag *BeaconDag) SetStart(blockHash common.Hash256) {
	dag.synced = false
	newStart := dag.Nodes[blockHash]
	dag.Start = newStart
	dag.ForkChoice.OnStartChange()
}

func (dag *BeaconDag) SyncChanges() {
	// Find all the changes made in the aggregator and apply them to the DAG.
	changes := make([]ScoreChange, 0)
	for k, v := range dag.agor.LatestAggregates {
		if v.PrevWeight != v.Weight {
			// get delta
			delta := int64(v.Weight) - int64(v.PrevWeight)
			// resolve difference in weight
			v.PrevWeight = v.Weight
			// remember the change, append it to our "to do" list of changes
			changes = append(changes, ScoreChange{Target: dag.Nodes[k], ScoreDelta: delta})
		}
	}
	dag.ForkChoice.ApplyScoreChanges(changes)
	dag.synced = true
}

func (dag *BeaconDag) HeadFn() common.Hash256 {
	// Make sure changes have been synced
	if !dag.synced {
		dag.SyncChanges()
	}
	// return the head
	return dag.ForkChoice.HeadFn().Key
}

func (dag *BeaconDag) Cleanup() {
	// cleanup aggregator
	dag.agor.Cleanup()
	// TODO prune DAG
}
