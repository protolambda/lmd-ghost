package viz

import (
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"lmd-ghost/sim"
	"os"
)

func CreateVizGraph(path string, chain *sim.SimChain) {
	writeNodesCSV(path + ".nodes.csv", chain)
	writeEdgesCSV(path + ".edges.csv", chain)
}

func check(err error, msg string) {
	if err != nil {
		panic(err)
	}
}

// encodes hash-256 to a hexadecimal string, 64 chars, no "0x" prefix
func hashToHexStr(hash sim.Hash256) string {
	dst := make([]byte, hex.EncodedLen(len(hash)))
	hex.Encode(dst, hash[:])
	return string(dst)
}

func writeNodesCSV(path string, chain *sim.SimChain) {
	file, err := os.Create(path)
	check(err, "Could not create nodes-CSV file")

	writer := csv.NewWriter(file)

	check(writer.Write([]string{"ID","Label","Slot","x","Proposer","BlockType"}), "failed to write nodes-CSV header")

	for hash, block := range chain.Blocks {
		id := hashToHexStr(hash)
		blockType := "normal"
		if hash == chain.Head {
			blockType = "head"
		} else if hash == chain.Justified {
			blockType = "justified"
		}
		check(writer.Write([]string{
			id, id, // id and label
			fmt.Sprintf("%d", block.Slot),
			fmt.Sprintf("%d", block.Slot + 1),// x: slot + 1, graphing software want coordinates 1 - N ...
			fmt.Sprintf("%d", block.Proposer),
			blockType,
			// TODO maybe also add votes to graph?
			//  (Problem: would require protolambda LMD-GHOST version, doesn't work for others)
		}), "failed to write CSV node for block " + id)
	}
	writer.Flush()
	check(file.Close(), "could not close nodes-CSV file")
}

func writeEdgesCSV(path string, chain *sim.SimChain) {
	file, err := os.Create(path)
	check(err, "Could not create edges-CSV file")

	writer := csv.NewWriter(file)

	check(writer.Write([]string{"Source","Target"}), "failed to write edges-CSV header")

	for hash, block := range chain.Blocks {
		// TODO we could also mark blocks in the path from justified <-> head,
		//  if we we're using the protolambda LMD-GHOST version
		id := hashToHexStr(hash)
		parentId := hashToHexStr(block.ParentHash)
		check(writer.Write([]string{parentId, id}), "failed to write CSV edge for block " + id)
	}

	writer.Flush()
	check(file.Close(), "could not close edges-CSV file")
}
