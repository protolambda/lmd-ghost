package attestation

import "lmd-ghost/eth2/common"

type Attestation struct {

	BeaconBlockRoot common.Hash256

	Attester common.ValidatorID

	Weight uint64
}
