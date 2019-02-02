package sim

import (
	"math/rand"
)

// The effective identifier of a block
type Hash256 [32]uint8

// Pubkey (or reference to one in a register) in real-world
type ValidatorID int64

type Block struct {
	ParentHash Hash256

	Hash Hash256

	Proposer ValidatorID

	Slot uint32

	Children []*Block
}

const EPOCH_LENGTH = 64
const LATENCY_FACTOR = 0.9
const MAX_SLOT_SKIP = 4

type ForkChoice interface {
	SetChain(chain *SimChain)
	BlockIn(block *Block)
	/// Note: the target attestion in the chain update will be updated with the given hash after calling this.
	// This enables you to access the previous latest attestation of the validator
	AttestIn(blockHash Hash256, attester ValidatorID)
	HeadFn() Hash256
}

type GetForkChoice func() ForkChoice

type SimChain struct {
	RNG *rand.Rand

	// every validator is "active". No validators enter/exit in this simulation.
	Validators []ValidatorID

	Blocks map[Hash256]*Block

	// The latest message of each (active) validator, some validators may not have one
	Targets map[ValidatorID]Hash256

	Justified Hash256
	Head      Hash256

	ForkChoice ForkChoice

	Slot  uint64
	Epoch uint64
}

func NewChain(validatorCount int, forkChoice ForkChoice) *SimChain {
	if validatorCount % EPOCH_LENGTH != 0 {
		panic("validator count should be nicely divisible by the epoch-length in this simulation.")
	}

	// simple recognizable non-zero hash as origin point (or "genesis") for the simulated chain.
	origin := Hash256{1}

	ch := &SimChain{
		RNG:        rand.New(rand.NewSource(1234)),
		Validators: make([]ValidatorID, validatorCount, validatorCount),
		Blocks:     make(map[Hash256]*Block),
		Targets:    make(map[ValidatorID]Hash256),
		Justified:  origin,
		Head:       origin,
		ForkChoice: forkChoice,
	}
	originBlock := Block{Hash:origin, Proposer: 0, Slot: 0, Children: make([]*Block, 0)}
	ch.Blocks[origin] = &originBlock
	for i := 0; i < validatorCount; i++ {
		ch.Validators[i] = ValidatorID(i)
	}
	return ch
}

func (ch *SimChain) SimJustify() {
	// TODO
}

func (ch *SimChain) Justify(blockHash Hash256) {
	ch.Justified = blockHash

	// TODO cleanup chain?
}

/// Goes up (towards slot 0) the tree by a few steps (upCount, more with more latency) and then back down a random path.
func (ch *SimChain) getRandomTarget() *Block {
	upCount := 0
	target := ch.Head
	for {
		targetBlock := ch.Blocks[target]
		if targetBlock.Slot > 0 && ch.RNG.Float64() < LATENCY_FACTOR {
			target = targetBlock.ParentHash
			upCount++
		} else {
			break
		}
	}
	downCount := ch.RNG.Intn(upCount + 1)
	targetBlock := ch.Blocks[target]
	for i := 0; i < downCount; i++ {
		if len(targetBlock.Children) > 0 {
			targetBlock = targetBlock.Children[ch.RNG.Intn(len(targetBlock.Children))]
		} else {
			break
		}
	}
	return targetBlock
}

func (ch *SimChain) HandleProposedBlock(blockHash Hash256, parentHash Hash256, proposer ValidatorID) {
	bl := Block{Hash: blockHash, ParentHash: parentHash, Proposer: proposer, Children: make([]*Block, 0)}
	parentBlock := ch.Blocks[bl.ParentHash]
	parentBlock.Children = append(parentBlock.Children, &bl)
	// slot is incremented by 1
	bl.Slot = parentBlock.Slot + 1
	// add block
	ch.Blocks[bl.Hash] = &bl
	// make our fork-choice mechanism aware of the new block
	ch.ForkChoice.BlockIn(&bl)
	// Update the target of the proposer
	ch.HandleAttestation(bl.Hash, proposer)
}

func (ch *SimChain) SimNewBlock() {
	// random parent block, derived from the current head, but perturbed; latency may introduce a fork in the chain
	parentBlock := ch.getRandomTarget()

	slot := parentBlock.Slot + uint32(ch.RNG.Intn(MAX_SLOT_SKIP) + 1)

	// In spec: get the first committee for the slot being proposed, and select member within based on slot. Committees are shuffled each epoch.
	//
	// Here: get a random validator, but with epoch influencing the choice: within the same epoch, a validator cannot propose a block in different slots.
	// How: select a random subset of the validators, size EPOCH_LENGTH, with and pick a validator based on the current slot.
	// This guarantees that a validator cannot propose twice within the same epoch.
	// And modify the slot, with an offset based on the epoch, to make every epoch a little different. Not secure, but sufficient for simulation (I think).
	proposerIndex := uint32(ch.RNG.Intn(len(ch.Validators)/EPOCH_LENGTH))*EPOCH_LENGTH + ((slot + (slot / EPOCH_LENGTH)) % EPOCH_LENGTH)
	proposer := ch.Validators[proposerIndex]


	// random block-hash
	blockHash := Hash256{}
	ch.RNG.Read(blockHash[:])

	// process it
	ch.HandleProposedBlock(blockHash, parentBlock.Hash, proposer)
}

func (ch *SimChain) HandleAttestation(target Hash256, id ValidatorID) {
	// make our fork-choice mechanism aware of the new attestation
	ch.ForkChoice.AttestIn(target, id)
	// Update the target of the attester
	ch.Targets[id] = target
}

func (ch *SimChain) SimNewAttestation() {
	// select a random validator (every validator is allowed to attest here)
	attester := ch.Validators[ch.RNG.Intn(len(ch.Validators))]
	// Connect to a random target block
	target := ch.getRandomTarget()
	ch.HandleAttestation(target.Hash, attester)
}

func (ch *SimChain) UpdateHead() {
	ch.Head = ch.ForkChoice.HeadFn()
}
