package sim

import (
	"lmd-ghost/eth2/attestations/attestation"
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/chain"
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/dag"
	"lmd-ghost/eth2/data/validator"
	"lmd-ghost/eth2/fork_choice"
	"lmd-ghost/eth2/state"
	"math/rand"
)

type Block struct {

	Children []*Block
}

const EPOCH_LENGTH = 64
const LATENCY_FACTOR = 0.8
const SLOT_SKIP_CHANCE = 0.4


type Simulation struct {
	RNG *rand.Rand

	Chain *chain.BeaconChain
}

func NewSimulation(validatorCount int, initForkChoice fork_choice.InitForkChoice) *Simulation {
	if validatorCount % EPOCH_LENGTH != 0 {
		panic("validator count should be nicely divisible by the epoch-length in this simulation.")
	}

	genesisBlock := &block.BeaconBlock{
		ParentHash: common.Hash256{0},
		Hash: common.Hash256{1},
		Proposer: 0,
		Slot: 0,
	}
	genesisState := &state.BeaconState{
		Slot: 0,
		ValidatorRegistry: make([]*validator.Validator, validatorCount, validatorCount),
	}
	for i := 0; i < validatorCount; i++ {
		// TODO: better simulated balances?
		genesisState.ValidatorRegistry[i] = &validator.Validator{Id: common.ValidatorID(i), Balance: 10}
	}

	ch, err := chain.NewBeaconChain(genesisBlock, genesisState, initForkChoice)
	if err != nil {
		panic("Failed to initialize chain for simulation.")
	}

	s := &Simulation{
		RNG:        rand.New(rand.NewSource(1234)),
		Chain: ch,
	}
	return s
}

/// Goes up (towards slot 0) the tree by a few steps (upCount, more with more latency) and then back down a random path.
func (s *Simulation) getRandomTarget() *dag.DagNode {
	upCount := 0
	target := s.Chain.Dag.Nodes[s.Chain.Head]
	for {
		if target.Slot > 0 && s.RNG.Float64() < LATENCY_FACTOR {
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

	st, err := s.Chain.Storage.GetPostState(parentBlock.Key)
	if err != nil {
		panic("Could not fetch state for simulation")
	}

	// randomly skip some slots
	for s.RNG.Float64() < SLOT_SKIP_CHANCE {
		if err := st.NextSlot(); err != nil {
			panic("Could not update state to new slot")
		}
	}

	// init the fake randomness of this state.
	st.Seed = int64(s.RNG.Uint64())
	proposer := st.GetProposer()

	// random block-hash
	blockHash := common.Hash256{}
	s.RNG.Read(blockHash[:])

	// create the block
	bl := &block.BeaconBlock{ParentHash: parentBlock.Key, Hash: blockHash, Proposer: proposer.Id, Slot: st.Slot}

	// add it to the chain
	if err := s.Chain.BlockIn(bl); err != nil {
		panic("Could not insert simulated new block")
	}

	// make the proposer attest its own block
	at := &attestation.Attestation{BeaconBlockRoot: bl.Hash, Attester: bl.Proposer, Weight: proposer.Balance}
	if err := s.Chain.AttestationIn(at); err != nil {
		panic("Could not insert simulated attestation")
	}
}

func (s *Simulation) SimNewAttestation() {
	// get random block
	target := s.getRandomTarget()

	st, err := s.Chain.Storage.GetPostState(target.Key)
	if err != nil {
		panic("Could not fetch state for simulation")
	}

	// select a random validator (every validator is allowed to attest here)
	attester := st.ValidatorRegistry[s.RNG.Intn(len(st.ValidatorRegistry))]

	// make the attestation happen
	at := &attestation.Attestation{BeaconBlockRoot: target.Key, Attester: attester.Id, Weight: attester.Balance}
	if err := s.Chain.AttestationIn(at); err != nil {
		panic("Could not insert simulated attestation")
	}
}

func (s *Simulation) UpdateHead() {
	s.Chain.UpdateHead()
}
