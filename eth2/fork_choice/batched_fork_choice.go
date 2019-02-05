package fork_choice

import (
	"lmd-ghost/eth2/block"
	"lmd-ghost/eth2/common"
	"lmd-ghost/eth2/dag"
	"lmd-ghost/eth2/data/attestation"
)

type LatestTarget struct {
	Slot uint64
	BlockRoot common.Hash256
	IndexInBatch uint64
}

type BatchedForkChoice struct {

	forkChoice ForkChoice

	// map of target -> aggregate attestation
	batches map[common.Hash256]*attestation.Attestation

	// the latest target of each validator that is participating in one of the batched attestations.
	latestTargets map[common.ValidatorID]LatestTarget
}

func NewBatchedForkChoice(forkChoice ForkChoice) *BatchedForkChoice {
	if forkChoice == nil {
		// TODO should we error in this decorater pattern?
		return nil
	}
	res := new(BatchedForkChoice)
	res.forkChoice = forkChoice

	// init batches map, and targets map
	res.Cleanup()

	return res
}


func (b *BatchedForkChoice) SetDag(dag *dag.BeaconDag) {
	b.forkChoice.SetDag(dag)
}

func (b *BatchedForkChoice) BlockIn(block *block.BeaconBlock) {
	b.forkChoice.BlockIn(block)
}

// ditches the old batches and target mappings, and initializes new ones.
func (b *BatchedForkChoice) Cleanup() {
	b.batches = make(map[common.Hash256]*attestation.Attestation)
	b.latestTargets = make(map[common.ValidatorID]LatestTarget)
}

func (b *BatchedForkChoice) addWeightedAttester(wa *attestation.WeightedAttester, root common.Hash256) {
	batch, batchExists := b.batches[root]
	// check if we have to create a batch first.
	if !batchExists {
		batch = &attestation.Attestation{
			Slot: wa.Slot,
			BeaconBlockRoot: root,
			WeightedAttesters: make([]*attestation.WeightedAttester, 0),
			SumWeight: 0,
		}
		b.batches[root] = batch
	}
	// get the index of the to-be-appended weighted-attester
	batchIndex := uint64(len(batch.WeightedAttesters))
	// add it to the batch
	batch.WeightedAttesters = append(batch.WeightedAttesters, wa)
	batch.SumWeight += wa.Weight
	// We're adding a new attestation
	b.latestTargets[wa.Attester] = LatestTarget{
		Slot: wa.Slot,
		BlockRoot: root,
		IndexInBatch: batchIndex,
	}
}

func (b *BatchedForkChoice) AttestationIn(atIn *attestation.Attestation) {
	for _, wa := range atIn.WeightedAttesters {
		latestTarget, hasPrev := b.latestTargets[wa.Attester]
		if hasPrev {
			if latestTarget.Slot < atIn.Slot {
				batch, ok := b.batches[latestTarget.BlockRoot]
				if !ok {
					panic("Inconsistent data: latest-targets should always map to a valid batch. Are you doing things concurrently?")
				}
				prevAttestestation := batch.WeightedAttesters[latestTarget.IndexInBatch]
				if prevAttestestation.Slot > wa.Slot {
					// We're going to ignore it. The batch may have a lower slot threshold,
					// but it includes an attestation for the attester that is newer than atIn.
					continue
				}

				// if the target changed, we move it to the new batch (aggregate attestation)
				if latestTarget.BlockRoot != atIn.BeaconBlockRoot {
					// Remove from old batch
					// Simply put the last attestation here, and shorten the length by 1
					lastIndex := len(batch.WeightedAttesters) - 1
					lastAttestation := batch.WeightedAttesters[lastIndex]
					prevAttestestation.Attester = lastAttestation.Attester
					prevAttestestation.Weight = lastAttestation.Weight
					batch.WeightedAttesters = batch.WeightedAttesters[:lastIndex]

					// Add to new batch
					// we don't return here, it's added like it's new
					b.addWeightedAttester(wa, atIn.BeaconBlockRoot)

					// update target
					latestTarget.Slot = atIn.Slot
					latestTarget.BlockRoot = atIn.BeaconBlockRoot
					continue
				} else if wa.Weight != prevAttestestation.Weight {
					// if only just the weight changed, we update just the weight + sum + slot

					// We're modifying the attestation weight!
					delta := prevAttestestation.Weight
					prevAttestestation.Weight = wa.Weight
					batch.SumWeight -= delta

					// update target
					latestTarget.Slot = atIn.Slot
					continue
				} else {
					// False alarm, nothing has changed
					continue
				}
			} else {
				// We're ignoring the attestation! It's too old!
				// (we prefer the first observed observation over a newer, if the slot height is equal)
				continue
			}
		} else {
			b.addWeightedAttester(wa, atIn.BeaconBlockRoot)
			continue
		}
	}
}

func (b *BatchedForkChoice) PushAttestationBatch() {
	// make batched attestations effective by piping in all batched attestations, and cleaning up
	for _, v := range b.batches {
		b.forkChoice.AttestationIn(v)
	}
	b.Cleanup()
}

func (b *BatchedForkChoice) SetStart(blockHash common.Hash256) {
	b.forkChoice.SetStart(blockHash)
}

func (b *BatchedForkChoice) HeadFn() common.Hash256 {
	// apply batched attestations
	b.PushAttestationBatch()
	return b.forkChoice.HeadFn()
}
