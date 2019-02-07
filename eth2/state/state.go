package state

import (
	"lmd-ghost/eth2/attestations/attestation"
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/common/constants"
	"lmd-ghost/eth2/data/validator"
	"math/rand"
)

type BeaconState struct {

	Slot uint64

	// Real-world implementation would have a much more secure RANDAO system.
	// Here we just hack in a seed to get the validators with.
	Seed int64

	ValidatorRegistry []*validator.Validator

	// The latest message of each (active) validator,
	// some validators may not have one, some may point to the same attestation (aggregate).
	Targets map[common.ValidatorID]*attestation.Attestation

}

func (st *BeaconState) GetProposer() *validator.Validator {
	// In spec: get the first committee for the slot being proposed,
	// and select member within based on slot. Committees are shuffled each epoch.
	// Here: just get based on epoch-relative slot number, and shuffle complete list every epoch
	return st.ValidatorRegistry[st.Slot % constants.EPOCH_LENGTH]
}

func (st *BeaconState) NextSlot() error {
	st.Slot += 1

	// TODO real client: process batched-block-roots

	if st.Slot % constants.EPOCH_LENGTH == 0 {
		if err := st.NextEpoch(); err != nil {
			return err
		}
	}

	return nil
}

func (st *BeaconState) NextEpoch() error {
	// shuffle validators etc
	rng := rand.New(rand.NewSource(st.Seed))
	// TODO: number-theoretic shuffling would be much better
	rng.Shuffle(len(st.ValidatorRegistry), func(i int, j int) {
		st.ValidatorRegistry[i], st.ValidatorRegistry[j] = st.ValidatorRegistry[j], st.ValidatorRegistry[i]
	})
	// TODO real client: handle all other epoch processing.
	return nil
}
