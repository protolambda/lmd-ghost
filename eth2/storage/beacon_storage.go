package storage

import (
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/common"
)

/// Very simple storage, to abstract away block-storage from the implementation,
//  making it easier to integrate the advanced parts like fork-choice etc. into a real client.
type BeaconStorage struct {

	blocks map[common.Hash256]*block.BeaconBlock

}

func NewBeaconStorage() *BeaconStorage {
	res := &BeaconStorage{blocks: make(map[common.Hash256]*block.BeaconBlock)}
	return res
}

func (st *BeaconStorage) GetBlock(blockHash common.Hash256) (*block.BeaconBlock, error) {
	return st.blocks[blockHash], nil
}

func (st *BeaconStorage) PutBlock(block *block.BeaconBlock) error {
	st.blocks[block.Hash] = block
	return nil
}

