// Package store acts as an storage engine
package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/gandhisamay/lite-db/memtable"
	"github.com/gandhisamay/lite-db/wal"
)

type Store struct {
	wal       walWriter
	writeChan chan WriteRequest
	batchChan chan []WriteRequest
	mem       *memtable.Memtable
	imm       *memtable.Memtable
	timeout   int
	batchSize int
}

// walWriter is the part of the WAL used by the asynchronous writer.
// Keeping this as an interface makes the writer unit-testable.
type walWriter interface {
	Append(wal.Record) error
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
		mem:       memtable.NewMemtable(),
		imm:       nil,
		writeChan: make(chan WriteRequest),
		batchChan: make(chan []WriteRequest),
		batchSize: 5,
	}

	if err := st.Replay(); err != nil {
		walFile.Close()
		return nil, err
	}

	go st.process()
	go st.write()

	return st, nil
}

// Replay rebuilds the store's active memtable from its WAL.
func (st *Store) Replay() error {
	return st.wal.Replay(st.applyRecord)
}

// Append writes an entry to the store's WAL.
func (st *Store) Append(entry WriteRequest) error {
	st.writeChan <- entry
	return nil
}

func (st *Store) Get(key string) (string, bool) {
	return st.mem.Get(key)
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
	batch := make([]WriteRequest, 0, st.batchSize)

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

				batch = make([]WriteRequest, 0, st.batchSize)
			}

		case <-timer.C:
			if len(batch) > 0 {
				st.batchChan <- batch
				batch = make([]WriteRequest, 0, st.batchSize)
			}
		}
	}
}

func (st *Store) process() error {
	for batch := range st.batchChan {

		for _, entry := range batch {
			err := st.wal.Append(wal.Record{
				Operation: uint8(entry.Operation),
				Key:       entry.Key,
				Value:     entry.Value,
			})
			if err != nil {
				return err
			}
		}

		fmt.Println("Fsyncing")
		err := st.wal.Fsync()
		if err != nil {
			return err
		}

		// now for th entire batch, update the memtable, and if the memtable is full, we create a new memtable
		st.updateMemtables(batch)
	}

	return nil
}

func (st *Store) updateMemtables(batch []WriteRequest) {
	for _, record := range batch {
		st.applyRequest(record)
	}
}

func (st *Store) applyRecord(payload []byte) {
	fields := strings.Fields(string(payload))
	if len(fields) == 0 {
		return
	}
	if len(fields) < 2 {
		return
	}

	request := WriteRequest{Key: fields[1]}
	switch fields[0] {
	case "SET":
		if len(fields) < 3 {
			return
		}
		request.Operation = OpSet
		request.Value = fields[2]
	case "DELETE":
		request.Operation = OpDelete
	default:
		return
	}

	st.applyRequest(request)
}

func (st *Store) applyRequest(request WriteRequest) {
	switch request.Operation {
	case OpSet:
		st.mem.Set(request.Key, request.Value)
	case OpDelete:
		st.mem.Delete(request.Key)
	default:
		return
	}

	if st.mem.Size() >= int(memtable.MaxMemTableSize) {
		st.imm = st.mem
		st.mem = memtable.NewMemtable()
	}
}
