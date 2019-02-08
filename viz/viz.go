package viz

import (
	"encoding/csv"
	"fmt"
	"lmd-ghost/eth2/chain"
	"log"
	"os"
)

func CreateVizGraph(path string, ch *chain.BeaconChain) {
	writeNodesCSV(path + ".nodes.csv", ch)
	writeEdgesCSV(path + ".edges.csv", ch)
}

func check(err error, msg string) {
	if err != nil {
		// TODO combine error + extra msg in a nice way
		log.Println(msg)
		panic(err)
	}
}

func writeNodesCSV(path string, ch *chain.BeaconChain) {
	file, err := os.Create(path)
	check(err, "Could not create nodes-CSV file")

	writer := csv.NewWriter(file)

	check(writer.Write([]string{"ID","Label","Slot","x","Proposer","BlockType"}), "failed to write nodes-CSV header")

	for hash := range ch.Dag.Nodes {
		id := hash.String()
		blockType := "normal"
		if hash == ch.Head {
			blockType = "head"
		} else if hash == ch.Dag.Justified.Key {
			blockType = "justified"
		} else if hash == ch.Dag.Finalized.Key {
			blockType = "finalized"
		}
		block, err := ch.Storage.GetBlock(hash)
		if err != nil {
			panic("Could not find block from DAG in storage")
		}
		check(writer.Write([]string{
			id, id, // id and label
			fmt.Sprintf("%d", block.Slot),
			fmt.Sprintf("%d", block.Slot + 1),// x: slot + 1, graphing software want coordinates 1 - N ...
			fmt.Sprintf("%d", block.Proposer),
			blockType,
			// TODO maybe also add votes to graph?
			//  (Problem: would require stateful LMD-GHOST version, doesn't work for others)
		}), "failed to write CSV node for block " + id)
	}
	writer.Flush()
	check(file.Close(), "could not close nodes-CSV file")
}

func writeEdgesCSV(path string, ch *chain.BeaconChain) {
	file, err := os.Create(path)
	check(err, "Could not create edges-CSV file")

	writer := csv.NewWriter(file)

	check(writer.Write([]string{"Source","Target"}), "failed to write edges-CSV header")

	for hash, block := range ch.Dag.Nodes {
		if block.Parent == nil {
			continue
		}
		// TODO we could also mark blocks in the path from justified <-> head,
		//  if we we're using the stateful LMD-GHOST version
		id := hash.String()
		parentId := block.Parent.Key.String()
		check(writer.Write([]string{parentId, id}), "failed to write CSV edge for block " + id)
	}

	writer.Flush()
	check(file.Close(), "could not close edges-CSV file")
}
