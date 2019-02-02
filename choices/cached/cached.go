package cached

import (
	"lmd-ghost/sim"
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

/// Just only the cache part of the implementation of vitalik
type CachedLMDGhost struct {

	chain *sim.SimChain

	cache map[CacheKey]sim.Hash256

	// slot -> hash -> ancestor
	ancestors map[uint8]map[sim.Hash256]sim.Hash256

	maxKnownSlot uint32
}

func (gh *CachedLMDGhost) SetChain(chain *sim.SimChain) {
	gh.chain = chain
}

func NewCachedLMDGhost() sim.ForkChoice {
	res := new(CachedLMDGhost)
	res.cache = make(map[CacheKey]sim.Hash256)
	res.ancestors = make(map[uint8]map[sim.Hash256]sim.Hash256)
	for i := uint8(0); i < 16; i++ {
		res.ancestors[i] = make(map[sim.Hash256]sim.Hash256)
	}
	return res
}

/// The spec get_ancestor, but with caching, and skipping ahead logarithmically
func (gh *CachedLMDGhost) getAncestor(block *sim.Block, slot uint32) *sim.Block {

	if slot >= block.Slot {
		if slot > block.Slot {
			return nil
		} else {
			return block
		}
	}

	// construct key
	cacheKey := CacheKey{}
	copy(cacheKey[:32], block.Hash[:])
	cacheKey[32] = uint8(slot >> 24)
	cacheKey[33] = uint8(slot >> 16)
	cacheKey[34] = uint8(slot >> 8)
	cacheKey[35] = uint8(slot)

	// check cache
	if res, ok := gh.cache[cacheKey]; ok {
		// hit!
		return gh.chain.Blocks[res]
	}

	if x := gh.chain.Blocks[gh.ancestors[logz[block.Slot - slot - 1]][block.Hash]]; x == nil {
		panic("Ancestors data is invalid")
	}

	// this will be the output
	// skip ahead logarithmically to find the ancestor, and dive in recursively
	skipHash := gh.ancestors[logz[block.Slot - slot - 1]][block.Hash]
	skipBlock := gh.chain.Blocks[skipHash]
	o := gh.getAncestor(skipBlock, slot)

	if o.Slot != slot {
		panic("Found ancestor is at wrong height")
	}

	// cache this, so we never have to handle beyond this point again.
	gh.cache[cacheKey] = o.Hash

	return o
}
func (gh *CachedLMDGhost) AttestIn(hash256 sim.Hash256) {
	// free, at cost of head-function.
}

func (gh *CachedLMDGhost) BlockIn(block *sim.Block) {
	// update the ancestor data (used for logarithmic lookup)
	for i := uint8(0); i < 16; i++ {
		if block.Slot % (1 << i) == 0 {
			gh.ancestors[i][block.Hash] = block.ParentHash
		} else {
			gh.ancestors[i][block.Hash] = gh.ancestors[i][block.ParentHash]
		}
	}

	// update maximum known slot
	if block.Slot > gh.maxKnownSlot {
		gh.maxKnownSlot = block.Slot
	}
}

/// Retrieves the head by *recursively* looking for the highest voted block
//   at *every* block in the path from start to head.
func (gh *CachedLMDGhost) HeadFn() sim.Hash256 {
	// Minor difference:
	// Normally you would have to filter for the active validators, and get their targets.
	// We can just iterate over the values in the sim-chain.
	// This difference only really matters when there's many validators inactive,
	//  and the client implementation doesn't store them separately.

	head := gh.chain.Blocks[gh.chain.Justified]
	for {
		if len(head.Children) == 0 {
			return head.Hash
		}
		bestItem := head.Children[0]
		var bestScore uint32 = 0
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

func (gh *CachedLMDGhost) getVoteCount(block *sim.Block) uint32 {
	count := uint32(0)
	for _, target := range gh.chain.Targets {
		if anc := gh.getAncestor(gh.chain.Blocks[target], block.Slot); anc != nil && anc.Hash == block.Hash {
			count++
		}
	}
	return count
}