package chain

import (
	"errors"
	"lmd-ghost/eth2/attestations/attestation"
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/dag"
	"lmd-ghost/eth2/fork_choice"
	"lmd-ghost/eth2/storage"
)

type BeaconChain struct {

	// Access other most-current variables through storage or dag, using head as reference.
	Head      common.Hash256

	// The inner-source of continuously-changing truth: the data stored within and between (i.e. state) the blocks.
	Storage    *storage.BeaconStorage

	// The outer-source of continuously-changing truth: the collection of blocks, structured.
	Dag        *dag.BeaconDag

}

func NewBeaconChain(genesisBlock *block.BeaconBlock, initForkChoice fork_choice.InitForkChoice) (*BeaconChain, error) {
	res := &BeaconChain{
		Head: genesisBlock.Hash,
		Storage: storage.NewBeaconStorage(),
		Dag: dag.NewBeaconDag(initForkChoice),
	}
	if err := res.Storage.PutBlock(genesisBlock); err != nil {
		return nil, err
	}
	// For a real client:
	//if err := res.Storage.PutPostState(genesisBlock.Hash, genesisState); err != nil {
	//	return nil, err
	//}
	res.Dag.Start = &dag.DagNode{
		Parent: nil,
		// expected branch factor is 2 (??), capacity of 8 should be fine? (TODO)
		Children: make([]*dag.DagNode, 0, 8),
		Key: genesisBlock.Hash,
		Slot: genesisBlock.Slot,
		Weight: 0,
	}
	return res, nil
}

func (ch *BeaconChain) BlockIn(block *block.BeaconBlock) error {
	// For a real implementation:
	//// preparation
	//// ======================
	//// get state of parent block, verify the existence of the parent block
	//st, err := ch.Storage.GetPostState(block.ParentHash)
	//if err != nil {
	//	return err
	//}
	//if st == nil {
	//	return errors.New(fmt.Sprintf("incoming block %s has parent %s that has not been processed", block.Hash, block.ParentHash))
	//}
	//
	//// skip to slot block.slot - 1
	//for i := st.Slot; i < block.Slot; i++ {
	//	if err := st.NextSlot(); err != nil {
	//		return errors.New(fmt.Sprintf("failed to progress state, continued from parent block %s, up to slot %d, during pre-processing state for block %s", block.ParentHash, i, block.Hash))
	//	}
	//}
	//
	//// processing
	//// ======================
	//// process block
	//if err := block.ProcessBlock(st); err != nil {
	//	return err
	//}
	//// continue last slot
	//if err := st.NextSlot(); err != nil {
	//	return err
	//}
	//
	//// post-processing
	//// ======================
	//
	// save the block and the state
	if err := ch.Storage.PutBlock(block); err != nil {
		return errors.New("failed to save processed block to storage")
	}
	//if err := ch.Storage.PutPostState(block.Hash, st); err != nil {
	//	return errors.New("failed to save processed block to storage")
	//}

	ch.Dag.BlockIn(block)

	ch.UpdateHead()

	return nil
}

func (ch *BeaconChain) AttestationIn(attestation *attestation.Attestation) error {
	// missing here: verify attestation
	// real implementation would save the attestation, for later slashing etc.
	ch.Dag.AttestationIn(attestation)
	return nil
}

func (ch *BeaconChain) UpdateHead() {
	// determine the head
	ch.Head = ch.Dag.HeadFn()
}