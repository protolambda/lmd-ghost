package fork_choice

import (
	"lmd-ghost/eth2/dag"
)

type ScoreChange struct {
	Target *dag.DagNode
	ScoreDelta int64
}

type ForkChoice interface {
	SetDag(dag *dag.BeaconDag)
	NodeIn(block *dag.DagNode)
	ApplyScoreChanges(changes []ScoreChange)
	StartIn(newStart *dag.DagNode)
	HeadFn() *dag.DagNode
}

type ConstructForkChoice func() ForkChoice
