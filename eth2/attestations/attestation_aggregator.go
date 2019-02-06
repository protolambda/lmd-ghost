package attestations

import (
	"lmd-ghost/eth2/attestations/attestation"
	"lmd-ghost/eth2/common"
)

type SlotLookupFn func(blockHash common.Hash256) uint64

type AggregatedAttestation struct {

	Target common.Hash256
	Weight int64
	// Remember previous weight, if prev != current, then there was a change,
	// only the dag can make an update, and set prev to current. Effectively like a "dirty" flag.
	PrevWeight int64

	// TODO: could contain a list signatures or something.
}

func NewAggregatedAttestation(target common.Hash256) *AggregatedAttestation {
	res := new(AggregatedAttestation)
	res.Target = target
	return res
}

func (at *AggregatedAttestation) RemoveAttestation(atOut *attestation.Attestation) {
	// TODO: remove attester from signers list

	at.Weight -= int64(atOut.Weight)
}

func (at *AggregatedAttestation) AddAttestation(atIn *attestation.Attestation) {
	// TODO: add attester to signers list

	at.Weight += int64(atIn.Weight)
}

type AttestationsAggregator struct {

	// aggregation: target -> sum of all attestations
	LatestAggregates map[common.Hash256]*AggregatedAttestation
	// lookup: validator -> target + weight contributed by validator
	LatestTargets map[common.ValidatorID]*attestation.Attestation

	SlotLookup SlotLookupFn
}

func NewAttestationsAggregator(slotLookup SlotLookupFn) *AttestationsAggregator {
	res := &AttestationsAggregator{
		LatestAggregates: make(map[common.Hash256]*AggregatedAttestation),
		LatestTargets: make(map[common.ValidatorID]*attestation.Attestation),
		SlotLookup: slotLookup,
	}
	return res
}

func (agor *AttestationsAggregator) createAgIfNonExists(key common.Hash256) *AggregatedAttestation {
	newAg, knownAg := agor.LatestAggregates[key]
	// check if we have to create a new aggregate
	if !knownAg {
		newAg = NewAggregatedAttestation(key)
		agor.LatestAggregates[key] = newAg
	}
	return newAg
}

func (agor *AttestationsAggregator) AttestationIn(atIn *attestation.Attestation) {
	prevContrib, hasPrevContrib := agor.LatestTargets[atIn.Attester]
	if hasPrevContrib {

		prevAg := agor.LatestAggregates[prevContrib.BeaconBlockRoot]
		if agor.SlotLookup(prevContrib.BeaconBlockRoot) > agor.SlotLookup(atIn.BeaconBlockRoot) {
			// We're going to ignore it. Too old, it's not later.
			return
		}

		// if the target changed, we move it to another aggregate
		if prevContrib.BeaconBlockRoot != atIn.BeaconBlockRoot {

			// remove from old aggregate
			prevAg.RemoveAttestation(atIn)

			// add to new aggregate, create aggregate if it does not exist yet
			newAg := agor.createAgIfNonExists(atIn.BeaconBlockRoot)
			newAg.AddAttestation(atIn)

			// update target
			agor.LatestTargets[atIn.Attester] = atIn
			return
		} else if atIn.Weight != prevContrib.Weight {
			// if only just the weight changed, we update just the weight + latest target

			// add the change in our contribution to the bigger total
			prevAg.Weight += int64(atIn.Weight) - int64(prevContrib.Weight)

			// update target
			agor.LatestTargets[atIn.Attester] = atIn
			return
		} else {
			// False alarm, nothing has changed
			return
		}
	} else {
		// add to new aggregate, create aggregate if it does not exist yet
		newAg := agor.createAgIfNonExists(atIn.BeaconBlockRoot)
		newAg.AddAttestation(atIn)

		// update target
		agor.LatestTargets[atIn.Attester] = atIn
	}
}

func (agor *AttestationsAggregator) Cleanup() {
	aliveTargets := make(map[common.Hash256]bool)
	for _, v := range agor.LatestTargets {
		aliveTargets[v.BeaconBlockRoot] = true
	}
	for k, v := range agor.LatestAggregates {
		// Check if aggregate is unprocessed; in this case it doesn't matter if it's a current target or not,
		//  it needs to be processed first.
		// So: if it's processed, or not an alive target, then delete it
		if v.PrevWeight == v.Weight || !aliveTargets[k] {
			// safe in Go
			delete(agor.LatestAggregates, k)
		}
	}
}
