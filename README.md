# LMD-GHOST 

Comparison of the different LMD-GHOST implementations, by @protolambda.

Implementations:

- Naive but readable spec implementation, original in Python/pseudocode [here](https://github.com/ethereum/eth2.0-specs/blob/master/specs/core/0_beacon-chain.md#beacon-chain-fork-choice-rule)
- Optimized LMD-ghost by Vitalik Buterin, original in Python [here](https://github.com/ethereum/research/blob/master/ghost/ghost.py)
- Cached spec, similar to the spec, but with the cache feature from the implementation by Vitalik.
- Simple attestation-backpropagation DAG, by @protolambda.
- Stateful sum-vote DAG with propagation cut-off, by @protolambda.
- Suggestions welcome!

TODO: graph results, for different parameter sets.

