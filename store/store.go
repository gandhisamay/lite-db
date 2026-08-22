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
	writeChan chan string
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
		writeChan: make(chan string),
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
	st.writeChan <- entry
	return nil
}

// Close closes the store and its WAL.
func (st *Store) Close() error {
	// TODO: resolve writeChan concurrency bug
	close(st.writeChan)
	return st.wal.Close()
}

// one concurrency bug that still exists is, what happens, if our system accepts
// a write after writeQueue channel is closed, we need to fix that as well
func (st *Store) write() {
	defer close(st.batchChan)
	batch := make([]string, 0, st.batchSize)

	for value := range st.writeChan {

		batch = append(batch, value)

		if len(batch) == st.batchSize {
			// process this array, make a cpy
			st.batchChan <- batch

			batch = make([]string, 0, st.batchSize)
		}

	}

	// this handles the case when write queue is closed
	st.batchChan <- batch
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
