package block

import (
	"lmd-ghost/eth2/common"
)

type BeaconBlock struct {

	ParentHash common.Hash256

	Hash common.Hash256

	Proposer common.ValidatorID

	Slot uint64

	// Lots of other stuff from the spec could be added here.

}


