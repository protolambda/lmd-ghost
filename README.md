# LMD-GHOST 

Comparison of the different LMD-GHOST implementations, by @protolambda.

Implementations:

- Naive but readable spec implementation, original in Python/pseudocode [here](https://github.com/ethereum/eth2.0-specs/blob/master/specs/core/0_beacon-chain.md#beacon-chain-fork-choice-rule)
- Cached spec, similar to the spec, but with the cache feature from the implementation by Vitalik.
- Optimized LMD-GHOST by Vitalik Buterin, slightly modified to fit spec, original in Python [here](https://github.com/ethereum/research/blob/master/ghost/ghost.py)
- Simple attestation-backpropagation DAG with best-target inherit, and majority cut-off, by @protolambda.
- Stateful DAG with partial propagation cut-off, by @protolambda.
- Sum-vote DAG by @protolambda, removed (but previously implemented) in favor of arbitrary attestation weights. See below.
- Suggestions welcome!


## Simulation

There is a simulation, in the `sim` package, with a config struct. The simulation is started from the `lmd_ghost_experiment.go` file, where the config values are specified.

Config: check out `sim/config.go` for documentation on all the different config options.

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
4. Scores are simple, but highly inefficient: We loop through every attestation, and then if the attestation has the child as an ancestor, it adds up to the total vote count for the child
5. The best child wins, and repeat 2 - 4 from there, until there is no children left.
6. Final winner is the head.

However, since aggregation is implemented, it's not as inefficient, since ancestor lookups are limited by the number of targets, instead of the number of attestations. 

### Cached spec: `cached`

Feature design by Vitalik, extracted to look at it separately by @protolambda.

So what if we could optimize that ancestor-lookup, to make the spec more efficient? Well, we can.

We do so with two measures:

- We create a lookup array for every single block that is considered. The lookup is logarithmic: we less and less big spans of blocks to find the ancestor
- We cache the results.

Now we can do the same as in the spec, but the ancestor lookup will be cheap.
Still, we have a problem where we consider *every* target, every time we make a choice between child-nodes, every step in the path from the justified block, to the eventual head.

#### Drawbacks

- The cache needs to be pruned
- The ancestor data needs to be pruned, and remainder needs to be updated every time.
 Either per finalization, or justification. Per-justification results in better lookup speeds. But the cost of doing an update may not be worth it.
- We have to use **heights** (a.k.a. distance from genesis in number of blocks), instead of slots.
 The implementation of Vitalik that this is extracted from is made to work for an older version of the spec,
 and does not seem to account for large gaps (i.e. multiple empty slots) between blocks.
- Pruning, updating, and tracking block-height adds quite a lot to complexity, caching is hard to get right. 

### Optimized LMD-GHOST by Vitalik Buterin: `vitalik`

Aside from the logarithmic ancestor lookup with caching, there is more optimizations designed by Vitalik.

These include:

1. Simple yet useful "fast-pathing": if a node has one child, then we don't have to lookup anything to know that this child will be the best child.
2. Majority voting, or a "clear winner": if a node at a given height has the majority of the votes at the height, then it does not matter what pre-curses it, it will be part of the highest-weighted path from the justified node to the head.
3. Logarithmic majority vote lookup: we don't want to check all heights, since this is costly, hence only lookup a few heights, in smaller steps. The ancestor-optimization from before is used to get votes at a height relatively quick.
4. Pruning: Given that everything is done in one go, and we do not want to consider all attestations at every depth, we prune away attestations for branches that are not part of the path towards the head.
5. Different from original: attestations for blocks are fully batched now, so there's no "latest-targets", but a "latest-scores". Computation is not limited by number of attestations, but number of blocks.

#### Drawbacks

All drawbacks from `cached`, a subset of the features in this implementation.

### Simple attestation propagation DAG: `simple_back_prop`

> Note: I'm calling it a DAG, since pruning could be slot-based, which would means nodes older than the justified slot are thrown out,
 splitting the initial tree in separate disconnected trees. Non-canonical components in the DAG will eventually be pruned.

Another way to handle weight changes is to focus more on the "DAG": every block is a node, with one input (parent block), and children (blocks having the node as parent).
So to find the best path, all we need to do is:
 
1. Propagate the weights from the target nodes, up to the root of the DAG (justified block).
2. Propagation is just adding the votes for the child-nodes, and updating the vote-count for the node having these children.

And then there are a few optimizations:

1. Best-target inheritance: When propagating up, we iterate over every child anyway, so we might as well remember the best-target of the node. (best-target being the hypothetical head, if this node would be part of the canonical chain).
2. Batch by-Height: When propagating up, we can batch changes per height. I.e. first process the maximum known height, then one height up, and so on: every node will only be touched once.
3. Majority cut-off: if we reach a node that is weighted higher than half of the DAG, we can stop all the work, and return its best-target. This prevents us from going back far in time in long chains.

The first optimization could be extended: we can propagate the best target (i.e. leaf node) as well. So we don't have to walk back, we can just pick the best leaf from the start.
This is done in the below more advanced implementation.

#### Drawbacks

- The per-slot/height processing consumes quite some memory (relatively), since we create a lot of maps.

#### Possible improvements

- Optimize you data-structure of unfinalized blocks to make per-height/slot iteration efficient. No need to re-construct it every time you want to find the head.
- Change to a per-height (i.e. distance to genesis) approach, instead of per slot. Effectively skip empty slots, making the implementation use less memory.

### State-ful DAG

This is the port of the below sum-vote DAG, to support aggregation of attestations, and arbitrary weighting.
However, due to the nature of the approach, most of the previous optimizations could not be ported.

This approach keeps track of the scores in the DAG itself, not in the computation function of the HEAD.

The primary benefit of this approach is the full information you have about *every block in the DAG*, without losing too much of the performance.
This information includes:
- LMD-GHOST weight
- Best child-node of any node in the dag.
- Best target-node of any node in the dag.

Head-lookup is simply `O(1)`: get the best target node for the starting node.

Pruning: If you're already pruning your collection of blocks (the DAG), you won't need to do additional work for pruning.

#### Drawbacks

To achieve `O(1)` during the "computation" of the head, insertions have to be processed to update the state.

Computation:
- Insertions of score changes are propagated back towards the root of the DAG. Batching/aggregation helps a lot here to reduce workload.
- Insertions of blocks trigger a propagation of the best-target.
 This cuts off as soon as a node in the path towards the root node is not the best child of its parent node.

#### Optimizations

The primary operation cost in the DAG is to check if a child node becomes the best child node, or the opposite (if the best child node is not the best anymore).
Although these operations themselves are optimized with quick swapping / timely checks, it is good to avoid.
By checking for a majority weight (i.e. `node Weight + change in Score > total dag Weight / 2 + possible change`) during back-propagation of weight changes,
 we can determine that the best-target will not change anymore during further propagation, and only the weight adjustment needs to be propagated further.

Often, an attestation moves up one or more blocks on the same branch, and may not change in weight.
This can be quickly recognized, as the best-target would not be different between the two attested nodes.
In such situation, it is easy to only apply a partial back-propagation, up to the point where both can be combined.
If the combined weight cancels out, further back-propagation can be avoided. But the combination can already be regarded as a win.

More advanced dissolving (i.e. combine two changes into none) optimizations from the previous version of this algorithm could not be ported, due to the aggregation going on,
 and weighting generally making dissolving less likely. However, one could still try to optimize for combining two changes at two different targets,
  into one change once the back-propagation of these targets hits the fork between the two.


### Sum-vote DAG with propagation cut-off

**Deprecated.** *Documented here for archiving purposes.*

Previously we did not consider weighting (e.g. balances of attesters) or aggregating (similar to weighting, but no single attester identity behind a vote change).

This approach is a big trade-off between computation-time when adding attestations, and determining the head.
The optimizations largely benefit from the small atomic changes in non-weighted non-aggregated LMD-GHOST.
To the point where the approach itself wins from any optimization you could make to the non-stateful approaches, in this version of LMD-GHOST.
Unfortunately, aggregation is critical for larger networks, hence this approach is deprecated.

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
Either die hard and keep building your own chain (change in attestations dissolved very quick),
 or stay as close as possible to the head (change in attestation dissolved reasonably quick).


### Your algorithm?

Suggestions are welcome! Please submit an issue or a PR.


## License

Algorithms licensed under MIT. See license file.

Also note that Vitalik's algorithm is a port (with some updates) from the original in the Ethereum research repo, which is also licensed to MIT, but to Vitalik.
Original of Vitalik can be found here: https://github.com/ethereum/research/blob/master/ghost/ghost.py



