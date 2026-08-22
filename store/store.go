// Package store acts as an storage engine
package store

import (
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

	return &Store{
		wal:        walFile,
		data:       make(map[string]string),
		writeQueue: make(chan string),
		batchChan:  make(chan []string),
	}, nil
}

// Replay replays the store's WAL through fn.
func (st *Store) Replay(fn func([]byte)) error {
	return st.wal.Replay(fn)
}

// Append writes an entry to the store's WAL.
func (st *Store) Append(entry string) error {
	return st.wal.Append(entry)
}

// Fsync makes all entries currently written to the WAL durable.
func (st *Store) Fsync() error {
	return st.wal.Fsync()
}

// Close closes the store and its WAL.
func (st *Store) Close() error {
	return st.wal.Close()
}

// Write function will keep running indefinitely
func (st *Store) Write(data string) {
	batch := make([]string, 0, st.batchSize)

	go st.process()

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
			st.wal.Append(entry)
		}

		err := st.wal.Fsync()
		if err != nil {
			return err
		}
	}
}
