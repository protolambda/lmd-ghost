package sim

import (
	"lmd-ghost/eth2/attestations/attestation"
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/chain"
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/common/constants"
	"lmd-ghost/eth2/dag"
	"lmd-ghost/eth2/fork_choice/choices/cached"
	"lmd-ghost/eth2/fork_choice/choices/proto_array"
	"lmd-ghost/eth2/fork_choice/choices/simple_back_prop"
	"lmd-ghost/eth2/fork_choice/choices/spec"
	"lmd-ghost/eth2/fork_choice/choices/stateful"
	"lmd-ghost/eth2/fork_choice/choices/vitalik"
	"lmd-ghost/viz"
	"log"
	"math/rand"
)


var forkRules = map[string]dag.InitForkChoice {
	"spec": spec.NewSpecLMDGhost,
	"vitalik": vitalik.NewVitaliksOptimizedLMDGhost,
	"cached": cached.NewCachedLMDGhost,
	"simple_back_prop": simple_back_prop.NewSimpleBackPropLMDGhost,
	"stateful": stateful.NewStatefulLMDGhost,
	"proto_array": proto_array.NewProtoArrayLMDGhost,
}


type Simulation struct {
	RNG *rand.Rand

	Chain *chain.BeaconChain

	Config *SimConfig
}

func NewSimulation(c *SimConfig) *Simulation {

	initForkChoice := forkRules[c.ForkChoiceRule]

	genesisBlock := &block.BeaconBlock{
		ParentHash: common.Hash256{0},
		Hash: common.Hash256{1},
		Proposer: 0,
		Slot: constants.GENESIS_SLOT,
	}

	ch, err := chain.NewBeaconChain(genesisBlock, initForkChoice)
	if err != nil {
		panic("Failed to initialize chain for simulation.")
	}

	s := &Simulation{
		RNG:        rand.New(rand.NewSource(1234)),
		Chain: ch,
		Config: c,
	}
	return s
}

/// Goes up (towards slot 0) the tree by a few steps (upCount, more with more latency) and then back down a random path.
func (s *Simulation) getRandomTarget() *dag.DagNode {
	upCount := 0
	target := s.Chain.Dag.Nodes[s.Chain.Head]
	for {
		if target.Parent != nil && s.RNG.Float64() < s.Config.LatencyFactor {
			target = target.Parent
			upCount++
		} else {
			break
		}
	}
	downCount := s.RNG.Intn(upCount + 1)
	for i := 0; i < downCount; i++ {
		if len(target.Children) > 0 {
			target = target.Children[s.RNG.Intn(len(target.Children))]
		} else {
			break
		}
	}
	return target
}

func (s *Simulation) SimNewBlock() {
	// random parent block, derived from the current head, but perturbed; latency may introduce a fork in the chain
	parentBlock := s.getRandomTarget()

	blockSlot := parentBlock.Slot + 1
	for i := 0; i < 10; i++ {
		if s.RNG.Float64() > s.Config.SlotSkipChance {
			break
		}
		blockSlot++
	}

	// get a random proposer
	// [divergence from spec: there's a slight chance that a proposer proposes twice in the same epoch]
	proposer := common.ValidatorID(s.RNG.Intn(int(s.Config.ValidatorCount)))

	// random block-hash
	blockHash := common.Hash256{}
	s.RNG.Read(blockHash[:])

	// create the block
	bl := &block.BeaconBlock{ParentHash: parentBlock.Key, Hash: blockHash, Proposer: proposer, Slot: blockSlot}

	// add it to the chain
	if err := s.Chain.BlockIn(bl); err != nil {
		panic("Could not insert simulated new block")
	}

	weight := s.Config.BaseAttestWeight
	if s.Config.MaxExtraAttestWeight != 0 {
		weight += uint64(s.RNG.Intn(int(s.Config.MaxExtraAttestWeight)))
	}

	// make the proposer attest its own block
	at := &attestation.Attestation{BeaconBlockRoot: bl.Hash, Attester: bl.Proposer, Weight: uint64(weight)}
	if err := s.Chain.AttestationIn(at); err != nil {
		panic("Could not insert simulated attestation")
	}
}

func (s *Simulation) SimNewAttestation() {
	// get random block
	target := s.getRandomTarget()

	// select a random validator (every validator is allowed to attest here)
	attester := common.ValidatorID(s.RNG.Intn(int(s.Config.ValidatorCount)))

	weight := s.Config.BaseAttestWeight
	if s.Config.MaxExtraAttestWeight != 0 {
		weight += uint64(s.RNG.Intn(int(s.Config.MaxExtraAttestWeight)))
	}

	// make the attestation happen
	at := &attestation.Attestation{BeaconBlockRoot: target.Key, Attester: attester, Weight: uint64(weight)}
	if err := s.Chain.AttestationIn(at); err != nil {
		panic("Could not insert simulated attestation")
	}
}

// TODO parametrize latency, simulated attestations per block, and slot-skip
func (s *Simulation) RunSim() {
	// log every 5% of the simulated amount of blocks
	logInterval := s.Config.Blocks / 20
	// update the head 10 times during attestation processing.
	headUpdateInterval := s.Config.AttestationsPerBlock / 10
	attestationCounter := uint64(0)
	if s.Config.FinalizeEpochsAgo < s.Config.JustifyEpochsAgo {
		panic("invalid config: finalization happens quicker than justification in config")
	}
	if s.Config.FinalizeEpochsAgo < 1 {
		panic("finalization is too quick")
	}
	if s.Config.JustifyEpochsAgo < 1 {
		panic("justification is too quick")
	}
	for n := uint64(0); n < s.Config.Blocks; n++ {
		head := s.Chain.Dag.Nodes[s.Chain.Head]
		epoch := head.Slot / constants.EPOCH_LENGTH
		if epoch > s.Config.FinalizeEpochsAgo {
			finalizedEpoch := epoch - s.Config.FinalizeEpochsAgo
			if finalizedEpoch > s.Chain.Dag.Finalized.Slot/constants.EPOCH_LENGTH {
				f := s.Chain.Dag.Justified
				for f != nil && f.Slot/constants.EPOCH_LENGTH > finalizedEpoch {
					f = f.Parent
				}
				if f != nil && f != s.Chain.Dag.Finalized {
					s.Chain.Dag.Finalize(f.Key)
				}
			}
		}
		if epoch > s.Config.JustifyEpochsAgo {
			justifiedEpoch := epoch - s.Config.JustifyEpochsAgo
			if justifiedEpoch > s.Chain.Dag.Justified.Slot/constants.EPOCH_LENGTH {
				j := head
				for j != nil && j.Slot/constants.EPOCH_LENGTH > justifiedEpoch {
					j = j.Parent
				}
				if j != nil && j != s.Chain.Dag.Justified {
					s.Chain.Dag.Justify(j.Key)
				}
			}
		}

		if n % logInterval == 0 {
			log.Printf("total %d blocks added, %d blocks in dag, head at slot: %d, processed %d attestations.\n",
				n, len(s.Chain.Dag.Nodes), s.Chain.Dag.Nodes[s.Chain.Head].Slot - constants.GENESIS_SLOT, attestationCounter)
		}
		for a := uint64(0); a < s.Config.AttestationsPerBlock; a++ {
			s.SimNewAttestation()
			if a % headUpdateInterval == headUpdateInterval - 1 {
				s.Chain.UpdateHead()
			}
		}
		attestationCounter += s.Config.AttestationsPerBlock
		// head will update after adding a block
		s.SimNewBlock()
	}
	log.Printf("total %d blocks added, %d blocks in dag, head at slot: %d, processed %d attestations.\n",
		s.Config.Blocks, len(s.Chain.Dag.Nodes), s.Chain.Dag.Nodes[s.Chain.Head].Slot - constants.GENESIS_SLOT, attestationCounter)
}

func (s *Simulation) SaveNetworkGraph() {
	simName := s.Config.String()
	viz.CreateVizGraph("out/" + simName, s.Chain)
}