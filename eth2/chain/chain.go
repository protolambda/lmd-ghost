package chain

import (
	"errors"
	"fmt"
	"lmd-ghost/eth2/attestations/attestation"
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/dag"
	"lmd-ghost/eth2/storage"
)

type BeaconChain struct {

	// Access other most-current variables through storage or dag, using head as reference.
	Head      common.Hash256

	// The inner-source of continuously-changing truth: the data stored within and between (i.e. state) the blocks.
	Storage    storage.BeaconStorage

	// The outer-source of continuously-changing truth: the collection of blocks, structured.
	Dag        dag.BeaconDag

}

func (ch *BeaconChain) BlockIn(block *block.BeaconBlock) error {
	// verify the existence of the parent block

	// preparation
	// ======================
	// get state of parent block
	state, err := ch.Storage.GetPostState(block.ParentHash)
	if err != nil {
		return err
	}
	if state == nil {
		return errors.New(fmt.Sprintf("incoming block %s has parent %s that has not been processed", block.Hash, block.ParentHash))
	}

	// skip to slot block.slot - 1
	for i := state.Slot; i < block.Slot; i++ {
		if err := state.NextSlot(); err != nil {
			return errors.New(fmt.Sprintf("failed to progress state, continued from parent block %s, up to slot %d, during pre-processing state for block %s", block.ParentHash, i, block.Hash))
		}
	}

	// processing
	// ======================
	// process block
	if err := block.ProcessBlock(state); err != nil {
		return err
	}
	// continue last slot
	if err := state.NextSlot(); err != nil {
		return err
	}

	// post-processing
	// ======================

	// save the block and the state
	if err := ch.Storage.PutBlock(block); err != nil {
		return errors.New("failed to save processed block to storage")
	}
	if err := ch.Storage.PutPostState(block.Hash, state); err != nil {
		return errors.New("failed to save processed block to storage")
	}

	// determine the head
	ch.Dag.BlockIn(block)
	ch.Head = ch.Dag.HeadFn()

	return nil
}

func (ch *BeaconChain) AttestationIn(attestation *attestation.Attestation) error {
	// TODO verify attestation
	ch.Dag.AttestationIn(attestation)
	return nil
}
