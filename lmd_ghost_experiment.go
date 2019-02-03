package main

import (
	"fmt"
	"lmd-ghost/choices/cached"
	"lmd-ghost/choices/protolambda"
	"lmd-ghost/choices/simple_back_prop"
	"lmd-ghost/choices/spec"
	"lmd-ghost/choices/vitalik"
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

var forkRules = map[string]sim.GetForkChoice {
	"spec": spec.NewSpecLMDGhost,
	"vitalik": vitalik.NewVitaliksOptimizedLMDGhost,
	"cached": cached.NewCachedLMDGhost,
	"simple-back-prop": simple_back_prop.NewSimpleBackPropLMDGhost,
	"protolambda": protolambda.NewProtolambdaLMDGhost,
}


// TODO parametrize latency, simulated attestations per block, and slot-skip
func runSim(blocks int, validatorCount int, name string) {
	simName := fmt.Sprintf("%s__%d_blocks__%d_validators", name, blocks, validatorCount)
	defer track(runningtime(simName))
	getForkChoice := forkRules[name]
	forkChoice := getForkChoice()
	chain := sim.NewChain(validatorCount, forkChoice)
	forkChoice.SetChain(chain)
	logInterval := blocks / 20
	for n := 0; n < blocks; n++ {
		if n % logInterval == 0 {
			log.Printf("total %d blocks, head at slot: %d\n", len(chain.Blocks), chain.Blocks[chain.Head].Slot)
		}
		chain.SimNewBlock()
		for a := 0; a < 100; a++ {
			chain.SimNewAttestation()
		}
		chain.UpdateHead()
	}
	viz.CreateVizGraph("out/" + simName, chain)
}

func main()  {
	runSim(10000, 64*10, "protolambda")
}
