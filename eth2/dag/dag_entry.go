package dag

import "lmd-ghost/eth2/common"

type DagEntry struct {

	Parent *DagEntry

	Children []*DagEntry

	Key common.Hash256

	Slot uint64

	// TODO store extra data in dag itself, for different fork-choice implementations?
	Extra interface{}

}