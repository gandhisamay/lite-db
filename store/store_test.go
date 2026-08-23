package store

import (
	"sync"
	"testing"
	"time"

	"github.com/gandhisamay/lite-db/memtable"
	"github.com/gandhisamay/lite-db/wal"
)

type fakeWAL struct {
	mu      sync.Mutex
	entries []wal.Record
	fsyncs  int
	fsyncCh chan struct{}
}

func (w *fakeWAL) Append(entry wal.Record) error {
	w.mu.Lock()
	w.entries = append(w.entries, entry)
	w.mu.Unlock()
	return nil
}

func (w *fakeWAL) Fsync() error {
	w.mu.Lock()
	w.fsyncs++
	w.mu.Unlock()
	w.fsyncCh <- struct{}{}
	return nil
}

func (w *fakeWAL) Replay(func([]byte)) error { return nil }
func (w *fakeWAL) Close() error              { return nil }

func (w *fakeWAL) stats() (int, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.entries), w.fsyncs
}

func TestBatchOfFiveIsFsyncedOnce(t *testing.T) {
	fake := &fakeWAL{fsyncCh: make(chan struct{}, 1)}
	st := &Store{
		wal:       fake,
		writeChan: make(chan WriteRequest),
		batchChan: make(chan []WriteRequest),
		mem:       memtable.NewMemtable(),
		batchSize: 5,
	}

	processDone := make(chan struct{})
	go func() {
		st.process()
		close(processDone)
	}()
	go st.write()

	for i := range 5 {
		st.writeChan <- WriteRequest{
			Operation: OpSet,
			Key:       string(rune('a' + i)),
			Value:     "value",
		}
	}

	select {
	case <-fake.fsyncCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for group commit fsync")
	}

	entries, fsyncs := fake.stats()
	if entries != 5 {
		t.Fatalf("got %d WAL entries, want 5", entries)
	}
	if fsyncs != 1 {
		t.Fatalf("got %d fsyncs, want 1", fsyncs)
	}

	close(st.writeChan)
	select {
	case <-processDone:
	case <-time.After(time.Second):
		t.Fatal("writer pipeline did not stop")
	}
}
