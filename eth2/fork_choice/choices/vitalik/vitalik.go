package vitalik

import (
	"lmd-ghost/eth2/dag"
)

/*

Note that Vitalik's algorithm is a port (with some updates) from the original (written in Python) in the Ethereum research repo, which is also licensed to MIT, but to Vitalik.
Original of Vitalik can be found here: https://github.com/ethereum/research/blob/master/ghost/ghost.py

 */

type CacheKey [32 + 4]uint8

// Trick to get a quick conversion array, gets the log of a number
const logzLen = 100000
var logz = [logzLen]uint8{0, 0}
func init() {
	for i := 2; i < logzLen; i++ {
		logz[i] = logz[i / 2] + 1
	}
}

/// Vitaliks optimized version of the spec implementation
/// Orignal python version here: https://github.com/ethereum/research/blob/master/ghost/ghost.py
type VitaliksOptimizedLMDGhost struct {

	dag *dag.BeaconDag

	latestScores map[*dag.DagNode]int64

	cache map[CacheKey]*dag.DagNode

	// slot -> block-ref -> ancestor
	ancestors [16]map[*dag.DagNode]*dag.DagNode

	maxKnownHeight uint64
}

func NewVitaliksOptimizedLMDGhost(d *dag.BeaconDag) dag.ForkChoice {
	res := &VitaliksOptimizedLMDGhost{
		dag: d,
		latestScores: make(map[*dag.DagNode]int64),
		cache: make(map[CacheKey]*dag.DagNode),
		ancestors: [16]map[*dag.DagNode]*dag.DagNode{},
		maxKnownHeight: 0,
	}
	for i := uint8(0); i < 16; i++ {
		res.ancestors[i] = make(map[*dag.DagNode]*dag.DagNode)
	}
	return res
}

func (gh *VitaliksOptimizedLMDGhost) ApplyScoreChanges(changes []dag.ScoreChange) {
	for _, v := range changes {
		if v.Target.Slot >= gh.dag.Finalized.Slot {
			gh.latestScores[v.Target] += v.ScoreDelta
		}
	}
	// delete targets that have a 0 score
	for k, v := range gh.latestScores {
		if v == 0 {
			// deletion during map iteration, safe in Go
			delete(gh.latestScores, k)
		}
	}
}

func (gh *VitaliksOptimizedLMDGhost) OnNewNode(block *dag.DagNode) {
	startHeight := gh.dag.Finalized.Height
	// update the ancestor data (used for logarithmic lookup)
	for i := uint8(0); i < 16; i++ {
		if (block.Height - startHeight) % (1 << i) == 0 {
			gh.ancestors[i][block] = block.Parent
		} else {
			gh.ancestors[i][block] = gh.ancestors[i][block.Parent]
		}
	}

	// update maximum known slot
	if block.Height > gh.maxKnownHeight {
		gh.maxKnownHeight = block.Height
	}
}

/// Similar to the spec get_ancestor,
/// but using height instead of slot numbers to enable skipping ahead logarithmically, and with caching.
func (gh *VitaliksOptimizedLMDGhost) getAncestor(block *dag.DagNode, height uint64) *dag.DagNode {

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


func (gh *VitaliksOptimizedLMDGhost) getPowerOf2Below(x uint64) uint64 {
	// simply logz it, and 2^e this, to get the closes power of 2
	return 1 << logz[x]
}

func (gh *VitaliksOptimizedLMDGhost) getClearWinner(latestVotes map[*dag.DagNode]int64, height uint64) *dag.DagNode {
	// get the total vote count at this height
	totalVoteCount := int64(0)
	// map of vote-counts for every hash at this height
	atHeight := make(map[*dag.DagNode]int64)
	for t, v := range latestVotes {
		anc := gh.getAncestor(t, height)
		if anc != nil {
			atHeight[anc] = atHeight[anc] + v
			totalVoteCount += v
		}
	}
	for k, v := range atHeight {
		if v >= totalVoteCount / 2 {
			return k
		}
	}
	return nil
}

func (gh *VitaliksOptimizedLMDGhost) OnPrune() {
	minSlot := gh.dag.Finalized.Slot
	// prune old latest_scores
	for k := range gh.latestScores {
		if k.Slot < minSlot {
			delete(gh.latestScores, k)
		}
	}
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

func (gh *VitaliksOptimizedLMDGhost) HeadFn() *dag.DagNode {
	// Trick: At first we consider all targets (latest attestations), but later we start forgetting attestations
	//  that do not affect the remaining path-finding from start to head.
	// Modification from original: we keep track of total attestation-score per target block, instead of all attestations.
	// Hence, we can just copy the map.
	latestVotes := make(map[*dag.DagNode]int64)
	for t, w := range gh.latestScores {
		// Copy weight
		latestVotes[t] = w
	}
	head := gh.dag.Justified
	for {
		// short var "c": head.Children
		if len(head.Children) == 0 {
			return head
		}
		// Trick: check every depth for a clear 50% winner. This enables us to skip ahead towards the leafs of the tree.
		// And do so from leaf-level, back towards 0, to get the most out of this trick.
		// But not the very end, as this will likely not have a majority vote.
		step := gh.getPowerOf2Below(gh.maxKnownHeight - head.Height) / 2
		for step > 0 {
			possibleClearWinner := gh.getClearWinner(latestVotes, (head.Height - (head.Height % step) + step)
			if possibleClearWinner != nil {
				head = possibleClearWinner
				break
			}
			// Go back logarithmically
			step /= 2
		}

		if step > 0 {
			// nothing
		} else if len(head.Children) == 1 {
			// Another trick: if there's only 1 child, then you don't have to do any fork-choice at all, just pick it.
			// Dubbed a "only-child fast-path"
			head = head.Children[0]
		} else {
			// This process is similar to getVoteCount in the spec implementation,
			//  but we add up votes for every child with just 1 iteration through all latest-votes.
			childScores := make(map[*dag.DagNode]int64)
			for t, w := range latestVotes {
				if child := gh.getAncestor(t, head.Height + 1); child != nil {
					childScores[child] += w
				}
			}

			// Choose the best child
			// Mod from the original implementation, that did something with the hashes, for binary LMD-GHOST.
			bestItem := head.Children[0]
			var bestScore int64 = 0
			for child, childScore := range childScores {
				if childScore > bestScore {
					bestScore = childScore
					bestItem = child
				}
			}

			head = bestItem
		}

		// No definitive head has been found yet, continue path-finding, after doing some post-processing for this round.

		// Post-process; optimize the graph by removing votes that do not belong to the current head.
		deletes := make([]*dag.DagNode, 0)
		for k := range latestVotes {
			if anc := gh.getAncestor(k, head.Height); anc == nil || anc != head {
				deletes = append(deletes, k)
			}
		}
		for _, k := range deletes {
			delete(latestVotes, k)
		}
	}
}
