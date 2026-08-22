// Package store acts as an storage engine
package store

import (
	"fmt"
	"time"

	"com.db.beginner/wal"
)

type Store struct {
	// now we have this ready, let's build upon this now
	wal       walWriter
	data      map[string]string
	writeChan chan string
	batchChan chan []string
	timeout   int
	batchSize int
}

// walWriter is the part of the WAL used by the asynchronous writer.
// Keeping this as an interface makes the writer unit-testable.
type walWriter interface {
	Append(string) error
	Fsync() error
	Replay(func([]byte)) error
	Close() error
}

// Open creates a store and opens its write-ahead log.
func Open() (*Store, error) {
	walFile, err := wal.Open()
	if err != nil {
		return nil, err
	}

	st := &Store{
		wal:       walFile,
		data:      make(map[string]string),
		writeChan: make(chan string),
		batchChan: make(chan []string),
		batchSize: 5,
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
	close(st.writeChan)
	return st.wal.Close()
}

// one concurrency bug that still exists is, what happens, if our system accepts
// a write after writeQueue channel is closed, we need to fix that as well
func (st *Store) write() {
	defer close(st.batchChan)
	batch := make([]string, 0, st.batchSize)

	timer := time.NewTimer(time.Hour)
	timer.Stop()

	for {
		select {
		case value, ok := <-st.writeChan:

			if !ok {
				// channel is closed, clear the partial batch
				if len(batch) > 0 {
					st.batchChan <- batch
				}
				return
			}

			if len(batch) == 0 {

				// some boilerplate code related to production timer behaviour
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(time.Millisecond)
			}

			batch = append(batch, value)

			if len(batch) == st.batchSize {
				// process this array, make a cpy
				st.batchChan <- batch

				batch = make([]string, 0, st.batchSize)
			}

		case <-timer.C:
			if len(batch) > 0 {
				st.batchChan <- batch
				batch = make([]string, 0, st.batchSize)
			}
		}
	}
}

func (st *Store) process() error {
	for batch := range st.batchChan {

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

	return nil
}
