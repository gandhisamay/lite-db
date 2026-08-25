// Package sst holds the logic for sst implementation
package sst

import (
	"encoding/binary"

	"github.com/gandhisamay/lite-db/memtable"
)

func WriteToSST(mem *memtable.Memtable) {
	// in progress
}

func buildBytePayload(entry memtable.Record) []byte {
	buffer := make([]byte, 9+len(entry.Key)+len(entry.Value))

	binary.LittleEndian.PutUint32(buffer[:4], uint32(len(entry.Key)))

	if entry.Deleted {
		binary.LittleEndian.PutUint32(buffer[4:8], 0)
		buffer[8] = 1
	} else {
		binary.LittleEndian.PutUint32(buffer[4:8], uint32(len(entry.Value)))
		buffer[8] = 0
	}

	copy(buffer[9:9+len(entry.Key)], []byte(entry.Key))

	if !entry.Deleted {
		copy(buffer[9+len(entry.Key):], []byte(entry.Value))
	}

	return buffer
}
