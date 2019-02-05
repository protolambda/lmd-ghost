package state

import (
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/common/constants"
	"lmd-ghost/eth2/data/attestation"
	"lmd-ghost/eth2/data/validator"
)

type BeaconState struct {

	Slot uint64

	ValidatorRegistry []validator.Validator

	// The latest message of each (active) validator,
	// some validators may not have one, some may point to the same attestation (aggregate).
	Targets map[common.ValidatorID]*attestation.Attestation

}

func (state *BeaconState) NextSlot() error {
	state.Slot += 1

	// TODO process batched-block-roots

	if state.Slot % constants.EPOCH_LENGTH == 0 {
		if err := state.NextEpoch(); err != nil {
			return err
		}
	}

	return nil
}

func (state *BeaconState) NextEpoch() error {
	// TODO shuffle validators etc.
	return nil
}