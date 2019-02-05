package attestation

import "lmd-ghost/eth2/common"

type WeightedAttester struct {
	Attester common.ValidatorID
	// note: signed. Makes separating additions/removals in an attestation target change easy.
	Weight int64
	Slot uint64
}

type Attestation struct {

	// the minimum attestated slot in all of WeightedAttesters, needs to be kept consistent
	Slot uint64

	BeaconBlockRoot common.Hash256

	/// list of validators responsible for this (Aggregated or not) attestation
	WeightedAttesters []*WeightedAttester

	/// sum of weights in WeightedAttesters, needs to be kept consistent
	SumWeight int64

	// TODO other spec variables
}
