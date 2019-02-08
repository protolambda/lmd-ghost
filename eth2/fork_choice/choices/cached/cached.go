package cached

import (
	"lmd-ghost/eth2/dag"
)

type CacheKey [32 + 4]uint8

// Trick to get a quick conversion array, gets the log of a number
const logzLen = 10000
var logz = [logzLen]uint8{0, 0}
func init() {
	for i := 2; i < logzLen; i++ {
		logz[i] = logz[i / 2] + 1
	}
}

/// Just only the cache part of the implementation of Vitalik
type CachedLMDGhost struct {

	dag *dag.BeaconDag

	latestScores map[*dag.DagNode]int64

	cache map[CacheKey]*dag.DagNode

	// slot -> hash -> ancestor
	ancestors [16]map[*dag.DagNode]*dag.DagNode

}

func NewCachedLMDGhost(d *dag.BeaconDag) dag.ForkChoice {
	res := &CachedLMDGhost{
		dag:          d,
		latestScores: make(map[*dag.DagNode]int64),
		cache: make(map[CacheKey]*dag.DagNode),
		ancestors: [16]map[*dag.DagNode]*dag.DagNode{},
	}
	for i := uint8(0); i < 16; i++ {
		res.ancestors[i] = make(map[*dag.DagNode]*dag.DagNode)
	}
	return res
}

/// Similar to the spec get_ancestor,
/// but using height instead of slot numbers to enable skipping ahead logarithmically, and with caching.
func (gh *CachedLMDGhost) getAncestor(block *dag.DagNode, height uint64) *dag.DagNode {

	if height >= block.Height {
		if height > block.Height {
			return nil
		} else {
			return block
		}
	}

	// construct key
	cacheKey := CacheKey{}
	copy(cacheKey[:32], block.Key[:])
	cacheKey[32] = uint8(height >> 24)
	cacheKey[33] = uint8(height >> 16)
	cacheKey[34] = uint8(height >> 8)
	cacheKey[35] = uint8(height)

	// check cache
	if res, ok := gh.cache[cacheKey]; ok {
		// hit!
		return res
	}

	if x := gh.ancestors[logz[block.Height - height - 1]][block]; x == nil {
		panic("Ancestors data is invalid")
	}

	// this will be the output
	// skip ahead logarithmically to find the ancestor, and dive in recursively
	skipBlock := gh.ancestors[logz[block.Height - height - 1]][block]
	o := gh.getAncestor(skipBlock, height)

	if o.Height != height {
		panic("Found ancestor is at wrong height")
	}

	// cache this, so we never have to handle beyond this point again.
	gh.cache[cacheKey] = o

	return o
}

func (gh *CachedLMDGhost) ApplyScoreChanges(changes []dag.ScoreChange) {
	for _, v := range changes {
		gh.latestScores[v.Target] += v.ScoreDelta
	}
	// delete targets that have a 0 score
	for k, v := range gh.latestScores {
		if v == 0 {
			// deletion during map iteration, safe in Go
			delete(gh.latestScores, k)
		}
	}
}

func (gh *CachedLMDGhost) OnNewNode(block *dag.DagNode) {
	startHeight := gh.dag.Start.Height
	// update the ancestor data (used for logarithmic lookup)
	for i := uint8(0); i < 16; i++ {
		if (block.Height - startHeight) % (1 << i) == 0 {
			gh.ancestors[i][block] = block.Parent
		} else {
			gh.ancestors[i][block] = gh.ancestors[i][block.Parent]
		}
	}
}

func (gh *CachedLMDGhost) OnStartChange() {
	minSlot := gh.dag.Start.Slot
	// prune cache (based on slot), and re-init ancestor data for non-pruned data
	for k, v := range gh.cache {
		if v.Slot < minSlot {
			// deletion during iteration here is safe in Go
			delete(gh.cache, k)
		}
	}
	// prune away old ancestor data
	for _, ancMap := range gh.ancestors {
		for k, v := range ancMap {
			if v.Slot < minSlot {
				delete(ancMap, k)
			}
		}
	}
	// now update all ancestor data.
	for _, v := range gh.dag.Nodes {
		gh.OnNewNode(v)
	}
}

/// Retrieves the head by *recursively* looking for the highest voted block
//   at *every* block in the path from start to head.
func (gh *CachedLMDGhost) HeadFn() *dag.DagNode {
	// Minor difference:
	// Normally you would have to filter for the active validators, and get their targets.
	// We can just iterate over the values in the common-chain.
	// This difference only really matters when there's many validators inactive,
	//  and the client implementation doesn't store them separately.

	head := gh.dag.Start
	for {
		if len(head.Children) == 0 {
			return head
		}
		bestItem := head.Children[0]
		var bestScore int64 = 0
		for _, child := range head.Children {
			childVotes := gh.getVoteCount(child)
			if childVotes > bestScore {
				bestScore = childVotes
				bestItem = child
			}
		}
		head = bestItem
	}
}

func (gh *CachedLMDGhost) getVoteCount(block *dag.DagNode) int64 {
	totalWeight := int64(0)
	for target, weight := range gh.latestScores {
		if anc := gh.getAncestor(target, block.Height); anc != nil && anc == target {
			totalWeight += weight
		}
	}
	return totalWeight
}
