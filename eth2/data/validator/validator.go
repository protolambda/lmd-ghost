package validator

import "lmd-ghost/eth2/common"

type Validator struct {

	// this would be the pub-key (or a reference of one) in a real implementation
	Id common.ValidatorID

	Balance uint64

	// TODO implement entries/exits?
}
