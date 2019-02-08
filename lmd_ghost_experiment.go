package main

import (
	"lmd-ghost/sim"
	"log"
	"time"
)


func main()  {
	config := &sim.SimConfig{
		ValidatorCount: 400,
		LatencyFactor: 0.4,
		SlotSkipChance: 0,
		BaseAttestWeight: 100000,
		MaxExtraAttestWeight: 10000,
		Blocks: 100,
		AttestationsPerBlock: 100,
		ForkChoiceRule: "vitalik",
	}

	s := sim.NewSimulation(config)
	name := config.String()

	log.Println("Start:	", name)
	startTime := time.Now()
	s.RunSim()
	endTime := time.Now()
	log.Println("End: ", name, "took", endTime.Sub(startTime))

	// Optional: write the network graph of the chain to a nodes and edges CSV
	// s.SaveNetworkGraph()

}
