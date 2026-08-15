// Package wal holds all the wal related logic
package wal

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
)

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

func (wal *Wal) Append(log string) error {
	buf := buildByteArrPayload(log)

	_, err := wal.file.Write(buf)
	if err != nil {
		return err
	}

	//   TODO: we should not fsync on every write, this is an inefficient approach
	err = wal.file.Sync()
	if err != nil {
		return err
	}

	return nil
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

		fn(payload)

	}

	return nil
}

func buildByteArrPayload(log string) []byte {
	payload := []byte(log)
	buf := make([]byte, 4+len(payload))

	binary.LittleEndian.PutUint32(buf[:4], uint32(len(payload)))

	copy(buf[4:], payload)

	return buf
}
