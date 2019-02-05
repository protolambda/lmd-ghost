package fork_choice

import (
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/dag"
	"lmd-ghost/eth2/data/attestation"
)

type ForkChoice interface {
	SetDag(dag *dag.BeaconDag)
	BlockIn(block *block.BeaconBlock)
	AttestationIn(attestation *attestation.Attestation)
	SetStart(blockHash common.Hash256)
	HeadFn() common.Hash256
}

type ConstructForkChoice func() ForkChoice
