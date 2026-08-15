package database

import (
	"encoding/binary"
	"fmt"
	"os"
)

func buildByteArrPayload(log string) []byte {
	payload := []byte(log)
	buf := make([]byte, 4+len(payload))

	binary.LittleEndian.PutUint32(buf[:4], uint32(len(payload)))

	copy(buf[4:], payload)

	return buf
}

func WriteToWal(log string, file *os.File) error {
	buf := buildByteArrPayload(log)
	fmt.Println(buf)

	_, err := file.Write(buf)
	if err != nil {
		return err
	}

	//   TODO: we should not fsync on every write, this is an inefficient approach
	err = file.Sync()
	if err != nil {
		return err
	}

	return nil
}
