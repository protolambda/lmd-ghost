package dag

import (
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/data/attestation"
	"lmd-ghost/eth2/fork_choice"
)

/// Beacon-Dag: a collection of the blocks in the canonical chain, and all its unfinalized branches.

type BeaconDag struct {

	// The main component: chooses which truth to follow.
	ForkChoice fork_choice.ForkChoice

}


func (dag *BeaconDag) BlockIn(block *block.BeaconBlock) {

}

func (dag *BeaconDag) AttestationIn(attestation *attestation.Attestation) {

}


func (b *BatchedForkChoice) SetStart(blockHash common.Hash256) {
	b.forkChoice.SetStart(blockHash)
}

func (b *BatchedForkChoice) HeadFn() common.Hash256 {
	return b.forkChoice.HeadFn()
}
