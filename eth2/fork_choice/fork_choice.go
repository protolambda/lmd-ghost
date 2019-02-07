package fork_choice

import (
	"lmd-ghost/eth2/dag"
)

type ScoreChange struct {
	Target *dag.DagNode
	ScoreDelta int64
}

type ForkChoice interface {
	OnNewNode(node *dag.DagNode)
	ApplyScoreChanges(changes []ScoreChange)
	OnStartChange(newStart *dag.DagNode)
	HeadFn() *dag.DagNode
}

type InitForkChoice func(dag *dag.BeaconDag) ForkChoice
