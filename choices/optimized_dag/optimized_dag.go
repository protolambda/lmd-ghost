package optimized_dag

import "lmd-ghost/sim"

/// A simple take on using a DAG for the fork-choice.
/// Stores entries in DAG, but re-propagates target votes every time the head is computed.
type SimpleDagLMDGhost struct {

	chain *sim.SimChain

}

func (gh *SimpleDagLMDGhost) AttestIn(hash256 sim.Hash256) {

}

func (gh *SimpleDagLMDGhost) BlockIn(hash256 sim.Hash256) {

}

func (gh *SimpleDagLMDGhost) HeadFn() sim.Hash256 {
	// TODO
	return sim.Hash256{}
}
