package protolambda

import (
	"lmd-ghost/sim"
)

type Node struct {
	Parent *Node
	Block *sim.Block
	BestTarget *sim.Block
	// the first index is reserved for the "best" child
	Children []*Node
	// the index of this node, in the children list of the parent. If it's 0, then we're the best child
	IndexAsChild uint32
	Votes int32
}

func (n *Node) AddVote() {
	// increment the actual vote count
	n.Votes += 1
	//log.Printf("Added vote to %d, new count: %d\n", n.Block.Hash[0], n.Votes)

	if n.Parent != nil {
		// if we're not the best child of the parent, than we have a chance to become it.
		if n.IndexAsChild != 0 {
			if oldBest := n.Parent.Children[0]; n.Votes > oldBest.Votes {
				// the best has been overthrown, now swap it out
				n.Parent.Children[0] = n
				n.Parent.Children[n.IndexAsChild] = oldBest
				oldBest.IndexAsChild = n.IndexAsChild
				n.IndexAsChild = 0

				// Update the target of the parent, it inherits it from this node
				n.Parent.BestTarget = n.BestTarget
			}
		}

		// Propagate the vote
		n.Parent.AddVote()
	}
}

func (n *Node) RemoveVote() {
	// decrement the actual vote count
	n.Votes -= 1
	//log.Printf("Removed vote from %d, new count: %d\n", n.Block.Hash[0], n.Votes)

	if n.Votes < 0 {
		panic("Warning: removed too many votes!")
	}

	if n.Parent != nil {
		// if we're the best child of the parent, than we have a chance to lose our position to the second best.
		if n.IndexAsChild == 0 && len(n.Parent.Children) > 1 {
			// TODO: with some sorting it may be faster to keep track of the 2nd best child
			newBest := n
			for i := 1; i < len(n.Parent.Children); i++ {
				c := n.Parent.Children[i]
				if c.Votes > newBest.Votes {
					newBest = c
				}
			}
			// check if we actually had to update the new-best
			if newBest != n {
				// we have been overthrown, now swap in the new best
				n.Parent.Children[0] = newBest
				n.Parent.Children[newBest.IndexAsChild] = n
				n.IndexAsChild = newBest.IndexAsChild
				newBest.IndexAsChild = 0

				// Update the target of the parent, it inherits it from the new-best
				n.Parent.BestTarget = newBest.BestTarget
			}
		}

		// Propagate the vote removal
		n.Parent.RemoveVote()
	}
}

/// A simple take on using a DAG for the fork-choice.
/// Stores entries in DAG, but re-propagates target votes every time the head is computed.
type ProtolambdaLMDGhost struct {

	chain *sim.SimChain

	nodes map[sim.Hash256]*Node
}

func NewProtolambdaLMDGhost() sim.ForkChoice {
	res := &ProtolambdaLMDGhost{nodes: make(map[sim.Hash256]*Node)}
	return res
}

func (gh *ProtolambdaLMDGhost) SetChain(chain *sim.SimChain) {
	gh.chain = chain
	// add the origin block
	gh.BlockIn(chain.Blocks[chain.Justified])
}

func (gh *ProtolambdaLMDGhost) AttestIn(blockHash sim.Hash256, attester sim.ValidatorID) {
	// TODO combine add/remove for cutoff effect in recursive update

	// remove previous attest by validator, if there is any
	prevTarget, hasPrev := gh.chain.Targets[attester]
	//log.Println("=======================")

	//for k, n := range gh.nodes {
	//	log.Printf("%d: %d\n", k[0], n.Votes)
	//}
	//log.Printf("Attesation: %d prev %d, by %d\n", blockHash[0], prevTarget[0], attester)
	if prevTarget == blockHash {
		// nothing to do, attest does not change vote
		return
	}

	if hasPrev {
		gh.nodes[prevTarget].RemoveVote()
	}

	// add new attest by validator
	gh.nodes[blockHash].AddVote()
}

func (gh *ProtolambdaLMDGhost) BlockIn(block *sim.Block) {
	// *Note*: The data-structure is completely the same every time a block is added.
	//  Just one node more + a change to its parent.
	node := &Node{Block: block, Votes: 0}
	if block.Slot != 0 {
		node.Parent = gh.nodes[block.ParentHash]
		node.IndexAsChild = uint32(len(node.Parent.Children))
		node.Parent.Children = append(node.Parent.Children, node)
		// If this is the only/first node that is added,
		//  then it does not need attestations, it will just be the new target.
		if len(node.Parent.Children) == 1 {
			// propagate the new best-target up, as far as necessary
			p := node.Parent
			c := node
			for p != nil {
				if c.IndexAsChild == 0 {
					p.BestTarget = block
					c = p
					p = p.Parent
				} else {
					break
				}
			}
		}
	}
	// best end-target is the block itself
	node.BestTarget = block
	// expected branch factor is 2 (??), capacity of 8 should be fine? (TODO)
	node.Children = make([]*Node, 0, 8)
	// Add the node to the collection
	gh.nodes[block.Hash] = node
}

func (gh *ProtolambdaLMDGhost) HeadFn() sim.Hash256 {
	// All the work has already been done, just pick the best-target of the root node.
	// *Bonus*: And this works for *every* node in the graph!
	// Changing the root is costless (except for pruning away old nodes, which also needs to be done for other algos)
	return gh.nodes[gh.chain.Justified].BestTarget.Hash
}
