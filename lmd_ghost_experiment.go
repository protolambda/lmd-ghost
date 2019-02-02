package main

import (
	"lmd-ghost/choices/cached"
	"lmd-ghost/choices/protolambda"
	"lmd-ghost/choices/simple_back_prop"
	"lmd-ghost/choices/spec"
	"lmd-ghost/choices/vitalik"
	"lmd-ghost/sim"
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

// TODO parametrize validator count and simulated attestations per block
func runSim(blocks int, getForkChoice sim.GetForkChoice) {
	forkChoice := getForkChoice()
	chain := sim.NewChain(64*20, forkChoice)
	forkChoice.SetChain(chain)
	interval := blocks / 20
	for n := 0; n < blocks; n++ {
		if n % interval == 0 {
			log.Printf("total %d blocks, head at slot: %d\n", len(chain.Blocks), chain.Blocks[chain.Head].Slot)
		}
		if len(chain.Blocks) > blocks {
			panic("Too many blocks")
		}
		chain.SimNewBlock()
		for a := 0; a < 100; a++ {
			chain.SimNewAttestation()
		}
		chain.UpdateHead()
	}
}

func RunSpec() {
	defer track(runningtime("spec"))
	runSim(1000, spec.NewSpecLMDGhost)
}

func RunVitalik() {
	defer track(runningtime("vitalik"))
	runSim(50000, vitalik.NewVitaliksOptimizedLMDGhost)
}

func RunCached() {
	defer track(runningtime("cached"))
	runSim(3000, cached.NewCachedLMDGhost)
}

func RunSimpleBackProp() {
	defer track(runningtime("simple-back-prop"))
	runSim(20000, simple_back_prop.NewSimpleBackPropLMDGhost)
}

func RunProtolambda() {
	defer track(runningtime("protolambda"))
	runSim(10000, protolambda.NewProtolambdaLMDGhost)
}


func main()  {
	//RunSpec()
	//RunVitalik()
	//RunCached()
	//RunSimpleBackProp()
	RunProtolambda()
}
