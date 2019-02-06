package dag

import "lmd-ghost/eth2/common"

type DagNode struct {

	Parent *DagNode

	Children []*DagNode

	Key common.Hash256

	Slot uint64

	// Note: unused in some algorithms. E.g. spec implementation.
	Weight int64

	// TODO store extra data in dag itself, for different fork-choice implementations?
	Extra interface{}

}