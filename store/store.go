// Package store acts as an storage engine
package store

import (
	"fmt"

	"com.db.beginner/wal"
)

type Store struct {
	// now we have this ready, let's build upon this now
	wal        *wal.Wal
	data       map[string]string
	writeQueue chan string
	batchChan  chan []string
	timeout    int
	batchSize  int
}

// Open creates a store and opens its write-ahead log.
func Open() (*Store, error) {
	walFile, err := wal.Open()
	if err != nil {
		return nil, err
	}

	st := &Store{
		wal:        walFile,
		data:       make(map[string]string),
		writeQueue: make(chan string),
		batchChan:  make(chan []string),
		batchSize:  5,
	}

	go st.process()
	go st.write()

	return st, nil
}

// Replay replays the store's WAL through fn.
func (st *Store) Replay(fn func([]byte)) error {
	return st.wal.Replay(fn)
}

// Append writes an entry to the store's WAL.
func (st *Store) Append(entry string) error {
	st.writeQueue <- entry
	return nil
}

// Close closes the store and its WAL.
func (st *Store) Close() error {
	// before closing this, we must process the batchChan and writeQueue
	// close(st.batchChan)
	// close(st.writeQueue)
	// return st.wal.Close()
	return nil
}

// Write function will keep running indefinitely
// concurrency issues here
// 1. If the batchChan is closed, but we still try to write into it, then we are screwed
// so we must make sure, that the remaining part in the batch is flushed before, we actually
// batch channel is closed, similarly for the write queue, same logic is applicable
func (st *Store) write() {
	batch := make([]string, 0, st.batchSize)

	for value := range st.writeQueue {

		batch = append(batch, value)

		if len(batch) == st.batchSize {
			// process this array, make a cpy
			st.batchChan <- batch

			batch = make([]string, 0, st.batchSize)
		}

	}
}

func (st *Store) process() error {
	for {
		batch := <-st.batchChan

		for _, entry := range batch {
			err := st.wal.Append(entry)
			if err != nil {
				return err
			}
		}

		fmt.Println("Fsyncing")
		err := st.wal.Fsync()
		if err != nil {
			return err
		}
	}
}
