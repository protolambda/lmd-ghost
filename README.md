# LMD-GHOST 

Comparison of the different LMD-GHOST implementations, by @protolambda.

Implementations:

- Naive but readable spec implementation, original in Python/pseudocode [here](https://github.com/ethereum/eth2.0-specs/blob/master/specs/core/0_beacon-chain.md#beacon-chain-fork-choice-rule)
- Cached spec, similar to the spec, but with the cache feature from the implementation by Vitalik.
- Optimized LMD-GHOST by Vitalik Buterin, original in Python [here](https://github.com/ethereum/research/blob/master/ghost/ghost.py)
- Simple attestation-backpropagation DAG, by @protolambda.
- Stateful sum-vote DAG with propagation cut-off, by @protolambda.
- Suggestions welcome!


## Simulation

There is a simulated chain, in the `sim` package, with some constants & parameters. And then there is a few more parameters in the usage of this chain in the `lmd_ghost_experiment.go` file.

The simulation code is a work-in-progress, we are discussing parameters in an issue on the specs repo, here: https://github.com/ethereum/eth2.0-specs/issues/570


## Network Graph

In the `viz` package is code to write the full simulated chain in two CSV files: one for nodes, one for edges.
This data can be imported to some graph visualizer such as Gephi.
Use a geo-layout based on slot number to get a starting point, afterwards you can apply other layout algorithms to separate nodes on the same height from eachother.

TODO: graph results, for different parameter sets. (after/during discussion which parameter sets should be considered).


## Implementations

### Spec implementation: `spec`

The spec is very straightforward:

1. Start at the justified block
2. Compare every child, to choose the best one
3. Comparison is based on scores
4. Scores are simple, but highly inefficient:
 -- We loop through every attestation
   ** If the attestation has the child as an ancestor, it adds up to the total vote count for the child
5. The best child wins, and repeat 2 - 4 from there, until there is no children left.
6. Final winner is the head.


### Cached spec: `cached`

Feature design by Vitalik, extracted to look at it separately by @protolambda.

So what if we could optimize that ancestor-lookup, to make the spec more efficient? Well, we can.

We do so with two measures:

- We create a lookup array for every single block that is considered. The lookup is logarithmic: we less and less big spans of blocks to find the ancestor
- We cache the results.

Now we can do the same as in the spec, but the ancestor lookup will be cheap. Still, we have a problem where we consider *every* attestation, every time we make a choice between child-nodes, every step in the path from the justified block, to the eventual head.


### Optimized LMD-GHOST by Vitalik Buterin: `vitalik`

Aside from the logarithmic ancestor lookup with caching, there is more optimizations designed by Vitalik.

These include:

1. Simple yet useful "fast-pathing": if a node has one child, then we don't have to lookup anything to know that this child will be the best child.
2. Majority voting, or a "clear winner": if a node at a given height has the majority of the votes at the height, then it does not matter what pre-curses it, it will be part of the highest-weighted path from the justified node to the head.
3. Logarithmic majority vote lookup: we don't want to check all heights, since this is costly, hence only lookup a few heights, in smaller steps. The ancestor-optimization from before is used to get votes at a height relatively quick.
4. Best-child determination based on some bitmask magic: unclear, and also considers balances. This was ported from the original python implementation, but may be outdated.
5. Pruning: Given that everything is done in one go, and we do not want to consider all attestations at every depth, we prune away attestations for branches that are not part of the path towards the head.


### Simple attestation propagation DAG: `simple_back_prop`

> Note: I'm calling it a DAG, since pruning could be slot-based, which would means nodes older than the justified slot are thrown out,
 splitting the initial tree in separate disconnected trees. Non-canonical components in the DAG will eventually be pruned.

Another way to handle votes is to focus more on the "DAG": every block is a node, with one input (parent block), and children (blocks having the node as parent).
So to find the best path, all we need to do is:
 
1. propagate the attestations from the leaf nodes, up to the root of the DAG (justified block).
2. Propagation is just adding the votes for the child-nodes, and updating the vote-count for the node having these children.

And then there are two optimizations:

1. When propagating up, we iterate over every child anyway, so we might as well remember the best child of the node.
2. When propagating up, we can batch changes per height. I.e. first process the maximum known height, then one height up, and so on: every node will only be touched once.

The first optimization could be extended: we can propagate the best target (i.e. leaf node) as well. So we don't have to walk back, we can just pick the best leaf from the start.
This is done in the below more advanced implementation.


### Stateful sum-vote DAG with propagation cut-off: `protolambda`

A different approach to LMD-GHOST by @protolambda.

The real trade-off between computation-time when adding attestations, and determining the head.

#### The benefits

- Instead of doing any work in the computation of the head, nothing is done, the lookup is basically free; `O(1)`.
- And as a side-effect of the data-structure being used, the same thing applies when you want to look-up the head for any other starting-point than the justified block, it's also `O(1)`.
- And the same also applies if you just want to know the best child of a block, also `O(1)`.
- This could be significant for pruning away data in a more efficient way, to be researched.

#### The trade-off

The trade-off is: we have to process attestations, and block-additions.
Luckily, since these are so small changes, they can be optimized heavily.

#### How it works

An attestation could mean two things:

1. The validator has never attested before, so it's effectively a `+1` on the tree.
2. The validator has a previous attestation target (which we are already forced to remember by the spec), which will be removed. This is effectively a `-1 +1`.

1 is very easy, but a bit costly: all we need to do is propagate a vote, and make sure to update the best-child/target on the way down to the root. This is `O(N)`.

2 seems complicated, but may actually be cheaper than 1! Thought about the `-1 +1`? Well, `-1 +1 = 0`.
The only thing is we have to find the point in the DAG where they dissolve each other.

This is rather easy: zig-zag between the two branches, propagating both the `-1` and `+1` one node back to the root, until we arrived at the same node.
At this point we can stop the zig-zag, since the total weight for this node does not change.

However, one thing may have changed (but most likely won't): the new branch (`+1`) could have overtaken the old one (`-1`).
This can be checked efficiently, and all we have to do is propagate this change back up the tree.
And even this may be cut-off, as we may arrive at an earlier fork where our branch is not the best choice: no target change from there!

Adding blocks is also slightly more costly, since we have to account for target changes; even without votes, if it's the only child, it will change the target.
Again, we can propagate this the same way as we propagated the best-target change in the `-1 +1` case, and cut-off at some point if the block is not the new head.

Overall this data-structure works really well when validators don't change their attestation by much, which is incentivized by the slashing conditions.
Either die hard and keep building your own chain, or stay as close as possible to the head.


### Your algorithm?

Suggestions are welcome! Please submit an issue or a PR.


## License

Algorithms licensed under MIT. See license file.

Also note that Vitalik's algorithm is a port (with some updates) from the original in the Ethereum research repo, which is also licensed to MIT, but to Vitalik.
Original of Vitalik can be found here: https://github.com/ethereum/research/blob/master/ghost/ghost.py



