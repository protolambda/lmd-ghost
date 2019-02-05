package storage

import (
	"lmd-ghost/eth2/storage/block"
	"lmd-ghost/eth2/storage/state"
)

/// very simple storage, to abstract away state and block-storage from the implementation,
//  making it easier to integrate the advanced parts like fork-choice etc. into a real client.
type BeaconStorage struct {

	block.BlockStorage

	state.StateStorage
}

func NewBeaconStorage() *BeaconStorage {
	// create storage
	res := new(BeaconStorage)
	// inititalize all storage facilities
	res.BlockStorage.Init()

	return res
}
