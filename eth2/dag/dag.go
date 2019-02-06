package dag

import (
	"lmd-ghost/eth2/attestations"
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/attestations/attestation"
	"lmd-ghost/eth2/fork_choice"
)

/// Beacon-Dag: a collection of the blocks in the canonical chain, and all its unfinalized branches.

type BeaconDag struct {

	// The main component: chooses which truth to follow.
	ForkChoice fork_choice.ForkChoice

	// Aggegate, the effective "latest-targets", but every attestation is grouped by block.
	agor attestations.AttestationsAggregator

	Nodes map[common.Hash256]*DagNode
	lastAtID VoteIdentity

	Start *DagNode
}


func (dag *BeaconDag) BlockIn(block *block.BeaconBlock) {
	// TODO create node
}

func (dag *BeaconDag) AttestationIn(atIn *attestation.Attestation) {
	dag.agor.AttestationIn(atIn)
}

func (dag *BeaconDag) SetStart(blockHash common.Hash256) {
	newStart := dag.Nodes[blockHash]
	dag.ForkChoice.StartIn(newStart)
	// change old start after signifying the new start, the fork-choice gets an opportunity to retrieve both nodes.
	dag.Start = newStart
}

func (dag *BeaconDag) SyncChanges() {
	changes := make([]fork_choice.ScoreChange, 0)
	for k, v := range dag.agor.LatestAggregates {
		if v.PrevWeight != v.Weight {
			// get delta
			delta := v.Weight - v.PrevWeight
			// resolve difference in weight
			v.PrevWeight = v.Weight
			// remember the change, append it to our "to do" list of changes
			changes = append(changes, fork_choice.ScoreChange{Target: dag.Nodes[k], ScoreDelta: delta})
		}
	}
	dag.ForkChoice.ApplyScoreChanges(changes)
}

func (dag *BeaconDag) HeadFn() common.Hash256 {
	return dag.ForkChoice.HeadFn().Key
}

func (dag *BeaconDag) Cleanup() {
	// cleanup aggregator
	dag.agor.Cleanup()
	// TODO prune DAG
}
