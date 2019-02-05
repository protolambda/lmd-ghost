package vitalik

import (
	"lmd-ghost/sim"
	"math/big"
)

/*

NOTE: This implementation is a port of the research work by vitalik, originally written in Python.
The MIT license does not apply to this work. Due to the lack of a license file in the research repository,
 you would have to ask Vitalik to use this algorithm under your own license.

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

	chain *sim.SimChain

	cache map[CacheKey]sim.Hash256

	// slot -> hash -> ancestor
	ancestors map[uint8]map[sim.Hash256]sim.Hash256

	maxKnownSlot uint32
}

func NewVitaliksOptimizedLMDGhost() sim.ForkChoice {
	res := new(VitaliksOptimizedLMDGhost)
	res.cache = make(map[CacheKey]sim.Hash256)
	res.ancestors = make(map[uint8]map[sim.Hash256]sim.Hash256)
	for i := uint8(0); i < 16; i++ {
		res.ancestors[i] = make(map[sim.Hash256]sim.Hash256)
	}
	return res
}

func (gh *VitaliksOptimizedLMDGhost) SetChain(chain *sim.SimChain) {
	gh.chain = chain
}

func (gh *VitaliksOptimizedLMDGhost) AttestIn(blockHash sim.Hash256, attester sim.ValidatorID) {
	// free, at cost of head function. Latest attestation map is maintained in the chain struct.
}

func (gh *VitaliksOptimizedLMDGhost) BlockIn(block *sim.Block) {
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

/// The spec get_ancestor, but with caching, and skipping ahead logarithmically
func (gh *VitaliksOptimizedLMDGhost) getAncestor(block *sim.Block, slot uint32) *sim.Block {

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

// Port of a rather strange looking function written by Vitalik.
// Best guess (@protolambda) is that the uniformness of the hashes is exploited to try and find a bias towards an entry
// that belongs to the highest-voted 50% of a random sampling, in a lot of cases. Enough to warrant a choice for it.
// Ask Vitalik for his own reasoning, lol.
func (gh *VitaliksOptimizedLMDGhost) chooseBestChild(votes map[sim.Hash256]float64) *sim.Block {
	bitmask := big.NewInt(0)
	for bit := int(255); bit >= 0; bit-- {
		zeroVotes := float64(0)
		oneVotes := float64(0)
		var singleCandidate *sim.Block
		hasNoSingleCandidate := false
		for k, v := range votes {
			candidateAsInt := new(big.Int)
			candidateAsInt.SetBytes(k[:])
			if new(big.Int).Rsh(candidateAsInt, uint(bit+1)).Cmp(bitmask) != 0 {
				continue
			}
			if new(big.Int).Rsh(candidateAsInt, uint(bit)).Bit(0) == 0 {
				zeroVotes += v
			} else {
				oneVotes += v
			}
			if singleCandidate == nil && !hasNoSingleCandidate {
				singleCandidate = gh.chain.Blocks[k]
			} else {
				hasNoSingleCandidate = true
			}
		}
		bitmask.Lsh(bitmask, 1)
		if oneVotes > zeroVotes {
			bitmask.SetBit(bitmask, 0, 1)
		} else {
			bitmask.SetBit(bitmask, 0, 0)
		}

		if singleCandidate != nil {
			return singleCandidate
		}
	}
	return nil
}

func (gh *VitaliksOptimizedLMDGhost) getPowerOf2Below(x uint32) uint32 {
	// simply logz it, and 2^e this, to get the closes power of 2
	return 1 << logz[x]
}

func (gh *VitaliksOptimizedLMDGhost) getClearWinner(latestVotes map[sim.Hash256]uint32, slot uint32) *sim.Block {
	// get the total vote count at this height
	totalVoteCount := uint32(0)
	// map of vote-counts for every hash at this height
	atHeight := make(map[sim.Hash256]uint32)
	for t, v := range latestVotes {
		tBlock := gh.chain.Blocks[t]
		anc := gh.getAncestor(tBlock, slot)
		if anc != nil {
			atHeight[anc.Hash] = atHeight[anc.Hash] + v
			totalVoteCount += v
		}
	}
	for k, v := range atHeight {
		if v >= totalVoteCount / 2 {
			return gh.chain.Blocks[k]
		}
	}
	return nil
}

func (gh *VitaliksOptimizedLMDGhost) HeadFn() sim.Hash256 {
	// Trick: At first we consider all targets (latest attestations), but later we start forgetting attestations
	//  that do not affect the remaining path-finding from start to head.
	latestVotes := make(map[sim.Hash256]uint32)
	for _, t := range gh.chain.Targets {
		// every attestation counts as 1 in latest spec (v0.1).
		// No balances involved, like in the original version of this lmd implementation of LMD-GHOST.
		latestVotes[t] = latestVotes[t] + 1
	}
	head := gh.chain.Blocks[gh.chain.Justified]
	for {
		// short var "c": head.Children
		if len(head.Children) == 0 {
			return head.Hash
		}
		// Trick: check every depth for a clear 50% winner. This enables us to skip ahead towards the leafs of the tree.
		// And do so from leaf-level, back towards 0, to get the most out of this trick.
		// But not the very end, as this will likely not have a majority vote.
		step := gh.getPowerOf2Below(gh.maxKnownSlot - head.Slot) / 2
		for step > 0 {
			possibleClearWinner := gh.getClearWinner(latestVotes, (head.Slot + gh.maxKnownSlot)/2)
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
			childVotes := make(map[sim.Hash256]float64)
			for _, c := range head.Children {
				childVotes[c.Hash] = 0.01
			}
			for t, v := range latestVotes {
				child := gh.getAncestor(gh.chain.Blocks[t], head.Slot + 1)
				if child != nil {
					childVotes[child.Hash] = childVotes[child.Hash] + float64(v)
				}
			}
			head = gh.chooseBestChild(childVotes)
		}

		// No definitive head has been found yet, continue path-finding, after doing some post-processing for this round.

		// Post-process; optimize the graph by removing votes that do not belong to the current head.
		deletes := make([]sim.Hash256, 0)
		for k := range latestVotes {
			if anc := gh.getAncestor(gh.chain.Blocks[k], head.Slot); anc == nil || anc.Hash != head.Hash {
				deletes = append(deletes, k)
			}
		}
		for _, k := range deletes {
			delete(latestVotes, k)
		}
	}
}
