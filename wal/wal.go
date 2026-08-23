// Package wal holds all the wal related logic
package wal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
)

// Record is the WAL-facing representation of a store write request.
type Record struct {
	Operation uint8
	Key       string
	Value     string
}

type Wal struct {
	file *os.File
	path string
}

func Open() (*Wal, error) {
	walFilePath := "data.wal"

	file, err := os.OpenFile(walFilePath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}

	return &Wal{
		file: file,
		path: walFilePath,
	}, nil
}

func (wal *Wal) Close() error {
	return wal.file.Close()
}

func (wal *Wal) Append(record Record) error {
	payload := recordPayload(record)

	buf := buildByteArrPayload(payload)

	_, err := wal.file.Write(buf)
	return err
}

func (wal *Wal) Fsync() error {
	return wal.file.Sync()
}

func (wal *Wal) Replay(fn func([]byte)) error {
	// now we read the logs from the wal file, and then we can process it.
	if wal.file == nil {
		return errors.New("wal not yet initialised")
	}

	for {
		var lengthPrefix [4]byte

		_, err := io.ReadFull(wal.file, lengthPrefix[:])

		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err
		}

		length := binary.LittleEndian.Uint32(lengthPrefix[:])

		payload := make([]byte, length)

		_, err = io.ReadFull(wal.file, payload)
		if err != nil {
			return err
		}

		var checkSumStored [4]byte

		_, err = io.ReadFull(wal.file, checkSumStored[:])
		if err != nil {
			return err
		}
		checkSumStoredBinary := binary.LittleEndian.Uint32(checkSumStored[:])

		checkSumComputed := crc32.ChecksumIEEE(payload)

		if checkSumStoredBinary != checkSumComputed {
			// we have an issue
			message := fmt.Sprintf("data corrupted for payload: %s and checksum: %d", payload, checkSumStoredBinary)
			return errors.New(message)
		}

		fn(payload)
	}
}

func buildByteArrPayload(payload string) []byte {
	buf := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(payload)))
	binary.LittleEndian.PutUint32(buf[4+len(payload):], crc32.ChecksumIEEE([]byte(payload)))

	copy(buf[4:], payload)

	return buf
}

func recordPayload(record Record) string {
	return fmt.Sprintf("%d %s %s\n", record.Operation, record.Key, record.Value)
}
