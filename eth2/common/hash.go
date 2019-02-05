package common

import "encoding/hex"

// The effective identifier of a block
type Hash256 [32]uint8

// encodes hash-256 to a hexadecimal string, 64 chars, no "0x" prefix
func (h Hash256) String() string {
	dst := make([]byte, hex.EncodedLen(len(h)))
	hex.Encode(dst, h[:])
	return string(dst)
}

