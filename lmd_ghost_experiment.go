package main

import (
	"fmt"
	"lmd-ghost/eth2/fork_choice"
	"lmd-ghost/eth2/fork_choice/choices/cached"
	"lmd-ghost/eth2/fork_choice/choices/simple_back_prop"
	"lmd-ghost/eth2/fork_choice/choices/spec"
	"lmd-ghost/eth2/fork_choice/choices/stateful"
	"lmd-ghost/eth2/fork_choice/choices/vitalik"
	"lmd-ghost/sim"
	"lmd-ghost/viz"
	"log"
	"time"
)

func runningtime(s string) (string, time.Time) {
	log.Println("Start:	", s)
	return s, time.Now()
}

func track(s string, startTime time.Time) {
	endTime := time.Now()
	log.Println("End:	", s, "took", endTime.Sub(startTime))
}

var forkRules = map[string]fork_choice.InitForkChoice {
	"spec": spec.NewSpecLMDGhost,
	"vitalik": vitalik.NewVitaliksOptimizedLMDGhost,
	"cached": cached.NewCachedLMDGhost,
	"simple-back-prop": simple_back_prop.NewSimpleBackPropLMDGhost,
	"stateful": stateful.NewStatefulLMDGhost,
}


// TODO parametrize latency, simulated attestations per block, and slot-skip
func runSim(blocks int, validatorCount int, attestationsPerBlock int, name string) {
	simName := fmt.Sprintf("%s__%d_blocks__%d_validators", name, blocks, validatorCount)
	defer track(runningtime(simName))
	initForkChoice := forkRules[name]
	s := sim.NewSimulation(validatorCount, initForkChoice)
	// log every 5% of the simulated amount of blocks
	logInterval := blocks / 20
	// update the head 10 times during attestation processing.
	headUpdateInterval := attestationsPerBlock / 10
	attestationCounter := 0
	for n := 0; n < blocks; n++ {
		if n % logInterval == 0 {
			log.Printf("total %d blocks, head at slot: %d, processed %d attestations.\n", len(s.Chain.Dag.Nodes), s.Chain.Dag.Nodes[s.Chain.Head].Slot, attestationCounter)
		}
		for a := 0; a < attestationsPerBlock; a++ {
			s.SimNewAttestation()
			if a % headUpdateInterval == headUpdateInterval - 1 {
				s.Chain.UpdateHead()
			}
		}
		attestationCounter += attestationsPerBlock
		// head will update after adding a block
		s.SimNewBlock()
	}
	viz.CreateVizGraph("out/" + simName, s.Chain)
}

func main()  {
	runSim(100, 64*10, 1000, "stateful")
}
