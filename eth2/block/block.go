package block

import (
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/state"
)

type BeaconBlock struct {

	ParentHash common.Hash256

	Hash common.Hash256

	Proposer common.ValidatorID

	Slot uint64

	// Lots of other stuff from the spec could be added here.

}

func (block *BeaconBlock) ProcessBlock(state *state.BeaconState) error {

	// TODO validation: check if data is ok/complete/signed
	// TODO processing: apply block to state.
	// TODO verification: check if the computed state-root of the state matches the state-root of the block.

	return nil
}

