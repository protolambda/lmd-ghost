package sim

import (
	"fmt"
	"strings"
)

type SimConfig struct {
	ValidatorCount uint64
	LatencyFactor float64
	SlotSkipChance float64
	BaseAttestWeight uint64
	MaxExtraAttestWeight uint64
	Blocks uint64
	FinalizeEpochsAgo uint64
	JustifyEpochsAgo uint64
	AttestationsPerBlock uint64
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
