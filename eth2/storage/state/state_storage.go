package state

import (
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/state"
)

// Note: state-storage only saves post-states for blocks, nothing is saved for empty slots.
type StateStorage struct {

	states map[common.Hash256]*state.BeaconState

}

func (st *StateStorage) Init() {
	st.states = make(map[common.Hash256]*state.BeaconState)
}

// returns the post-state of a block. Changes are to the state are not persisted without PutPostState()
func (st *StateStorage) GetPostState(blockHash common.Hash256) (*state.BeaconState, error) {
	// TODO: this is not sufficient, it must be a copy / not affect original.
	return st.states[blockHash], nil
}

func (st *StateStorage) PutPostState(blockHash common.Hash256, state *state.BeaconState) error {
	st.states[blockHash] = state
	return nil
}

func (st *StateStorage) HasPostState(stateHash common.Hash256) (bool, error) {
	_, ok := st.states[stateHash]
	return ok, nil
}

