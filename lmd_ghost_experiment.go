package main

import (
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

func main()  {
	config := &sim.SimConfig{
		ValidatorCount: 100,
		LatencyFactor: 0.8,
		SlotSkipChance: 0.4,
		BaseAttestWeight: 100000,
		MaxExtraAttestWeight: 10000,
		Blocks: 100,
		AttestationsPerBlock: 1000,
		ForkChoiceRule: "stateful",
	}

	s := sim.NewSimulation(config)

	defer track(runningtime(config.String()))
	s.RunSim()
}
