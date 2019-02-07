package block

import (
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/common"
)

type BlockStorage struct {

	blocks map[common.Hash256]*block.BeaconBlock

}

func NewBlockStorage() *BlockStorage {
	res := &BlockStorage{blocks: make(map[common.Hash256]*block.BeaconBlock)}
	return res
}

func (st *BlockStorage) GetBlock(blockHash common.Hash256) (*block.BeaconBlock, error) {
	return st.blocks[blockHash], nil
}

func (st *BlockStorage) PutBlock(block *block.BeaconBlock) error {
	st.blocks[block.Hash] = block
}
