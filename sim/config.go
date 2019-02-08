package sim

import (
	"fmt"
	"strings"
)

type SimConfig struct {
	// Static amount of validators in simulation. Can be very high, since targets are batched.
	ValidatorCount uint64
	// Latency factor, an idea taken from the simulation by Vitalik. The higher the factor, the closer proposals are to the head.
	LatencyFactor float64
	// The chance to skip a slot, repeats max. 10 times.
	SlotSkipChance float64
	// Every attestation will have at least this weight
	BaseAttestWeight uint64
	// In addition to the base weight, randomly add 0 - max_extra. Uniform distribution.
	MaxExtraAttestWeight uint64
	// The amount of blocks to simulate. Not consecutive, but total additions to the tree starting from genesis. Genesis excluded.
	Blocks uint64
	// Distance in epochs, from head, to finalize up to. Finalization results in pruning of the DAG.
	FinalizeEpochsAgo uint64
	// Distance in epochs, from head, to justify up to. Justification changes the starting point from where the fork-rule is executed.
	JustifyEpochsAgo uint64
	// Amount of individual attestations to simulate and add per simulated block. Attestations are batched. This may include double attestations by the same validator. Batching will reduce it to one.
	AttestationsPerBlock uint64
	// The name of the fork-choice rule. Generally, names are the same as the packages. Mapping is defined in sim/simulation.go.
	ForkChoiceRule string
}

func (c *SimConfig) String() string {
	return strings.Replace(
		fmt.Sprintf("v%d_lf%f_sc%f_bw%d_ew%d_bl%d_atpb%d_fork-%s",
		c.ValidatorCount, c.LatencyFactor, c.SlotSkipChance,
		c.BaseAttestWeight, c.MaxExtraAttestWeight, c.Blocks,
		c.AttestationsPerBlock, c.ForkChoiceRule),
		".", "_", -1)
}
